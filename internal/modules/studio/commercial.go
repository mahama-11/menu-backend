package studio

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"menu-service/internal/models"
	"menu-service/internal/platform"

	"gorm.io/gorm"
)

func (s *Service) createChargeIntentForJob(job *models.GenerationJob) error {
	if !s.cfg.BillingEnabled || s.platform == nil || job.Mode == "batch" {
		return nil
	}
	if _, err := s.repo.FindChargeIntentByJobID(job.ID); err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	chargeMode, billableItemCode := s.billableItemForMode(job.Mode)
	intent := &models.StudioChargeIntent{
		ID:               fmt.Sprintf("intent_%d", time.Now().UnixNano()),
		JobID:            job.ID,
		BatchRootID:      job.BatchRootID,
		UserID:           job.UserID,
		OrganizationID:   job.OrganizationID,
		ProductCode:      s.cfg.ProductCode,
		ChargeMode:       chargeMode,
		ResourceType:     s.cfg.ResourceType,
		BillableItemCode: billableItemCode,
		EstimatedUnits:   1,
		FinalUnits:       0,
		ReservationKey:   fmt.Sprintf("studio:reservation:%s", job.ID),
		FinalizationID:   fmt.Sprintf("studio:finalization:%s", job.ID),
		EventID:          fmt.Sprintf("studio:event:%s", job.ID),
		Provider:         job.Provider,
		Status:           "created",
		Metadata: mustEncodeJSON(map[string]any{
			"job_mode":                    job.Mode,
			"requested_variants":          job.RequestedVariants,
			"source_asset_ids":            decodeStringSlice(job.SourceAssetIDs),
			"charge_priority_asset_codes": s.chargePriorityAssetCodes(),
		}),
	}
	reservation, err := s.platform.ReserveResources(platform.ReserveInput{
		ResourceType:       s.cfg.ResourceType,
		BillingSubjectType: "organization",
		BillingSubjectID:   job.OrganizationID,
		BillableItemCode:   billableItemCode,
		ReservationKey:     intent.ReservationKey,
		Units:              1,
		ReferenceID:        intent.ID,
		Metadata:           mustEncodeJSON(map[string]any{"job_id": job.ID}),
	})
	if err != nil {
		intent.Status = "failed_need_reconcile"
		intent.FailureCode = "RESERVE_FAILED"
		intent.FailureMessage = err.Error()
		_ = s.repo.CreateChargeIntent(intent)
		return err
	}
	now := time.Now()
	intent.ReservationID = reservation.ID
	intent.Status = "reserved"
	intent.ReservedAt = &now
	if err := s.repo.CreateChargeIntent(intent); err != nil {
		if _, releaseErr := s.platform.ReleaseReservation(reservation.ID); releaseErr != nil {
			return fmt.Errorf("persist charge intent: %w (reservation release also failed: %v)", err, releaseErr)
		}
		return fmt.Errorf("persist charge intent: %w", err)
	}
	return nil
}

func (s *Service) finalizeChargeIntent(job *models.GenerationJob) error {
	if !s.cfg.BillingEnabled || s.platform == nil || job.Mode == "batch" {
		return nil
	}
	intent, err := s.repo.FindChargeIntentByJobID(job.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if intent.Status == "settled" {
		return nil
	}
	input := platform.FinalizeInput{
		FinalizationID: intent.FinalizationID,
		ReservationID:  intent.ReservationID,
		IngestEventInput: platform.IngestEventInput{
			EventID:            intent.EventID,
			SourceType:         "studio_job",
			SourceID:           job.ID,
			SourceAction:       job.Mode,
			ProductCode:        s.cfg.ProductCode,
			OrgID:              job.OrganizationID,
			UserID:             job.UserID,
			BillableItemCode:   intent.BillableItemCode,
			ChargeGroupID:      firstNonEmpty(job.BatchRootID, job.ID),
			BillingSubjectType: "organization",
			BillingSubjectID:   job.OrganizationID,
			UsageUnits:         1,
			Unit:               "action",
			Dimensions:         mustEncodeJSON(s.billingDimensions(job)),
			OccurredAt:         time.Now().UTC().Format(time.RFC3339),
		},
	}
	result, err := s.platform.FinalizeMetering(input)
	if err != nil {
		intent.Status = "failed_need_reconcile"
		intent.FailureCode = "FINALIZE_FAILED"
		intent.FailureMessage = err.Error()
		return s.repo.SaveChargeIntent(intent)
	}
	now := time.Now()
	intent.FinalUnits = 1
	intent.Status = "settled"
	intent.FinalizedAt = &now
	intent.ProviderJobID = job.ProviderJobID
	if result.Settlement != nil {
		intent.SettlementID = result.Settlement.ID
		intent.Metadata = mergeJSON(intent.Metadata, map[string]any{
			"settlement": map[string]any{
				"id":                result.Settlement.ID,
				"currency":          result.Settlement.Currency,
				"gross_amount":      result.Settlement.GrossAmount,
				"discount_amount":   result.Settlement.DiscountAmount,
				"net_amount":        result.Settlement.NetAmount,
				"quota_consumed":    result.Settlement.QuotaConsumed,
				"credits_consumed":  result.Settlement.CreditsConsumed,
				"wallet_asset_code": result.Settlement.WalletAssetCode,
				"wallet_debited":    result.Settlement.WalletDebited,
			},
		})
	}
	return s.repo.SaveChargeIntent(intent)
}

func (s *Service) releaseChargeIntent(job *models.GenerationJob) error {
	if !s.cfg.BillingEnabled || s.platform == nil || job.Mode == "batch" {
		return nil
	}
	intent, err := s.repo.FindChargeIntentByJobID(job.ID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if intent.Status == "released" || intent.Status == "settled" || intent.ReservationID == "" {
		return nil
	}
	if _, err := s.platform.ReleaseReservation(intent.ReservationID); err != nil {
		intent.Status = "failed_need_reconcile"
		intent.FailureCode = "RELEASE_FAILED"
		intent.FailureMessage = err.Error()
		return s.repo.SaveChargeIntent(intent)
	}
	now := time.Now()
	intent.Status = "released"
	intent.ReleasedAt = &now
	return s.repo.SaveChargeIntent(intent)
}

func (s *Service) billableItemForMode(mode string) (string, string) {
	switch mode {
	case "refinement":
		return mode, s.cfg.RefinementBillableItem
	case "variation":
		return mode, s.cfg.VariationBillableItem
	default:
		return "single", s.cfg.SingleBillableItem
	}
}

func (s *Service) billingDimensions(job *models.GenerationJob) map[string]any {
	var params map[string]any
	if raw := job.ParamsSnapshot; raw != "" {
		_ = json.Unmarshal([]byte(raw), &params)
	}
	if params == nil {
		params = map[string]any{}
	}
	params["provider"] = job.Provider
	params["mode"] = job.Mode
	params["requested_variants"] = job.RequestedVariants
	return params
}
