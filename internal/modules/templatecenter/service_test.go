package templatecenter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/models"
	"menu-service/internal/platform"
	"menu-service/internal/repository"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newTemplateCenterTestService(t *testing.T) (*Service, *repository.TemplateCenterRepository, *repository.StudioRepository) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), fmt.Sprintf("templatecenter-%s.db", t.Name()))
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.TemplateCatalog{}, &models.TemplateCatalogVersion{}, &models.TemplateCatalogExample{}, &models.TemplateFavorite{}, &models.TemplateUsageEvent{}, &models.StylePreset{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	templateRepo := repository.NewTemplateCenterRepository(db)
	studioRepo := repository.NewStudioRepository(db)
	service := NewService(templateRepo, studioRepo, nil, nil)
	if err := service.Bootstrap(); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	return service, templateRepo, studioRepo
}

func TestTemplateCenterBootstrapSeedsLocalCatalogEvenWithPlatformProjectionClient(t *testing.T) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "templatecenter-platform-bootstrap.db")
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&models.TemplateCatalog{}, &models.TemplateCatalogVersion{}, &models.TemplateCatalogExample{}, &models.TemplateFavorite{}, &models.TemplateUsageEvent{}, &models.StylePreset{}); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
	service := NewService(repository.NewTemplateCenterRepository(db), repository.NewStudioRepository(db), nil, platform.New(config.PlatformConfig{BaseURL: "http://127.0.0.1:1", Timeout: time.Millisecond}))
	if err := service.Bootstrap(); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	items, err := service.ListCatalogs("user-1", "org-1", ListCatalogInput{Plan: "pro", Source: "local"})
	if err != nil {
		t.Fatalf("ListCatalogs local: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected local template seed to be present for platform upstream sync")
	}
}

func TestTemplateCenterBootstrapAndMeta(t *testing.T) {
	service, _, _ := newTemplateCenterTestService(t)
	items, err := service.ListCatalogs("user-1", "org-1", ListCatalogInput{Plan: "pro"})
	if err != nil {
		t.Fatalf("ListCatalogs: %v", err)
	}
	if len(items) < 12 {
		t.Fatalf("expected seeded templates, got %d", len(items))
	}
	meta := service.Meta()
	if len(meta.Cuisines) == 0 || len(meta.Platforms) == 0 || len(meta.Moods) == 0 {
		t.Fatalf("expected populated meta: %+v", meta)
	}
}

func TestTemplateCenterBootstrapExposesFourSlotTemplateContract(t *testing.T) {
	service, _, _ := newTemplateCenterTestService(t)

	items, err := service.ListCatalogs("user-1", "org-1", ListCatalogInput{Plan: "pro"})
	if err != nil {
		t.Fatalf("ListCatalogs: %v", err)
	}
	var menuTemplate *TemplateCatalogSummary
	for idx := range items {
		if items[idx].TemplateID == "TPL-MENU-001" {
			menuTemplate = &items[idx]
			break
		}
	}
	if menuTemplate == nil {
		t.Fatalf("expected TPL-MENU-001 in catalog")
	}
	assertFourSlotContract(t, menuTemplate.InputSlots, menuTemplate.StrategyPolicy)

	detail, err := service.GetCatalogDetail("user-1", "org-1", "TPL-MENU-001", "pro")
	if err != nil {
		t.Fatalf("GetCatalogDetail: %v", err)
	}
	assertFourSlotContract(t, detail.InputSlots, detail.StrategyPolicy)
	if len(detail.TargetOutputs) == 0 {
		t.Fatalf("expected target_outputs in detail: %+v", detail)
	}

	useResult, err := service.UseTemplate("user-1", "org-1", "TPL-MENU-001", "pro", UseTemplateInput{TargetPlatform: "print_a4", Language: "en"})
	if err != nil {
		t.Fatalf("UseTemplate: %v", err)
	}
	assertFourSlotContract(t, useResult.InputSlots, useResult.ResolvedStrategy)
	if useResult.PrefilledJob.InputMode != "multi_image" || useResult.PrefilledJob.GenerationStrategy != "multi_image" || useResult.PrefilledJob.Provider != "comfyui_bridge" {
		t.Fatalf("expected multi_image handoff routed to comfyui_bridge, got %+v", useResult.PrefilledJob)
	}
}

