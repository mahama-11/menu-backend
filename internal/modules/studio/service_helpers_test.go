package studio

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/models"
	"menu-service/internal/platform"
	"menu-service/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func seedStudioTemplateVersion(t *testing.T, db *gorm.DB, catalogID, versionID string, prompts map[string]string, executionProfile string) {
	t.Helper()
	if err := db.Create(&models.TemplateCatalog{
		ID:               catalogID,
		Slug:             strings.ToLower(catalogID),
		Name:             catalogID,
		Status:           "active",
		Scope:            "public",
		PlanRequired:     "basic",
		CurrentVersionID: versionID,
	}).Error; err != nil {
		t.Fatalf("create template catalog: %v", err)
	}
	promptJSON, _ := json.Marshal(prompts)
	if err := db.Create(&models.TemplateCatalogVersion{
		ID:                   versionID,
		TemplateCatalogID:    catalogID,
		VersionNo:            1,
		Status:               "active",
		PromptTemplatesJSON:  string(promptJSON),
		ExecutionProfileJSON: executionProfile,
	}).Error; err != nil {
		t.Fatalf("create template version: %v", err)
	}
}

func newStudioTestService(t *testing.T) *Service {
	service, mock := newBilledStudioTestService(t)
	t.Cleanup(mock.server.Close)
	return service
}

func newBilledStudioTestService(t *testing.T) (*Service, *platformMockServer) {
	t.Helper()
	mock := newPlatformMockServer(t)
	client := platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-test",
		InternalServiceSecret: "test-secret",
	})
	service, _ := newStudioTestServiceWithConfig(t, config.StudioConfig{
		ProductCode:            "menu",
		ResourceType:           "credits",
		SingleBillableItem:     "menu.render.call",
		RefinementBillableItem: "menu.render.call",
		VariationBillableItem:  "menu.render.call",
	}, client)
	return service, mock
}

func newStudioTestServiceWithConfig(t *testing.T, cfg config.StudioConfig, platformClient *platform.Client) (*Service, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(
		&models.AuditLog{},
		&models.UserPreference{},
		&models.Activity{},
		&models.StudioAsset{},
		&models.StylePreset{},
		&models.GenerationJob{},
		&models.GenerationVariant{},
		&models.StudioChargeIntent{},
		&models.SharePost{},
		&models.TemplateCatalog{},
		&models.TemplateCatalogVersion{},
		&models.TemplateCatalogExample{},
		&models.TemplateFavorite{},
		&models.TemplateUsageEvent{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewService(
		repository.NewStudioRepository(db),
		repository.NewTemplateCenterRepository(db),
		repository.NewShareRepository(db),
		repository.NewUserRepository(db),
		nil,
		platformClient,
		config.AppConfig{
			CreditsAssetCode:   "MENU_CREDIT",
			RewardAssetCode:    "MENU_PROMO_CREDIT",
			AllowanceAssetCode: "MENU_MONTHLY_ALLOWANCE",
		},
		cfg,
		config.SecurityConfig{
			JWTSecret:     "jwt-test-secret",
			EncryptionKey: "enc-test-secret",
		},
	), db
}

type platformMockServer struct {
	server        *httptest.Server
	mu            sync.Mutex
	resolveCount  int
	runtimeCount  int
	lastRuntime   map[string]any
	sessionCount  int
	reserveCount  int
	finalizeCount int
	releaseCount  int
	channelCount  int
}

func newPlatformMockServer(t *testing.T) *platformMockServer {
	t.Helper()
	mock := &platformMockServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/v1/commercial/route/resolve", mock.handleResolveRoute)
	mux.HandleFunc("/internal/v1/storage/assets", mock.handleUploadAsset)
	mux.HandleFunc("/internal/v1/runtime/charge-sessions", mock.handleCreateChargeSession)
	mux.HandleFunc("/internal/v1/runtime/jobs", mock.handleCreateRuntimeJob)
	mux.HandleFunc("/internal/v1/controls/reservations", mock.handleReserve)
	mux.HandleFunc("/internal/v1/metering/finalizations", mock.handleFinalize)
	mux.HandleFunc("/internal/v1/controls/reservations/", mock.handleReservationAction)
	mux.HandleFunc("/internal/v1/incentives/channel-events/charges", mock.handleChannelCharge)
	mock.server = httptest.NewServer(mux)
	return mock
}

func (m *platformMockServer) handleUploadAsset(w http.ResponseWriter, _ *http.Request) {
	writePlatformSuccess(w, map[string]any{
		"storage_key": "menu/studio-assets/test.png",
		"mime_type":   "image/png",
		"file_size":   5,
	})
}

func (m *platformMockServer) reserveCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.reserveCount
}

func (m *platformMockServer) finalizeCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.finalizeCount
}

