package channel

import (
	"errors"
	"sort"

	"menu-service/internal/platform"
)

const menuProductCode = "menu"

type Service struct {
	platform *platform.Client
}

type PartnerSummary struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	PartnerType string `json:"partner_type"`
	Status      string `json:"status"`
	RiskLevel   string `json:"risk_level"`
}

type ProgramSummary struct {
	ID          string `json:"id"`
	ProgramCode string `json:"program_code"`
	Name        string `json:"name"`
	ProgramType string `json:"program_type"`
	Status      string `json:"status"`
}

type BindingView struct {
	Binding platform.ChannelBinding `json:"binding"`
	Partner *PartnerSummary         `json:"partner,omitempty"`
	Program *ProgramSummary         `json:"program,omitempty"`
}

type CommissionView struct {
	Partner *PartnerSummary                  `json:"partner,omitempty"`
	Program *ProgramSummary                  `json:"program,omitempty"`
	Ledger  platform.ChannelCommissionLedger `json:"ledger"`
}

type SettlementView struct {
	Partner *PartnerSummary                  `json:"partner,omitempty"`
	Program *ProgramSummary                  `json:"program,omitempty"`
	Batch   *platform.ChannelSettlementBatch `json:"batch,omitempty"`
	Item    platform.ChannelSettlementItem   `json:"item"`
}

type AdjustmentView struct {
	Partner *PartnerSummary                            `json:"partner,omitempty"`
	Program *ProgramSummary                            `json:"program,omitempty"`
	Item    platform.ChannelCommissionAdjustmentLedger `json:"item"`
}

type PreviewInput struct {
	UserID                    string `json:"user_id,omitempty"`
	PolicyVersionID           string `json:"policy_version_id,omitempty"`
	RegionCode                string `json:"region_code,omitempty"`
	PartnerTier               string `json:"partner_tier,omitempty"`
	BillableItemCode          string `json:"billable_item_code,omitempty"`
	AppliesTo                 string `json:"applies_to,omitempty"`
	SourceChargeID            string `json:"source_charge_id,omitempty"`
	SourceOrderID             string `json:"source_order_id,omitempty"`
	Currency                  string `json:"currency,omitempty"`
	GrossAmount               int64  `json:"gross_amount,omitempty"`
	DiscountAmount            int64  `json:"discount_amount,omitempty"`
	PaidAmount                int64  `json:"paid_amount,omitempty"`
	RefundedAmount            int64  `json:"refunded_amount,omitempty"`
	NetCollectedAmount        int64  `json:"net_collected_amount,omitempty"`
	PaymentFeeAmount          int64  `json:"payment_fee_amount,omitempty"`
	TaxAmount                 int64  `json:"tax_amount,omitempty"`
	ServiceDeliveryCostAmount int64  `json:"service_delivery_cost_amount,omitempty"`
	InfraVariableCostAmount   int64  `json:"infra_variable_cost_amount,omitempty"`
	RiskReserveAmount         int64  `json:"risk_reserve_amount,omitempty"`
	ManualAdjustmentAmount    int64  `json:"manual_adjustment_amount,omitempty"`
	OccurredAt                string `json:"occurred_at,omitempty"`
	CommissionRecognitionAt   string `json:"commission_recognition_at,omitempty"`
	SnapshotBasis             string `json:"snapshot_basis,omitempty"`
	Dimensions                string `json:"dimensions,omitempty"`
	Metadata                  string `json:"metadata,omitempty"`
}

type PreviewView struct {
	Partner *PartnerSummary                          `json:"partner,omitempty"`
	Program *ProgramSummary                          `json:"program,omitempty"`
	Result  *platform.ChannelPolicyResolutionPreview `json:"result"`
}

type CreateAdjustmentInput struct {
	ChannelPartnerID         string `json:"channel_partner_id" binding:"required"`
	ChannelProgramID         string `json:"channel_program_id" binding:"required"`
	SourceCommissionLedgerID string `json:"source_commission_ledger_id,omitempty"`
	SourceProfitSnapshotID   string `json:"source_profit_snapshot_id,omitempty"`
	AdjustmentType           string `json:"adjustment_type" binding:"required"`
	Currency                 string `json:"currency,omitempty"`
	AdjustmentAmount         int64  `json:"adjustment_amount" binding:"required"`
	ReasonCode               string `json:"reason_code" binding:"required"`
	EffectiveAt              string `json:"effective_at,omitempty"`
	OperatorID               string `json:"operator_id,omitempty"`
	Metadata                 string `json:"metadata,omitempty"`
}

