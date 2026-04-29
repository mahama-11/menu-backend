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

	service := NewService(nil, nil, studioRepo, client, nil, nil)
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
	service := NewService(nil, nil, nil, nil, nil, audit.NewService(repo))
	result, err := service.AuditHistory("user-1", "org-1", "", "", 20, 0)
	if err != nil {
		t.Fatalf("AuditHistory() error = %v", err)
	}
	if result.Total != 2 || len(result.Items) != 2 || result.Items[0].ID != "audit-2" {
		t.Fatalf("unexpected audit history: %+v", result)
	}
}

func TestCommercialOfferings_LoadsOfferingsAndWalletSummary(t *testing.T) {
	mock := newUserPlatformMockServer(t)
	defer mock.server.Close()

	client := platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-user-test",
		InternalServiceSecret: "test-secret",
	})

	service := NewService(nil, nil, nil, client, nil, nil)
	result, err := service.CommercialOfferings("org-1")
	if err != nil {
		t.Fatalf("CommercialOfferings() error = %v", err)
	}
	if result.ProductCode != "menu" || result.Offerings == nil || result.Offerings.Product == nil {
		t.Fatalf("unexpected offerings payload: %+v", result)
	}
	if len(result.Offerings.Packages) != 2 {
		t.Fatalf("packages = %d, want 2", len(result.Offerings.Packages))
	}
	if result.WalletSummary == nil || result.WalletSummary.BillingSubjectID != "org-1" {
		t.Fatalf("unexpected wallet summary: %+v", result.WalletSummary)
	}
}

func TestAssignCommercialPackage_SubscriptionGrantsQuota(t *testing.T) {
	mock := newUserPlatformMockServer(t)
	defer mock.server.Close()

	client := platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-user-test",
		InternalServiceSecret: "test-secret",
	})

	service := NewService(nil, nil, nil, client, nil, nil)
	result, err := service.AssignCommercialPackage("admin-user", "org-admin", AssignCommercialPackageInput{
		PackageCode: "menu.pkg.sub.basic.monthly",
		TargetOrgID: "org-subscription",
	})
	if err != nil {
		t.Fatalf("AssignCommercialPackage() error = %v", err)
	}
	if result.FulfillmentMode != "entitlement_grant" || result.Amount != 300 || result.GrantedQuotaUnits != 300 {
		t.Fatalf("unexpected subscription assignment result: %+v", result)
	}
	if result.WalletSummary == nil {
		t.Fatalf("missing subscription assignment artifacts: %+v", result)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.quotaGrants) != 1 || mock.quotaGrants[0].BillingSubjectID != "org-subscription" {
		t.Fatalf("quota grant request not captured: %+v", mock.quotaGrants)
	}
	if mock.quotaGrants[0].BillableItemCode != "menu.render.call" || mock.quotaGrants[0].Units != 300 {
		t.Fatalf("unexpected quota grant request: %+v", mock.quotaGrants[0])
	}
}

func TestAssignCommercialPackage_PermanentPackGrantsQuota(t *testing.T) {
	mock := newUserPlatformMockServer(t)
	defer mock.server.Close()

	client := platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-user-test",
		InternalServiceSecret: "test-secret",
	})

	service := NewService(nil, nil, nil, client, nil, nil)
	result, err := service.AssignCommercialPackage("admin-user", "org-admin", AssignCommercialPackageInput{
		PackageCode: "menu.pkg.pack.permanent.basic",
		TargetOrgID: "org-pack",
	})
	if err != nil {
		t.Fatalf("AssignCommercialPackage() error = %v", err)
	}
	if result.FulfillmentMode != "entitlement_grant" || result.Amount != 100 || result.GrantedQuotaUnits != 100 {
		t.Fatalf("unexpected permanent assignment result: %+v", result)
	}
	if result.WalletSummary == nil {
		t.Fatalf("missing permanent assignment artifacts: %+v", result)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.quotaGrants) != 1 || mock.quotaGrants[0].BillingSubjectID != "org-pack" {
		t.Fatalf("quota grant request not captured: %+v", mock.quotaGrants)
	}
	if mock.quotaGrants[0].BillableItemCode != "menu.render.call" || mock.quotaGrants[0].Units != 100 {
		t.Fatalf("unexpected quota grant request: %+v", mock.quotaGrants[0])
	}
}

