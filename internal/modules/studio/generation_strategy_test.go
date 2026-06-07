package studio

import (
	"encoding/json"
	"testing"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/platform"
)

func TestCreateGenerationJob_NormalizesAskForRequiredInputToImageToImage(t *testing.T) {
	mock := newPlatformMockServer(t)
	defer mock.server.Close()
	service, _ := newStudioTestServiceWithConfig(t, config.StudioConfig{
		ProductCode:            "menu",
		ResourceType:           "credits",
		SingleBillableItem:     "menu.render.call",
		RefinementBillableItem: "menu.render.call",
		VariationBillableItem:  "menu.render.call",
		DefaultProvider:        "volcengine",
	}, platform.New(config.PlatformConfig{BaseURL: mock.server.URL, Timeout: time.Second, ServiceName: "menu-test", InternalServiceSecret: "test-secret"}))
	asset := mustRegisterStudioAsset(t, service, "dish.png")

	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:               "single",
		InputMode:          "ask_for_required_input",
		GenerationStrategy: "ask_for_required_input",
		SourceAssets: []StudioSourceAssetInput{{
			AssetID: asset.ID,
			Role:    "dish_photo",
		}},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}
	if job.InputMode != "image_to_image" || job.GenerationStrategy != "image_to_image" {
		t.Fatalf("expected image_to_image strategy, got input_mode=%q generation_strategy=%q", job.InputMode, job.GenerationStrategy)
	}
	manifest := decodeLastRuntimeManifest(t, mock)
	if manifest["input_mode"] != "image_to_image" || manifest["generation_strategy"] != "image_to_image" {
		t.Fatalf("runtime manifest did not receive normalized strategy: %+v", manifest)
	}
}

func TestCreateGenerationJob_RoleAwareMultiImageManifestAndProvider(t *testing.T) {
	mock := newPlatformMockServer(t)
	defer mock.server.Close()
	service, _ := newStudioTestServiceWithConfig(t, config.StudioConfig{
		ProductCode:            "menu",
		ResourceType:           "credits",
		SingleBillableItem:     "menu.render.call",
		RefinementBillableItem: "menu.render.call",
		VariationBillableItem:  "menu.render.call",
		DefaultProvider:        "volcengine",
	}, platform.New(config.PlatformConfig{BaseURL: mock.server.URL, Timeout: time.Second, ServiceName: "menu-test", InternalServiceSecret: "test-secret"}))
	assets := []AssetSummary{
		*mustRegisterStudioAsset(t, service, "dish.png"),
		*mustRegisterStudioAsset(t, service, "brand.png"),
		*mustRegisterStudioAsset(t, service, "menu.png"),
		*mustRegisterStudioAsset(t, service, "style.png"),
	}
	roles := []string{"dish_photo", "brand_logo", "menu_reference", "style_reference"}
	sourceAssets := make([]StudioSourceAssetInput, 0, len(assets))
	for i, asset := range assets {
		sourceAssets = append(sourceAssets, StudioSourceAssetInput{AssetID: asset.ID, Role: roles[i]})
	}

	job, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:               "single",
		InputMode:          "ask_for_required_input",
		GenerationStrategy: "ask_for_required_input",
		SourceAssets:       sourceAssets,
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob() error = %v", err)
	}
	if job.InputMode != "multi_image" || job.Provider != "comfyui_bridge" {
		t.Fatalf("expected multi_image routed to comfyui_bridge, got %+v", job)
	}
	payload := mock.lastRuntimePayload()
	if payload["provider_code"] != "comfyui_bridge" {
		t.Fatalf("expected provider_code comfyui_bridge, got %+v", payload)
	}
	manifest := decodeLastRuntimeManifest(t, mock)
	if manifest["input_mode"] != "multi_image" || manifest["generation_strategy"] != "multi_image" {
		t.Fatalf("expected multi_image manifest, got %+v", manifest)
	}
	rawSources, ok := manifest["source_assets"].([]any)
	if !ok || len(rawSources) != 4 {
		t.Fatalf("expected four manifest source_assets, got %+v", manifest["source_assets"])
	}
	seenRoles := map[string]bool{}
	for _, raw := range rawSources {
		item, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("unexpected source asset shape: %+v", raw)
		}
		seenRoles[stringMapValue(item, "role")] = true
	}
	for _, role := range roles {
		if !seenRoles[role] {
			t.Fatalf("manifest missing role %s: %+v", role, rawSources)
		}
	}
}