type Overview struct {
	Partners           []PartnerSummary `json:"partners"`
	CurrentBindings    []BindingView    `json:"current_bindings"`
	TotalCommission    int64            `json:"total_commission"`
	PendingCommission  int64            `json:"pending_commission"`
	EarnedCommission   int64            `json:"earned_commission"`
	SettledCommission  int64            `json:"settled_commission"`
	ReversedCommission int64            `json:"reversed_commission"`
	PendingClawback    int64            `json:"pending_clawback"`
	AppliedClawback    int64            `json:"applied_clawback"`
	SettlementCount    int              `json:"settlement_count"`
	RecentSettlements  []SettlementView `json:"recent_settlements"`
}

func NewService(platformClient *platform.Client) *Service {
	return &Service{platform: platformClient}
}

func (s *Service) CurrentBinding(orgID string) ([]BindingView, error) {
	bindings, err := s.platform.ListChannelBindings(menuProductCode, orgID, "")
	if err != nil {
		return nil, err
	}
	partners, programs, err := s.loadPartnerProgramMaps()
	if err != nil {
		return nil, err
	}
	sort.Slice(bindings, func(i, j int) bool {
		return bindings[i].CreatedAt.After(bindings[j].CreatedAt)
	})
	out := make([]BindingView, 0, len(bindings))
	for _, binding := range bindings {
		out = append(out, BindingView{
			Binding: binding,
			Partner: partners[binding.ChannelPartnerID],
			Program: programs[binding.ChannelProgramID],
		})
	}
	return out, nil
}

func (s *Service) Overview(orgID string) (*Overview, error) {
	partners, err := s.resolvePartnersForOrg(orgID)
	if err != nil {
		return nil, err
	}
	currentBindings, err := s.CurrentBinding(orgID)
	if err != nil {
		return nil, err
	}
	if len(partners) == 0 {
		return &Overview{CurrentBindings: currentBindings}, nil
	}

	partnerViews := make([]PartnerSummary, 0, len(partners))
	partnerMap := make(map[string]*PartnerSummary, len(partners))
	for _, item := range partners {
		summary := mapPartner(item)
		partnerViews = append(partnerViews, summary)
		partnerMap[item.ID] = &summary
	}
	sort.Slice(partnerViews, func(i, j int) bool {
		return partnerViews[i].Code < partnerViews[j].Code
	})

	commissions, err := s.listPartnerCommissions(partners)
	if err != nil {
		return nil, err
	}
	clawbacks, err := s.listPartnerClawbacks(partners)
	if err != nil {
		return nil, err
	}
	settlements, err := s.ListSettlements(orgID, "")
	if err != nil {
		return nil, err
	}

	out := &Overview{
		Partners:        partnerViews,
		CurrentBindings: currentBindings,
		SettlementCount: len(settlements),
	}
	for _, item := range commissions {
		out.TotalCommission += item.CommissionAmount
		switch item.Status {
		case "pending":
			out.PendingCommission += item.CommissionAmount
		case "earned", "settlement_in_progress":
			out.EarnedCommission += item.CommissionAmount
		case "settled":
			out.SettledCommission += item.CommissionAmount
		case "reversed", "void":
			out.ReversedCommission += item.CommissionAmount
		}
	}
	for _, item := range clawbacks {
		switch item.Status {
		case "pending":
			out.PendingClawback += item.ClawbackAmount
		case "applied":
			out.AppliedClawback += item.ClawbackAmount
		}
	}
	if len(settlements) > 5 {
		out.RecentSettlements = settlements[:5]
	} else {
		out.RecentSettlements = settlements
	}
	_ = partnerMap
	return out, nil
}

