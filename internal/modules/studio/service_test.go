package studio

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/models"
	"menu-service/internal/platform"
	"menu-service/internal/repository"
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

func TestCreateGenerationJob_WithCreativeSourceMetadata(t *testing.T) {
	mock := newPlatformMockServer(t)
	service, db := newStudioTestServiceWithConfig(t, config.StudioConfig{
		ProductCode:            "menu",
		ResourceType:           "quota",
		SingleBillableItem:     "menu.render.call",
		RefinementBillableItem: "menu.render.call",
		VariationBillableItem:  "menu.render.call",
		DefaultProvider:        "volcengine",
	}, platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-test",
		InternalServiceSecret: "test-secret",
	}))
	t.Cleanup(mock.server.Close)
	seedStudioTemplateVersion(t, db, "TPL-TH-001", "TPL-TH-001-v1", map[string]string{"en": "template prompt from official template"}, `{"provider":"volcengine"}`)
	asset, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "template.jpg",
		SourceURL:  "https://cdn.example.com/template.jpg",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}

	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		SourceAssetIDs: []string{asset.ID},
		Prompt:         "hero dish on premium plate",
		Metadata: map[string]any{
			"creative_source": map[string]any{
				"source_type":         "template",
				"source_id":           "TPL-TH-001",
				"title":               "Tom Yum Hero",
				"target_platform":     "instagram_feed",
				"template_id":         "TPL-TH-001",
				"template_version_id": "TPL-TH-001-v1",
			},
			"execution_profile": map[string]any{
				"provider":     "volcengine",
				"style_prompt": "premium plating, sharp focus",
				"parameter_profile": map[string]any{
					"stylization": 55,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}

	if job.Metadata["template_catalog_id"] != "TPL-TH-001" {
		t.Fatalf("expected template_catalog_id in metadata, got %+v", job.Metadata)
	}
	creativeSource, ok := job.Metadata["creative_source"].(map[string]any)
	if !ok || creativeSource["source_type"] != "template" {
		t.Fatalf("expected creative_source metadata, got %+v", job.Metadata)
	}
	if job.Provider != "volcengine" {
		t.Fatalf("expected provider from execution profile, got %+v", job)
	}
	if job.CreativeSource == nil || job.CreativeSource.SourceType != "template" || job.CreativeSource.TemplateID != "TPL-TH-001" {
		t.Fatalf("expected top-level creative source, got %+v", job.CreativeSource)
	}
	if job.PromptSnapshot.SystemPrompt != "template prompt from official template" {
		t.Fatalf("expected template prompt in system layer, got %+v", job.PromptSnapshot)
	}
	if job.PromptSnapshot.UserPrompt != "hero dish on premium plate" {
		t.Fatalf("expected user prompt in user layer, got %+v", job.PromptSnapshot)
	}
	if job.PromptSnapshot.StylePrompt != "premium plating, sharp focus" {
		t.Fatalf("expected style prompt from metadata execution profile, got %+v", job.PromptSnapshot)
	}
	if job.PromptSnapshot.PromptTemplate != "template prompt from official template\n\npremium plating, sharp focus\n\nhero dish on premium plate" {
		t.Fatalf("expected composed prompt in prompt snapshot, got %+v", job.PromptSnapshot)
	}
	if job.PromptSnapshot.ParameterProfile["stylization"] != float64(55) {
		t.Fatalf("expected parameter profile, got %+v", job.PromptSnapshot)
	}
}

func TestCreateGenerationJob_DefaultProviderUsesPlatformBinding(t *testing.T) {
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
		DefaultProvider:        "default",
	}, client)
	t.Cleanup(mock.server.Close)

	asset, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "template.jpg",
		SourceURL:  "https://cdn.example.com/template.jpg",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}

	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		Provider:       "default",
		SourceAssetIDs: []string{asset.ID},
		Prompt:         "hero dish",
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}
	stored, err := service.repo.FindGenerationJobByID("org-1", job.JobID)
	if err != nil {
		t.Fatalf("FindGenerationJobByID() error = %v", err)
	}
	if stored.RuntimeJobID == "" {
		t.Fatalf("expected runtime job id, got %+v", stored)
	}
	payload := mock.lastRuntimePayload()
	if payload == nil {
		t.Fatalf("expected runtime payload to be captured")
	}
	if value, exists := payload["provider_code"]; exists && value != "" {
		t.Fatalf("expected provider_code to be omitted for default provider, got %+v", payload)
	}
}