func TestCreateGenerationJob_RejectsMoreThanFourSourceAssets(t *testing.T) {
	service := newStudioTestService(t)
	sourceAssets := make([]StudioSourceAssetInput, 0, 5)
	for i := 0; i < 5; i++ {
		asset := mustRegisterStudioAsset(t, service, "asset.png")
		sourceAssets = append(sourceAssets, StudioSourceAssetInput{AssetID: asset.ID, Role: "style_reference"})
	}
	_, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{Mode: "single", SourceAssets: sourceAssets})
	if err == nil {
		t.Fatalf("expected source asset limit error")
	}
	domainErr, ok := asStudioDomainError(err)
	if !ok || domainErr.Code != "STUDIO_SOURCE_ASSETS_LIMIT_EXCEEDED" {
		t.Fatalf("expected STUDIO_SOURCE_ASSETS_LIMIT_EXCEEDED, got %#v", err)
	}
}

func TestCreateGenerationJob_RejectsExplicitUnsupportedMultiImageProvider(t *testing.T) {
	service := newStudioTestService(t)
	first := mustRegisterStudioAsset(t, service, "a.png")
	second := mustRegisterStudioAsset(t, service, "b.png")
	_, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode:               "single",
		Provider:           "volcengine",
		InputMode:          "multi_image",
		GenerationStrategy: "multi_image",
		SourceAssets: []StudioSourceAssetInput{
			{AssetID: first.ID, Role: "dish_photo"},
			{AssetID: second.ID, Role: "style_reference"},
		},
	})
	if err == nil {
		t.Fatalf("expected unsupported provider error")
	}
	domainErr, ok := asStudioDomainError(err)
	if !ok || domainErr.Code != "STUDIO_MULTI_IMAGE_PROVIDER_UNSUPPORTED" {
		t.Fatalf("expected STUDIO_MULTI_IMAGE_PROVIDER_UNSUPPORTED, got %#v", err)
	}
}

func TestCreateGenerationJob_BatchFanoutKeepsChildStrategyImageToImage(t *testing.T) {
	mock := newPlatformMockServer(t)
	defer mock.server.Close()
	service, _ := newStudioTestServiceWithConfig(t, config.StudioConfig{
		ProductCode:            "menu",
		ResourceType:           "credits",
		SingleBillableItem:     "menu.render.call",
		RefinementBillableItem: "menu.render.call",
		VariationBillableItem:  "menu.render.call",
		DefaultProvider:        "volcengine",
	}, platform.New(config.PlatformConfig{BaseURL: mock.server.URL, Timeout: time.Second, ServiceName: "menu-test", InternalServiceSecret: "test-secret"}))
	first := mustRegisterStudioAsset(t, service, "batch-a.png")
	second := mustRegisterStudioAsset(t, service, "batch-b.png")

	root, err := service.CreateGenerationJob("user-1", "org-1", CreateGenerationJobInput{
		Mode: "batch",
		SourceAssets: []StudioSourceAssetInput{
			{AssetID: first.ID, Role: "dish_photo"},
			{AssetID: second.ID, Role: "style_reference"},
		},
	})
	if err != nil {
		t.Fatalf("CreateGenerationJob(batch) error = %v", err)
	}
	if root.InputMode != "image_to_image" || root.GenerationStrategy != "image_to_image" {
		t.Fatalf("batch root should stay image_to_image fanout, got input_mode=%q generation_strategy=%q", root.InputMode, root.GenerationStrategy)
	}
	payload := mock.lastRuntimePayload()
	if payload["provider_code"] != "volcengine" {
		t.Fatalf("batch child should keep default single-image provider, got payload=%+v", payload)
	}
	manifest := decodeLastRuntimeManifest(t, mock)
	if manifest["input_mode"] != "image_to_image" || manifest["generation_strategy"] != "image_to_image" {
		t.Fatalf("batch child runtime manifest should stay image_to_image, got %+v", manifest)
	}
	rawSourceIDs, ok := manifest["source_asset_ids"].([]any)
	if !ok || len(rawSourceIDs) != 1 {
		t.Fatalf("batch child runtime manifest should contain exactly one source asset, got %+v", manifest["source_asset_ids"])
	}
}

func mustRegisterStudioAsset(t *testing.T, service *Service, fileName string) *AssetSummary {
	t.Helper()
	asset, err := service.RegisterAsset("user-1", "org-1", RegisterAssetInput{
		AssetType:  "source",
		SourceType: "upload",
		FileName:   fileName,
		SourceURL:  "https://cdn.example.com/" + fileName,
	})
	if err != nil {
		t.Fatalf("RegisterAsset(%s) error = %v", fileName, err)
	}
	return asset
}

func decodeLastRuntimeManifest(t *testing.T, mock *platformMockServer) map[string]any {
	t.Helper()
	payload := mock.lastRuntimePayload()
	if payload == nil {
		t.Fatalf("expected runtime payload")
	}
	rawManifest, ok := payload["input_manifest"].(string)
	if !ok || rawManifest == "" {
		t.Fatalf("expected input_manifest string, got %+v", payload)
	}
	var manifest map[string]any
	if err := json.Unmarshal([]byte(rawManifest), &manifest); err != nil {
		t.Fatalf("decode input_manifest: %v", err)
	}
	return manifest
}
