package studio

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestCreateStylePresetAndGenerationJob(t *testing.T) {
	service := newStudioTestService(t)

	asset, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "menu.jpg",
		SourceURL:  "https://cdn.example.com/menu.jpg",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}

	style, err := service.CreateStylePreset("user-1", "org-1", CreateStylePresetInput{
		Name:           "Japanese Indoor",
		Visibility:     "organization",
		PreviewAssetID: asset.ID,
		Dimensions: []StyleDimension{
			{Type: "cuisine", Key: "japanese", Label: "Japanese"},
			{Type: "scene", Key: "indoor", Label: "Indoor"},
		},
		Tags: []string{"japanese", "indoor"},
		ExecutionProfile: StyleExecutionProfile{
			Provider:       "mock",
			Model:          "menu-v1",
			PromptTemplate: "make it japanese indoor",
		},
	})
	if err != nil {
		t.Fatalf("CreateStylePreset() error = %v", err)
	}

	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		StylePresetID:  style.StyleID,
		SourceAssetIDs: []string{asset.ID},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}

	if job.Mode != "single" || job.Status != "queued" {
		t.Fatalf("unexpected job summary: %+v", job)
	}
	if len(job.SourceAssetIDs) != 1 || job.SourceAssetIDs[0] != asset.ID {
		t.Fatalf("unexpected source assets: %+v", job.SourceAssetIDs)
	}
	if job.Stage != "queued" || job.Provider == "" {
		t.Fatalf("unexpected orchestration fields: %+v", job)
	}
}

func TestRecordJobResultsAndSelectVariant(t *testing.T) {
	service := newStudioTestService(t)

	source, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "dish.png",
		SourceURL:  "https://cdn.example.com/dish.png",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}

	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		SourceAssetIDs: []string{source.ID},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}

	completed, err := service.RecordJobResults("user-1", "org-1", job.JobID, RecordJobResultsInput{
		Status:   "completed",
		Progress: 80,
		Variants: []RecordJobVariantInput{
			{
				Index:      0,
				Status:     "ready",
				IsSelected: true,
				Asset: RegisterAssetInput{
					AssetType:  "generated",
					SourceType: "generated",
					FileName:   "dish-jp.png",
					SourceURL:  "https://cdn.example.com/dish-jp.png",
				},
			},
			{
				Index:  1,
				Status: "ready",
				Asset: RegisterAssetInput{
					AssetType:  "generated",
					SourceType: "generated",
					FileName:   "dish-kr.png",
					SourceURL:  "https://cdn.example.com/dish-kr.png",
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("RecordJobResults() error = %v", err)
	}
	if completed.Status != "completed" || completed.Progress != 100 || len(completed.Variants) != 2 {
		t.Fatalf("unexpected completed job: %+v", completed)
	}

	selectedVariantID := completed.Variants[1].VariantID
	selected, err := service.SelectVariant("user-1", "org-1", job.JobID, selectedVariantID)
	if err != nil {
		t.Fatalf("SelectVariant() error = %v", err)
	}
	if selected.SelectedVariantID != selectedVariantID {
		t.Fatalf("selected variant = %s, want %s", selected.SelectedVariantID, selectedVariantID)
	}
}

func TestCreateGenerationJob_IsIdempotent(t *testing.T) {
	service := newStudioTestService(t)
	source, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "dish.png",
		SourceURL:  "https://cdn.example.com/dish.png",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}
	first, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		IdempotencyKey: "idem-1",
		SourceAssetIDs: []string{source.ID},
	})
	if err != nil {
		t.Fatalf("first CreateGenerationJob() error = %v", err)
	}
	second, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		IdempotencyKey: "idem-1",
		SourceAssetIDs: []string{source.ID},
	})
	if err != nil {
		t.Fatalf("second CreateGenerationJob() error = %v", err)
	}
	if first.JobID != second.JobID {
		t.Fatalf("idempotent create produced two jobs: %s vs %s", first.JobID, second.JobID)
	}
}