func assertFourSlotContract(t *testing.T, slots []TemplateInputSlot, strategy map[string]any) {
	t.Helper()
	roles := make([]string, 0, len(slots))
	for _, slot := range slots {
		if slot.Required {
			roles = append(roles, slot.Role)
		}
	}
	expected := []string{"dish_photo", "brand_logo", "menu_reference", "style_reference"}
	if len(roles) != len(expected) {
		t.Fatalf("expected four required slots, got roles=%+v slots=%+v", roles, slots)
	}
	for idx, role := range expected {
		if roles[idx] != role {
			t.Fatalf("unexpected slot roles: got %+v want %+v", roles, expected)
		}
	}
	if strategy["input_mode"] != "multi_image" || strategy["generation_strategy"] != "multi_image" || strategy["provider"] != "comfyui_bridge" {
		t.Fatalf("expected multi_image strategy policy, got %+v", strategy)
	}
}

func TestTemplateCenterUseCopyAndFavorite(t *testing.T) {
	service, _, studioRepo := newTemplateCenterTestService(t)
	useResult, err := service.UseTemplate("user-1", "org-1", "TPL-TH-001", "basic", UseTemplateInput{
		TargetPlatform: "instagram_feed",
		Language:       "en",
		UploadImageURL: "https://example.com/dish.jpg",
	})
	if err != nil {
		t.Fatalf("UseTemplate: %v", err)
	}
	if useResult.TargetRoute == "" || useResult.PrefilledJob.Provider == "" {
		t.Fatalf("unexpected use result: %+v", useResult)
	}
	creativeSource, ok := useResult.PrefilledJob.Metadata["creative_source"].(map[string]any)
	if !ok || creativeSource["source_type"] != "template" {
		t.Fatalf("expected creative_source metadata, got %+v", useResult.PrefilledJob.Metadata)
	}
	if creativeSource["template_id"] != "TPL-TH-001" {
		t.Fatalf("unexpected template creative source: %+v", creativeSource)
	}

	copied, err := service.CopyToMyTemplates("user-1", "org-1", "TPL-TH-001", CopyTemplateInput{Name: "My Tom Yum"})
	if err != nil {
		t.Fatalf("CopyToMyTemplates: %v", err)
	}
	style, err := studioRepo.FindStylePresetByID("org-1", copied.StyleID)
	if err != nil {
		t.Fatalf("FindStylePresetByID: %v", err)
	}
	if style.SourceCatalogID != "TPL-TH-001" || style.SourceVersionID == "" {
		t.Fatalf("unexpected copied style preset: %+v", style)
	}

	if err := service.SetFavorite("user-1", "org-1", "TPL-TH-001"); err != nil {
		t.Fatalf("SetFavorite: %v", err)
	}
	favorites, err := service.ListFavorites("user-1", "org-1", "basic")
	if err != nil || len(favorites) != 1 {
		t.Fatalf("ListFavorites: %+v err=%v", favorites, err)
	}
	if err := service.RemoveFavorite("user-1", "org-1", "TPL-TH-001"); err != nil {
		t.Fatalf("RemoveFavorite: %v", err)
	}
	favorites, err = service.ListFavorites("user-1", "org-1", "basic")
	if err != nil || len(favorites) != 0 {
		t.Fatalf("expected empty favorites: %+v err=%v", favorites, err)
	}
}

func TestTemplateCenterPlanLockingAndFilters(t *testing.T) {
	service, _, _ := newTemplateCenterTestService(t)
	items, err := service.ListCatalogs("user-1", "org-1", ListCatalogInput{
		Cuisine:  "japanese",
		Platform: "line_oa",
		Plan:     "basic",
	})
	if err != nil {
		t.Fatalf("ListCatalogs filter: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected filtered catalog items")
	}
	locked := false
	for _, item := range items {
		if item.TemplateID == "TPL-JP-001" && item.Locked {
			locked = true
		}
	}
	if !locked {
		t.Fatalf("expected pro template to be locked for basic plan: %+v", items)
	}
	if _, err := service.UseTemplate("user-1", "org-1", "TPL-JP-001", "basic", UseTemplateInput{TargetPlatform: "instagram_feed"}); err == nil {
		t.Fatalf("expected plan lock error")
	}
}