func TestSimulateCommercialConsumption_ClosesAssignmentToSettlementLoop(t *testing.T) {
	mock := newUserPlatformMockServer(t)
	defer mock.server.Close()

	client := platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-user-test",
		InternalServiceSecret: "test-secret",
	})

	service := NewService(nil, nil, nil, client, nil, nil)
	if _, err := service.AssignCommercialPackage("admin-user", "org-admin", AssignCommercialPackageInput{
		PackageCode: "menu.pkg.sub.basic.monthly",
		TargetOrgID: "org-e2e",
		CycleKey:    "2026-04",
	}); err != nil {
		t.Fatalf("AssignCommercialPackage() error = %v", err)
	}

	result, err := service.SimulateCommercialConsumption("admin-user", "org-admin", SimulateCommercialConsumptionInput{
		TargetOrgID: "org-e2e",
		Units:       1,
	})
	if err != nil {
		t.Fatalf("SimulateCommercialConsumption() error = %v", err)
	}
	if result.BeforeWallet == nil || result.AfterWallet == nil || result.Settlement == nil || result.Reservation == nil {
		t.Fatalf("unexpected simulate consumption result: %+v", result)
	}
	if result.BeforeWallet.AllowanceBalance != result.AfterWallet.AllowanceBalance {
		t.Fatalf("quota consumption should not mutate wallet allowance summary: before=%+v after=%+v", result.BeforeWallet, result.AfterWallet)
	}
	if result.Settlement.QuotaConsumed != 1 || result.Settlement.WalletDebited != 0 {
		t.Fatalf("unexpected settlement result: %+v", result.Settlement)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.reservations) != 1 {
		t.Fatalf("reservations = %d, want 1", len(mock.reservations))
	}
	if len(mock.finalizations) != 1 {
		t.Fatalf("finalizations = %d, want 1", len(mock.finalizations))
	}
}

func TestCommercialOrderPayment_ConfirmsAndFulfillsSubscription(t *testing.T) {
	mock := newUserPlatformMockServer(t)
	defer mock.server.Close()

	client := platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-user-test",
		InternalServiceSecret: "test-secret",
	})
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s-commercial?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	commercialRepo := repository.NewCommercialRepository(db)
	if err := commercialRepo.AutoMigrate(); err != nil {
		t.Fatalf("commercial migrate: %v", err)
	}

	service := NewService(nil, commercialRepo, nil, client, nil, nil)
	mock.mu.Lock()
	mock.summaryByOrg["org-1"] = buildWalletSummaryState(
		"org-1",
		map[string]walletAssetState{
			"MENU_CASH": {
				AssetType:     "cash_balance",
				LifecycleType: "permanent",
				Balance:       1000000,
			},
		},
	)
	mock.mu.Unlock()
	orderView, err := service.CreateCommercialOrder("user-1", "org-1", CreateCommercialOrderInput{
		PackageCode: "menu.pkg.sub.basic.monthly",
	})
	if err != nil {
		t.Fatalf("CreateCommercialOrder() error = %v", err)
	}
	if orderView.Order == nil || orderView.Order.Status != "pending_payment" || orderView.Order.TotalAmount != 900 {
		t.Fatalf("unexpected order create result: %+v", orderView)
	}

	result, err := service.ConfirmCommercialOrderPayment("user-1", "org-1", orderView.Order.ID, ConfirmCommercialOrderPaymentInput{
		PaymentMethod: "promptpay",
		ProviderCode:  "manual_success",
	})
	if err != nil {
		t.Fatalf("ConfirmCommercialOrderPayment() error = %v", err)
	}
	if result.Order == nil || result.Order.Status != "fulfilled" || result.Order.PaymentStatus != "succeeded" || result.Order.FulfillmentStatus != "succeeded" {
		t.Fatalf("unexpected fulfilled order result: %+v", result)
	}
	if result.Payment == nil || result.Payment.Status != "succeeded" {
		t.Fatalf("unexpected payment result: %+v", result.Payment)
	}
	if result.Fulfillment == nil || result.Fulfillment.FulfillmentMode != "entitlement_grant" || result.Fulfillment.Amount != 300 {
		t.Fatalf("unexpected fulfillment result: %+v", result.Fulfillment)
	}
	if result.WalletSummary == nil {
		t.Fatalf("unexpected wallet summary after fulfillment: %+v", result.WalletSummary)
	}
	if result.WalletSummary.PermanentBalance != 999100 {
		t.Fatalf("unexpected cash balance after payment: %+v", result.WalletSummary)
	}

	loaded, err := service.GetCommercialOrder("org-1", orderView.Order.ID)
	if err != nil {
		t.Fatalf("GetCommercialOrder() error = %v", err)
	}
	if loaded.Payment == nil || loaded.Fulfillment == nil {
		t.Fatalf("expected payment and fulfillment in loaded order: %+v", loaded)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()
	if len(mock.quotaGrants) == 0 || mock.quotaGrants[0].BillingSubjectID != "org-1" {
		t.Fatalf("expected quota grant fulfillment request: %+v", mock.quotaGrants)
	}
}

