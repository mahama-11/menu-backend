package user

import (
	"fmt"
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
