package referral

import (
	"sort"

	"menu-service/internal/config"
	"menu-service/internal/platform"
	"menu-service/pkg/logger"
)

const menuProductCode = "menu"

type Service struct {
	platform           *platform.Client
	creditsAssetCode   string
	rewardAssetCode    string
	allowanceAssetCode string
}

var defaultAssetDefinitions = func(creditsAssetCode, rewardAssetCode, allowanceAssetCode string) []platform.CreateAssetDefinitionInput {
	return []platform.CreateAssetDefinitionInput{
		{
			AssetCode:     creditsAssetCode,
			ProductCode:   menuProductCode,
			AssetType:     "cash_balance",
			LifecycleType: "permanent",
			Status:        "active",
			Description:   "Menu permanent wallet balance",
			Metadata:      `{"seeded_by":"menu_bootstrap","seed_version":"v1","asset_role":"primary_balance"}`,
		},
		{
			AssetCode:         rewardAssetCode,
			ProductCode:       menuProductCode,
			AssetType:         "reward_credit",
			LifecycleType:     "expiring",
			DefaultExpireDays: 30,
			Status:            "active",
			Description:       "Menu promotional reward credit",
			Metadata:          `{"seeded_by":"menu_bootstrap","seed_version":"v1","asset_role":"promo_reward"}`,
		},
		{
			AssetCode:     allowanceAssetCode,
			ProductCode:   menuProductCode,
			AssetType:     "subscription_allowance",
			LifecycleType: "cycle_reset",
			ResetCycle:    "monthly",
			Status:        "active",
			Description:   "Menu monthly subscription allowance",
			Metadata:      `{"seeded_by":"menu_bootstrap","seed_version":"v1","asset_role":"monthly_allowance"}`,
		},
	}
}

func defaultReferralPrograms(rewardAssetCode string) []platform.CreateReferralProgramInput {
	return []platform.CreateReferralProgramInput{
		{
			ProductCode:           menuProductCode,
			ProgramCode:           "menu_signup_default",
			Name:                  "Menu Signup Referral",
			TriggerType:           "signup",
			CommissionPolicy:      "fixed_amount",
			CommissionCurrency:    rewardAssetCode,
			CommissionFixedAmount: 20,
			SettlementDelayDays:   0,
			AllowRepeat:           false,
			Status:                "active",
			Metadata:              `{"seeded_by":"menu_bootstrap","seed_version":"v1"}`,
		},
	}
}

type Overview struct {
	Programs             []platform.ReferralProgram    `json:"programs"`
	Codes                []platform.ReferralCode       `json:"codes"`
	Conversions          []platform.ReferralConversion `json:"conversions"`
	Commissions          []platform.CommissionLedger   `json:"commissions"`
	TotalConversions     int                           `json:"total_conversions"`
	TrackedConversions   int                           `json:"tracked_conversions"`
	EarnedConversions    int                           `json:"earned_conversions"`
	ReversedConversions  int                           `json:"reversed_conversions"`
	TotalCommission      int64                         `json:"total_commission"`
	EarnedCommission     int64                         `json:"earned_commission"`
	PendingCommission    int64                         `json:"pending_commission"`
	ReversedCommission   int64                         `json:"reversed_commission"`
	RedeemableCommission int64                         `json:"redeemable_commission"`
	RedeemedCommission   int64                         `json:"redeemed_commission"`
}

type CreateCodeInput struct {
	ProgramCode string `json:"program_code" binding:"required"`
	Code        string `json:"code"`
	Metadata    string `json:"metadata"`
}

type RedeemInput struct {
	CommissionIDs []string `json:"commission_ids"`
	Metadata      string   `json:"metadata"`
}

func NewService(platformClient *platform.Client, appCfg config.AppConfig) *Service {
	return &Service{
		platform:           platformClient,
		creditsAssetCode:   appCfg.CreditsAssetCode,
		rewardAssetCode:    appCfg.RewardAssetCode,
		allowanceAssetCode: appCfg.AllowanceAssetCode,
	}
}

func (s *Service) Bootstrap() error {
	if err := s.bootstrapAssetDefinitions(); err != nil {
		return err
	}
	existing, err := s.platform.ListReferralPrograms(menuProductCode, "")
	if err != nil {
		return err
	}
	byCode := make(map[string]platform.ReferralProgram, len(existing))
	for _, item := range existing {
		byCode[item.ProgramCode] = item
	}
	for _, program := range defaultReferralPrograms(s.rewardAssetCode) {
		if item, ok := byCode[program.ProgramCode]; ok {
			logger.With("program_code", item.ProgramCode, "status", item.Status).Info("referral.bootstrap.program_exists")
			continue
		}
		created, createErr := s.platform.CreateReferralProgram(program)
		if createErr != nil {
			return createErr
		}
		logger.With("program_code", created.ProgramCode, "trigger_type", created.TriggerType).Info("referral.bootstrap.program_created")
	}
	return nil
}