func TestCommercialOrderPayment_UsesCreditsWhenRequested(t *testing.T) {
	mock := newUserPlatformMockServer(t)
	defer mock.server.Close()

	client := platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-user-test",
		InternalServiceSecret: "test-secret",
	})
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s-commercial-credit?mode=memory&cache=shared", t.Name())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	commercialRepo := repository.NewCommercialRepository(db)
	if err := commercialRepo.AutoMigrate(); err != nil {
		t.Fatalf("commercial migrate: %v", err)
	}

	service := NewService(nil, commercialRepo, nil, client, nil, nil)
	mock.mu.Lock()
	mock.summaryByOrg["org-1"] = buildWalletSummaryState(
		"org-1",
		map[string]walletAssetState{
			"MENU_CREDIT": {
				AssetType:     "wallet_credit",
				LifecycleType: "permanent",
				Balance:       1000,
			},
		},
	)
	mock.mu.Unlock()
	orderView, err := service.CreateCommercialOrder("user-1", "org-1", CreateCommercialOrderInput{
		PackageCode: "menu.pkg.sub.basic.monthly",
	})
	if err != nil {
		t.Fatalf("CreateCommercialOrder() error = %v", err)
	}

	result, err := service.ConfirmCommercialOrderPayment("user-1", "org-1", orderView.Order.ID, ConfirmCommercialOrderPaymentInput{
		PaymentMethod:    "wallet_credits",
		ProviderCode:     "platform_wallet",
		PaymentAssetCode: "MENU_CREDIT",
	})
	if err != nil {
		t.Fatalf("ConfirmCommercialOrderPayment() error = %v", err)
	}
	if result.Payment == nil || result.Payment.Amount != 90 || result.Payment.Currency != "MENU_CREDIT" {
		t.Fatalf("unexpected credit payment result: %+v", result.Payment)
	}
	if result.WalletSummary == nil || result.WalletSummary.PermanentBalance != 910 {
		t.Fatalf("unexpected wallet summary after credit payment: %+v", result.WalletSummary)
	}
}

type userPlatformMockServer struct {
	server             *httptest.Server
	mu                 sync.Mutex
	lastLedgerPost     *platform.PostWalletLedgerInput
	lastCycleAllowance *platform.GrantCycleAllowanceInput
	quotaGrants        []platform.GrantQuotaInput
	capabilityGrants   []platform.GrantCapabilityInput
	summaryByOrg       map[string]map[string]any
	reservations       []platform.ReserveInput
	finalizations      []platform.FinalizeInput
}

type walletAssetState struct {
	AssetType     string
	LifecycleType string
	Balance       int64
}

func newUserPlatformMockServer(t *testing.T) *userPlatformMockServer {
	t.Helper()
	mock := &userPlatformMockServer{summaryByOrg: map[string]map[string]any{}}
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/v1/catalog/offerings", mock.handleCatalogOfferings)
	mux.HandleFunc("/internal/v1/controls/quota/policies", mock.handleQuotaPolicies)
	mux.HandleFunc("/internal/v1/controls/quota/grants", mock.handleQuotaGrants)
	mux.HandleFunc("/internal/v1/controls/capability/policies", mock.handleCapabilityPolicies)
	mux.HandleFunc("/internal/v1/controls/capability/grants", mock.handleCapabilityGrants)
	mux.HandleFunc("/internal/v1/controls/reservations", mock.handleReservations)
	mux.HandleFunc("/internal/v1/wallet/accounts", mock.handleWalletAccounts)
	mux.HandleFunc("/internal/v1/wallet/ledger", mock.handleWalletLedger)
	mux.HandleFunc("/internal/v1/wallet/summary", mock.handleWalletSummary)
	mux.HandleFunc("/internal/v1/wallet/cycle-allowances", mock.handleWalletCycleAllowances)
	mux.HandleFunc("/internal/v1/metering/finalizations", mock.handleMeteringFinalizations)
	mux.HandleFunc("/internal/v1/incentives/rewards", mock.handleRewards)
	mux.HandleFunc("/internal/v1/incentives/commissions", mock.handleCommissions)
	mock.server = httptest.NewServer(mux)
	return mock
}

