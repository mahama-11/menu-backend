package auth

import (
	"menu-service/internal/config"
	"menu-service/internal/models"
	authz "menu-service/internal/modules/authz"
	"menu-service/internal/platform"
	"menu-service/internal/repository"
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
	AssetCode     string     `json:"asset_code"`
	Balance       int64      `json:"balance"`
	Rewarded      int64      `json:"rewarded"`
	RewardGranted bool       `json:"reward_granted"`
	RewardError   string     `json:"reward_error,omitempty"`
	MaxCredits    *int64     `json:"max_credits"`
	PlanName      string     `json:"plan_name"`
	PlanTier      string     `json:"plan_tier"`
	ResetDate     *time.Time `json:"reset_date"`
	BillingModel  string     `json:"billing_model"`
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
	accounts, err := s.platform.ListWalletAccounts("organization", orgID)
	if err != nil {
		return 0
	}
	for _, account := range accounts {
		if account.AssetCode == s.appCfg.CreditsAssetCode {
			return account.Balance
		}
	}
	return 0
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
	return CreditsSummary{
		AssetCode:    s.appCfg.CreditsAssetCode,
		Balance:      s.lookupCreditsBalance(orgID),
		MaxCredits:   nil,
		PlanName:     planName,
		PlanTier:     planTier,
		ResetDate:    nil,
		BillingModel: "wallet_bonus",
	}
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

func currentOrgName(user platform.PlatformUserProfile) string {
	for _, org := range user.Orgs {
		if org.ID == user.OrgID {
			return org.Name
		}
	}
	return ""
}