func TestHandleDispatchTask_DispatchesQueuedJob(t *testing.T) {
	service := newStudioTestService(t)
	source, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "dish.png",
		SourceURL:  "https://cdn.example.com/dish.png",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}
	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		Provider:       "manual",
		SourceAssetIDs: []string{source.ID},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}
	if dispatchErr := service.HandleDispatchTask(context.Background(), job.JobID); dispatchErr != nil {
		t.Fatalf("HandleDispatchTask() error = %v", dispatchErr)
	}
	dispatched, err := service.GetGenerationJob("org-1", job.JobID)
	if err != nil {
		t.Fatalf("GetGenerationJob() error = %v", err)
	}
	if dispatched.Status != "processing" || dispatched.Stage != "provider_accepted" || dispatched.ProviderJobID == "" {
		t.Fatalf("unexpected dispatched job: %+v", dispatched)
	}
}

func TestCreateBatchJob_AggregatesChildren(t *testing.T) {
	service := newStudioTestService(t)
	first, _ := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "a.png",
		SourceURL:  "https://cdn.example.com/a.png",
	})
	second, _ := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "b.png",
		SourceURL:  "https://cdn.example.com/b.png",
	})
	root, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "batch",
		SourceAssetIDs: []string{first.ID, second.ID},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob(batch) error = %v", err)
	}
	if root.ChildJobCount != 2 || len(root.ChildJobs) != 2 {
		t.Fatalf("unexpected batch root: %+v", root)
	}
	for _, child := range root.ChildJobs {
		if _, recordErr := service.RecordJobResults("user-1", "org-1", child.JobID, RecordJobResultsInput{
			Status:   "completed",
			Progress: 100,
		}); recordErr != nil {
			t.Fatalf("RecordJobResults(%s) error = %v", child.JobID, recordErr)
		}
	}
	completed, err := service.GetGenerationJob("org-1", root.JobID)
	if err != nil {
		t.Fatalf("GetGenerationJob(root) error = %v", err)
	}
	if completed.Status != "completed" || completed.Progress != 100 {
		t.Fatalf("unexpected aggregated root: %+v", completed)
	}
}

func TestUpdateJobRuntime_UpdatesStageAndHeartbeat(t *testing.T) {
	service := newStudioTestService(t)
	source, _ := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "c.png",
		SourceURL:  "https://cdn.example.com/c.png",
	})
	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		SourceAssetIDs: []string{source.ID},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}
	progress := 35
	eta := 42
	updated, err := service.UpdateJobRuntime(job.JobID, UpdateJobRuntimeInput{
		Status:        "processing",
		Stage:         "running",
		StageMessage:  "Provider is rendering",
		Progress:      &progress,
		EtaSeconds:    &eta,
		ProviderJobID: "provider-job-1",
	})
	if err != nil {
		t.Fatalf("UpdateJobRuntime() error = %v", err)
	}
	if updated.Stage != "running" || updated.Progress != 35 || updated.ProviderJobID != "provider-job-1" {
		t.Fatalf("unexpected runtime update: %+v", updated)
	}
	if updated.HeartbeatAt == nil {
		t.Fatalf("expected heartbeat timestamp")
	}
}

func TestCreateGenerationJob_WithBillingCreatesReservedChargeIntent(t *testing.T) {
	service, platformMock := newBilledStudioTestService(t)
	source, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "billable.png",
		SourceURL:  "https://cdn.example.com/billable.png",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}
	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		SourceAssetIDs: []string{source.ID},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}
	intent, err := service.repo.FindChargeIntentByJobID(job.JobID)
	if err != nil {
		t.Fatalf("FindChargeIntentByJobID() error = %v", err)
	}
	if intent.ID == "" || intent.Status != "reserved" || intent.ReservationID == "" {
		t.Fatalf("unexpected reserved intent: %+v", intent)
	}
	if job.Charge == nil || !job.Charge.Billable || job.Charge.Status != "reserved" || job.Charge.BillableItemCode != "menu.generate.single" {
		t.Fatalf("unexpected charge summary: %+v", job.Charge)
	}
	if strings.Join(job.Charge.ChargePriorityAssetCodes, ",") != "MENU_MONTHLY_ALLOWANCE,MENU_PROMO_CREDIT,MENU_CREDIT" {
		t.Fatalf("unexpected charge priority: %+v", job.Charge.ChargePriorityAssetCodes)
	}
	if platformMock.reserveCalls() != 1 {
		t.Fatalf("reserveCalls = %d, want 1", platformMock.reserveCalls())
	}
}

