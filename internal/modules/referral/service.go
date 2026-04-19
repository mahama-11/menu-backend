package referral

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/platform"
	"menu-service/pkg/logger"
)

const menuProductCode = "menu"

type Service struct {
	platform           *platform.Client
	frontendBaseURL    string
	creditsAssetCode   string
	rewardAssetCode    string
	allowanceAssetCode string
}

type ReferralCodeSummary struct {
	ID          string         `json:"id"`
	ProgramID   string         `json:"program_id"`
	ProductCode string         `json:"product_code"`
	Code        string         `json:"code"`
	Status      string         `json:"status"`
	Metadata    map[string]any `json:"metadata,omitempty"`
	InviteURL   string         `json:"invite_url,omitempty"`
	SignupURL   string         `json:"signup_url,omitempty"`
	ShareText   string         `json:"share_text,omitempty"`
	CreatedAt   string         `json:"created_at"`
	UpdatedAt   string         `json:"updated_at"`
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
	Codes                []ReferralCodeSummary         `json:"codes"`
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
	RedeemTargetAssetCode string                       `json:"redeem_target_asset_code"`
	InviteBaseURL         string                       `json:"invite_base_url,omitempty"`
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
		frontendBaseURL:    strings.TrimRight(appCfg.FrontendBaseURL, "/"),
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

func (s *Service) ListCodes(orgID, programCode, status string) ([]ReferralCodeSummary, error) {
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
	out := make([]ReferralCodeSummary, 0, len(items))
	for _, item := range items {
		out = append(out, s.mapReferralCode(item))
	}
	return out, nil
}

func (s *Service) CreateCode(orgID string, input CreateCodeInput) (*ReferralCodeSummary, error) {
	item, err := s.platform.CreateReferralCode(platform.CreateReferralCodeInput{
		ProgramCode:         input.ProgramCode,
		Code:                input.Code,
		PromoterSubjectType: "organization",
		PromoterSubjectID:   orgID,
		Status:              "active",
		Metadata:            input.Metadata,
	})
	if err != nil {
		return nil, err
	}
	mapped := s.mapReferralCode(*item)
	return &mapped, nil
}

func (s *Service) EnsureCode(orgID string, input CreateCodeInput) (*ReferralCodeSummary, error) {
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
		Programs:              programs,
		Codes:                 codes,
		Conversions:           conversions,
		Commissions:           commissions,
		RedeemTargetAssetCode: s.rewardAssetCode,
		InviteBaseURL:         s.frontendBaseURL,
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

func (s *Service) mapReferralCode(item platform.ReferralCode) ReferralCodeSummary {
	signupURL := s.buildSignupURL(item.Code)
	return ReferralCodeSummary{
		ID:          item.ID,
		ProgramID:   item.ProgramID,
		ProductCode: item.ProductCode,
		Code:        item.Code,
		Status:      item.Status,
		Metadata:    decodeStringMap(item.Metadata),
		InviteURL:   signupURL,
		SignupURL:   signupURL,
		ShareText:   s.buildShareText(item.Code, signupURL),
		CreatedAt:   item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *Service) buildSignupURL(code string) string {
	if strings.TrimSpace(code) == "" {
		return ""
	}
	base := defaultString(s.frontendBaseURL, "http://localhost:5173")
	return fmt.Sprintf("%s/signup?referral_code=%s", base, url.QueryEscape(code))
}

func (s *Service) buildShareText(code, signupURL string) string {
	if signupURL == "" {
		return ""
	}
	return fmt.Sprintf("Use my Menu invite code %s to join: %s", code, signupURL)
}

func decodeStringMap(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
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