func (s *Service) ListPrograms(status string) ([]platform.ReferralProgram, error) {
	items, err := s.platform.ListReferralPrograms(menuProductCode, defaultString(status, "active"))
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ProgramCode < items[j].ProgramCode
	})
	return items, nil
}

func (s *Service) ListCodes(orgID, programCode, status string) ([]platform.ReferralCode, error) {
	programID, err := s.resolveProgramID(programCode)
	if err != nil {
		return nil, err
	}
	items, err := s.platform.ListReferralCodes(programID, "organization", orgID, status)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Service) CreateCode(orgID string, input CreateCodeInput) (*platform.ReferralCode, error) {
	return s.platform.CreateReferralCode(platform.CreateReferralCodeInput{
		ProgramCode:         input.ProgramCode,
		Code:                input.Code,
		PromoterSubjectType: "organization",
		PromoterSubjectID:   orgID,
		Status:              "active",
		Metadata:            input.Metadata,
	})
}

func (s *Service) EnsureCode(orgID string, input CreateCodeInput) (*platform.ReferralCode, error) {
	items, err := s.ListCodes(orgID, input.ProgramCode, "active")
	if err != nil {
		return nil, err
	}
	if len(items) > 0 {
		return &items[0], nil
	}
	return s.CreateCode(orgID, input)
}

func (s *Service) ResolveCode(code string) (*platform.ResolvedReferralCode, error) {
	return s.platform.ResolveReferralCode(code, menuProductCode)
}

func (s *Service) ListConversions(orgID, status string) ([]platform.ReferralConversion, error) {
	items, err := s.platform.ListReferralConversions(menuProductCode, "organization", orgID, status)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Service) ListCommissions(orgID, status string) ([]platform.CommissionLedger, error) {
	items, err := s.platform.ListCommissions(menuProductCode, "organization", orgID, status)
	if err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items, nil
}

func (s *Service) Overview(orgID, conversionStatus, commissionStatus string) (*Overview, error) {
	programs, err := s.ListPrograms("active")
	if err != nil {
		return nil, err
	}
	codes, err := s.ListCodes(orgID, "", "")
	if err != nil {
		return nil, err
	}
	conversions, err := s.ListConversions(orgID, conversionStatus)
	if err != nil {
		return nil, err
	}
	commissions, err := s.ListCommissions(orgID, commissionStatus)
	if err != nil {
		return nil, err
	}
	out := &Overview{
		Programs:    programs,
		Codes:       codes,
		Conversions: conversions,
		Commissions: commissions,
	}
	for _, item := range conversions {
		out.TotalConversions++
		switch item.Status {
		case "commission_earned":
			out.EarnedConversions++
		case "reward_issued":
			out.EarnedConversions++
		case "reversed":
			out.ReversedConversions++
		default:
			out.TrackedConversions++
		}
	}
	for _, item := range commissions {
		out.TotalCommission += item.Amount
		switch item.Status {
		case "earned":
			out.EarnedCommission += item.Amount
			out.RedeemableCommission += item.Amount
		case "redeemed":
			out.RedeemedCommission += item.Amount
		case "reversed":
			out.ReversedCommission += item.Amount
		default:
			out.PendingCommission += item.Amount
		}
	}
	return out, nil
}

func (s *Service) RedeemCommissions(orgID string, input RedeemInput) (*platform.RedeemCommissionsResult, error) {
	return s.platform.RedeemCommissions(platform.RedeemCommissionsInput{
		ProductCode:            menuProductCode,
		BeneficiarySubjectType: "organization",
		BeneficiarySubjectID:   orgID,
		AssetCode:              s.rewardAssetCode,
		CommissionIDs:          input.CommissionIDs,
		Metadata:               input.Metadata,
	})
}

func (s *Service) bootstrapAssetDefinitions() error {
	existing, err := s.platform.ListAssetDefinitions(menuProductCode, "", "")
	if err != nil {
		return err
	}
	byCode := make(map[string]platform.AssetDefinition, len(existing))
	for _, item := range existing {
		byCode[item.AssetCode] = item
	}
	for _, asset := range defaultAssetDefinitions(s.creditsAssetCode, s.rewardAssetCode, s.allowanceAssetCode) {
		if item, ok := byCode[asset.AssetCode]; ok {
			logger.With("asset_code", item.AssetCode, "lifecycle_type", item.LifecycleType).Info("referral.bootstrap.asset_exists")
			continue
		}
		created, createErr := s.platform.CreateAssetDefinition(asset)
		if createErr != nil {
			return createErr
		}
		logger.With("asset_code", created.AssetCode, "lifecycle_type", created.LifecycleType).Info("referral.bootstrap.asset_created")
	}
	return nil
}

func (s *Service) resolveProgramID(programCode string) (string, error) {
	if programCode == "" {
		return "", nil
	}
	items, err := s.platform.ListReferralPrograms(menuProductCode, "")
	if err != nil {
		return "", err
	}
	for _, item := range items {
		if item.ProgramCode == programCode {
			return item.ID, nil
		}
	}
	return "", nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