func TestRecordJobResults_WithBillingFinalizesChargeIntent(t *testing.T) {
	service, platformMock := newBilledStudioTestService(t)
	source, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "done.png",
		SourceURL:  "https://cdn.example.com/done.png",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}
	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		SourceAssetIDs: []string{source.ID},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}
	updated, recordErr := service.RecordJobResults("user-1", "org-1", job.JobID, RecordJobResultsInput{
		Status:   "completed",
		Progress: 100,
	})
	if recordErr != nil {
		t.Fatalf("RecordJobResults() error = %v", recordErr)
	}
	intent, err := service.repo.FindChargeIntentByJobID(job.JobID)
	if err != nil {
		t.Fatalf("FindChargeIntentByJobID() error = %v", err)
	}
	if intent.Status != "settled" || intent.FinalUnits != 1 || intent.SettlementID == "" {
		t.Fatalf("unexpected settled intent: %+v", intent)
	}
	if updated.Charge == nil || updated.Charge.Status != "settled" || updated.Charge.SettlementID == "" || updated.Charge.WalletAssetCode != "MENU_CREDIT" || updated.Charge.WalletDebited != 100 {
		t.Fatalf("unexpected finalized charge summary: %+v", updated.Charge)
	}
	if platformMock.finalizeCalls() != 1 {
		t.Fatalf("finalizeCalls = %d, want 1", platformMock.finalizeCalls())
	}
}

func TestCancelGenerationJob_WithBillingReleasesChargeIntent(t *testing.T) {
	service, platformMock := newBilledStudioTestService(t)
	source, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "cancel.png",
		SourceURL:  "https://cdn.example.com/cancel.png",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}
	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		SourceAssetIDs: []string{source.ID},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}
	updated, cancelErr := service.CancelGenerationJob("org-1", job.JobID)
	if cancelErr != nil {
		t.Fatalf("CancelGenerationJob() error = %v", cancelErr)
	}
	intent, err := service.repo.FindChargeIntentByJobID(job.JobID)
	if err != nil {
		t.Fatalf("FindChargeIntentByJobID() error = %v", err)
	}
	if intent.Status != "released" || intent.ReleasedAt == nil {
		t.Fatalf("unexpected released intent: %+v", intent)
	}
	if updated.Charge == nil || updated.Charge.Status != "released" {
		t.Fatalf("unexpected released charge summary: %+v", updated.Charge)
	}
	if platformMock.releaseCalls() != 1 {
		t.Fatalf("releaseCalls = %d, want 1", platformMock.releaseCalls())
	}
}

