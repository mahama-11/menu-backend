package user

import (
	"encoding/json"
	"sort"
	"strings"
	"time"

	"menu-service/internal/models"
	audit "menu-service/internal/modules/audit"
	"menu-service/internal/modules/auth"
	"menu-service/internal/platform"
	"menu-service/internal/repository"
)

type Service struct {
	repo     *repository.UserRepository
	studio   *repository.StudioRepository
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

type WalletHistoryEntry struct {
	ID               string         `json:"id"`
	Category         string         `json:"category"`
	Title            string         `json:"title"`
	Description      string         `json:"description,omitempty"`
	Direction        string         `json:"direction"`
	Amount           int64          `json:"amount"`
	AssetCode        string         `json:"asset_code,omitempty"`
	Currency         string         `json:"currency,omitempty"`
	Status           string         `json:"status"`
	OccurredAt       string         `json:"occurred_at"`
	ReferenceType    string         `json:"reference_type,omitempty"`
	ReferenceID      string         `json:"reference_id,omitempty"`
	JobID            string         `json:"job_id,omitempty"`
	EventID          string         `json:"event_id,omitempty"`
	SettlementID     string         `json:"settlement_id,omitempty"`
	BillableItemCode string         `json:"billable_item_code,omitempty"`
	ChargeMode       string         `json:"charge_mode,omitempty"`
	QuotaConsumed    int64          `json:"quota_consumed,omitempty"`
	CreditsConsumed  int64          `json:"credits_consumed,omitempty"`
	WalletDebited    int64          `json:"wallet_debited,omitempty"`
	FlowStatus       string         `json:"flow_status,omitempty"`
	Metadata         map[string]any `json:"metadata,omitempty"`
}

type WalletHistoryResult struct {
	Items []WalletHistoryEntry `json:"items"`
}

func NewService(repo *repository.UserRepository, studioRepo *repository.StudioRepository, platformClient *platform.Client, authService *auth.Service, auditService *audit.Service) *Service {
	return &Service{repo: repo, studio: studioRepo, platform: platformClient, auth: authService, audit: auditService}
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

func (s *Service) WalletSummary(orgID string) (*auth.WalletSummary, error) {
	return s.auth.WalletSummary(orgID)
}

func (s *Service) AuditHistory(userID, orgID, targetType, status string, limit, offset int) (*audit.HistoryResult, error) {
	if s.audit == nil {
		return &audit.HistoryResult{Items: []audit.HistoryItem{}, Total: 0}, nil
	}
	return s.audit.History(orgID, userID, targetType, status, limit, offset)
}

func (s *Service) WalletHistory(orgID string, limit int) (*WalletHistoryResult, error) {
	entries := make([]WalletHistoryEntry, 0)

	if s.platform != nil {
		rewards, err := s.platform.ListRewards("menu", "organization", orgID)
		if err != nil {
			return nil, err
		}
		for _, item := range rewards {
			entries = append(entries, mapRewardHistory(item))
		}

		commissions, err := s.platform.ListCommissions("menu", "organization", orgID, "")
		if err != nil {
			return nil, err
		}
		for _, item := range commissions {
			entries = append(entries, mapCommissionHistory(item))
		}

		accounts, err := s.platform.ListWalletAccounts("organization", orgID)
		if err != nil {
			return nil, err
		}
		for _, account := range accounts {
			ledgers, ledgerErr := s.platform.ListWalletLedger(account.ID)
			if ledgerErr != nil {
				return nil, ledgerErr
			}
			for _, item := range ledgers {
				if entry, ok := mapWalletLedgerHistory(item); ok {
					entries = append(entries, entry)
				}
			}
		}
	}

	if s.studio != nil {
		intents, err := s.studio.ListChargeIntents(orgID, 0)
		if err != nil {
			return nil, err
		}
		for _, item := range intents {
			entries = append(entries, mapChargeIntentHistory(item))
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].OccurredAt > entries[j].OccurredAt
	})
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	return &WalletHistoryResult{Items: entries}, nil
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

func mapRewardHistory(item platform.RewardLedger) WalletHistoryEntry {
	category := "reward"
	title := "Reward credit issued"
	if item.RewardType == "commission_redeem" {
		category = "redeem"
		title = "Commission redeemed to credits"
	}
	return WalletHistoryEntry{
		ID:            item.ID,
		Category:      category,
		Title:         title,
		Direction:     "credit",
		Amount:        item.Amount,
		AssetCode:     item.AssetCode,
		Status:        item.Status,
		OccurredAt:    item.CreatedAt.UTC().Format(time.RFC3339),
		ReferenceType: item.ReferenceType,
		ReferenceID:   item.ReferenceID,
		Metadata:      decodeMap(item.Metadata),
	}
}

func mapCommissionHistory(item platform.CommissionLedger) WalletHistoryEntry {
	title := "Referral commission earned"
	switch item.Status {
	case "redeemed":
		title = "Referral commission redeemed"
	case "reversed":
		title = "Referral commission reversed"
	}
	return WalletHistoryEntry{
		ID:            item.ID,
		Category:      "commission",
		Title:         title,
		Direction:     "info",
		Amount:        item.Amount,
		Currency:      item.Currency,
		Status:        item.Status,
		OccurredAt:    item.CreatedAt.UTC().Format(time.RFC3339),
		ReferenceType: item.ReferenceType,
		ReferenceID:   item.ReferenceID,
		Metadata:      decodeMap(item.Metadata),
	}
}

func mapWalletLedgerHistory(item platform.WalletLedger) (WalletHistoryEntry, bool) {
	switch item.Reason {
	case "reward_issue", "metering_settlement":
		return WalletHistoryEntry{}, false
	case "asset_expire":
		return WalletHistoryEntry{
			ID:            item.ID,
			Category:      "expiration",
			Title:         "Credits expired",
			Direction:     "debit",
			Amount:        item.Amount,
			AssetCode:     item.AssetCode,
			Status:        item.Status,
			OccurredAt:    item.CreatedAt.UTC().Format(time.RFC3339),
			ReferenceType: item.ReferenceType,
			ReferenceID:   item.ReferenceID,
			Metadata:      decodeMap(item.Metadata),
		}, true
	default:
		title := "Balance adjustment"
		category := "wallet_adjustment"
		if item.Direction == "credit" {
			title = "Credits recharged"
			category = "recharge"
		}
		return WalletHistoryEntry{
			ID:            item.ID,
			Category:      category,
			Title:         title,
			Direction:     normalizeDirection(item.Direction),
			Amount:        item.Amount,
			AssetCode:     item.AssetCode,
			Status:        item.Status,
			OccurredAt:    item.CreatedAt.UTC().Format(time.RFC3339),
			ReferenceType: item.ReferenceType,
			ReferenceID:   item.ReferenceID,
			Metadata:      decodeMap(item.Metadata),
		}, true
	}
}

func mapChargeIntentHistory(item models.StudioChargeIntent) WalletHistoryEntry {
	metadata := decodeMap(item.Metadata)
	settlement, _ := metadata["settlement"].(map[string]any)
	title := "Studio generation charge"
	switch item.Status {
	case "released":
		title = "Studio reservation released"
	case "failed_need_reconcile":
		title = "Studio charge needs reconciliation"
	}
	occurredAt := item.UpdatedAt.UTC().Format(time.RFC3339)
	if item.FinalizedAt != nil {
		occurredAt = item.FinalizedAt.UTC().Format(time.RFC3339)
	} else if item.ReleasedAt != nil {
		occurredAt = item.ReleasedAt.UTC().Format(time.RFC3339)
	} else if item.ReservedAt != nil {
		occurredAt = item.ReservedAt.UTC().Format(time.RFC3339)
	}
	return WalletHistoryEntry{
		ID:               item.ID,
		Category:         "charge",
		Title:            title,
		Direction:        "debit",
		Amount:           int64MapValue(settlement, "net_amount"),
		AssetCode:        stringMapValue(settlement, "wallet_asset_code"),
		Currency:         stringMapValue(settlement, "currency"),
		Status:           item.Status,
		OccurredAt:       occurredAt,
		ReferenceType:    "studio_job",
		ReferenceID:      item.JobID,
		JobID:            item.JobID,
		EventID:          item.EventID,
		SettlementID:     item.SettlementID,
		BillableItemCode: item.BillableItemCode,
		ChargeMode:       item.ChargeMode,
		QuotaConsumed:    int64MapValue(settlement, "quota_consumed"),
		CreditsConsumed:  int64MapValue(settlement, "credits_consumed"),
		WalletDebited:    int64MapValue(settlement, "wallet_debited"),
		FlowStatus:       item.Status,
		Metadata:         metadata,
	}
}

func decodeMap(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func stringMapValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, ok := values[key]
	if !ok {
		return ""
	}
	str, _ := value.(string)
	return str
}

func int64MapValue(values map[string]any, key string) int64 {
	if values == nil {
		return 0
	}
	value, ok := values[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func normalizeDirection(direction string) string {
	switch direction {
	case "credit":
		return "credit"
	case "debit":
		return "debit"
	default:
		return "info"
	}
}
