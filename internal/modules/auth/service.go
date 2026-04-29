package auth

import (
	"fmt"
	"menu-service/internal/config"
	"menu-service/internal/models"
	authz "menu-service/internal/modules/authz"
	"menu-service/internal/platform"
	"menu-service/internal/repository"
	"menu-service/pkg/logger"
	"strings"
	"time"
)

type Service struct {
	platform *platform.Client
	repo     *repository.UserRepository
	authz    *authz.Service
	appCfg   config.AppConfig
}

type RegisterInput struct {
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=6"`
	RestaurantName string `json:"restaurant_name" binding:"required,min=2"`
	Role           string `json:"role"`
	ReferralCode   string `json:"referral_code,omitempty"`
	ChannelCode    string `json:"channel_code,omitempty"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type UserSummary struct {
	ID                 string `json:"id"`
	Email              string `json:"email"`
	FullName           string `json:"full_name"`
	OrgID              string `json:"org_id"`
	OrgName            string `json:"org_name"`
	OrgRole            string `json:"org_role"`
	PlanID             string `json:"plan_id"`
	Status             string `json:"status"`
	SelectedRole       string `json:"selected_role,omitempty"`
	RestaurantName     string `json:"restaurant_name"`
	LanguagePreference string `json:"language_preference,omitempty"`
}

type CreditsSummary struct {
	AssetCode        string     `json:"asset_code"`
	Balance          int64      `json:"balance"`
	PermanentBalance int64      `json:"permanent_balance"`
	RewardBalance    int64      `json:"reward_balance"`
	AllowanceBalance int64      `json:"allowance_balance"`
	Rewarded         int64      `json:"rewarded"`
	RewardGranted    bool       `json:"reward_granted"`
	RewardError      string     `json:"reward_error,omitempty"`
	MaxCredits       *int64     `json:"max_credits"`
	PlanName         string     `json:"plan_name"`
	PlanTier         string     `json:"plan_tier"`
	ResetDate        *time.Time `json:"reset_date"`
	BillingModel     string     `json:"billing_model"`
}

type WalletSummary struct {
	BillingSubjectType string                        `json:"billing_subject_type"`
	BillingSubjectID   string                        `json:"billing_subject_id"`
	ProductCode        string                        `json:"product_code"`
	TotalBalance       int64                         `json:"total_balance"`
	PermanentBalance   int64                         `json:"permanent_balance"`
	RewardBalance      int64                         `json:"reward_balance"`
	AllowanceBalance   int64                         `json:"allowance_balance"`
	Assets             []platform.WalletAssetSummary `json:"assets"`
}

type AuthResult struct {
	AccessToken string         `json:"access_token"`
	User        UserSummary    `json:"user"`
	Credits     CreditsSummary `json:"credits"`
	Access      AccessSummary  `json:"access"`
}

type SessionResult struct {
	Authenticated bool           `json:"authenticated"`
	User          UserSummary    `json:"user"`
	Credits       CreditsSummary `json:"credits"`
	Access        AccessSummary  `json:"access"`
}

type AccessSummary struct {
	ActiveOrgID         string   `json:"active_org_id"`
	HasMenuAccess       bool     `json:"has_menu_access"`
	MenuRoles           []string `json:"menu_roles"`
	MenuPermissions     []string `json:"menu_permissions"`
	PlatformPermissions []string `json:"platform_permissions,omitempty"`
}

func NewService(platformClient *platform.Client, repo *repository.UserRepository, authzService *authz.Service, appCfg config.AppConfig) *Service {
	return &Service{platform: platformClient, repo: repo, authz: authzService, appCfg: appCfg}
}

func (s *Service) Register(input RegisterInput) (*AuthResult, error) {
	authResult, err := s.platform.Register(platform.AuthRegisterInput{
		FullName: input.RestaurantName,
		Email:    input.Email,
		Company:  input.RestaurantName,
		Password: input.Password,
	})
	if err != nil {
		return nil, err
	}

	result := &AuthResult{
		AccessToken: authResult.AccessToken,
		User:        s.buildUserSummary(authResult.User, input.RestaurantName, input.Role),
		Credits:     s.buildCreditsSummary(authResult.User.OrgID, authResult.User.PlanID),
	}
	if s.repo != nil {
		_, _ = s.repo.UpsertPreference(authResult.User.ID, authResult.User.OrgID, "en")
	}

	if s.appCfg.SignupBonusCredits > 0 {
		_, rewardErr := s.platform.CreateReward(platform.CreateRewardInput{
			ProductCode:            "menu",
			CampaignCode:           "menu_signup_p0",
			RewardType:             "signup_bonus",
			BeneficiarySubjectType: "organization",
			BeneficiarySubjectID:   authResult.User.OrgID,
			AssetCode:              s.appCfg.CreditsAssetCode,
			Amount:                 s.appCfg.SignupBonusCredits,
			Status:                 "issued",
			ReferenceType:          "signup",
			ReferenceID:            authResult.User.ID,
			Metadata:               input.Role,
		})
		if rewardErr != nil {
			result.Credits.RewardError = rewardErr.Error()
		} else {
			result.Credits.RewardGranted = true
			result.Credits.Rewarded = s.appCfg.SignupBonusCredits
			if s.repo != nil {
				_ = s.repo.CreateActivity(&models.Activity{
					UserID:         authResult.User.ID,
					OrganizationID: authResult.User.OrgID,
					ActionType:     "signup_bonus",
					ActionName:     "Welcome Bonus",
					Status:         "succeeded",
					CreditsUsed:    0,
				})
			}
		}
	}

	if input.ReferralCode != "" {
		if referralErr := s.createSignupReferralConversion(authResult.User, input); referralErr != nil {
			if s.repo != nil {
				_ = s.repo.CreateActivity(&models.Activity{
					UserID:         authResult.User.ID,
					OrganizationID: authResult.User.OrgID,
					ActionType:     "referral_signup_tracking",
					ActionName:     "Referral Signup Tracking",
					Status:         "failed",
					CreditsUsed:    0,
					ErrorMessage:   referralErr.Error(),
				})
			}
		} else if s.repo != nil {
			_ = s.repo.CreateActivity(&models.Activity{
				UserID:         authResult.User.ID,
				OrganizationID: authResult.User.OrgID,
				ActionType:     "referral_signup_tracking",
				ActionName:     "Referral Signup Tracking",
				Status:         "succeeded",
				CreditsUsed:    0,
				EventID:        input.ReferralCode,
			})
		}
	}

	if input.ChannelCode != "" {
		if channelErr := s.createSignupChannelBinding(authResult.User, input); channelErr != nil {
			logger.With("org_id", authResult.User.OrgID, "channel_code", input.ChannelCode, "error", channelErr).Error("auth.register.channel_binding_failed")
			if s.repo != nil {
				_ = s.repo.CreateActivity(&models.Activity{
					UserID:         authResult.User.ID,
					OrganizationID: authResult.User.OrgID,
					ActionType:     "channel_signup_binding",
					ActionName:     "Channel Signup Binding",
					Status:         "failed",
					CreditsUsed:    0,
					ErrorMessage:   channelErr.Error(),
					EventID:        input.ChannelCode,
				})
			}
		} else if s.repo != nil {
			_ = s.repo.CreateActivity(&models.Activity{
				UserID:         authResult.User.ID,
				OrganizationID: authResult.User.OrgID,
				ActionType:     "channel_signup_binding",
				ActionName:     "Channel Signup Binding",
				Status:         "succeeded",
				CreditsUsed:    0,
				EventID:        input.ChannelCode,
			})
		}
	}

	result.Credits.Balance = s.lookupCreditsBalance(authResult.User.OrgID)
	result.Access = s.buildBestEffortAccessSummary(authResult.User.ID, authResult.User.OrgID)
	return result, nil
}

func (s *Service) Login(input LoginInput) (*AuthResult, error) {
	authResult, err := s.platform.Login(platform.AuthLoginInput{
		Email:    input.Email,
		Password: input.Password,
	})
	if err != nil {
		return nil, err
	}
	access, err := s.buildAccessSummary(authResult.User.ID, authResult.User.OrgID)
	if err != nil {
		return nil, err
	}
	return &AuthResult{
		AccessToken: authResult.AccessToken,
		User:        s.buildUserSummary(authResult.User, currentOrgName(authResult.User), ""),
		Credits:     s.buildCreditsSummary(authResult.User.OrgID, authResult.User.PlanID),
		Access:      access,
	}, nil
}

func (s *Service) Session(userID, orgID string) (*SessionResult, error) {
	profile, err := s.platform.GetUserProfile(userID, orgID)
	if err != nil {
		return nil, err
	}
	access, err := s.buildAccessSummary(profile.ID, profile.OrgID)
	if err != nil {
		return nil, err
	}
	return &SessionResult{
		Authenticated: true,
		User:          s.buildUserSummary(*profile, currentOrgName(*profile), ""),
		Credits:       s.buildCreditsSummary(profile.OrgID, profile.PlanID),
		Access:        access,
	}, nil
}

func (s *Service) Credits(userID, orgID string) (*CreditsSummary, error) {
	profile, err := s.platform.GetUserProfile(userID, orgID)
	if err != nil {
		return nil, err
	}
	out := s.buildCreditsSummary(profile.OrgID, profile.PlanID)
	return &out, nil
}

func (s *Service) lookupCreditsBalance(orgID string) int64 {
	summary, err := s.platform.GetWalletSummary("organization", orgID, "menu")
	if err != nil {
		return 0
	}
	var total int64
	for _, asset := range summary.Assets {
		if isSpendableCreditsAsset(asset) {
			total += asset.AvailableBalance
		}
	}
	return total
}

func (s *Service) buildUserSummary(user platform.PlatformUserProfile, restaurantName, selectedRole string) UserSummary {
	return UserSummary{
		ID:                 user.ID,
		Email:              user.Email,
		FullName:           user.FullName,
		OrgID:              user.OrgID,
		OrgName:            currentOrgName(user),
		OrgRole:            user.OrgRole,
		PlanID:             user.PlanID,
		Status:             user.Status,
		SelectedRole:       selectedRole,
		RestaurantName:     restaurantName,
		LanguagePreference: s.lookupLanguagePreference(user.ID, user.OrgID),
	}
}

func (s *Service) buildCreditsSummary(orgID, planID string) CreditsSummary {
	planName, planTier := normalizePlan(planID)
	walletSummary, _ := s.platform.GetWalletSummary("organization", orgID, "menu")
	var totalBalance, permanentBalance, rewardBalance, allowanceBalance int64
	if walletSummary != nil {
		for _, asset := range walletSummary.Assets {
			if !isSpendableCreditsAsset(asset) {
				continue
			}
			totalBalance += asset.AvailableBalance
			switch asset.LifecycleType {
			case "cycle_reset":
				allowanceBalance += asset.AvailableBalance
			case "expiring":
				rewardBalance += asset.AvailableBalance
			default:
				permanentBalance += asset.AvailableBalance
			}
		}
	}
	return CreditsSummary{
		AssetCode:        s.appCfg.CreditsAssetCode,
		Balance:          totalBalance,
		PermanentBalance: permanentBalance,
		RewardBalance:    rewardBalance,
		AllowanceBalance: allowanceBalance,
		MaxCredits:       nil,
		PlanName:         planName,
		PlanTier:         planTier,
		ResetDate:        nil,
		BillingModel:     "wallet_bonus",
	}
}

func (s *Service) WalletSummary(orgID string) (*WalletSummary, error) {
	item, err := s.platform.GetWalletSummary("organization", orgID, "menu")
	if err != nil {
		return nil, err
	}
	return &WalletSummary{
		BillingSubjectType: item.BillingSubjectType,
		BillingSubjectID:   item.BillingSubjectID,
		ProductCode:        item.ProductCode,
		TotalBalance:       item.TotalBalance,
		PermanentBalance:   item.PermanentBalance,
		RewardBalance:      item.RewardBalance,
		AllowanceBalance:   item.AllowanceBalance,
		Assets:             item.Assets,
	}, nil
}

func (s *Service) lookupLanguagePreference(userID, orgID string) string {
	if s.repo == nil {
		return "en"
	}
	item, err := s.repo.GetPreference(userID, orgID)
	if err != nil || item.LanguagePreference == "" {
		return "en"
	}
	return item.LanguagePreference
}

func normalizePlan(planID string) (string, string) {
	switch planID {
	case "starter", "":
		return "Basic", "free"
	case "pro_monthly", "pro":
		return "Pro", "pro"
	case "growth":
		return "Growth", "growth"
	default:
		return planID, "custom"
	}
}

func (s *Service) buildAccessSummary(userID, orgID string) (AccessSummary, error) {
	if s.authz == nil {
		return AccessSummary{
			ActiveOrgID:     orgID,
			HasMenuAccess:   false,
			MenuRoles:       []string{},
			MenuPermissions: []string{},
		}, nil
	}
	ctx, err := s.authz.Resolve(userID, orgID)
	if err != nil {
		return AccessSummary{}, err
	}
	return AccessSummary{
		ActiveOrgID:         ctx.OrgID,
		HasMenuAccess:       containsString(ctx.MenuPermissions, "menu.access"),
		MenuRoles:           ctx.MenuRoles,
		MenuPermissions:     ctx.MenuPermissions,
		PlatformPermissions: ctx.PlatformPermissions,
	}, nil
}

func (s *Service) buildBestEffortAccessSummary(userID, orgID string) AccessSummary {
	summary, err := s.buildAccessSummary(userID, orgID)
	if err != nil {
		return AccessSummary{
			ActiveOrgID:     orgID,
			HasMenuAccess:   false,
			MenuRoles:       []string{},
			MenuPermissions: []string{},
		}
	}
	return summary
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func isSpendableCreditsAsset(asset platform.WalletAssetSummary) bool {
	if asset.AssetCode == "MENU_CASH" || asset.AssetType == "cash_balance" {
		return false
	}
	return true
}

func (s *Service) createSignupReferralConversion(user platform.PlatformUserProfile, input RegisterInput) error {
	_, err := s.platform.CreateReferralConversion(platform.CreateReferralConversionInput{
		ReferralCode:          input.ReferralCode,
		ProductCode:           "menu",
		TriggerType:           "signup",
		ReferredSubjectType:   "organization",
		ReferredSubjectID:     user.OrgID,
		SettlementSubjectType: "organization",
		SettlementSubjectID:   user.OrgID,
		ReferenceType:         "menu_signup",
		ReferenceID:           user.ID,
		Metadata:              input.Role,
	})
	return err
}

func (s *Service) createSignupChannelBinding(user platform.PlatformUserProfile, input RegisterInput) error {
	partners, err := s.platform.ListChannelPartners("active")
	if err != nil {
		return err
	}
	channelCode := strings.TrimSpace(input.ChannelCode)
	var partner *platform.ChannelPartner
	for i := range partners {
		if strings.EqualFold(partners[i].Code, channelCode) {
			partner = &partners[i]
			break
		}
	}
	if partner == nil {
		return fmt.Errorf("channel partner code not found: %s", channelCode)
	}
	programs, err := s.platform.ListChannelPrograms("menu", "active")
	if err != nil {
		return err
	}
	var program *platform.ChannelProgram
	for i := range programs {
		if programs[i].ProgramType == "channel_revenue_share" {
			program = &programs[i]
			break
		}
	}
	if program == nil {
		return fmt.Errorf("active channel revenue share program not found")
	}
	_, err = s.platform.CreateChannelBinding(platform.CreateChannelBindingInput{
		ProductCode:      "menu",
		OrgID:            user.OrgID,
		ChannelPartnerID: partner.ID,
		ChannelProgramID: program.ID,
		BindingSource:    "signup_code",
		SourceCode:       channelCode,
		SourceRefID:      user.ID,
		Status:           "active",
		CreatedBy:        "menu_auth_register",
		Metadata:         input.Role,
	})
	if err != nil && platform.IsConflict(err) {
		return nil
	}
	return err
}

func currentOrgName(user platform.PlatformUserProfile) string {
	for _, org := range user.Orgs {
		if org.ID == user.OrgID {
			return org.Name
		}
	}
	return ""
}
