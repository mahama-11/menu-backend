package models

import "time"

type StudioAsset struct {
	ID             string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	UserID         string    `gorm:"type:varchar(64);index:idx_studio_asset_org_user,priority:2;not null" json:"user_id"`
	OrganizationID string    `gorm:"type:varchar(64);index:idx_studio_asset_org_user,priority:1;not null" json:"organization_id"`
	AssetType      string    `gorm:"type:varchar(32);index;not null" json:"asset_type"`
	SourceType     string    `gorm:"type:varchar(32);index;not null" json:"source_type"`
	Status         string    `gorm:"type:varchar(32);index;not null" json:"status"`
	FileName       string    `gorm:"type:varchar(255)" json:"file_name"`
	MimeType       string    `gorm:"type:varchar(128)" json:"mime_type"`
	StorageKey     string    `gorm:"type:text" json:"storage_key"`
	SourceURL      string    `gorm:"type:text" json:"source_url"`
	PreviewURL     string    `gorm:"type:text" json:"preview_url"`
	Width          int       `json:"width"`
	Height         int       `json:"height"`
	FileSize       int64     `json:"file_size"`
	Metadata       string    `gorm:"type:text" json:"metadata"`
	CreatedAt      time.Time `gorm:"index;autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type StylePreset struct {
	ID               string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	OrganizationID   string    `gorm:"type:varchar(64);index:idx_style_preset_org_visibility,priority:1;not null" json:"organization_id"`
	CreatedByUserID  string    `gorm:"type:varchar(64);index;not null" json:"created_by_user_id"`
	Name             string    `gorm:"type:varchar(128);index;not null" json:"name"`
	Description      string    `gorm:"type:text" json:"description"`
	Visibility       string    `gorm:"type:varchar(32);index:idx_style_preset_org_visibility,priority:2;not null" json:"visibility"`
	Status           string    `gorm:"type:varchar(32);index;not null" json:"status"`
	Version          int       `gorm:"not null;default:1" json:"version"`
	ParentStyleID    string    `gorm:"type:varchar(64);index" json:"parent_style_id"`
	PreviewAssetID   string    `gorm:"type:varchar(64);index" json:"preview_asset_id"`
	DimensionsJSON   string    `gorm:"type:text" json:"dimensions_json"`
	TagsJSON         string    `gorm:"type:text" json:"tags_json"`
	ExecutionProfile string    `gorm:"type:text" json:"execution_profile"`
	Metadata         string    `gorm:"type:text" json:"metadata"`
	CreatedAt        time.Time `gorm:"index;autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type GenerationJob struct {
	ID                string     `gorm:"type:varchar(64);primaryKey" json:"id"`
	UserID            string     `gorm:"type:varchar(64);index:idx_generation_job_org_user_created,priority:2;uniqueIndex:idx_generation_job_idempotency,priority:2;not null" json:"user_id"`
	OrganizationID    string     `gorm:"type:varchar(64);index:idx_generation_job_org_user_created,priority:1;uniqueIndex:idx_generation_job_idempotency,priority:1;not null" json:"organization_id"`
	Mode              string     `gorm:"type:varchar(32);index;not null" json:"mode"`
	Status            string     `gorm:"type:varchar(32);index;not null" json:"status"`
	Stage             string     `gorm:"type:varchar(32);index;not null" json:"stage"`
	StageMessage      string     `gorm:"type:text" json:"stage_message"`
	Provider          string     `gorm:"type:varchar(64);index;not null" json:"provider"`
	ProviderJobID     string     `gorm:"type:varchar(128);index" json:"provider_job_id"`
	IdempotencyKey    *string    `gorm:"type:varchar(128);uniqueIndex:idx_generation_job_idempotency,priority:3" json:"idempotency_key,omitempty"`
	StylePresetID     string     `gorm:"type:varchar(64);index" json:"style_preset_id"`
	ParentJobID       string     `gorm:"type:varchar(64);index" json:"parent_job_id"`
	BatchRootID       string     `gorm:"type:varchar(64);index" json:"batch_root_id"`
	ParentVariantID   string     `gorm:"type:varchar(64);index" json:"parent_variant_id"`
	SourceAssetIDs    string     `gorm:"type:text" json:"source_asset_ids"`
	RequestedVariants int        `gorm:"not null;default:1" json:"requested_variants"`
	ChildJobCount     int        `gorm:"not null;default:0" json:"child_job_count"`
	Progress          int        `gorm:"not null;default:0" json:"progress"`
	QueuePosition     int        `json:"queue_position"`
	EtaSeconds        int        `json:"eta_seconds"`
	PromptSnapshot    string     `gorm:"type:text" json:"prompt_snapshot"`
	ParamsSnapshot    string     `gorm:"type:text" json:"params_snapshot"`
	ErrorCode         string     `gorm:"type:varchar(64)" json:"error_code"`
	ErrorMessage      string     `gorm:"type:text" json:"error_message"`
	SelectedVariantID string     `gorm:"type:varchar(64);index" json:"selected_variant_id"`
	Metadata          string     `gorm:"type:text" json:"metadata"`
	AttemptCount      int        `gorm:"not null;default:0" json:"attempt_count"`
	MaxAttempts       int        `gorm:"not null;default:3" json:"max_attempts"`
	NextRetryAt       *time.Time `gorm:"index" json:"next_retry_at,omitempty"`
	TimeoutAt         *time.Time `gorm:"index" json:"timeout_at,omitempty"`
	HeartbeatAt       *time.Time `gorm:"index" json:"heartbeat_at,omitempty"`
	CreatedAt         time.Time  `gorm:"index:idx_generation_job_org_user_created,priority:3,sort:desc;autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
	CompletedAt       *time.Time `json:"completed_at,omitempty"`
	CanceledAt        *time.Time `json:"canceled_at,omitempty"`
}

type GenerationVariant struct {
	ID              string    `gorm:"type:varchar(64);primaryKey" json:"id"`
	JobID           string    `gorm:"type:varchar(64);uniqueIndex:idx_generation_variant_job_index,priority:1;not null" json:"job_id"`
	AssetID         string    `gorm:"type:varchar(64);index" json:"asset_id"`
	ParentVariantID string    `gorm:"type:varchar(64);index" json:"parent_variant_id"`
	Status          string    `gorm:"type:varchar(32);index;not null" json:"status"`
	VariantIndex    int       `gorm:"uniqueIndex:idx_generation_variant_job_index,priority:2;not null" json:"variant_index"`
	Score           float64   `json:"score"`
	IsSelected      bool      `gorm:"index" json:"is_selected"`
	Metadata        string    `gorm:"type:text" json:"metadata"`
	CreatedAt       time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type StudioChargeIntent struct {
	ID                string     `gorm:"type:varchar(64);primaryKey" json:"id"`
	JobID             string     `gorm:"type:varchar(64);uniqueIndex;not null" json:"job_id"`
	BatchRootID       string     `gorm:"type:varchar(64);index" json:"batch_root_id"`
	UserID            string     `gorm:"type:varchar(64);index;not null" json:"user_id"`
	OrganizationID    string     `gorm:"type:varchar(64);index;not null" json:"organization_id"`
	ProductCode       string     `gorm:"type:varchar(64);index;not null" json:"product_code"`
	ChargeMode        string     `gorm:"type:varchar(32);index;not null" json:"charge_mode"`
	ResourceType      string     `gorm:"type:varchar(32);index;not null" json:"resource_type"`
	BillableItemCode  string     `gorm:"type:varchar(128);index;not null" json:"billable_item_code"`
	EstimatedUnits    int64      `json:"estimated_units"`
	FinalUnits        int64      `json:"final_units"`
	ReservationID     string     `gorm:"type:varchar(64);index" json:"reservation_id"`
	ReservationKey    string     `gorm:"type:varchar(128);uniqueIndex;not null" json:"reservation_key"`
	FinalizationID    string     `gorm:"type:varchar(128);uniqueIndex" json:"finalization_id"`
	EventID           string     `gorm:"type:varchar(128);uniqueIndex;not null" json:"event_id"`
	SettlementID      string     `gorm:"type:varchar(64);index" json:"settlement_id"`
	Provider          string     `gorm:"type:varchar(64);index" json:"provider"`
	ProviderJobID     string     `gorm:"type:varchar(128);index" json:"provider_job_id"`
	Status            string     `gorm:"type:varchar(32);index;not null" json:"status"`
	FailureCode       string     `gorm:"type:varchar(64)" json:"failure_code"`
	FailureMessage    string     `gorm:"type:text" json:"failure_message"`
	Metadata          string     `gorm:"type:text" json:"metadata"`
	ReservedAt        *time.Time `json:"reserved_at,omitempty"`
	FinalizedAt       *time.Time `json:"finalized_at,omitempty"`
	ReleasedAt        *time.Time `json:"released_at,omitempty"`
	CreatedAt         time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}