func (m *userPlatformMockServer) handleReservations(w http.ResponseWriter, r *http.Request) {
	var req platform.ReserveInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	m.reservations = append(m.reservations, req)
	m.mu.Unlock()
	writeUserPlatformSuccess(w, map[string]any{
		"id":                   "reservation-1",
		"resource_type":        req.ResourceType,
		"billing_subject_type": req.BillingSubjectType,
		"billing_subject_id":   req.BillingSubjectID,
		"billable_item_code":   req.BillableItemCode,
		"reservation_key":      req.ReservationKey,
		"finalization_id":      nil,
		"units":                req.Units,
		"status":               "reserved",
		"reference_id":         req.ReferenceID,
		"metadata":             req.Metadata,
		"created_at":           time.Now().UTC().Format(time.RFC3339),
		"updated_at":           time.Now().UTC().Format(time.RFC3339),
	})
}

func (m *userPlatformMockServer) handleCatalogOfferings(w http.ResponseWriter, r *http.Request) {
	writeUserPlatformSuccess(w, map[string]any{
		"product": map[string]any{
			"id":         "prod-menu",
			"code":       "menu",
			"name":       "Menu",
			"status":     "active",
			"owner_team": "menu",
			"metadata":   "{}",
			"created_at": time.Now().UTC().Format(time.RFC3339),
			"updated_at": time.Now().UTC().Format(time.RFC3339),
		},
		"skus": []map[string]any{
			{
				"id":           "sku-sub-basic",
				"product_id":   "prod-menu",
				"code":         "menu.sku.sub.basic.monthly",
				"name":         "Menu Basic Monthly",
				"sku_type":     "subscription",
				"billing_mode": "recurring",
				"currency":     "CNY",
				"list_price":   900,
				"status":       "active",
				"metadata":     `{"package_code":"menu.pkg.sub.basic.monthly"}`,
				"created_at":   time.Now().UTC().Format(time.RFC3339),
				"updated_at":   time.Now().UTC().Format(time.RFC3339),
			},
			{
				"id":           "sku-pack-basic",
				"product_id":   "prod-menu",
				"code":         "menu.sku.pack.permanent.basic",
				"name":         "Menu Permanent Basic Pack",
				"sku_type":     "resource_pack",
				"billing_mode": "one_time",
				"currency":     "CNY",
				"list_price":   1900,
				"status":       "active",
				"metadata":     `{"package_code":"menu.pkg.pack.permanent.basic","quota_units":100}`,
				"created_at":   time.Now().UTC().Format(time.RFC3339),
				"updated_at":   time.Now().UTC().Format(time.RFC3339),
			},
		},
		"packages": []map[string]any{
			{
				"id":           "pkg-sub-basic",
				"product_id":   "prod-menu",
				"code":         "menu.pkg.sub.basic.monthly",
				"name":         "Basic Monthly Package",
				"package_type": "subscription",
				"status":       "active",
				"metadata":     `{"sku_code":"menu.sku.sub.basic.monthly","monthly_calls":300}`,
				"created_at":   time.Now().UTC().Format(time.RFC3339),
				"updated_at":   time.Now().UTC().Format(time.RFC3339),
			},
			{
				"id":           "pkg-pack-basic",
				"product_id":   "prod-menu",
				"code":         "menu.pkg.pack.permanent.basic",
				"name":         "Permanent Basic Pack",
				"package_type": "permanent_pack",
				"status":       "active",
				"metadata":     `{"sku_code":"menu.sku.pack.permanent.basic","quota_units":100}`,
				"created_at":   time.Now().UTC().Format(time.RFC3339),
				"updated_at":   time.Now().UTC().Format(time.RFC3339),
			},
		},
		"billable_items": []map[string]any{
			{
				"id":               "billable-1",
				"product_id":       "prod-menu",
				"code":             "menu_studio_single_generate",
				"name":             "Menu Studio Single Generate",
				"meter_unit":       "action",
				"billing_scope":    "organization",
				"settlement_mode":  "included_then_overage",
				"pricing_behavior": "fixed",
				"status":           "active",
				"metadata":         "{}",
				"created_at":       time.Now().UTC().Format(time.RFC3339),
				"updated_at":       time.Now().UTC().Format(time.RFC3339),
			},
		},
		"rate_cards": []map[string]any{
			{
				"id":           "rate-1",
				"product_id":   "prod-menu",
				"code":         "menu.sku.sub.basic.monthly.v1",
				"target_type":  "sku",
				"target_id":    "sku-sub-basic",
				"price_model":  "flat",
				"currency":     "CNY",
				"price_config": `{"unit_amount":900}`,
				"version":      1,
				"status":       "active",
				"metadata":     `{"package_code":"menu.pkg.sub.basic.monthly"}`,
				"created_at":   time.Now().UTC().Format(time.RFC3339),
				"updated_at":   time.Now().UTC().Format(time.RFC3339),
			},
		},
		"asset_definitions": []map[string]any{
			{
				"asset_code":          "MENU_CASH",
				"product_code":        "menu",
				"asset_type":          "cash_balance",
				"lifecycle_type":      "permanent",
				"default_expire_days": 0,
				"reset_cycle":         "",
				"status":              "active",
				"description":         "Menu cash balance",
				"metadata":            "{}",
				"created_at":          time.Now().UTC().Format(time.RFC3339),
				"updated_at":          time.Now().UTC().Format(time.RFC3339),
			},
			{
				"asset_code":          "MENU_MONTHLY_ALLOWANCE",
				"product_code":        "menu",
				"asset_type":          "subscription_allowance",
				"lifecycle_type":      "cycle_reset",
				"default_expire_days": 0,
				"reset_cycle":         "monthly",
				"status":              "active",
				"description":         "Menu monthly allowance",
				"metadata":            "{}",
				"created_at":          time.Now().UTC().Format(time.RFC3339),
				"updated_at":          time.Now().UTC().Format(time.RFC3339),
			},
			{
				"asset_code":          "MENU_CREDIT",
				"product_code":        "menu",
				"asset_type":          "credit",
				"lifecycle_type":      "permanent",
				"default_expire_days": 0,
				"reset_cycle":         "",
				"status":              "active",
				"description":         "Menu permanent credit",
				"metadata":            "{}",
				"created_at":          time.Now().UTC().Format(time.RFC3339),
				"updated_at":          time.Now().UTC().Format(time.RFC3339),
			},
		},
		"allowance_policies": []map[string]any{},
	})
}