func (m *platformMockServer) releaseCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.releaseCount
}

func (m *platformMockServer) channelChargeCalls() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.channelCount
}

func (m *platformMockServer) handleResolveRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	m.mu.Lock()
	m.resolveCount++
	call := m.resolveCount
	m.mu.Unlock()
	writePlatformSuccess(w, map[string]any{
		"billing_profile_key":   "menu-service",
		"routing_policy_key":    "studio-default",
		"merchant_account_id":   "merchant-1",
		"provider_channel":      "volcengine",
		"route_snapshot":        fmt.Sprintf("{\"route\":\"snapshot-%d\"}", call),
		"settlement_currency":   "CNY",
		"wallet_asset_code":     "MENU_CREDIT",
		"commission_asset_code": "COMMISSION_LEDGER",
	})
}

func (m *platformMockServer) handleCreateChargeSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	m.mu.Lock()
	m.sessionCount++
	call := m.sessionCount
	m.mu.Unlock()
	writePlatformSuccess(w, map[string]any{
		"id":                 fmt.Sprintf("charge-session-%d", call),
		"source_type":        "menu_generation_job",
		"source_id":          fmt.Sprintf("job-%d", call),
		"product_code":       "menu-service",
		"organization_id":    "org-1",
		"user_id":            "user-1",
		"billable_item_code": "menu.render.call",
		"resource_type":      "credits",
		"status":             "created",
		"reservation_key":    fmt.Sprintf("studio:reservation:job-%d", call),
		"estimated_units":    1,
		"route_snapshot":     "{}",
		"metadata":           "{}",
		"created_at":         time.Now().UTC().Format(time.RFC3339),
		"updated_at":         time.Now().UTC().Format(time.RFC3339),
	})
}

func (m *platformMockServer) handleCreateRuntimeJob(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	var payload map[string]any
	_ = json.NewDecoder(r.Body).Decode(&payload)
	m.mu.Lock()
	m.runtimeCount++
	call := m.runtimeCount
	m.lastRuntime = payload
	m.mu.Unlock()
	writePlatformSuccess(w, map[string]any{
		"id":                fmt.Sprintf("runtime-job-%d", call),
		"product_code":      "menu-service",
		"task_type":         "image_generation",
		"provider_code":     "mock",
		"provider_mode":     "sync",
		"organization_id":   "org-1",
		"user_id":           "user-1",
		"source_type":       "menu_generation_job",
		"source_id":         fmt.Sprintf("job-%d", call),
		"charge_session_id": fmt.Sprintf("charge-session-%d", call),
		"status":            "queued",
		"stage":             "queued",
		"stage_message":     "Runtime job queued",
		"input_manifest":    "{}",
		"output_manifest":   "",
		"route_snapshot":    "{}",
		"metadata":          "{}",
		"priority":          0,
		"attempt_count":     0,
		"max_attempts":      3,
		"created_at":        time.Now().UTC().Format(time.RFC3339),
		"updated_at":        time.Now().UTC().Format(time.RFC3339),
	})
}