func (s *Service) ListCommissions(orgID, status string) ([]CommissionView, error) {
	partners, err := s.resolvePartnersForOrg(orgID)
	if err != nil {
		return nil, err
	}
	if len(partners) == 0 {
		return []CommissionView{}, nil
	}
	programs, err := s.listProgramsMap()
	if err != nil {
		return nil, err
	}
	out := make([]CommissionView, 0)
	for _, partner := range partners {
		items, err := s.platform.ListChannelCommissions(menuProductCode, partner.ID, status)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			out = append(out, CommissionView{
				Partner: ptr(mapPartner(partner)),
				Program: programs[item.ChannelProgramID],
				Ledger:  item,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Ledger.CreatedAt.After(out[j].Ledger.CreatedAt)
	})
	return out, nil
}

func (s *Service) ListSettlements(orgID, status string) ([]SettlementView, error) {
	partners, err := s.resolvePartnersForOrg(orgID)
	if err != nil {
		return nil, err
	}
	if len(partners) == 0 {
		return []SettlementView{}, nil
	}
	programs, err := s.listProgramsMap()
	if err != nil {
		return nil, err
	}
	out := make([]SettlementView, 0)
	for _, partner := range partners {
		items, err := s.platform.ListChannelSettlementItems("", partner.ID, status)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			var batch *platform.ChannelSettlementBatch
			detail, detailErr := s.platform.GetChannelSettlementBatch(item.SettlementBatchID)
			if detailErr == nil {
				batch = &detail.Batch
			}
			var program *ProgramSummary
			if batch != nil {
				program = programs[batch.ChannelProgramID]
			}
			out = append(out, SettlementView{
				Partner: ptr(mapPartner(partner)),
				Program: program,
				Batch:   batch,
				Item:    item,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		left := out[i].Item.CreatedAt
		right := out[j].Item.CreatedAt
		if out[i].Batch != nil && out[i].Batch.PeriodEnd.After(left) {
			left = out[i].Batch.PeriodEnd
		}
		if out[j].Batch != nil && out[j].Batch.PeriodEnd.After(right) {
			right = out[j].Batch.PeriodEnd
		}
		return left.After(right)
	})
	return out, nil
}

func (s *Service) ListAdjustments(orgID, status string) ([]AdjustmentView, error) {
	partners, err := s.resolvePartnersForOrg(orgID)
	if err != nil {
		return nil, err
	}
	if len(partners) == 0 {
		return []AdjustmentView{}, nil
	}
	programs, err := s.listProgramsMap()
	if err != nil {
		return nil, err
	}
	out := make([]AdjustmentView, 0)
	for _, partner := range partners {
		items, listErr := s.platform.ListChannelAdjustments(menuProductCode, partner.ID, status)
		if listErr != nil {
			return nil, listErr
		}
		for _, item := range items {
			out = append(out, AdjustmentView{
				Partner: ptr(mapPartner(partner)),
				Program: programs[item.ChannelProgramID],
				Item:    item,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Item.CreatedAt.After(out[j].Item.CreatedAt)
	})
	return out, nil
}

func (s *Service) CreateAdjustment(orgID, operatorID string, input CreateAdjustmentInput) (*AdjustmentView, error) {
	partners, err := s.resolvePartnersForOrg(orgID)
	if err != nil {
		return nil, err
	}
	var matchedPartner *platform.ChannelPartner
	for _, item := range partners {
		if item.ID == input.ChannelPartnerID {
			copyItem := item
			matchedPartner = &copyItem
			break
		}
	}
	if matchedPartner == nil {
		return nil, errors.New("channel partner is not linked to current organization")
	}
	item, err := s.platform.CreateChannelAdjustment(platform.CreateChannelCommissionAdjustmentInput{
		ProductCode:              menuProductCode,
		ChannelPartnerID:         input.ChannelPartnerID,
		ChannelProgramID:         input.ChannelProgramID,
		SourceCommissionLedgerID: input.SourceCommissionLedgerID,
		SourceProfitSnapshotID:   input.SourceProfitSnapshotID,
		AdjustmentType:           input.AdjustmentType,
		Currency:                 input.Currency,
		AdjustmentAmount:         input.AdjustmentAmount,
		ReasonCode:               input.ReasonCode,
		EffectiveAt:              input.EffectiveAt,
		OperatorID:               firstNonEmpty(input.OperatorID, operatorID),
		Metadata:                 input.Metadata,
	})
	if err != nil {
		return nil, err
	}
	programs, err := s.listProgramsMap()
	if err != nil {
		return nil, err
	}
	view := &AdjustmentView{
		Partner: ptr(mapPartner(*matchedPartner)),
		Program: programs[item.ChannelProgramID],
		Item:    *item,
	}
	return view, nil
}

func (s *Service) Preview(orgID string, input PreviewInput) (*PreviewView, error) {
	result, err := s.platform.PreviewChannelPolicyResolution(platform.RecordChannelChargeInput{
		ProductCode:               menuProductCode,
		OrgID:                     orgID,
		UserID:                    input.UserID,
		PolicyVersionID:           input.PolicyVersionID,
		RegionCode:                input.RegionCode,
		PartnerTier:               input.PartnerTier,
		BillableItemCode:          input.BillableItemCode,
		AppliesTo:                 firstNonEmpty(input.AppliesTo, "usage_charge"),
		SourceChargeID:            firstNonEmpty(input.SourceChargeID, "preview-charge"),
		SourceOrderID:             input.SourceOrderID,
		Currency:                  input.Currency,
		GrossAmount:               input.GrossAmount,
		DiscountAmount:            input.DiscountAmount,
		PaidAmount:                input.PaidAmount,
		RefundedAmount:            input.RefundedAmount,
		NetCollectedAmount:        input.NetCollectedAmount,
		PaymentFeeAmount:          input.PaymentFeeAmount,
		TaxAmount:                 input.TaxAmount,
		ServiceDeliveryCostAmount: input.ServiceDeliveryCostAmount,
		InfraVariableCostAmount:   input.InfraVariableCostAmount,
		RiskReserveAmount:         input.RiskReserveAmount,
		ManualAdjustmentAmount:    input.ManualAdjustmentAmount,
		OccurredAt:                input.OccurredAt,
		CommissionRecognitionAt:   input.CommissionRecognitionAt,
		SnapshotBasis:             input.SnapshotBasis,
		Dimensions:                input.Dimensions,
		Metadata:                  input.Metadata,
	})
	if err != nil {
		return nil, err
	}
	partnerMap, programMap, err := s.loadPartnerProgramMaps()
	if err != nil {
		return nil, err
	}
	view := &PreviewView{Result: result}
	if result.ChannelID != "" {
		view.Partner = partnerMap[result.ChannelID]
	}
	if result.ChannelProgramID != "" {
		view.Program = programMap[result.ChannelProgramID]
	}
	return view, nil
}

func (s *Service) resolvePartnersForOrg(orgID string) ([]platform.ChannelPartner, error) {
	items, err := s.platform.ListChannelPartners("")
	if err != nil {
		return nil, err
	}
	out := make([]platform.ChannelPartner, 0)
	for _, item := range items {
		if item.SettlementSubjectType == "organization" && item.SettlementSubjectID == orgID {
			out = append(out, item)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Status == out[j].Status {
			return out[i].CreatedAt.After(out[j].CreatedAt)
		}
		return out[i].Status == "active"
	})
	return out, nil
}

func (s *Service) loadPartnerProgramMaps() (map[string]*PartnerSummary, map[string]*ProgramSummary, error) {
	partners, err := s.platform.ListChannelPartners("")
	if err != nil {
		return nil, nil, err
	}
	programs, err := s.platform.ListChannelPrograms(menuProductCode, "")
	if err != nil {
		return nil, nil, err
	}
	partnerMap := make(map[string]*PartnerSummary, len(partners))
	for _, item := range partners {
		summary := mapPartner(item)
		partnerMap[item.ID] = &summary
	}
	programMap := make(map[string]*ProgramSummary, len(programs))
	for _, item := range programs {
		summary := mapProgram(item)
		programMap[item.ID] = &summary
	}
	return partnerMap, programMap, nil
}

func (s *Service) listProgramsMap() (map[string]*ProgramSummary, error) {
	items, err := s.platform.ListChannelPrograms(menuProductCode, "")
	if err != nil {
		return nil, err
	}
	out := make(map[string]*ProgramSummary, len(items))
	for _, item := range items {
		summary := mapProgram(item)
		out[item.ID] = &summary
	}
	return out, nil
}

func (s *Service) listPartnerCommissions(partners []platform.ChannelPartner) ([]platform.ChannelCommissionLedger, error) {
	out := make([]platform.ChannelCommissionLedger, 0)
	for _, partner := range partners {
		items, err := s.platform.ListChannelCommissions(menuProductCode, partner.ID, "")
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	return out, nil
}

func (s *Service) listPartnerClawbacks(partners []platform.ChannelPartner) ([]platform.ChannelClawbackLedger, error) {
	out := make([]platform.ChannelClawbackLedger, 0)
	for _, partner := range partners {
		items, err := s.platform.ListChannelClawbacks(menuProductCode, partner.ID, "")
		if err != nil {
			return nil, err
		}
		out = append(out, items...)
	}
	return out, nil
}

func mapPartner(item platform.ChannelPartner) PartnerSummary {
	return PartnerSummary{
		ID:          item.ID,
		Code:        item.Code,
		Name:        item.Name,
		PartnerType: item.PartnerType,
		Status:      item.Status,
		RiskLevel:   item.RiskLevel,
	}
}

func mapProgram(item platform.ChannelProgram) ProgramSummary {
	return ProgramSummary{
		ID:          item.ID,
		ProgramCode: item.ProgramCode,
		Name:        item.Name,
		ProgramType: item.ProgramType,
		Status:      item.Status,
	}
}

func ptr[T any](value T) *T {
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
