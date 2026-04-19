package user

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/models"
	audit "menu-service/internal/modules/audit"
	"menu-service/internal/platform"
	"menu-service/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestWalletHistory_AggregatesPlatformAndStudioEntries(t *testing.T) {
	mock := newUserPlatformMockServer(t)
	defer mock.server.Close()

	client := platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-user-test",
		InternalServiceSecret: "test-secret",
	})
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if migrateErr := db.AutoMigrate(&models.StudioChargeIntent{}); migrateErr != nil {
		t.Fatalf("auto migrate: %v", migrateErr)
	}
	studioRepo := repository.NewStudioRepository(db)
	finalizedAt := time.Now().UTC()
	if createErr := studioRepo.CreateChargeIntent(&models.StudioChargeIntent{
		ID:               "intent-1",
		JobID:            "job-1",
		UserID:           "user-1",
		OrganizationID:   "org-1",
		ProductCode:      "menu",
		ChargeMode:       "single",
		ResourceType:     "credits",
		BillableItemCode: "menu_studio_single_generate",
		EstimatedUnits:   1,
		FinalUnits:       1,
		ReservationKey:   "res-key-1",
		FinalizationID:   "fin-1",
		EventID:          "evt-1",
		Status:           "settled",
		SettlementID:     "settlement-1",
		Metadata:         `{"settlement":{"currency":"MENU_CREDIT","net_amount":100,"wallet_asset_code":"MENU_CREDIT","wallet_debited":100}}`,
		FinalizedAt:      &finalizedAt,
	}); createErr != nil {
		t.Fatalf("create charge intent: %v", createErr)
	}

	service := NewService(nil, studioRepo, client, nil, nil)
	result, err := service.WalletHistory("org-1", 20)
	if err != nil {
		t.Fatalf("WalletHistory() error = %v", err)
	}
	if len(result.Items) < 4 {
		t.Fatalf("wallet history items = %d, want at least 4", len(result.Items))
	}
	foundCharge := false
	foundReward := false
	foundCommission := false
	foundExpire := false
	for _, item := range result.Items {
		switch item.Category {
		case "charge":
			foundCharge = item.JobID == "job-1" && item.Amount == 100
		case "reward":
			foundReward = item.AssetCode == "MENU_PROMO_CREDIT"
		case "commission":
			foundCommission = item.Currency == "MENU_PROMO_CREDIT"
		case "expiration":
			foundExpire = item.AssetCode == "MENU_PROMO_CREDIT"
		}
	}
	if !foundCharge || !foundReward || !foundCommission || !foundExpire {
		t.Fatalf("unexpected history aggregation: %+v", result.Items)
	}
}

