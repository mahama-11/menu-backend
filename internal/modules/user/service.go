package user

import (
	"menu-service/internal/models"
	audit "menu-service/internal/modules/audit"
	"menu-service/internal/modules/auth"
	"menu-service/internal/platform"
	"menu-service/internal/repository"
)

type Service struct {
	repo     *repository.UserRepository
	platform *platform.Client
	auth     *auth.Service
	audit    *audit.Service
}

type ActivityItem struct {
	ID           string `json:"id"`
	ActionType   string `json:"action_type"`
	ActionName   string `json:"action_name"`
	Status       string `json:"status"`
	CreditsUsed  int64  `json:"credits_used"`
	CreatedAt    string `json:"created_at"`
	ResultURL    string `json:"result_url,omitempty"`
	EventID      string `json:"event_id,omitempty"`
	JobID        string `json:"job_id,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

type ActivitiesResult struct {
	Activities []ActivityItem `json:"activities"`
	TotalCount int64          `json:"total_count"`
}

type UpdateProfileInput struct {
	Name               string `json:"name"`
	RestaurantName     string `json:"restaurant_name"`
	LanguagePreference string `json:"language_preference" binding:"omitempty,oneof=en zh th"`
}

func NewService(repo *repository.UserRepository, platformClient *platform.Client, authService *auth.Service, auditService *audit.Service) *Service {
	return &Service{repo: repo, platform: platformClient, auth: authService, audit: auditService}
}

func (s *Service) Activities(userID, orgID string, limit, offset int) (*ActivitiesResult, error) {
	items, total, err := s.repo.ListActivities(userID, orgID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]ActivityItem, 0, len(items))
	for _, item := range items {
		out = append(out, mapActivity(item))
	}
	return &ActivitiesResult{Activities: out, TotalCount: total}, nil
}

func (s *Service) Profile(userID, orgID string) (*auth.UserSummary, error) {
	session, err := s.auth.Session(userID, orgID)
	if err != nil {
		return nil, err
	}
	return &session.User, nil
}

func (s *Service) Credits(userID, orgID string) (*auth.CreditsSummary, error) {
	return s.auth.Credits(userID, orgID)
}

func (s *Service) UpdateProfile(userID, orgID string, input UpdateProfileInput) (*auth.UserSummary, error) {
	if input.Name != "" {
		if _, err := s.platform.UpdateUserProfile(userID, platform.UpdateUserProfileInput{FullName: input.Name}); err != nil {
			return nil, err
		}
	}
	if input.RestaurantName != "" {
		if err := s.platform.UpdateOrganizationProfile(orgID, platform.UpdateOrganizationProfileInput{Name: input.RestaurantName}); err != nil {
			return nil, err
		}
	}
	if input.LanguagePreference != "" {
		if _, err := s.repo.UpsertPreference(userID, orgID, input.LanguagePreference); err != nil {
			return nil, err
		}
	}
	return s.Profile(userID, orgID)
}

func mapActivity(item models.Activity) ActivityItem {
	return ActivityItem{
		ID:           item.ID,
		ActionType:   item.ActionType,
		ActionName:   item.ActionName,
		Status:       item.Status,
		CreditsUsed:  item.CreditsUsed,
		CreatedAt:    item.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		ResultURL:    item.ResultURL,
		EventID:      item.EventID,
		JobID:        item.JobID,
		ErrorMessage: item.ErrorMessage,
	}
}