func (m *userPlatformMockServer) handleQuotaPolicies(w http.ResponseWriter, r *http.Request) {
	packageCode := r.URL.Query().Get("package_code")
	items := []map[string]any{}
	switch packageCode {
	case "menu.pkg.sub.basic.monthly":
		items = append(items, map[string]any{
			"id": "quota_policy_menu_sub_basic_monthly", "product_code": "menu", "package_code": packageCode,
			"billable_item_code": "menu.render.call", "grant_mode": "cycle_reset", "units": 300, "reset_cycle": "monthly",
			"status": "active", "metadata": `{"tier":"basic"}`, "created_at": time.Now().UTC().Format(time.RFC3339), "updated_at": time.Now().UTC().Format(time.RFC3339),
		})
	case "menu.pkg.pack.permanent.basic":
		items = append(items, map[string]any{
			"id": "quota_policy_menu_pack_permanent_basic", "product_code": "menu", "package_code": packageCode,
			"billable_item_code": "menu.render.call", "grant_mode": "one_time", "units": 100, "reset_cycle": "",
			"status": "active", "metadata": `{"tier":"pack_basic"}`, "created_at": time.Now().UTC().Format(time.RFC3339), "updated_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
	writeUserPlatformSuccess(w, map[string]any{"items": items})
}

func (m *userPlatformMockServer) handleQuotaGrants(w http.ResponseWriter, r *http.Request) {
	var req platform.GrantQuotaInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	m.quotaGrants = append(m.quotaGrants, req)
	m.mu.Unlock()
	writeUserPlatformSuccess(w, map[string]any{
		"id": "quota-ledger-1", "billing_subject_type": req.BillingSubjectType, "billing_subject_id": req.BillingSubjectID,
		"billable_item_code": req.BillableItemCode, "units": req.Units, "direction": "grant", "status": "active",
		"created_at": time.Now().UTC().Format(time.RFC3339), "updated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (m *userPlatformMockServer) handleCapabilityPolicies(w http.ResponseWriter, r *http.Request) {
	packageCode := r.URL.Query().Get("package_code")
	items := []map[string]any{}
	if packageCode == "menu.pkg.sub.basic.monthly" {
		items = append(items, map[string]any{
			"id": "cap_policy_menu_sub_basic_template_scope", "product_code": "menu", "package_code": packageCode,
			"capability_code": "template_scope", "grant_value": "free_templates", "status": "active", "metadata": `{"tier":"basic"}`,
			"created_at": time.Now().UTC().Format(time.RFC3339), "updated_at": time.Now().UTC().Format(time.RFC3339),
		})
	}
	writeUserPlatformSuccess(w, map[string]any{"items": items})
}

func (m *userPlatformMockServer) handleCapabilityGrants(w http.ResponseWriter, r *http.Request) {
	var req platform.GrantCapabilityInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	m.capabilityGrants = append(m.capabilityGrants, req)
	m.mu.Unlock()
	writeUserPlatformSuccess(w, map[string]any{
		"id": "capability-grant-1", "product_code": req.ProductCode, "billing_subject_type": req.BillingSubjectType,
		"billing_subject_id": req.BillingSubjectID, "capability_code": req.CapabilityCode, "grant_value": req.GrantValue,
		"status": "active", "metadata": req.Metadata, "created_at": time.Now().UTC().Format(time.RFC3339), "updated_at": time.Now().UTC().Format(time.RFC3339),
	})
}

func (m *userPlatformMockServer) handleWalletAccounts(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("product_code") == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
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
	if r.Method == http.MethodPost {
		var req platform.PostWalletLedgerInput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		m.mu.Lock()
		m.lastLedgerPost = &req
		state := summaryToState(m.summaryByOrg[req.BillingSubjectID])
		item := state[req.AssetCode]
		if item.AssetType == "" {
			item.AssetType = defaultString(req.AssetType, "cash_balance")
		}
		if item.LifecycleType == "" {
			item.LifecycleType = "permanent"
		}
		if req.Direction == "debit" {
			if item.Balance < req.Amount {
				m.mu.Unlock()
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"code":       4001,
					"message":    "insufficient wallet balance",
					"error":      "insufficient wallet balance",
					"error_code": "WALLET_INSUFFICIENT_BALANCE",
					"timestamp":  time.Now().UnixMilli(),
				})
				return
			}
			item.Balance -= req.Amount
		} else {
			item.Balance += req.Amount
		}
		state[req.AssetCode] = item
		m.summaryByOrg[req.BillingSubjectID] = buildWalletSummaryState(req.BillingSubjectID, state)
		m.mu.Unlock()
		writeUserPlatformSuccess(w, map[string]any{
			"ledger": map[string]any{
				"id":                   "ledger-posted-1",
				"wallet_account_id":    "wallet-pack-1",
				"billing_subject_type": req.BillingSubjectType,
				"billing_subject_id":   req.BillingSubjectID,
				"asset_code":           req.AssetCode,
				"direction":            req.Direction,
				"amount":               req.Amount,
				"reason":               req.Reason,
				"reference_type":       req.ReferenceType,
				"reference_id":         req.ReferenceID,
				"status":               "posted",
				"metadata":             req.Metadata,
				"created_at":           time.Now().UTC().Format(time.RFC3339),
			},
			"account": map[string]any{
				"id":                   "wallet-pack-1",
				"billing_subject_type": req.BillingSubjectType,
				"billing_subject_id":   req.BillingSubjectID,
				"asset_code":           req.AssetCode,
				"asset_type":           req.AssetType,
				"balance":              summaryAssetBalance(m.summaryByOrg[req.BillingSubjectID], req.AssetCode),
				"status":               "active",
				"metadata":             "{}",
				"created_at":           time.Now().UTC().Format(time.RFC3339),
				"updated_at":           time.Now().UTC().Format(time.RFC3339),
			},
		})
		return
	}
	if r.URL.Query().Get("product_code") == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
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

func (m *userPlatformMockServer) handleWalletSummary(w http.ResponseWriter, r *http.Request) {
	orgID := r.URL.Query().Get("billing_subject_id")
	m.mu.Lock()
	summary, ok := m.summaryByOrg[orgID]
	m.mu.Unlock()
	if !ok {
		summary = map[string]any{
			"billing_subject_type": "organization",
			"billing_subject_id":   orgID,
			"product_code":         "menu",
			"total_balance":        50,
			"permanent_balance":    20,
			"reward_balance":       0,
			"allowance_balance":    30,
			"assets": []map[string]any{
				{
					"asset_code":        "MENU_MONTHLY_ALLOWANCE",
					"asset_type":        "subscription_allowance",
					"lifecycle_type":    "cycle_reset",
					"account_balance":   30,
					"available_balance": 30,
					"expiring_balance":  0,
					"next_expires_at":   nil,
				},
				{
					"asset_code":        "MENU_CREDIT",
					"asset_type":        "credit",
					"lifecycle_type":    "permanent",
					"account_balance":   20,
					"available_balance": 20,
					"expiring_balance":  0,
					"next_expires_at":   nil,
				},
			},
		}
	}
	writeUserPlatformSuccess(w, summary)
}

func (m *userPlatformMockServer) handleWalletCycleAllowances(w http.ResponseWriter, r *http.Request) {
	var req platform.GrantCycleAllowanceInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	m.lastCycleAllowance = &req
	state := summaryToState(m.summaryByOrg[req.BillingSubjectID])
	state[req.AssetCode] = walletAssetState{
		AssetType:     "subscription_allowance",
		LifecycleType: "cycle_reset",
		Balance:       req.Amount,
	}
	m.summaryByOrg[req.BillingSubjectID] = buildWalletSummaryState(req.BillingSubjectID, state)
	m.mu.Unlock()
	writeUserPlatformSuccess(w, map[string]any{
		"bucket": map[string]any{
			"id":                   "bucket-subscription-1",
			"wallet_account_id":    "wallet-subscription-1",
			"billing_subject_type": req.BillingSubjectType,
			"billing_subject_id":   req.BillingSubjectID,
			"asset_code":           req.AssetCode,
			"asset_type":           "subscription_allowance",
			"lifecycle_type":       "cycle_reset",
			"source_type":          "",
			"source_id":            "",
			"cycle_key":            req.CycleKey,
			"balance":              req.Amount,
			"status":               "active",
			"metadata":             req.Metadata,
			"created_at":           time.Now().UTC().Format(time.RFC3339),
			"updated_at":           time.Now().UTC().Format(time.RFC3339),
		},
		"account": map[string]any{
			"id":                   "wallet-subscription-1",
			"billing_subject_type": req.BillingSubjectType,
			"billing_subject_id":   req.BillingSubjectID,
			"asset_code":           req.AssetCode,
			"asset_type":           "subscription_allowance",
			"balance":              req.Amount,
			"status":               "active",
			"metadata":             "{}",
			"created_at":           time.Now().UTC().Format(time.RFC3339),
			"updated_at":           time.Now().UTC().Format(time.RFC3339),
		},
	})
}

func (m *userPlatformMockServer) handleMeteringFinalizations(w http.ResponseWriter, r *http.Request) {
	var req platform.FinalizeInput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	m.mu.Lock()
	m.finalizations = append(m.finalizations, req)
	m.mu.Unlock()
	writeUserPlatformSuccess(w, map[string]any{
		"reservation": map[string]any{
			"id":                   req.ReservationID,
			"resource_type":        "quota",
			"billing_subject_type": req.BillingSubjectType,
			"billing_subject_id":   req.BillingSubjectID,
			"billable_item_code":   req.BillableItemCode,
			"reservation_key":      fmt.Sprintf("reservation-key:%s", req.ReservationID),
			"finalization_id":      req.FinalizationID,
			"units":                req.UsageUnits,
			"status":               "committed",
			"reference_id":         req.SourceID,
			"metadata":             req.Dimensions,
			"created_at":           time.Now().UTC().Format(time.RFC3339),
			"updated_at":           time.Now().UTC().Format(time.RFC3339),
		},
		"event": map[string]any{
			"event_id": req.EventID,
			"status":   "settled",
		},
		"settlement": map[string]any{
			"id":                   "settlement-1",
			"event_id":             req.EventID,
			"request_id":           "req-1",
			"trace_id":             "trace-1",
			"billing_subject_type": req.BillingSubjectType,
			"billing_subject_id":   req.BillingSubjectID,
			"product_code":         req.ProductCode,
			"billable_item_code":   req.BillableItemCode,
			"billing_profile_id":   "bp-1",
			"commercial_entity_id": "ce-1",
			"merchant_account_id":  "ma-1",
			"settlement_mode":      "included_then_overage",
			"currency":             "MENU_CREDIT",
			"gross_amount":         req.UsageUnits,
			"discount_amount":      0,
			"net_amount":           req.UsageUnits,
			"quota_consumed":       req.UsageUnits,
			"credits_consumed":     0,
			"wallet_asset_code":    "",
			"wallet_debited":       0,
			"billing_amount":       0,
			"reward_amount":        0,
			"commission_amount":    0,
			"status":               "posted",
			"snapshot":             req.Dimensions,
			"created_at":           time.Now().UTC().Format(time.RFC3339),
			"updated_at":           time.Now().UTC().Format(time.RFC3339),
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

func int64Value(value any) int64 {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int64:
		return typed
	case float64:
		return int64(typed)
	default:
		return 0
	}
}

func rebuildSummaryAssets(permanentBalance, rewardBalance, allowanceBalance int64) []map[string]any {
	assets := make([]map[string]any, 0, 3)
	if allowanceBalance > 0 {
		assets = append(assets, map[string]any{
			"asset_code":        "MENU_MONTHLY_ALLOWANCE",
			"asset_type":        "subscription_allowance",
			"lifecycle_type":    "cycle_reset",
			"account_balance":   allowanceBalance,
			"available_balance": allowanceBalance,
			"expiring_balance":  0,
			"next_expires_at":   nil,
		})
	}
	if permanentBalance > 0 {
		assets = append(assets, map[string]any{
			"asset_code":        "MENU_CREDIT",
			"asset_type":        "credit",
			"lifecycle_type":    "permanent",
			"account_balance":   permanentBalance,
			"available_balance": permanentBalance,
			"expiring_balance":  0,
			"next_expires_at":   nil,
		})
	}
	if rewardBalance > 0 {
		assets = append(assets, map[string]any{
			"asset_code":        "MENU_PROMO_CREDIT",
			"asset_type":        "reward_credit",
			"lifecycle_type":    "expiring",
			"account_balance":   rewardBalance,
			"available_balance": rewardBalance,
			"expiring_balance":  rewardBalance,
			"next_expires_at":   time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339),
		})
	}
	return assets
}

func buildWalletSummaryState(orgID string, state map[string]walletAssetState) map[string]any {
	var totalBalance int64
	var permanentBalance int64
	var rewardBalance int64
	var allowanceBalance int64
	assets := make([]map[string]any, 0, len(state))
	for assetCode, item := range state {
		if item.Balance <= 0 {
			continue
		}
		totalBalance += item.Balance
		switch item.LifecycleType {
		case "cycle_reset":
			allowanceBalance += item.Balance
		case "expiring":
			rewardBalance += item.Balance
		default:
			permanentBalance += item.Balance
		}
		asset := map[string]any{
			"asset_code":        assetCode,
			"asset_type":        item.AssetType,
			"lifecycle_type":    item.LifecycleType,
			"account_balance":   item.Balance,
			"available_balance": item.Balance,
			"expiring_balance":  0,
			"next_expires_at":   nil,
		}
		if item.LifecycleType == "expiring" {
			asset["expiring_balance"] = item.Balance
			asset["next_expires_at"] = time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
		}
		assets = append(assets, asset)
	}
	return map[string]any{
		"billing_subject_type": "organization",
		"billing_subject_id":   orgID,
		"product_code":         "menu",
		"total_balance":        totalBalance,
		"permanent_balance":    permanentBalance,
		"reward_balance":       rewardBalance,
		"allowance_balance":    allowanceBalance,
		"assets":               assets,
	}
}

func summaryToState(summary map[string]any) map[string]walletAssetState {
	state := map[string]walletAssetState{}
	if summary == nil {
		return state
	}
	assets, _ := summary["assets"].([]map[string]any)
	if len(assets) == 0 {
		if typed, ok := summary["assets"].([]any); ok {
			for _, item := range typed {
				asset, ok := item.(map[string]any)
				if !ok {
					continue
				}
				assetCode, _ := asset["asset_code"].(string)
				state[assetCode] = walletAssetState{
					AssetType:     stringValue(asset["asset_type"]),
					LifecycleType: stringValue(asset["lifecycle_type"]),
					Balance:       int64Value(asset["available_balance"]),
				}
			}
			return state
		}
	}
	for _, asset := range assets {
		assetCode, _ := asset["asset_code"].(string)
		state[assetCode] = walletAssetState{
			AssetType:     stringValue(asset["asset_type"]),
			LifecycleType: stringValue(asset["lifecycle_type"]),
			Balance:       int64Value(asset["available_balance"]),
		}
	}
	return state
}

func summaryAssetBalance(summary map[string]any, assetCode string) int64 {
	state := summaryToState(summary)
	return state[assetCode].Balance
}

func stringValue(value any) string {
	typed, _ := value.(string)
	return typed
}