func TestAuditHistory_ReturnsNewestItems(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s-audit?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.AuditLog{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	repo := repository.NewAuditRepository(db)
	if err := repo.Create(&models.AuditLog{
		ID:          "audit-1",
		ActorUserID: "user-1",
		ActorOrgID:  "org-1",
		Action:      "studio.job.create",
		TargetType:  "generation_job",
		TargetID:    "job-1",
		Status:      "success",
		CreatedAt:   time.Now().Add(-time.Minute),
	}); err != nil {
		t.Fatalf("create audit 1: %v", err)
	}
	if err := repo.Create(&models.AuditLog{
		ID:          "audit-2",
		ActorUserID: "user-1",
		ActorOrgID:  "org-1",
		Action:      "menu.share.post.create",
		TargetType:  "share_post",
		TargetID:    "share-1",
		Status:      "success",
		CreatedAt:   time.Now(),
	}); err != nil {
		t.Fatalf("create audit 2: %v", err)
	}
	service := NewService(nil, nil, nil, nil, audit.NewService(repo))
	result, err := service.AuditHistory("user-1", "org-1", "", "", 20, 0)
	if err != nil {
		t.Fatalf("AuditHistory() error = %v", err)
	}
	if result.Total != 2 || len(result.Items) != 2 || result.Items[0].ID != "audit-2" {
		t.Fatalf("unexpected audit history: %+v", result)
	}
}

type userPlatformMockServer struct {
	server *httptest.Server
	mu     sync.Mutex
}

func newUserPlatformMockServer(t *testing.T) *userPlatformMockServer {
	t.Helper()
	mock := &userPlatformMockServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/v1/wallet/accounts", mock.handleWalletAccounts)
	mux.HandleFunc("/internal/v1/wallet/ledger", mock.handleWalletLedger)
	mux.HandleFunc("/internal/v1/incentives/rewards", mock.handleRewards)
	mux.HandleFunc("/internal/v1/incentives/commissions", mock.handleCommissions)
	mock.server = httptest.NewServer(mux)
	return mock
}

func (m *userPlatformMockServer) handleWalletAccounts(w http.ResponseWriter, r *http.Request) {
	writeUserPlatformSuccess(w, map[string]any{
		"items": []map[string]any{
			{
				"id":                   "wallet-1",
				"billing_subject_type": "organization",
				"billing_subject_id":   "org-1",
				"asset_code":           "MENU_PROMO_CREDIT",
				"asset_type":           "reward_credit",
				"balance":              50,
				"status":               "active",
				"metadata":             "{}",
				"created_at":           time.Now().UTC().Format(time.RFC3339),
				"updated_at":           time.Now().UTC().Format(time.RFC3339),
			},
		},
	})
}

func (m *userPlatformMockServer) handleWalletLedger(w http.ResponseWriter, r *http.Request) {
	writeUserPlatformSuccess(w, map[string]any{
		"items": []map[string]any{
			{
				"id":                   "ledger-expire-1",
				"wallet_account_id":    "wallet-1",
				"billing_subject_type": "organization",
				"billing_subject_id":   "org-1",
				"asset_code":           "MENU_PROMO_CREDIT",
				"direction":            "debit",
				"amount":               20,
				"reason":               "asset_expire",
				"reference_type":       "wallet_bucket",
				"reference_id":         "bucket-1",
				"status":               "posted",
				"metadata":             "{}",
				"created_at":           time.Now().Add(-time.Hour).UTC().Format(time.RFC3339),
			},
		},
	})
}

func (m *userPlatformMockServer) handleRewards(w http.ResponseWriter, r *http.Request) {
	writeUserPlatformSuccess(w, map[string]any{
		"items": []map[string]any{
			{
				"id":                       "reward-1",
				"product_code":             "menu",
				"campaign_code":            "signup",
				"reward_type":              "signup_bonus",
				"beneficiary_subject_type": "organization",
				"beneficiary_subject_id":   "org-1",
				"asset_code":               "MENU_PROMO_CREDIT",
				"amount":                   30,
				"status":                   "issued",
				"reference_type":           "signup",
				"reference_id":             "user-1",
				"metadata":                 "{}",
				"created_at":               time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
				"updated_at":               time.Now().Add(-2 * time.Hour).UTC().Format(time.RFC3339),
			},
		},
	})
}

func (m *userPlatformMockServer) handleCommissions(w http.ResponseWriter, r *http.Request) {
	writeUserPlatformSuccess(w, map[string]any{
		"items": []map[string]any{
			{
				"id":                       "commission-1",
				"product_code":             "menu",
				"commission_type":          "referral",
				"beneficiary_subject_type": "organization",
				"beneficiary_subject_id":   "org-1",
				"settlement_subject_type":  "organization",
				"settlement_subject_id":    "org-1",
				"currency":                 "MENU_PROMO_CREDIT",
				"amount":                   20,
				"status":                   "earned",
				"reference_type":           "signup",
				"reference_id":             "conversion-1",
				"metadata":                 "{}",
				"created_at":               time.Now().Add(-3 * time.Hour).UTC().Format(time.RFC3339),
				"updated_at":               time.Now().Add(-3 * time.Hour).UTC().Format(time.RFC3339),
			},
		},
	})
}

func writeUserPlatformSuccess(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":      0,
		"message":   "success",
		"timestamp": time.Now().UnixMilli(),
		"data":      data,
	})
}
