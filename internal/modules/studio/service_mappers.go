package studio

import (
	"errors"
	"slices"
	"time"

	"menu-service/internal/models"

	"gorm.io/gorm"
)

func (s *Service) mapAsset(item *models.StudioAsset) *AssetSummary {
	sourceURL := item.SourceURL
	previewURL := item.PreviewURL
	if item.StorageKey != "" {
		protectedURL := s.BuildSignedAssetContentURL(item.ID)
		sourceURL = protectedURL
		previewURL = protectedURL
	}
	return &AssetSummary{
		ID:         item.ID,
		AssetType:  item.AssetType,
		SourceType: item.SourceType,
		Status:     item.Status,
		FileName:   item.FileName,
		MimeType:   item.MimeType,
		StorageKey: item.StorageKey,
		SourceURL:  sourceURL,
		PreviewURL: previewURL,
		Width:      item.Width,
		Height:     item.Height,
		FileSize:   item.FileSize,
		Metadata:   decodeMap(item.Metadata),
		CreatedAt:  item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func mapStylePreset(item *models.StylePreset) *StylePresetSummary {
	return &StylePresetSummary{
		StyleID:          item.ID,
		Name:             item.Name,
		Description:      item.Description,
		Visibility:       item.Visibility,
		Status:           item.Status,
		Version:          item.Version,
		ParentStyleID:    item.ParentStyleID,
		PreviewAssetID:   item.PreviewAssetID,
		Dimensions:       decodeDimensions(item.DimensionsJSON),
		Tags:             decodeStringSlice(item.TagsJSON),
		ExecutionProfile: decodeExecutionProfile(item.ExecutionProfile),
		Metadata:         decodeMap(item.Metadata),
		CreatedByUserID:  item.CreatedByUserID,
		CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *Service) mapGenerationJob(item *models.GenerationJob, variants []models.GenerationVariant, childJobs []models.GenerationJob, chargeIntent *models.StudioChargeIntent) *GenerationJobSummary {
	out := &GenerationJobSummary{
		JobID:             item.ID,
		UserID:            item.UserID,
		Mode:              item.Mode,
		Status:            item.Status,
		Stage:             item.Stage,
		StageMessage:      item.StageMessage,
		Provider:          item.Provider,
		ProviderJobID:     item.ProviderJobID,
		IdempotencyKey:    derefString(item.IdempotencyKey),
		StylePresetID:     item.StylePresetID,
		ParentJobID:       item.ParentJobID,
		BatchRootID:       item.BatchRootID,
		ParentVariantID:   item.ParentVariantID,
		SourceAssetIDs:    decodeStringSlice(item.SourceAssetIDs),
		RequestedVariants: item.RequestedVariants,
		ChildJobCount:     item.ChildJobCount,
		Progress:          item.Progress,
		QueuePosition:     item.QueuePosition,
		EtaSeconds:        item.EtaSeconds,
		ErrorCode:         item.ErrorCode,
		ErrorMessage:      item.ErrorMessage,
		SelectedVariantID: item.SelectedVariantID,
		PromptSnapshot:    decodeExecutionProfile(item.PromptSnapshot),
		CreativeSource:    decodeCreativeSource(item.Metadata),
		ParamsSnapshot:    decodeMap(item.ParamsSnapshot),
		Metadata:          decodeMap(item.Metadata),
		CreatedAt:         item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         item.UpdatedAt.UTC().Format(time.RFC3339),
		Variants:          make([]GenerationVariantSummary, 0, len(variants)),
		AttemptCount:      item.AttemptCount,
		MaxAttempts:       item.MaxAttempts,
		ChildJobs:         make([]GenerationJobSummaryLite, 0, len(childJobs)),
	}
	out.Charge = s.mapGenerationJobCharge(item, chargeIntent)
	if item.CompletedAt != nil {
		value := item.CompletedAt.UTC().Format(time.RFC3339)
		out.CompletedAt = &value
	}
	if item.NextRetryAt != nil {
		value := item.NextRetryAt.UTC().Format(time.RFC3339)
		out.NextRetryAt = &value
	}
	if item.TimeoutAt != nil {
		value := item.TimeoutAt.UTC().Format(time.RFC3339)
		out.TimeoutAt = &value
	}
	if item.HeartbeatAt != nil {
		value := item.HeartbeatAt.UTC().Format(time.RFC3339)
		out.HeartbeatAt = &value
	}
	if item.CanceledAt != nil {
		value := item.CanceledAt.UTC().Format(time.RFC3339)
		out.CanceledAt = &value
	}
	for _, variant := range variants {
		var assetSummary *AssetSummary
		if variant.AssetID != "" {
			if asset, err := s.repo.FindAssetByID(item.OrganizationID, variant.AssetID); err == nil {
				assetSummary = s.mapAsset(asset)
			}
		}
		out.Variants = append(out.Variants, GenerationVariantSummary{
			VariantID:       variant.ID,
			AssetID:         variant.AssetID,
			Asset:           assetSummary,
			PreviewURL:      stringMapValue(decodeMap(variant.Metadata), "preview_url"),
			ParentVariantID: variant.ParentVariantID,
			Status:          variant.Status,
			Index:           variant.VariantIndex,
			Score:           variant.Score,
			IsSelected:      variant.IsSelected,
			Metadata:        decodeMap(variant.Metadata),
		})
	}
	for _, child := range childJobs {
		out.ChildJobs = append(out.ChildJobs, GenerationJobSummaryLite{
			JobID:        child.ID,
			Status:       child.Status,
			Stage:        child.Stage,
			StageMessage: child.StageMessage,
			Progress:     child.Progress,
			ErrorCode:    child.ErrorCode,
			ErrorMessage: child.ErrorMessage,
			Mode:         child.Mode,
			EtaSeconds:   child.EtaSeconds,
		})
	}
	return out
}

func (s *Service) mapGenerationJobCharge(job *models.GenerationJob, intent *models.StudioChargeIntent) *GenerationJobChargeSummary {
	summary := &GenerationJobChargeSummary{
		BillingEnabled:           true,
		Billable:                 job.Mode != "batch",
		ChargePriorityAssetCodes: s.chargePriorityAssetCodes(),
	}
	if !summary.Billable {
		return summary
	}
	chargeMode, billableItemCode := s.billableItemForMode(job.Mode)
	summary.ChargeMode = chargeMode
	summary.ResourceType = s.cfg.ResourceType
	summary.BillableItemCode = billableItemCode
	if intent == nil {
		return summary
	}
	summary.ChargeMode = firstNonEmpty(intent.ChargeMode, summary.ChargeMode)
	summary.ResourceType = firstNonEmpty(intent.ResourceType, summary.ResourceType)
	summary.BillableItemCode = firstNonEmpty(intent.BillableItemCode, summary.BillableItemCode)
	summary.Status = intent.Status
	summary.FailureCode = intent.FailureCode
	summary.FailureMessage = intent.FailureMessage
	summary.ReservationID = intent.ReservationID
	summary.SettlementID = intent.SettlementID
	summary.EstimatedUnits = intent.EstimatedUnits
	summary.FinalUnits = intent.FinalUnits

	metadata := decodeMap(intent.Metadata)
	if settlement, ok := metadata["settlement"].(map[string]any); ok {
		summary.Currency = stringMapValue(settlement, "currency")
		summary.WalletAssetCode = stringMapValue(settlement, "wallet_asset_code")
		summary.QuotaConsumed = int64MapValue(settlement, "quota_consumed")
		summary.CreditsConsumed = int64MapValue(settlement, "credits_consumed")
		summary.WalletDebited = int64MapValue(settlement, "wallet_debited")
		summary.GrossAmount = int64MapValue(settlement, "gross_amount")
		summary.DiscountAmount = int64MapValue(settlement, "discount_amount")
		summary.NetAmount = int64MapValue(settlement, "net_amount")
	}
	return summary
}

func (s *Service) chargePriorityAssetCodes() []string {
	out := make([]string, 0, 2)
	for _, code := range []string{
		s.appCfg.RewardAssetCode,
		s.appCfg.CreditsAssetCode,
	} {
		if code == "" {
			continue
		}
		if !slices.Contains(out, code) {
			out = append(out, code)
		}
	}
	return out
}

func (s *Service) mapAssetLibraryItem(userID, orgID string, item *models.StudioAsset, sharePost *models.SharePost) (*AssetLibraryItem, error) {
	_ = userID
	out := &AssetLibraryItem{
		Asset:      *s.mapAsset(item),
		OriginRole: item.AssetType,
		CanRefine:  item.Status == "ready" && (item.AssetType == "source" || item.AssetType == "generated"),
		CanShare:   item.Status == "ready" && item.AssetType == "generated",
	}
	if variant, err := s.repo.FindGenerationVariantByAssetID(item.ID); err == nil {
		out.VariantID = variant.ID
		out.ProducedByJobID = variant.JobID
		if job, jobErr := s.repo.FindGenerationJobByID(orgID, variant.JobID); jobErr == nil {
			out.LatestJob = mapJobLite(job)
		} else if !errors.Is(jobErr, gorm.ErrRecordNotFound) {
			return nil, jobErr
		}
		if variant.IsSelected {
			out.OriginRole = "selected_result"
		} else {
			out.OriginRole = "generated_result"
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	} else if job, jobErr := s.repo.FindLatestJobUsingAsset(orgID, item.ID); jobErr == nil {
		out.LatestJob = mapJobLite(job)
		out.ProducedByJobID = job.ID
		out.OriginRole = "source_asset"
	} else if !errors.Is(jobErr, gorm.ErrRecordNotFound) {
		return nil, jobErr
	}
	if sharePost != nil {
		out.Share = mapSharePostSummary(sharePost)
	}
	return out, nil
}

func (s *Service) mapJobHistoryItem(orgID string, job *GenerationJobSummary) (*JobHistoryItem, error) {
	out := &JobHistoryItem{Job: job}
	for _, assetID := range job.SourceAssetIDs {
		if assetID == "" {
			continue
		}
		item, err := s.repo.FindAssetByID(orgID, assetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		out.SourceAssets = append(out.SourceAssets, *s.mapAsset(item))
	}
	for _, variant := range job.Variants {
		if variant.AssetID == "" {
			continue
		}
		asset, err := s.repo.FindAssetByID(orgID, variant.AssetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		mapped := s.mapAsset(asset)
		out.ResultAssets = append(out.ResultAssets, *mapped)
		if variant.IsSelected || variant.VariantID == job.SelectedVariantID {
			out.SelectedAsset = mapped
		}
	}
	return out, nil
}

func mapJobLite(item *models.GenerationJob) *GenerationJobSummaryLite {
	if item == nil {
		return nil
	}
	return &GenerationJobSummaryLite{
		JobID:        item.ID,
		Status:       item.Status,
		Stage:        item.Stage,
		StageMessage: item.StageMessage,
		Progress:     item.Progress,
		ErrorCode:    item.ErrorCode,
		ErrorMessage: item.ErrorMessage,
		Mode:         item.Mode,
		EtaSeconds:   item.EtaSeconds,
	}
}

func mapSharePostSummary(item *models.SharePost) *SharePostSummary {
	if item == nil {
		return nil
	}
	out := &SharePostSummary{
		ShareID:       item.ID,
		Status:        item.Status,
		Visibility:    item.Visibility,
		ShareURL:      item.ShareURL,
		ViewCount:     item.ViewCount,
		LikeCount:     item.LikeCount,
		FavoriteCount: item.FavoriteCount,
		Metadata:      decodeMap(item.Metadata),
	}
	if item.PublishedAt != nil {
		value := item.PublishedAt.UTC().Format(time.RFC3339)
		out.PublishedAt = &value
	}
	return out
}