func (m *platformMockServer) lastRuntimePayload() map[string]any {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastRuntime == nil {
		return nil
	}
	out := make(map[string]any, len(m.lastRuntime))
	maps.Copy(out, m.lastRuntime)
	return out
}

func (m *platformMockServer) handleReserve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	m.mu.Lock()
	m.reserveCount++
	call := m.reserveCount
	m.mu.Unlock()
	writePlatformSuccess(w, map[string]any{
		"id":                   fmt.Sprintf("reservation-%d", call),
		"resource_type":        "credits",
		"billing_subject_type": "organization",
		"billing_subject_id":   "org-1",
		"billable_item_code":   "menu.render.call",
		"units":                1,
		"status":               "reserved",
		"reference_id":         fmt.Sprintf("intent-%d", call),
		"metadata":             "{}",
		"created_at":           time.Now().UTC().Format(time.RFC3339),
		"updated_at":           time.Now().UTC().Format(time.RFC3339),
	})
}

func (m *platformMockServer) handleFinalize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	m.mu.Lock()
	m.finalizeCount++
	call := m.finalizeCount
	m.mu.Unlock()
	writePlatformSuccess(w, map[string]any{
		"reservation": map[string]any{
			"id":              fmt.Sprintf("reservation-%d", call),
			"status":          "finalized",
			"resource_type":   "credits",
			"reference_id":    fmt.Sprintf("intent-%d", call),
			"metadata":        "{}",
			"created_at":      time.Now().UTC().Format(time.RFC3339),
			"updated_at":      time.Now().UTC().Format(time.RFC3339),
			"committed_at":    time.Now().UTC().Format(time.RFC3339),
			"reservation_key": nil,
		},
		"event": map[string]any{
			"event_id": fmt.Sprintf("evt-%d", call),
		},
		"settlement": map[string]any{
			"id":                fmt.Sprintf("settlement-%d", call),
			"currency":          "MENU_CREDIT",
			"gross_amount":      100,
			"discount_amount":   0,
			"net_amount":        100,
			"quota_consumed":    0,
			"credits_consumed":  0,
			"wallet_asset_code": "MENU_CREDIT",
			"wallet_debited":    100,
		},
	})
}

func (m *platformMockServer) handleReservationAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/release") {
		http.NotFound(w, r)
		return
	}
	m.mu.Lock()
	m.releaseCount++
	call := m.releaseCount
	m.mu.Unlock()
	writePlatformSuccess(w, map[string]any{
		"id":            fmt.Sprintf("reservation-%d", call),
		"resource_type": "credits",
		"status":        "released",
		"reference_id":  fmt.Sprintf("intent-%d", call),
		"metadata":      "{}",
		"created_at":    time.Now().UTC().Format(time.RFC3339),
		"updated_at":    time.Now().UTC().Format(time.RFC3339),
		"released_at":   time.Now().UTC().Format(time.RFC3339),
	})
}

func (m *platformMockServer) handleChannelCharge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.NotFound(w, r)
		return
	}
	m.mu.Lock()
	m.channelCount++
	call := m.channelCount
	m.mu.Unlock()
	writePlatformSuccess(w, map[string]any{
		"matched":            true,
		"idempotent":         false,
		"status":             "earned",
		"binding_id":         "binding-1",
		"channel_partner_id": "partner-1",
		"policy_id":          "policy-1",
		"ledger": map[string]any{
			"id":                fmt.Sprintf("channel-ledger-%d", call),
			"source_charge_id":  fmt.Sprintf("settlement-%d", call),
			"commission_amount": 10,
			"status":            "earned",
			"created_at":        time.Now().UTC().Format(time.RFC3339),
			"updated_at":        time.Now().UTC().Format(time.RFC3339),
		},
	})
}

func writePlatformSuccess(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":      0,
		"message":   "success",
		"timestamp": time.Now().UnixMilli(),
		"data":      data,
	})
}