func TestCreateGenerationJob_ComposesLayeredPromptFromExecutionProfile(t *testing.T) {
	mock := newPlatformMockServer(t)
	service, db := newStudioTestServiceWithConfig(t, config.StudioConfig{
		ProductCode:            "menu",
		ResourceType:           "quota",
		SingleBillableItem:     "menu.render.call",
		RefinementBillableItem: "menu.render.call",
		VariationBillableItem:  "menu.render.call",
		DefaultProvider:        "volcengine",
	}, platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-test",
		InternalServiceSecret: "test-secret",
	}))
	t.Cleanup(mock.server.Close)
	seedStudioTemplateVersion(t, db, "TPL-TH-002", "TPL-TH-002-v1", map[string]string{"zh": "泰式料理营销海报，主体清晰，适合社交媒体传播"}, `{"provider":"volcengine"}`)

	asset, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "prompt.jpg",
		SourceURL:  "https://cdn.example.com/prompt.jpg",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}

	_, err = service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:           "single",
		SourceAssetIDs: []string{asset.ID},
		Prompt:         "突出新品和限时促销",
		Params:         map[string]any{"language": "zh"},
		Metadata: map[string]any{
			"template_catalog_id": "TPL-TH-002",
			"template_version_id": "TPL-TH-002-v1",
			"execution_profile": map[string]any{
				"provider":     "volcengine",
				"style_prompt": "高饱和配色，暖光，俯拍构图",
			},
			"creative_source": map[string]any{
				"source_type":         "template",
				"source_id":           "TPL-TH-002",
				"template_id":         "TPL-TH-002",
				"template_version_id": "TPL-TH-002-v1",
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}

	payload := mock.lastRuntimePayload()
	if payload == nil {
		t.Fatalf("expected runtime payload to be captured")
	}
	rawManifest, _ := payload["input_manifest"].(string)
	var manifest map[string]any
	if err := json.Unmarshal([]byte(rawManifest), &manifest); err != nil {
		t.Fatalf("decode input_manifest: %v", err)
	}
	promptSnapshot, _ := manifest["prompt_snapshot"].(map[string]any)
	if promptSnapshot["system_prompt"] != "泰式料理营销海报，主体清晰，适合社交媒体传播" {
		t.Fatalf("unexpected system_prompt: %+v", promptSnapshot)
	}
	if promptSnapshot["style_prompt"] != "高饱和配色，暖光，俯拍构图" {
		t.Fatalf("unexpected style_prompt: %+v", promptSnapshot)
	}
	if promptSnapshot["user_prompt"] != "突出新品和限时促销" {
		t.Fatalf("unexpected user_prompt: %+v", promptSnapshot)
	}
	if promptSnapshot["prompt_template"] != "泰式料理营销海报，主体清晰，适合社交媒体传播\n\n高饱和配色，暖光，俯拍构图\n\n突出新品和限时促销" {
		t.Fatalf("unexpected composed prompt: %+v", promptSnapshot)
	}
}

func TestRegisterAsset_PersistsBase64ToLocalStorage(t *testing.T) {
	mock := newPlatformMockServer(t)
	service, _ := newStudioTestServiceWithConfig(t, config.StudioConfig{
		ProductCode:            "menu",
		ResourceType:           "credits",
		SingleBillableItem:     "menu.render.call",
		RefinementBillableItem: "menu.render.call",
		VariationBillableItem:  "menu.render.call",
		DefaultProvider:        "volcengine",
	}, platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-test",
		InternalServiceSecret: "test-secret",
	}))
	asset, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   "pig.png",
		MimeType:   "image/png",
		SourceURL:  "data:image/png;base64,aGVsbG8=",
	})
	if err != nil {
		t.Fatalf("RegisterAsset() error = %v", err)
	}
	if asset.StorageKey == "" || !strings.Contains(asset.SourceURL, "/api/v1/menu/studio/assets/") {
		t.Fatalf("expected protected asset url, got %+v", asset)
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
	if completed.Variants[0].Asset == nil || completed.Variants[0].Asset.SourceURL == "" {
		t.Fatalf("expected first variant asset to be hydrated, got %+v", completed.Variants[0])
	}
	if completed.Variants[0].PreviewURL == "" {
		t.Fatalf("expected first variant preview url, got %+v", completed.Variants[0])
	}
	if completed.Variants[1].Asset == nil || completed.Variants[1].Asset.SourceURL == "" {
		t.Fatalf("expected second variant asset to be hydrated, got %+v", completed.Variants[1])
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

func TestCreateGenerationJob_BootstrapsPlatformRuntime(t *testing.T) {
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
	if job.Status != "queued" || job.Stage != "queued" {
		t.Fatalf("expected queued job, got %+v", job)
	}
	if job.Charge == nil || job.Charge.BillableItemCode == "" {
		t.Fatalf("expected charge summary to be created, got %+v", job.Charge)
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
	if job.Charge == nil || !job.Charge.Billable || job.Charge.Status != "reserved" || job.Charge.BillableItemCode != "menu.render.call" {
		t.Fatalf("unexpected charge summary: %+v", job.Charge)
	}
	if strings.Join(job.Charge.ChargePriorityAssetCodes, ",") != "MENU_PROMO_CREDIT,MENU_CREDIT" {
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
	if platformMock.channelChargeCalls() != 1 {
		t.Fatalf("channelChargeCalls = %d, want 1", platformMock.channelChargeCalls())
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
	mock := newPlatformMockServer(t)
	t.Cleanup(mock.server.Close)
	service, db := newStudioTestServiceWithConfig(t, config.StudioConfig{
		ProductCode:            "menu",
		ResourceType:           "credits",
		SingleBillableItem:     "menu.render.call",
		RefinementBillableItem: "menu.render.call",
		VariationBillableItem:  "menu.render.call",
	}, platform.New(config.PlatformConfig{
		BaseURL:               mock.server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-test",
		InternalServiceSecret: "test-secret",
	}))
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
	if createErr := shareRepo.CreatePost(&models.SharePost{
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
	}); createErr != nil {
		t.Fatalf("CreatePost() error = %v", createErr)
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