func TestAssetLibrary_ReturnsGeneratedAssetAndShareState(t *testing.T) {
	service, db := newStudioTestServiceWithConfig(t, config.StudioConfig{}, nil)
	source, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "source.png",
		SourceURL:  "https://cdn.example.com/source.png",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}
	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		SourceAssetIDs: []string{source.ID},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}
	completed, err := service.RecordJobResults("user-1", "org-1", job.JobID, RecordJobResultsInput{
		Status:   "completed",
		Progress: 100,
		Variants: []RecordJobVariantInput{{
			Index:      0,
			Status:     "ready",
			IsSelected: true,
			Asset: RegisterAssetInput{
				AssetType:  "generated",
				SourceType: "generated",
				FileName:   "result.png",
				SourceURL:  "https://cdn.example.com/result.png",
			},
		}},
	})
	if err != nil {
		t.Fatalf("RecordJobResults() error = %v", err)
	}
	generatedAssetID := completed.Variants[0].AssetID
	shareRepo := repository.NewShareRepository(db)
	now := time.Now().UTC()
	if err := shareRepo.CreatePost(&models.SharePost{
		ID:             "share-1",
		OrganizationID: "org-1",
		UserID:         "user-1",
		AssetID:        generatedAssetID,
		JobID:          job.JobID,
		VariantID:      completed.Variants[0].VariantID,
		Visibility:     "public",
		Status:         "published",
		ShareToken:     "token-1",
		ShareURL:       "https://menu.example.com/share/token-1",
		PublishedAt:    &now,
	}); err != nil {
		t.Fatalf("CreatePost() error = %v", err)
	}
	result, err := service.AssetLibrary("user-1", "org-1", "generated", "", "", 20, 0)
	if err != nil {
		t.Fatalf("AssetLibrary() error = %v", err)
	}
	if result.Total != 1 || len(result.Items) != 1 {
		t.Fatalf("unexpected asset library result: %+v", result)
	}
	item := result.Items[0]
	if item.ProducedByJobID != job.JobID || item.Share == nil || item.Share.Status != "published" || !item.CanShare {
		t.Fatalf("unexpected asset library item: %+v", item)
	}
}

func TestJobHistory_ReturnsSourceAndSelectedAssets(t *testing.T) {
	service := newStudioTestService(t)
	source, _ := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "dish.png",
		SourceURL:  "https://cdn.example.com/dish.png",
	})
	job, _ := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		SourceAssetIDs: []string{source.ID},
	})
	_, err := service.RecordJobResults("user-1", "org-1", job.JobID, RecordJobResultsInput{
		Status:   "completed",
		Progress: 100,
		Variants: []RecordJobVariantInput{{
			Index:      0,
			Status:     "ready",
			IsSelected: true,
			Asset: RegisterAssetInput{
				AssetType:  "generated",
				SourceType: "generated",
				FileName:   "dish-result.png",
				SourceURL:  "https://cdn.example.com/dish-result.png",
			},
		}},
	})
	if err != nil {
		t.Fatalf("RecordJobResults() error = %v", err)
	}
	history, err := service.JobHistory("user-1", "org-1", "", 20, 0)
	if err != nil {
		t.Fatalf("JobHistory() error = %v", err)
	}
	if history.Total != 1 || len(history.Items) != 1 {
		t.Fatalf("unexpected job history result: %+v", history)
	}
	entry := history.Items[0]
	if len(entry.SourceAssets) != 1 || entry.SelectedAsset == nil || len(entry.ResultAssets) != 1 {
		t.Fatalf("unexpected job history entry: %+v", entry)
	}
}

func newStudioTestService(t *testing.T) *Service {
	service, _ := newStudioTestServiceWithConfig(t, config.StudioConfig{}, nil)
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
		BillingEnabled:         true,
		ProductCode:            "menu",
		ResourceType:           "credits",
		SingleBillableItem:     "menu.generate.single",
		RefinementBillableItem: "menu.generate.refinement",
		VariationBillableItem:  "menu.generate.variation",
	}, client)
	t.Cleanup(mock.server.Close)
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
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	return NewService(
		repository.NewStudioRepository(db),
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
	), db
}

type platformMockServer struct {
	server        *httptest.Server
	mu            sync.Mutex
	reserveCount  int
	finalizeCount int
	releaseCount  int
}

func newPlatformMockServer(t *testing.T) *platformMockServer {
	t.Helper()
	mock := &platformMockServer{}
	mux := http.NewServeMux()
	mux.HandleFunc("/internal/v1/controls/reservations", mock.handleReserve)
	mux.HandleFunc("/internal/v1/metering/finalizations", mock.handleFinalize)
	mux.HandleFunc("/internal/v1/controls/reservations/", mock.handleReservationAction)
	mock.server = httptest.NewServer(mux)
	return mock
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
		"billable_item_code":   "menu.generate.single",
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

func writePlatformSuccess(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":      0,
		"message":   "success",
		"timestamp": time.Now().UnixMilli(),
		"data":      data,
	})
}
