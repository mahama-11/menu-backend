package templatecenter

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"menu-service/internal/models"
	studio "menu-service/internal/modules/studio"
)

type TemplateMetaOption struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

type TemplatePlatformOption struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Ratio  string `json:"ratio,omitempty"`
	Format string `json:"format,omitempty"`
}

type TemplateMetaResult struct {
	Cuisines  []TemplateMetaOption     `json:"cuisines"`
	DishTypes []TemplateMetaOption     `json:"dish_types"`
	Platforms []TemplatePlatformOption `json:"platforms"`
	Moods     []TemplateMetaOption     `json:"moods"`
	Plans     []TemplateMetaOption     `json:"plans"`
}

type TemplateExampleSeed struct {
	ExampleType   string         `json:"example_type"`
	Title         string         `json:"title"`
	Description   string         `json:"description"`
	SourceRef     string         `json:"source_ref"`
	StorageKey    string         `json:"storage_key"`
	AssetID       string         `json:"asset_id"`
	PreviewURL    string         `json:"preview_url"`
	InputAssetURL string         `json:"input_asset_url"`
	OutputAssetURL string        `json:"output_asset_url"`
	Metadata      map[string]any `json:"metadata"`
	SortOrder     int            `json:"sort_order"`
}

type TemplateSeed struct {
	ID               string                 `json:"id"`
	Slug             string                 `json:"slug"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Cuisine          string                 `json:"cuisine"`
	DishType         string                 `json:"dish_type"`
	Plan             string                 `json:"plan"`
	CreditsCost      int64                  `json:"credits_cost"`
	Platforms        []string               `json:"platforms"`
	Moods            []string               `json:"moods"`
	Tags             []string               `json:"tags"`
	Layout           string                 `json:"layout"`
	Lighting         string                 `json:"lighting"`
	Props            []string               `json:"props"`
	PromptTemplates  map[string]string      `json:"prompt_templates"`
	CopyTemplates    map[string]any         `json:"copy_templates"`
	Hashtags         map[string][]string    `json:"hashtags"`
	ExportSpecs      map[string]any         `json:"export_specs"`
	InputSchema      map[string]any         `json:"input_schema"`
	ExecutionProfile map[string]any         `json:"execution_profile"`
	Examples         []TemplateExampleSeed  `json:"examples"`
	Metadata         map[string]any         `json:"metadata"`
}

type TemplateSeedLibrary struct {
	Version string             `json:"version"`
	Meta    TemplateMetaResult `json:"meta"`
	Templates []TemplateSeed   `json:"templates"`
}

//go:embed template_library.seed.json
var embeddedTemplateSeedJSON []byte

var templatePlatforms = map[string]TemplatePlatformOption{
	"instagram_feed":  {ID: "instagram_feed", Label: "Instagram Feed", Width: 1080, Height: 1080, Ratio: "1:1", Format: "jpg"},
	"instagram_story": {ID: "instagram_story", Label: "Instagram Story", Width: 1080, Height: 1920, Ratio: "9:16", Format: "jpg"},
	"instagram_reels": {ID: "instagram_reels", Label: "Instagram Reels", Width: 1080, Height: 1920, Ratio: "9:16", Format: "jpg"},
	"facebook_post":   {ID: "facebook_post", Label: "Facebook Post", Width: 1080, Height: 1350, Ratio: "4:5", Format: "jpg"},
	"facebook_cover":  {ID: "facebook_cover", Label: "Facebook Cover", Width: 820, Height: 312, Ratio: "205:78", Format: "jpg"},
	"line_oa":         {ID: "line_oa", Label: "LINE OA", Width: 1040, Height: 1040, Ratio: "1:1", Format: "jpg"},
	"tiktok":          {ID: "tiktok", Label: "TikTok", Width: 1080, Height: 1920, Ratio: "9:16", Format: "jpg"},
	"grabfood":        {ID: "grabfood", Label: "GrabFood Banner", Width: 1920, Height: 1080, Ratio: "16:9", Format: "jpg"},
	"foodpanda":       {ID: "foodpanda", Label: "foodpanda Banner", Width: 1920, Height: 1080, Ratio: "16:9", Format: "jpg"},
	"qr_menu":         {ID: "qr_menu", Label: "QR Menu", Width: 794, Height: 1123, Ratio: "A4", Format: "png"},
	"print_a4":        {ID: "print_a4", Label: "Print A4", Width: 794, Height: 1123, Ratio: "A4", Format: "png/pdf"},
	"print_a5":        {ID: "print_a5", Label: "Print A5", Width: 559, Height: 794, Ratio: "A5", Format: "png"},
}

func defaultTemplateMeta() TemplateMetaResult {
	library := loadEmbeddedTemplateSeedLibrary()
	if len(library.Meta.Cuisines) > 0 {
		return library.Meta
	}
	return TemplateMetaResult{}
}

func defaultTemplateSeeds() []TemplateSeed {
	return loadEmbeddedTemplateSeedLibrary().Templates
}

func (s TemplateSeed) catalogModel() *models.TemplateCatalog {
	return &models.TemplateCatalog{
		ID:             s.ID,
		Slug:           s.Slug,
		Name:           s.Name,
		Description:    s.Description,
		Status:         "active",
		Scope:          "official",
		Cuisine:        s.Cuisine,
		DishType:       s.DishType,
		PlanRequired:   s.Plan,
		CreditsCost:    s.CreditsCost,
		PlatformsJSON:  mustEncodeJSON(s.Platforms),
		MoodsJSON:      mustEncodeJSON(s.Moods),
		TagsJSON:       mustEncodeJSON(s.Tags),
		MetadataJSON:   mustEncodeJSON(mergeSeedMetadata(s.Metadata, map[string]any{"seeded": true})),
		RecommendScore: seedRecommendScore(s.Plan),
	}
}

func (s TemplateSeed) versionModel() *models.TemplateCatalogVersion {
	exportSpecs := map[string]any{}
	for key, value := range s.ExportSpecs {
		exportSpecs[key] = value
	}
	if len(exportSpecs) == 0 {
		for _, platformID := range s.Platforms {
			if spec, ok := templatePlatforms[platformID]; ok {
				exportSpecs[platformID] = spec
			}
		}
	}
	promptEN := s.PromptTemplates["en"]
	if promptEN == "" {
		promptEN = fmt.Sprintf("%s, %s, %s, %s, premium menu growth campaign visual", s.Name, s.Layout, s.Lighting, strings.Join(s.Props, ", "))
	}
	promptZH := defaultString(s.PromptTemplates["zh"], fmt.Sprintf("%s，布局：%s，光效：%s，道具：%s，高质量餐饮营销图", s.Name, s.Layout, s.Lighting, strings.Join(s.Props, "、")))
	promptTH := defaultString(s.PromptTemplates["th"], fmt.Sprintf("%s, layout %s, lighting %s, props %s", s.Name, s.Layout, s.Lighting, strings.Join(s.Props, ", ")))
	copyTemplates := s.CopyTemplates
	if len(copyTemplates) == 0 {
		copyTemplates = map[string]any{
		"en": map[string]string{"headline": s.Name, "body": "Turn this dish into a campaign-ready menu creative."},
		"zh": map[string]string{"headline": s.Name, "body": "将这道菜快速生成可直接用于投放与社媒传播的营销图。"},
		"th": map[string]string{"headline": s.Name, "body": "เปลี่ยนเมนูนี้ให้เป็นภาพโปรโมตพร้อมใช้งานทันที"},
	}
	}
	executionProfile := studio.StyleExecutionProfile{
		Provider:       "default",
		Model:          "menu-growth-v1",
		PromptTemplate: promptEN,
		ParameterProfile: map[string]any{
			"quality": "high",
			"style":   firstMood(s.Moods),
		},
		Variables: map[string]string{
			"template_id":    s.ID,
			"cuisine":        s.Cuisine,
			"dish_type":      s.DishType,
			"default_layout": s.Layout,
		},
	}
	if len(s.ExecutionProfile) > 0 {
		executionProfile = decodeExecutionProfile(mustEncodeJSON(s.ExecutionProfile))
		if executionProfile.PromptTemplate == "" {
			executionProfile.PromptTemplate = promptEN
		}
	}
	inputSchema := s.InputSchema
	if len(inputSchema) == 0 {
		inputSchema = map[string]any{
			"upload_image_url": true,
			"target_platforms": s.Platforms,
			"languages":        []string{"th", "zh", "en"},
		}
	}
	hashtags := s.Hashtags
	if len(hashtags) == 0 {
		hashtags = map[string][]string{"en": s.Tags, "zh": s.Tags, "th": s.Tags}
	}
	return &models.TemplateCatalogVersion{
		ID:                  s.ID + "-v1",
		TemplateCatalogID:   s.ID,
		VersionNo:           1,
		Status:              "published",
		Name:                s.Name + " v1",
		Summary:             s.Description,
		PromptTemplatesJSON: mustEncodeJSON(map[string]string{"en": promptEN, "zh": promptZH, "th": promptTH}),
		CopyTemplatesJSON:   mustEncodeJSON(copyTemplates),
		HashtagsJSON:        mustEncodeJSON(hashtags),
		DesignSpecJSON: mustEncodeJSON(map[string]any{
			"layout":   s.Layout,
			"lighting": s.Lighting,
			"props":    s.Props,
			"moods":    s.Moods,
		}),
		ExportSpecsJSON: mustEncodeJSON(exportSpecs),
		InputSchemaJSON: mustEncodeJSON(inputSchema),
		ExecutionProfileJSON: mustEncodeJSON(executionProfile),
		MetadataJSON: mustEncodeJSON(mergeSeedMetadata(s.Metadata, map[string]any{
			"seeded":      true,
			"cover_layout": s.Layout,
		})),
	}
}

func (s TemplateSeed) exampleModels(versionID string) []models.TemplateCatalogExample {
	items := make([]models.TemplateCatalogExample, 0, len(s.Examples))
	for idx, example := range s.Examples {
		items = append(items, models.TemplateCatalogExample{
			TemplateVersionID: versionID,
			ExampleType:       defaultString(example.ExampleType, "preview"),
			Title:             example.Title,
			Description:       example.Description,
			SourceRef:         example.SourceRef,
			StorageKey:        example.StorageKey,
			AssetID:           example.AssetID,
			PreviewURL:        example.PreviewURL,
			InputAssetURL:     example.InputAssetURL,
			OutputAssetURL:    example.OutputAssetURL,
			MetadataJSON:      mustEncodeJSON(example.Metadata),
			SortOrder:         idx + 1,
		})
	}
	return items
}

func examplesFromSeeds(versionID string, examples []TemplateExampleSeed) []models.TemplateCatalogExample {
	items := make([]models.TemplateCatalogExample, 0, len(examples))
	for idx, example := range examples {
		items = append(items, models.TemplateCatalogExample{
			TemplateVersionID: versionID,
			ExampleType:       defaultString(example.ExampleType, "preview"),
			Title:             example.Title,
			Description:       example.Description,
			SourceRef:         example.SourceRef,
			StorageKey:        example.StorageKey,
			AssetID:           example.AssetID,
			PreviewURL:        example.PreviewURL,
			InputAssetURL:     example.InputAssetURL,
			OutputAssetURL:    example.OutputAssetURL,
			MetadataJSON:      mustEncodeJSON(example.Metadata),
			SortOrder:         idx + 1,
		})
	}
	return items
}

func firstMood(moods []string) string {
	if len(moods) == 0 {
		return ""
	}
	return moods[0]
}

func seedRecommendScore(plan string) int {
	switch plan {
	case "growth":
		return 80
	case "pro":
		return 60
	default:
		return 40
	}
}

func loadEmbeddedTemplateSeedLibrary() TemplateSeedLibrary {
	var library TemplateSeedLibrary
	_ = json.Unmarshal(embeddedTemplateSeedJSON, &library)
	if library.Meta.Platforms == nil {
		library.Meta.Platforms = []TemplatePlatformOption{}
	}
	return library
}

func mergeSeedMetadata(seed map[string]any, extra map[string]any) map[string]any {
	out := map[string]any{}
	for key, value := range seed {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}
