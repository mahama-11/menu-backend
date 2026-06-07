package studio

type StyleDimension struct {
	Type  string `json:"type"`
	Key   string `json:"key"`
	Label string `json:"label"`
}

type StyleExecutionProfile struct {
	Provider               string            `json:"provider,omitempty"`
	Model                  string            `json:"model,omitempty"`
	SystemPrompt           string            `json:"system_prompt,omitempty"`
	StylePrompt            string            `json:"style_prompt,omitempty"`
	UserPrompt             string            `json:"user_prompt,omitempty"`
	PromptTemplate         string            `json:"prompt_template,omitempty"`
	NegativePromptTemplate string            `json:"negative_prompt_template,omitempty"`
	ParameterProfile       map[string]any    `json:"parameter_profile,omitempty"`
	Variables              map[string]string `json:"variables,omitempty"`
}

type AssetSummary struct {
	ID         string         `json:"id"`
	AssetType  string         `json:"asset_type"`
	SourceType string         `json:"source_type"`
	Status     string         `json:"status"`
	FileName   string         `json:"file_name"`
	MimeType   string         `json:"mime_type"`
	StorageKey string         `json:"storage_key"`
	SourceURL  string         `json:"source_url"`
	PreviewURL string         `json:"preview_url"`
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	FileSize   int64          `json:"file_size"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  string         `json:"created_at"`
	UpdatedAt  string         `json:"updated_at"`
}

type StylePresetSummary struct {
	StyleID          string                `json:"style_id"`
	Name             string                `json:"name"`
	Description      string                `json:"description,omitempty"`
	Visibility       string                `json:"visibility"`
	Status           string                `json:"status"`
	Version          int                   `json:"version"`
	ParentStyleID    string                `json:"parent_style_id,omitempty"`
	PreviewAssetID   string                `json:"preview_asset_id,omitempty"`
	Dimensions       []StyleDimension      `json:"dimensions"`
	Tags             []string              `json:"tags"`
	ExecutionProfile StyleExecutionProfile `json:"execution_profile"`
	Metadata         map[string]any        `json:"metadata,omitempty"`
	CreatedByUserID  string                `json:"created_by_user_id"`
	CreatedAt        string                `json:"created_at"`
	UpdatedAt        string                `json:"updated_at"`
}

type GenerationVariantSummary struct {
	VariantID       string         `json:"variant_id"`
	AssetID         string         `json:"asset_id,omitempty"`
	Asset           *AssetSummary  `json:"asset,omitempty"`
	PreviewURL      string         `json:"preview_url,omitempty"`
	ParentVariantID string         `json:"parent_variant_id,omitempty"`
	Status          string         `json:"status"`
	Index           int            `json:"index"`
	Score           float64        `json:"score"`
	IsSelected      bool           `json:"is_selected"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type GenerationJobSummary struct {
	JobID              string                      `json:"job_id"`
	UserID             string                      `json:"user_id"`
	Mode               string                      `json:"mode"`
	InputMode          string                      `json:"input_mode,omitempty"`
	GenerationStrategy string                      `json:"generation_strategy,omitempty"`
	Status             string                      `json:"status"`
	Stage              string                      `json:"stage"`
	StageMessage       string                      `json:"stage_message"`
	Provider           string                      `json:"provider"`
	ProviderJobID      string                      `json:"provider_job_id,omitempty"`
	IdempotencyKey     string                      `json:"idempotency_key,omitempty"`
	StylePresetID      string                      `json:"style_preset_id,omitempty"`
	ParentJobID        string                      `json:"parent_job_id,omitempty"`
	BatchRootID        string                      `json:"batch_root_id,omitempty"`
	ParentVariantID    string                      `json:"parent_variant_id,omitempty"`
	SourceAssetIDs     []string                    `json:"source_asset_ids"`
	RequestedVariants  int                         `json:"requested_variants"`
	ChildJobCount      int                         `json:"child_job_count,omitempty"`
	Progress           int                         `json:"progress"`
	QueuePosition      int                         `json:"queue_position,omitempty"`
	EtaSeconds         int                         `json:"eta_seconds,omitempty"`
	ErrorCode          string                      `json:"error_code,omitempty"`
	ErrorMessage       string                      `json:"error_message,omitempty"`
	SelectedVariantID  string                      `json:"selected_variant_id,omitempty"`
	PromptSnapshot     StyleExecutionProfile       `json:"prompt_snapshot"`
	CreativeSource     *CreativeSourceSnapshot     `json:"creative_source,omitempty"`
	ParamsSnapshot     map[string]any              `json:"params_snapshot,omitempty"`
	Metadata           map[string]any              `json:"metadata,omitempty"`
	Variants           []GenerationVariantSummary  `json:"variants,omitempty"`
	AttemptCount       int                         `json:"attempt_count"`
	MaxAttempts        int                         `json:"max_attempts"`
	NextRetryAt        *string                     `json:"next_retry_at,omitempty"`
	TimeoutAt          *string                     `json:"timeout_at,omitempty"`
	HeartbeatAt        *string                     `json:"heartbeat_at,omitempty"`
	CreatedAt          string                      `json:"created_at"`
	UpdatedAt          string                      `json:"updated_at"`
	CompletedAt        *string                     `json:"completed_at,omitempty"`
	CanceledAt         *string                     `json:"canceled_at,omitempty"`
	Charge             *GenerationJobChargeSummary `json:"charge,omitempty"`
	ChildJobs          []GenerationJobSummaryLite  `json:"child_jobs,omitempty"`
}

type SharePostSummary struct {
	ShareID       string         `json:"share_id"`
	Status        string         `json:"status"`
	Visibility    string         `json:"visibility"`
	ShareURL      string         `json:"share_url,omitempty"`
	ViewCount     int64          `json:"view_count"`
	LikeCount     int64          `json:"like_count"`
	FavoriteCount int64          `json:"favorite_count"`
	Metadata      map[string]any `json:"metadata,omitempty"`
	PublishedAt   *string        `json:"published_at,omitempty"`
}

type AssetLibraryItem struct {
	Asset           AssetSummary              `json:"asset"`
	OriginRole      string                    `json:"origin_role"`
	ProducedByJobID string                    `json:"produced_by_job_id,omitempty"`
	VariantID       string                    `json:"variant_id,omitempty"`
	LatestJob       *GenerationJobSummaryLite `json:"latest_job,omitempty"`
	CanRefine       bool                      `json:"can_refine"`
	CanShare        bool                      `json:"can_share"`
	Share           *SharePostSummary         `json:"share,omitempty"`
}

type AssetLibraryResult struct {
	Items []AssetLibraryItem `json:"items"`
	Total int64              `json:"total"`
}

type JobHistoryItem struct {
	Job           *GenerationJobSummary `json:"job"`
	SourceAssets  []AssetSummary        `json:"source_assets,omitempty"`
	ResultAssets  []AssetSummary        `json:"result_assets,omitempty"`
	SelectedAsset *AssetSummary         `json:"selected_asset,omitempty"`
}

type JobHistoryResult struct {
	Items []JobHistoryItem `json:"items"`
	Total int64            `json:"total"`
}

type GenerationJobSummaryLite struct {
	JobID        string `json:"job_id"`
	Status       string `json:"status"`
	Stage        string `json:"stage"`
	StageMessage string `json:"stage_message"`
	Progress     int    `json:"progress"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
	Mode         string `json:"mode"`
	EtaSeconds   int    `json:"eta_seconds,omitempty"`
}

type GenerationJobChargeSummary struct {
	BillingEnabled           bool     `json:"billing_enabled"`
	Billable                 bool     `json:"billable"`
	ChargeMode               string   `json:"charge_mode,omitempty"`
	ResourceType             string   `json:"resource_type,omitempty"`
	BillableItemCode         string   `json:"billable_item_code,omitempty"`
	Status                   string   `json:"status,omitempty"`
	FailureCode              string   `json:"failure_code,omitempty"`
	FailureMessage           string   `json:"failure_message,omitempty"`
	ReservationID            string   `json:"reservation_id,omitempty"`
	SettlementID             string   `json:"settlement_id,omitempty"`
	EstimatedUnits           int64    `json:"estimated_units,omitempty"`
	FinalUnits               int64    `json:"final_units,omitempty"`
	Currency                 string   `json:"currency,omitempty"`
	QuotaConsumed            int64    `json:"quota_consumed,omitempty"`
	CreditsConsumed          int64    `json:"credits_consumed,omitempty"`
	WalletAssetCode          string   `json:"wallet_asset_code,omitempty"`
	WalletDebited            int64    `json:"wallet_debited,omitempty"`
	GrossAmount              int64    `json:"gross_amount,omitempty"`
	DiscountAmount           int64    `json:"discount_amount,omitempty"`
	NetAmount                int64    `json:"net_amount,omitempty"`
	ChargePriorityAssetCodes []string `json:"charge_priority_asset_codes,omitempty"`
}

type RegisterAssetInput struct {
	AssetType  string         `json:"asset_type" binding:"required,oneof=source generated mask reference"`
	SourceType string         `json:"source_type" binding:"required,oneof=upload import generated"`
	FileName   string         `json:"file_name"`
	MimeType   string         `json:"mime_type"`
	StorageKey string         `json:"storage_key"`
	SourceURL  string         `json:"source_url"`
	PreviewURL string         `json:"preview_url"`
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	FileSize   int64          `json:"file_size"`
	Metadata   map[string]any `json:"metadata"`
}

type CreateStylePresetInput struct {
	Name             string                `json:"name" binding:"required"`
	Description      string                `json:"description"`
	Visibility       string                `json:"visibility" binding:"required,oneof=private organization public"`
	PreviewAssetID   string                `json:"preview_asset_id"`
	Dimensions       []StyleDimension      `json:"dimensions"`
	Tags             []string              `json:"tags"`
	ExecutionProfile StyleExecutionProfile `json:"execution_profile"`
	Metadata         map[string]any        `json:"metadata"`
}

type ForkStylePresetInput struct {
	Name             string                 `json:"name"`
	Visibility       string                 `json:"visibility" binding:"omitempty,oneof=private organization public"`
	ExecutionProfile *StyleExecutionProfile `json:"execution_profile,omitempty"`
}

type CreateGenerationJobInput struct {
	Mode               string                   `json:"mode" binding:"required,oneof=single batch variation refinement"`
	InputMode          string                   `json:"input_mode" binding:"omitempty,oneof=text_to_image image_to_image multi_image image_edit ask_for_required_input"`
	GenerationStrategy string                   `json:"generation_strategy" binding:"omitempty,oneof=text_to_image image_to_image multi_image image_edit ask_for_required_input"`
	Provider           string                   `json:"provider"`
	IdempotencyKey     string                   `json:"idempotency_key"`
	StylePresetID      string                   `json:"style_preset_id"`
	SourceAssetIDs     []string                 `json:"source_asset_ids"`
	SourceAssets       []StudioSourceAssetInput `json:"source_assets"`
	Prompt             string                   `json:"prompt"`
	ParentJobID        string                   `json:"parent_job_id"`
	ParentVariantID    string                   `json:"parent_variant_id"`
	RequestedVariants  int                      `json:"requested_variants"`
	Params             map[string]any           `json:"params"`
	Metadata           map[string]any           `json:"metadata"`
}

type CreativeSourceSnapshot struct {
	SourceType        string `json:"source_type,omitempty"`
	SourceID          string `json:"source_id,omitempty"`
	Title             string `json:"title,omitempty"`
	PlanRequired      string `json:"plan_required,omitempty"`
	CreditsCost       int64  `json:"credits_cost,omitempty"`
	TargetPlatform    string `json:"target_platform,omitempty"`
	TemplateID        string `json:"template_id,omitempty"`
	TemplateVersionID string `json:"template_version_id,omitempty"`
	StylePresetID     string `json:"style_preset_id,omitempty"`
}

type RecordJobResultsInput struct {
	Status       string                  `json:"status" binding:"required,oneof=processing completed failed canceled"`
	Progress     int                     `json:"progress"`
	ErrorCode    string                  `json:"error_code"`
	ErrorMessage string                  `json:"error_message"`
	Variants     []RecordJobVariantInput `json:"variants"`
	Metadata     map[string]any          `json:"metadata"`
}

type UpdateJobRuntimeInput struct {
	Status        string         `json:"status"`
	Stage         string         `json:"stage"`
	StageMessage  string         `json:"stage_message"`
	Progress      *int           `json:"progress"`
	QueuePosition *int           `json:"queue_position"`
	EtaSeconds    *int           `json:"eta_seconds"`
	ProviderJobID string         `json:"provider_job_id"`
	ErrorCode     string         `json:"error_code"`
	ErrorMessage  string         `json:"error_message"`
	Metadata      map[string]any `json:"metadata"`
}

type RecordJobVariantInput struct {
	Index           int                `json:"index" binding:"required"`
	ParentVariantID string             `json:"parent_variant_id"`
	Status          string             `json:"status" binding:"required,oneof=pending ready failed"`
	Score           float64            `json:"score"`
	IsSelected      bool               `json:"is_selected"`
	Asset           RegisterAssetInput `json:"asset"`
	Metadata        map[string]any     `json:"metadata"`
}

type SelectVariantInput struct {
	VariantID string `json:"variant_id" binding:"required"`
}