func TestTemplateCenterUsesPlatformProjectionForBusinessFlow(t *testing.T) {
	service, _, studioRepo := newTemplateCenterTestService(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writePlatformEnvelope := func(data any) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]any{
				"code":      0,
				"message":   "ok",
				"timestamp": time.Now().Unix(),
				"data":      data,
			})
		}
		switch r.URL.Path {
		case "/internal/v1/template-ops/catalog":
			writePlatformEnvelope(map[string]any{
				"items": []map[string]any{
					{
						"template_ref":    "menu:TPL-PLATFORM-001",
						"product_code":    "menu",
						"template_id":     "TPL-PLATFORM-001",
						"slug":            "platform-menu-template",
						"name":            "Platform Menu Template",
						"summary":         "Managed by platform",
						"cover_asset_id":  "asset-1",
						"recommend_score": 88,
						"tags":            []string{"platform", "thai"},
						"platforms":       []string{"instagram_feed"},
						"series":          "thai",
						"capability_type": "dish_photo",
						"modality":        "image",
						"scope":           "official",
						"managed_source":  "platform_projection",
						"raw": map[string]any{
							"cuisine":       "thai",
							"dish_type":     "soup",
							"plan_required": "basic",
							"credits_cost":  14,
							"moods":         []string{"appetizing"},
						},
					},
				},
				"total":  1,
				"limit":  50,
				"offset": 0,
			})
		case "/internal/v1/template-ops/catalog/menu:TPL-PLATFORM-001":
			writePlatformEnvelope(map[string]any{
				"item": map[string]any{
					"template_ref":    "menu:TPL-PLATFORM-001",
					"product_code":    "menu",
					"template_id":     "TPL-PLATFORM-001",
					"slug":            "platform-menu-template",
					"name":            "Platform Menu Template",
					"summary":         "Managed by platform",
					"cover_asset_id":  "asset-1",
					"recommend_score": 88,
					"tags":            []string{"platform", "thai"},
					"platforms":       []string{"instagram_feed"},
					"series":          "thai",
					"capability_type": "dish_photo",
					"modality":        "image",
					"scope":           "official",
					"managed_source":  "platform_projection",
					"raw": map[string]any{
						"cuisine":       "thai",
						"dish_type":     "soup",
						"plan_required": "basic",
						"credits_cost":  14,
						"moods":         []string{"appetizing"},
					},
				},
				"product": "menu",
				"detail_raw": map[string]any{
					"prompt_templates": map[string]any{},
					"copy_templates":   map[string]any{"en": map[string]any{"headline": "hello"}},
					"hashtags":         map[string]any{"en": []any{"#menu"}},
					"design_spec":      map[string]any{"layout": "hero"},
					"export_specs":     map[string]any{},
					"input_schema":     map[string]any{"upload_image_url": true},
					"layout":           "hero",
					"lighting":         "warm spotlight",
					"props":            []any{"lime", "chili"},
					"execution_profile": map[string]any{
						"provider": "default",
						"model":    "menu-growth-v1",
					},
					"examples": []any{
						map[string]any{
							"id":              "example-1",
							"exampleType":     "preview",
							"title":           "Preview One",
							"sourceRef":       "templates/menu/TPL-PLATFORM-001/example-1",
							"storageKey":      "menu/platform/example-1.png",
							"assetId":         "asset-1",
							"previewAssetUrl": "https://example.com/preview.png",
						},
					},
					"metadata": map[string]any{"managed_source": "platform_projection"},
				},
			})
		case "/internal/v1/controls/capability/resolve":
			writePlatformEnvelope(map[string]any{
				"product_code":          "menu",
				"billing_subject_type":  "organization",
				"billing_subject_id":    "org-1",
				"capability_code":       "template_scope",
				"grant_value":           "free_templates",
				"source":                "test",
				"matched_policy_id":     "policy-test",
				"matched_grant_scope":   "organization",
				"matched_grant_subject": "org-1",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service.platform = platform.New(config.PlatformConfig{
		BaseURL:               server.URL,
		Timeout:               2 * time.Second,
		InternalServiceSecret: "test-secret",
		ServiceName:           "menu-test",
	})

	items, err := service.ListCatalogs("user-1", "org-1", ListCatalogInput{Plan: "basic"})
	if err != nil {
		t.Fatalf("ListCatalogs platform: %v", err)
	}
	if len(items) != 1 || items[0].TemplateID != "TPL-PLATFORM-001" {
		t.Fatalf("unexpected platform items: %+v", items)
	}
	if items[0].Locked {
		t.Fatalf("expected organization capability to unlock official template: %+v", items[0])
	}

	publicItems, err := service.ListCatalogs("user-1", "", ListCatalogInput{Plan: "basic"})
	if err != nil {
		t.Fatalf("ListCatalogs public fallback: %v", err)
	}
	if len(publicItems) != 1 || !publicItems[0].Locked {
		t.Fatalf("expected public basic fallback to stay locked: %+v", publicItems)
	}

	detail, err := service.GetCatalogDetail("user-1", "org-1", "TPL-PLATFORM-001", "pro")
	if err != nil {
		t.Fatalf("GetCatalogDetail platform: %v", err)
	}
	if detail.ExecutionProfile.Provider != "default" || len(detail.Examples) != 1 {
		t.Fatalf("unexpected platform detail: %+v", detail)
	}

	if err := service.SetFavorite("user-1", "org-1", "TPL-PLATFORM-001"); err != nil {
		t.Fatalf("SetFavorite platform: %v", err)
	}
	favorites, err := service.ListFavorites("user-1", "org-1", "pro")
	if err != nil || len(favorites) != 1 || !favorites[0].IsFavorite {
		t.Fatalf("ListFavorites platform: %+v err=%v", favorites, err)
	}

	useResult, err := service.UseTemplate("user-1", "org-1", "TPL-PLATFORM-001", "pro", UseTemplateInput{
		TargetPlatform: "instagram_feed",
		Language:       "en",
	})
	if err != nil {
		t.Fatalf("UseTemplate platform: %v", err)
	}
	if useResult.TemplateVersionID == "" || useResult.PrefilledJob.Provider != "default" {
		t.Fatalf("unexpected platform use result: %+v", useResult)
	}
	executionProfile, _ := useResult.PrefilledJob.Metadata["execution_profile"].(map[string]any)
	stylePrompt, _ := executionProfile["style_prompt"].(string)
	if !strings.Contains(stylePrompt, "Layout: hero") || !strings.Contains(stylePrompt, "Lighting: warm spotlight") || !strings.Contains(stylePrompt, "Props: lime, chili") {
		t.Fatalf("expected flattened style_prompt fallback, got %+v", executionProfile)
	}
	exportSpec, ok := useResult.TemplateContext["export_spec"].(map[string]any)
	if !ok || exportSpec["format"] != "jpg" {
		t.Fatalf("expected fallback export spec from platform defaults, got %+v", useResult.TemplateContext["export_spec"])
	}

	copied, err := service.CopyToMyTemplates("user-1", "org-1", "TPL-PLATFORM-001", CopyTemplateInput{Name: "Copied From Platform"})
	if err != nil {
		t.Fatalf("CopyToMyTemplates platform: %v", err)
	}
	style, err := studioRepo.FindStylePresetByID("org-1", copied.StyleID)
	if err != nil {
		t.Fatalf("FindStylePresetByID platform: %v", err)
	}
	if style.SourceCatalogID != "TPL-PLATFORM-001" || style.SourceVersionID == "" {
		t.Fatalf("unexpected copied platform style: %+v", style)
	}
}
