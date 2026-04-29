package models

import "time"

type TemplateCatalog struct {
	ID               string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	Slug             string    `gorm:"type:varchar(128);uniqueIndex;not null" json:"slug"`
	Name             string    `gorm:"type:varchar(255);index;not null" json:"name"`
	Description      string    `gorm:"type:text" json:"description"`
	Status           string    `gorm:"type:varchar(32);index;not null" json:"status"`
	Scope            string    `gorm:"type:varchar(32);index;not null;default:official" json:"scope"`
	Cuisine          string    `gorm:"type:varchar(64);index" json:"cuisine"`
	DishType         string    `gorm:"type:varchar(64);index" json:"dish_type"`
	PlanRequired     string    `gorm:"type:varchar(32);index;not null;default:basic" json:"plan_required"`
	CreditsCost      int64     `gorm:"not null;default:0" json:"credits_cost"`
	CoverAssetID     string    `gorm:"type:varchar(64);index" json:"cover_asset_id"`
	PlatformsJSON    string    `gorm:"type:text" json:"platforms_json"`
	MoodsJSON        string    `gorm:"type:text" json:"moods_json"`
	TagsJSON         string    `gorm:"type:text" json:"tags_json"`
	MetadataJSON     string    `gorm:"type:text" json:"metadata_json"`
	RecommendScore   int       `gorm:"not null;default:0" json:"recommend_score"`
	SortOrder        int       `gorm:"not null;default:0" json:"sort_order"`
	CurrentVersionID string    `gorm:"type:varchar(64);index" json:"current_version_id"`
	CreatedAt        time.Time `gorm:"index;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type TemplateCatalogVersion struct {
	ID                   string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	TemplateCatalogID    string    `gorm:"type:varchar(64);index:idx_template_catalog_version_catalog,priority:1;not null" json:"template_catalog_id"`
	VersionNo            int       `gorm:"index:idx_template_catalog_version_catalog,priority:2;not null" json:"version_no"`
	Status               string    `gorm:"type:varchar(32);index;not null" json:"status"`
	Name                 string    `gorm:"type:varchar(255)" json:"name"`
	Summary              string    `gorm:"type:text" json:"summary"`
	PromptTemplatesJSON  string    `gorm:"type:text" json:"prompt_templates_json"`
	CopyTemplatesJSON    string    `gorm:"type:text" json:"copy_templates_json"`
	HashtagsJSON         string    `gorm:"type:text" json:"hashtags_json"`
	DesignSpecJSON       string    `gorm:"type:text" json:"design_spec_json"`
	ExportSpecsJSON      string    `gorm:"type:text" json:"export_specs_json"`
	InputSchemaJSON      string    `gorm:"type:text" json:"input_schema_json"`
	ExecutionProfileJSON string    `gorm:"type:text" json:"execution_profile_json"`
	MetadataJSON         string    `gorm:"type:text" json:"metadata_json"`
	CreatedAt            time.Time `gorm:"index;autoCreateTime" json:"created_at"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type TemplateCatalogExample struct {
	ID                string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	TemplateVersionID string    `gorm:"type:varchar(64);index;not null" json:"template_version_id"`
	ExampleType       string    `gorm:"type:varchar(32);index;not null" json:"example_type"`
	Title             string    `gorm:"type:varchar(255)" json:"title"`
	Description       string    `gorm:"type:text" json:"description"`
	SourceRef         string    `gorm:"type:varchar(255);index" json:"source_ref"`
	StorageKey        string    `gorm:"type:varchar(255);index" json:"storage_key"`
	AssetID           string    `gorm:"type:varchar(64);index" json:"asset_id"`
	PreviewURL        string    `gorm:"type:text" json:"preview_url"`
	InputAssetURL     string    `gorm:"type:text" json:"input_asset_url"`
	OutputAssetURL    string    `gorm:"type:text" json:"output_asset_url"`
	MetadataJSON      string    `gorm:"type:text" json:"metadata_json"`
	SortOrder         int       `gorm:"not null;default:0" json:"sort_order"`
	CreatedAt         time.Time `gorm:"index;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type TemplateFavorite struct {
	ID               string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	TemplateCatalogID string   `gorm:"type:varchar(64);uniqueIndex:idx_template_favorite_unique,priority:1;index;not null" json:"template_catalog_id"`
	UserID           string    `gorm:"type:varchar(64);uniqueIndex:idx_template_favorite_unique,priority:2;index;not null" json:"user_id"`
	OrganizationID   string    `gorm:"type:varchar(64);uniqueIndex:idx_template_favorite_unique,priority:3;index;not null" json:"organization_id"`
	CreatedAt        time.Time `gorm:"index;autoCreateTime" json:"created_at"`
}

type TemplateUsageEvent struct {
	ID                string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	TemplateCatalogID string    `gorm:"type:varchar(64);index;not null" json:"template_catalog_id"`
	TemplateVersionID string    `gorm:"type:varchar(64);index" json:"template_version_id"`
	UserID            string    `gorm:"type:varchar(64);index;not null" json:"user_id"`
	OrganizationID    string    `gorm:"type:varchar(64);index;not null" json:"organization_id"`
	EventType         string    `gorm:"type:varchar(32);index;not null" json:"event_type"`
	Status            string    `gorm:"type:varchar(32);index;not null;default:recorded" json:"status"`
	StylePresetID     string    `gorm:"type:varchar(64);index" json:"style_preset_id"`
	JobID             string    `gorm:"type:varchar(64);index" json:"job_id"`
	PayloadJSON       string    `gorm:"type:text" json:"payload_json"`
	CreatedAt         time.Time `gorm:"index;autoCreateTime" json:"created_at"`
}
