package studio

import (
	"encoding/json"
	"errors"
	"maps"
	"slices"
	"strings"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/models"
	audit "menu-service/internal/modules/audit"
	"menu-service/internal/platform"
	"menu-service/internal/repository"

	"gorm.io/gorm"
)

type Service struct {
	repo      *repository.StudioRepository
	shareRepo *repository.ShareRepository
	userRepo  *repository.UserRepository
	audit     *audit.Service
	platform  *platform.Client
	appCfg    config.AppConfig
	cfg       config.StudioConfig
	registry  *ProviderRegistry
	queue     JobQueue
}

type StyleDimension struct {
	Type  string `json:"type"`
	Key   string `json:"key"`
	Label string `json:"label"`
}

type StyleExecutionProfile struct {
	Provider               string            `json:"provider,omitempty"`
	Model                  string            `json:"model,omitempty"`
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
	ParentVariantID string         `json:"parent_variant_id,omitempty"`
	Status          string         `json:"status"`
	Index           int            `json:"index"`
	Score           float64        `json:"score"`
	IsSelected      bool           `json:"is_selected"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type GenerationJobSummary struct {
	JobID             string                      `json:"job_id"`
	UserID            string                      `json:"user_id"`
	Mode              string                      `json:"mode"`
	Status            string                      `json:"status"`
	Stage             string                      `json:"stage"`
	StageMessage      string                      `json:"stage_message"`
	Provider          string                      `json:"provider"`
	ProviderJobID     string                      `json:"provider_job_id,omitempty"`
	IdempotencyKey    string                      `json:"idempotency_key,omitempty"`
	StylePresetID     string                      `json:"style_preset_id,omitempty"`
	ParentJobID       string                      `json:"parent_job_id,omitempty"`
	BatchRootID       string                      `json:"batch_root_id,omitempty"`
	ParentVariantID   string                      `json:"parent_variant_id,omitempty"`
	SourceAssetIDs    []string                    `json:"source_asset_ids"`
	RequestedVariants int                         `json:"requested_variants"`
	ChildJobCount     int                         `json:"child_job_count,omitempty"`
	Progress          int                         `json:"progress"`
	QueuePosition     int                         `json:"queue_position,omitempty"`
	EtaSeconds        int                         `json:"eta_seconds,omitempty"`
	ErrorCode         string                      `json:"error_code,omitempty"`
	ErrorMessage      string                      `json:"error_message,omitempty"`
	SelectedVariantID string                      `json:"selected_variant_id,omitempty"`
	PromptSnapshot    StyleExecutionProfile       `json:"prompt_snapshot"`
	ParamsSnapshot    map[string]any              `json:"params_snapshot,omitempty"`
	Metadata          map[string]any              `json:"metadata,omitempty"`
	Variants          []GenerationVariantSummary  `json:"variants,omitempty"`
	AttemptCount      int                         `json:"attempt_count"`
	MaxAttempts       int                         `json:"max_attempts"`
	NextRetryAt       *string                     `json:"next_retry_at,omitempty"`
	TimeoutAt         *string                     `json:"timeout_at,omitempty"`
	HeartbeatAt       *string                     `json:"heartbeat_at,omitempty"`
	CreatedAt         string                      `json:"created_at"`
	UpdatedAt         string                      `json:"updated_at"`
	CompletedAt       *string                     `json:"completed_at,omitempty"`
	CanceledAt        *string                     `json:"canceled_at,omitempty"`
	Charge            *GenerationJobChargeSummary `json:"charge,omitempty"`
	ChildJobs         []GenerationJobSummaryLite  `json:"child_jobs,omitempty"`
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
	Mode              string         `json:"mode" binding:"required,oneof=single batch variation refinement"`
	Provider          string         `json:"provider"`
	IdempotencyKey    string         `json:"idempotency_key"`
	StylePresetID     string         `json:"style_preset_id"`
	SourceAssetIDs    []string       `json:"source_asset_ids" binding:"required,min=1"`
	ParentJobID       string         `json:"parent_job_id"`
	ParentVariantID   string         `json:"parent_variant_id"`
	RequestedVariants int            `json:"requested_variants"`
	Params            map[string]any `json:"params"`
	Metadata          map[string]any `json:"metadata"`
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

func NewService(repo *repository.StudioRepository, shareRepo *repository.ShareRepository, userRepo *repository.UserRepository, auditService *audit.Service, platformClient *platform.Client, appCfg config.AppConfig, cfg config.StudioConfig) *Service {
	return &Service{
		repo:      repo,
		shareRepo: shareRepo,
		userRepo:  userRepo,
		audit:     auditService,
		platform:  platformClient,
		appCfg:    appCfg,
		cfg:       defaultStudioConfig(cfg),
		registry:  NewProviderRegistry(),
		queue:     newNoopQueue(),
	}
}

func (s *Service) RegisterAsset(userID, orgID string, input RegisterAssetInput) (*AssetSummary, error) {
	item := &models.StudioAsset{
		UserID:         userID,
		OrganizationID: orgID,
		AssetType:      input.AssetType,
		SourceType:     input.SourceType,
		Status:         "ready",
		FileName:       input.FileName,
		MimeType:       input.MimeType,
		StorageKey:     input.StorageKey,
		SourceURL:      input.SourceURL,
		PreviewURL:     firstNonEmpty(input.PreviewURL, input.SourceURL),
		Width:          input.Width,
		Height:         input.Height,
		FileSize:       input.FileSize,
		Metadata:       mustEncodeJSON(input.Metadata),
	}
	if err := s.repo.CreateAsset(item); err != nil {
		return nil, err
	}
	_ = s.createActivity(userID, orgID, "studio.asset", "Register asset", "succeeded", 0, "", "")
	return mapAsset(item), nil
}

func (s *Service) ListAssets(userID, orgID, assetType, status string) ([]AssetSummary, error) {
	items, err := s.repo.ListAssets(orgID, "", assetType, status)
	if err != nil {
		return nil, err
	}
	out := make([]AssetSummary, 0, len(items))
	for _, item := range items {
		out = append(out, *mapAsset(&item))
	}
	return out, nil
}

func (s *Service) AssetLibrary(userID, orgID, assetType, status, query string, limit, offset int) (*AssetLibraryResult, error) {
	items, total, err := s.repo.ListAssetsPaginated(orgID, "", assetType, status, query, limit, offset)
	if err != nil {
		return nil, err
	}
	assetIDs := make([]string, 0, len(items))
	for _, item := range items {
		assetIDs = append(assetIDs, item.ID)
	}
	shareByAssetID := map[string]*models.SharePost{}
	if s.shareRepo != nil {
		posts, shareErr := s.shareRepo.FindPostsByAssetIDs(orgID, assetIDs)
		if shareErr != nil {
			return nil, shareErr
		}
		for _, post := range posts {
			if _, exists := shareByAssetID[post.AssetID]; !exists {
				postCopy := post
				shareByAssetID[post.AssetID] = &postCopy
			}
		}
	}
	out := make([]AssetLibraryItem, 0, len(items))
	for _, item := range items {
		libraryItem, itemErr := s.mapAssetLibraryItem(userID, orgID, &item, shareByAssetID[item.ID])
		if itemErr != nil {
			return nil, itemErr
		}
		out = append(out, *libraryItem)
	}
	return &AssetLibraryResult{Items: out, Total: total}, nil
}

func (s *Service) CreateStylePreset(userID, orgID string, input CreateStylePresetInput) (*StylePresetSummary, error) {
	if input.PreviewAssetID != "" {
		if _, err := s.repo.FindAssetByID(orgID, input.PreviewAssetID); err != nil {
			return nil, err
		}
	}
	item := &models.StylePreset{
		OrganizationID:   orgID,
		CreatedByUserID:  userID,
		Name:             strings.TrimSpace(input.Name),
		Description:      input.Description,
		Visibility:       input.Visibility,
		Status:           "active",
		Version:          1,
		ParentStyleID:    "",
		PreviewAssetID:   input.PreviewAssetID,
		DimensionsJSON:   mustEncodeJSON(input.Dimensions),
		TagsJSON:         mustEncodeJSON(normalizeTags(input.Tags)),
		ExecutionProfile: mustEncodeJSON(input.ExecutionProfile),
		Metadata:         mustEncodeJSON(input.Metadata),
	}
	if err := s.repo.CreateStylePreset(item); err != nil {
		return nil, err
	}
	return mapStylePreset(item), nil
}

func (s *Service) ListStylePresets(orgID, visibility, status string) ([]StylePresetSummary, error) {
	items, err := s.repo.ListStylePresets(orgID, visibility, defaultString(status, "active"))
	if err != nil {
		return nil, err
	}
	out := make([]StylePresetSummary, 0, len(items))
	for _, item := range items {
		out = append(out, *mapStylePreset(&item))
	}
	return out, nil
}

func (s *Service) GetStylePreset(orgID, styleID string) (*StylePresetSummary, error) {
	item, err := s.repo.FindStylePresetByID(orgID, styleID)
	if err != nil {
		return nil, err
	}
	return mapStylePreset(item), nil
}

func (s *Service) ForkStylePreset(userID, orgID, styleID string, input ForkStylePresetInput) (*StylePresetSummary, error) {
	parent, err := s.repo.FindStylePresetByID(orgID, styleID)
	if err != nil {
		return nil, err
	}
	profile := decodeExecutionProfile(parent.ExecutionProfile)
	if input.ExecutionProfile != nil {
		profile = *input.ExecutionProfile
	}
	item := &models.StylePreset{
		OrganizationID:   orgID,
		CreatedByUserID:  userID,
		Name:             firstNonEmpty(strings.TrimSpace(input.Name), parent.Name+" Copy"),
		Description:      parent.Description,
		Visibility:       firstNonEmpty(input.Visibility, "private"),
		Status:           "active",
		Version:          parent.Version + 1,
		ParentStyleID:    parent.ID,
		PreviewAssetID:   parent.PreviewAssetID,
		DimensionsJSON:   parent.DimensionsJSON,
		TagsJSON:         parent.TagsJSON,
		ExecutionProfile: mustEncodeJSON(profile),
		Metadata:         parent.Metadata,
	}
	if err := s.repo.CreateStylePreset(item); err != nil {
		return nil, err
	}
	return mapStylePreset(item), nil
}

func (s *Service) CreateGenerationJob(userID, orgID string, input CreateGenerationJobInput) (*GenerationJobSummary, error) {
	if input.IdempotencyKey != "" {
		if existing, err := s.repo.FindGenerationJobByIdempotencyKey(orgID, userID, input.IdempotencyKey); err == nil {
			return s.GetGenerationJob(orgID, existing.ID)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	if err := s.validateSourceAssets(orgID, input.SourceAssetIDs); err != nil {
		return nil, err
	}
	var promptSnapshot StyleExecutionProfile
	providerName := firstNonEmpty(input.Provider, s.cfg.DefaultProvider)
	if input.StylePresetID != "" {
		style, err := s.repo.FindStylePresetByID(orgID, input.StylePresetID)
		if err != nil {
			return nil, err
		}
		promptSnapshot = decodeExecutionProfile(style.ExecutionProfile)
		providerName = firstNonEmpty(input.Provider, promptSnapshot.Provider, s.cfg.DefaultProvider)
	}
	requestedVariants := input.RequestedVariants
	if requestedVariants <= 0 {
		requestedVariants = s.cfg.DefaultVariantCount
	}
	if input.Mode == "batch" && len(input.SourceAssetIDs) > 1 {
		var idempotencyKey *string
		if input.IdempotencyKey != "" {
			idempotencyKey = stringPtr(input.IdempotencyKey)
		}
		root := &models.GenerationJob{
			UserID:            userID,
			OrganizationID:    orgID,
			Mode:              "batch",
			Status:            "queued",
			Stage:             "queued",
			StageMessage:      "Batch job created and waiting for child dispatch",
			Provider:          providerName,
			IdempotencyKey:    idempotencyKey,
			StylePresetID:     input.StylePresetID,
			SourceAssetIDs:    mustEncodeJSON(input.SourceAssetIDs),
			RequestedVariants: requestedVariants,
			ChildJobCount:     len(input.SourceAssetIDs),
			Progress:          0,
			PromptSnapshot:    mustEncodeJSON(promptSnapshot),
			ParamsSnapshot:    mustEncodeJSON(input.Params),
			Metadata:          mustEncodeJSON(input.Metadata),
			MaxAttempts:       s.cfg.MaxAttempts,
		}
		if err := s.repo.CreateGenerationJob(root); err != nil {
			return nil, err
		}
		for index, assetID := range input.SourceAssetIDs {
			child := &models.GenerationJob{
				UserID:            userID,
				OrganizationID:    orgID,
				Mode:              "single",
				Status:            "queued",
				Stage:             "queued",
				StageMessage:      "Batch child job queued",
				Provider:          providerName,
				StylePresetID:     input.StylePresetID,
				ParentJobID:       root.ID,
				BatchRootID:       root.ID,
				ParentVariantID:   input.ParentVariantID,
				SourceAssetIDs:    mustEncodeJSON([]string{assetID}),
				RequestedVariants: requestedVariants,
				Progress:          0,
				PromptSnapshot:    mustEncodeJSON(promptSnapshot),
				ParamsSnapshot:    mustEncodeJSON(mergeMaps(input.Params, map[string]any{"batch_index": index})),
				Metadata:          mustEncodeJSON(input.Metadata),
				MaxAttempts:       s.cfg.MaxAttempts,
			}
			if err := s.repo.CreateGenerationJob(child); err != nil {
				return nil, err
			}
			if err := s.createChargeIntentForJob(child); err != nil {
				return nil, err
			}
			if err := s.queue.EnqueueDispatch(child.ID, 0); err != nil {
				return nil, err
			}
		}
		_ = s.createActivity(userID, orgID, "studio.job", "Create batch generation job", "queued", 0, "", root.ID)
		return s.GetGenerationJob(orgID, root.ID)
	}
	item := &models.GenerationJob{
		UserID:            userID,
		OrganizationID:    orgID,
		Mode:              input.Mode,
		Status:            "queued",
		Stage:             "queued",
		StageMessage:      "Job created and waiting for dispatch",
		Provider:          providerName,
		IdempotencyKey:    optionalString(input.IdempotencyKey),
		StylePresetID:     input.StylePresetID,
		ParentJobID:       input.ParentJobID,
		ParentVariantID:   input.ParentVariantID,
		BatchRootID:       "",
		SourceAssetIDs:    mustEncodeJSON(input.SourceAssetIDs),
		RequestedVariants: requestedVariants,
		Progress:          0,
		QueuePosition:     0,
		EtaSeconds:        0,
		PromptSnapshot:    mustEncodeJSON(promptSnapshot),
		ParamsSnapshot:    mustEncodeJSON(input.Params),
		SelectedVariantID: "",
		Metadata:          mustEncodeJSON(input.Metadata),
		MaxAttempts:       s.cfg.MaxAttempts,
	}
	if err := s.repo.CreateGenerationJob(item); err != nil {
		return nil, err
	}
	if err := s.createChargeIntentForJob(item); err != nil {
		return nil, err
	}
	if err := s.queue.EnqueueDispatch(item.ID, 0); err != nil {
		return nil, err
	}
	_ = s.createActivity(userID, orgID, "studio.job", "Create generation job", "queued", 0, "", item.ID)
	return s.GetGenerationJob(orgID, item.ID)
}

func (s *Service) ListGenerationJobs(userID, orgID, status string) ([]GenerationJobSummary, error) {
	items, err := s.repo.ListGenerationJobs(orgID, userID, status)
	if err != nil {
		return nil, err
	}
	out := make([]GenerationJobSummary, 0, len(items))
	for _, item := range items {
		job, err := s.GetGenerationJob(orgID, item.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, *job)
	}
	return out, nil
}

func (s *Service) JobHistory(userID, orgID, status string, limit, offset int) (*JobHistoryResult, error) {
	items, total, err := s.repo.ListGenerationJobsPaginated(orgID, userID, status, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]JobHistoryItem, 0, len(items))
	for _, item := range items {
		job, jobErr := s.GetGenerationJob(orgID, item.ID)
		if jobErr != nil {
			return nil, jobErr
		}
		entry, entryErr := s.mapJobHistoryItem(orgID, job)
		if entryErr != nil {
			return nil, entryErr
		}
		out = append(out, *entry)
	}
	return &JobHistoryResult{Items: out, Total: total}, nil
}

func (s *Service) GetGenerationJob(orgID, jobID string) (*GenerationJobSummary, error) {
	item, err := s.repo.FindGenerationJobByID(orgID, jobID)
	if err != nil {
		return nil, err
	}
	variants, err := s.repo.ListGenerationVariants(item.ID)
	if err != nil {
		return nil, err
	}
	childJobs, err := s.repo.ListChildGenerationJobs(item.ID)
	if err != nil {
		return nil, err
	}
	var chargeIntent *models.StudioChargeIntent
	if item.Mode != "batch" {
		if intent, chargeErr := s.repo.FindChargeIntentByJobID(item.ID); chargeErr == nil {
			chargeIntent = intent
		} else if !errors.Is(chargeErr, gorm.ErrRecordNotFound) {
			return nil, chargeErr
		}
	}
	return s.mapGenerationJob(item, variants, childJobs, chargeIntent), nil
}

func (s *Service) RecordJobResults(userID, orgID, jobID string, input RecordJobResultsInput) (*GenerationJobSummary, error) {
	job, err := s.repo.FindGenerationJobByID(orgID, jobID)
	if err != nil {
		return nil, err
	}
	job.Status = input.Status
	job.Progress = clampProgress(input.Progress, input.Status)
	job.Stage = mapStatusToStage(input.Status)
	job.StageMessage = defaultString(stringMapValue(input.Metadata, "stage_message"), defaultStageMessage(job.Stage, input.Status))
	job.ErrorCode = input.ErrorCode
	job.ErrorMessage = input.ErrorMessage
	job.Metadata = mergeJSON(job.Metadata, input.Metadata)
	now := time.Now()
	job.HeartbeatAt = &now
	job.TimeoutAt = nil
	job.NextRetryAt = nil
	if input.Status == "completed" {
		job.CompletedAt = &now
	}
	if input.Status == "canceled" {
		job.CanceledAt = &now
	}
	if err := s.repo.SaveGenerationJob(job); err != nil {
		return nil, err
	}
	for _, variantInput := range input.Variants {
		var assetID string
		if variantInput.Asset.SourceURL != "" || variantInput.Asset.StorageKey != "" {
			assetSummary, err := s.RegisterAsset(userID, orgID, variantInput.Asset)
			if err != nil {
				return nil, err
			}
			assetID = assetSummary.ID
		}
		variant, variantErr := s.repo.FindGenerationVariantByIndex(job.ID, variantInput.Index)
		if variantErr != nil && !errors.Is(variantErr, gorm.ErrRecordNotFound) {
			return nil, variantErr
		}
		if errors.Is(variantErr, gorm.ErrRecordNotFound) {
			variant = &models.GenerationVariant{
				JobID:        job.ID,
				VariantIndex: variantInput.Index,
			}
		}
		variant.AssetID = assetID
		variant.ParentVariantID = variantInput.ParentVariantID
		variant.Status = variantInput.Status
		variant.Score = variantInput.Score
		variant.IsSelected = variantInput.IsSelected
		variant.Metadata = mustEncodeJSON(variantInput.Metadata)
		if variant.ID == "" {
			if err := s.repo.CreateGenerationVariant(variant); err != nil {
				return nil, err
			}
		} else {
			if err := s.repo.SaveGenerationVariant(variant); err != nil {
				return nil, err
			}
		}
		if variant.IsSelected {
			job.SelectedVariantID = variant.ID
		}
	}
	if job.SelectedVariantID != "" {
		if err := s.repo.SaveGenerationJob(job); err != nil {
			return nil, err
		}
	}
	if input.Status == "completed" {
		_ = s.createActivity(userID, orgID, "studio.job", "Generation completed", "succeeded", 0, "", job.ID)
		_ = s.finalizeChargeIntent(job)
	}
	if input.Status == "failed" {
		_ = s.createActivity(userID, orgID, "studio.job", "Generation failed", "failed", 0, "", job.ID)
		_ = s.releaseChargeIntent(job)
	}
	if input.Status == "canceled" {
		_ = s.releaseChargeIntent(job)
	}
	if job.BatchRootID != "" {
		_ = s.refreshBatchRoot(orgID, job.BatchRootID)
	}
	return s.GetGenerationJob(orgID, jobID)
}

func (s *Service) SelectVariant(userID, orgID, jobID, variantID string) (*GenerationJobSummary, error) {
	job, err := s.repo.FindGenerationJobByID(orgID, jobID)
	if err != nil {
		return nil, err
	}
	variant, err := s.repo.FindGenerationVariant(jobID, variantID)
	if err != nil {
		return nil, err
	}
	if err := s.repo.ClearSelectedVariants(jobID); err != nil {
		return nil, err
	}
	variant.IsSelected = true
	if err := s.repo.SaveGenerationVariant(variant); err != nil {
		return nil, err
	}
	job.SelectedVariantID = variant.ID
	if err := s.repo.SaveGenerationJob(job); err != nil {
		return nil, err
	}
	assetURL := ""
	if variant.AssetID != "" {
		if asset, assetErr := s.repo.FindAssetByID(orgID, variant.AssetID); assetErr == nil {
			assetURL = firstNonEmpty(asset.SourceURL, asset.PreviewURL)
		}
	}
	_ = s.createActivity(userID, orgID, "studio.variant", "Select generation result", "succeeded", 0, assetURL, job.ID)
	return s.GetGenerationJob(orgID, jobID)
}

func (s *Service) UpdateJobRuntime(jobID string, input UpdateJobRuntimeInput) (*GenerationJobSummary, error) {
	job, err := s.repo.FindGenerationJobByIDGlobal(jobID)
	if err != nil {
		return nil, err
	}
	if input.Status != "" {
		job.Status = input.Status
	}
	if input.Stage != "" {
		job.Stage = input.Stage
	}
	if input.StageMessage != "" {
		job.StageMessage = input.StageMessage
	}
	if input.Progress != nil {
		job.Progress = clampProgress(*input.Progress, defaultString(job.Status, "processing"))
	}
	if input.QueuePosition != nil {
		job.QueuePosition = *input.QueuePosition
	}
	if input.EtaSeconds != nil {
		job.EtaSeconds = *input.EtaSeconds
	}
	if input.ProviderJobID != "" {
		job.ProviderJobID = input.ProviderJobID
	}
	if input.ErrorCode != "" {
		job.ErrorCode = input.ErrorCode
	}
	if input.ErrorMessage != "" {
		job.ErrorMessage = input.ErrorMessage
	}
	now := time.Now()
	job.HeartbeatAt = &now
	job.Metadata = mergeJSON(job.Metadata, input.Metadata)
	if input.Status == "completed" && job.CompletedAt == nil {
		job.CompletedAt = &now
	}
	if input.Status == "canceled" && job.CanceledAt == nil {
		job.CanceledAt = &now
	}
	if err := s.repo.SaveGenerationJob(job); err != nil {
		return nil, err
	}
	if job.BatchRootID != "" {
		_ = s.refreshBatchRoot(job.OrganizationID, job.BatchRootID)
	}
	return s.GetGenerationJob(job.OrganizationID, job.ID)
}

func (s *Service) RecordJobResultsInternal(jobID string, input RecordJobResultsInput) (*GenerationJobSummary, error) {
	job, err := s.repo.FindGenerationJobByIDGlobal(jobID)
	if err != nil {
		return nil, err
	}
	return s.RecordJobResults(job.UserID, job.OrganizationID, job.ID, input)
}

func (s *Service) validateSourceAssets(orgID string, assetIDs []string) error {
	for _, assetID := range assetIDs {
		if _, err := s.repo.FindAssetByID(orgID, assetID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) createActivity(userID, orgID, actionType, actionName, status string, creditsUsed int64, resultURL, jobID string) error {
	if s.userRepo == nil {
		return nil
	}
	return s.userRepo.CreateActivity(&models.Activity{
		UserID:         userID,
		OrganizationID: orgID,
		ActionType:     actionType,
		ActionName:     actionName,
		Status:         status,
		CreditsUsed:    creditsUsed,
		ResultURL:      resultURL,
		JobID:          jobID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})
}

func mapAsset(item *models.StudioAsset) *AssetSummary {
	return &AssetSummary{
		ID:         item.ID,
		AssetType:  item.AssetType,
		SourceType: item.SourceType,
		Status:     item.Status,
		FileName:   item.FileName,
		MimeType:   item.MimeType,
		StorageKey: item.StorageKey,
		SourceURL:  item.SourceURL,
		PreviewURL: item.PreviewURL,
		Width:      item.Width,
		Height:     item.Height,
		FileSize:   item.FileSize,
		Metadata:   decodeMap(item.Metadata),
		CreatedAt:  item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func mapStylePreset(item *models.StylePreset) *StylePresetSummary {
	return &StylePresetSummary{
		StyleID:          item.ID,
		Name:             item.Name,
		Description:      item.Description,
		Visibility:       item.Visibility,
		Status:           item.Status,
		Version:          item.Version,
		ParentStyleID:    item.ParentStyleID,
		PreviewAssetID:   item.PreviewAssetID,
		Dimensions:       decodeDimensions(item.DimensionsJSON),
		Tags:             decodeStringSlice(item.TagsJSON),
		ExecutionProfile: decodeExecutionProfile(item.ExecutionProfile),
		Metadata:         decodeMap(item.Metadata),
		CreatedByUserID:  item.CreatedByUserID,
		CreatedAt:        item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:        item.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func (s *Service) mapGenerationJob(item *models.GenerationJob, variants []models.GenerationVariant, childJobs []models.GenerationJob, chargeIntent *models.StudioChargeIntent) *GenerationJobSummary {
	out := &GenerationJobSummary{
		JobID:             item.ID,
		UserID:            item.UserID,
		Mode:              item.Mode,
		Status:            item.Status,
		Stage:             item.Stage,
		StageMessage:      item.StageMessage,
		Provider:          item.Provider,
		ProviderJobID:     item.ProviderJobID,
		IdempotencyKey:    derefString(item.IdempotencyKey),
		StylePresetID:     item.StylePresetID,
		ParentJobID:       item.ParentJobID,
		BatchRootID:       item.BatchRootID,
		ParentVariantID:   item.ParentVariantID,
		SourceAssetIDs:    decodeStringSlice(item.SourceAssetIDs),
		RequestedVariants: item.RequestedVariants,
		ChildJobCount:     item.ChildJobCount,
		Progress:          item.Progress,
		QueuePosition:     item.QueuePosition,
		EtaSeconds:        item.EtaSeconds,
		ErrorCode:         item.ErrorCode,
		ErrorMessage:      item.ErrorMessage,
		SelectedVariantID: item.SelectedVariantID,
		PromptSnapshot:    decodeExecutionProfile(item.PromptSnapshot),
		ParamsSnapshot:    decodeMap(item.ParamsSnapshot),
		Metadata:          decodeMap(item.Metadata),
		CreatedAt:         item.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:         item.UpdatedAt.UTC().Format(time.RFC3339),
		Variants:          make([]GenerationVariantSummary, 0, len(variants)),
		AttemptCount:      item.AttemptCount,
		MaxAttempts:       item.MaxAttempts,
		ChildJobs:         make([]GenerationJobSummaryLite, 0, len(childJobs)),
	}
	out.Charge = s.mapGenerationJobCharge(item, chargeIntent)
	if item.CompletedAt != nil {
		value := item.CompletedAt.UTC().Format(time.RFC3339)
		out.CompletedAt = &value
	}
	if item.NextRetryAt != nil {
		value := item.NextRetryAt.UTC().Format(time.RFC3339)
		out.NextRetryAt = &value
	}
	if item.TimeoutAt != nil {
		value := item.TimeoutAt.UTC().Format(time.RFC3339)
		out.TimeoutAt = &value
	}
	if item.HeartbeatAt != nil {
		value := item.HeartbeatAt.UTC().Format(time.RFC3339)
		out.HeartbeatAt = &value
	}
	if item.CanceledAt != nil {
		value := item.CanceledAt.UTC().Format(time.RFC3339)
		out.CanceledAt = &value
	}
	for _, variant := range variants {
		out.Variants = append(out.Variants, GenerationVariantSummary{
			VariantID:       variant.ID,
			AssetID:         variant.AssetID,
			ParentVariantID: variant.ParentVariantID,
			Status:          variant.Status,
			Index:           variant.VariantIndex,
			Score:           variant.Score,
			IsSelected:      variant.IsSelected,
			Metadata:        decodeMap(variant.Metadata),
		})
	}
	for _, child := range childJobs {
		out.ChildJobs = append(out.ChildJobs, GenerationJobSummaryLite{
			JobID:        child.ID,
			Status:       child.Status,
			Stage:        child.Stage,
			StageMessage: child.StageMessage,
			Progress:     child.Progress,
			ErrorCode:    child.ErrorCode,
			ErrorMessage: child.ErrorMessage,
			Mode:         child.Mode,
			EtaSeconds:   child.EtaSeconds,
		})
	}
	return out
}

func (s *Service) mapGenerationJobCharge(job *models.GenerationJob, intent *models.StudioChargeIntent) *GenerationJobChargeSummary {
	summary := &GenerationJobChargeSummary{
		BillingEnabled:           s.cfg.BillingEnabled,
		Billable:                 s.cfg.BillingEnabled && job.Mode != "batch",
		ChargePriorityAssetCodes: s.chargePriorityAssetCodes(),
	}
	if !summary.Billable {
		return summary
	}
	chargeMode, billableItemCode := s.billableItemForMode(job.Mode)
	summary.ChargeMode = chargeMode
	summary.ResourceType = s.cfg.ResourceType
	summary.BillableItemCode = billableItemCode
	if intent == nil {
		return summary
	}
	summary.ChargeMode = firstNonEmpty(intent.ChargeMode, summary.ChargeMode)
	summary.ResourceType = firstNonEmpty(intent.ResourceType, summary.ResourceType)
	summary.BillableItemCode = firstNonEmpty(intent.BillableItemCode, summary.BillableItemCode)
	summary.Status = intent.Status
	summary.FailureCode = intent.FailureCode
	summary.FailureMessage = intent.FailureMessage
	summary.ReservationID = intent.ReservationID
	summary.SettlementID = intent.SettlementID
	summary.EstimatedUnits = intent.EstimatedUnits
	summary.FinalUnits = intent.FinalUnits

	metadata := decodeMap(intent.Metadata)
	if settlement, ok := metadata["settlement"].(map[string]any); ok {
		summary.Currency = stringMapValue(settlement, "currency")
		summary.WalletAssetCode = stringMapValue(settlement, "wallet_asset_code")
		summary.QuotaConsumed = int64MapValue(settlement, "quota_consumed")
		summary.CreditsConsumed = int64MapValue(settlement, "credits_consumed")
		summary.WalletDebited = int64MapValue(settlement, "wallet_debited")
		summary.GrossAmount = int64MapValue(settlement, "gross_amount")
		summary.DiscountAmount = int64MapValue(settlement, "discount_amount")
		summary.NetAmount = int64MapValue(settlement, "net_amount")
	}
	return summary
}

func (s *Service) chargePriorityAssetCodes() []string {
	out := make([]string, 0, 3)
	for _, code := range []string{
		s.appCfg.AllowanceAssetCode,
		s.appCfg.RewardAssetCode,
		s.appCfg.CreditsAssetCode,
	} {
		if code == "" {
			continue
		}
		if !slices.Contains(out, code) {
			out = append(out, code)
		}
	}
	return out
}

func (s *Service) mapAssetLibraryItem(userID, orgID string, item *models.StudioAsset, sharePost *models.SharePost) (*AssetLibraryItem, error) {
	_ = userID
	out := &AssetLibraryItem{
		Asset:      *mapAsset(item),
		OriginRole: item.AssetType,
		CanRefine:  item.Status == "ready" && (item.AssetType == "source" || item.AssetType == "generated"),
		CanShare:   item.Status == "ready" && item.AssetType == "generated",
	}
	if variant, err := s.repo.FindGenerationVariantByAssetID(item.ID); err == nil {
		out.VariantID = variant.ID
		out.ProducedByJobID = variant.JobID
		if job, jobErr := s.repo.FindGenerationJobByID(orgID, variant.JobID); jobErr == nil {
			out.LatestJob = mapJobLite(job)
		} else if !errors.Is(jobErr, gorm.ErrRecordNotFound) {
			return nil, jobErr
		}
		if variant.IsSelected {
			out.OriginRole = "selected_result"
		} else {
			out.OriginRole = "generated_result"
		}
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	} else if job, jobErr := s.repo.FindLatestJobUsingAsset(orgID, item.ID); jobErr == nil {
		out.LatestJob = mapJobLite(job)
		out.ProducedByJobID = job.ID
		out.OriginRole = "source_asset"
	} else if !errors.Is(jobErr, gorm.ErrRecordNotFound) {
		return nil, jobErr
	}
	if sharePost != nil {
		out.Share = mapSharePostSummary(sharePost)
	}
	return out, nil
}

func (s *Service) mapJobHistoryItem(orgID string, job *GenerationJobSummary) (*JobHistoryItem, error) {
	out := &JobHistoryItem{Job: job}
	for _, assetID := range job.SourceAssetIDs {
		if assetID == "" {
			continue
		}
		item, err := s.repo.FindAssetByID(orgID, assetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		out.SourceAssets = append(out.SourceAssets, *mapAsset(item))
	}
	for _, variant := range job.Variants {
		if variant.AssetID == "" {
			continue
		}
		asset, err := s.repo.FindAssetByID(orgID, variant.AssetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue
			}
			return nil, err
		}
		mapped := mapAsset(asset)
		out.ResultAssets = append(out.ResultAssets, *mapped)
		if variant.IsSelected || variant.VariantID == job.SelectedVariantID {
			out.SelectedAsset = mapped
		}
	}
	return out, nil
}

func mapJobLite(item *models.GenerationJob) *GenerationJobSummaryLite {
	if item == nil {
		return nil
	}
	return &GenerationJobSummaryLite{
		JobID:        item.ID,
		Status:       item.Status,
		Stage:        item.Stage,
		StageMessage: item.StageMessage,
		Progress:     item.Progress,
		ErrorCode:    item.ErrorCode,
		ErrorMessage: item.ErrorMessage,
		Mode:         item.Mode,
		EtaSeconds:   item.EtaSeconds,
	}
}

func mapSharePostSummary(item *models.SharePost) *SharePostSummary {
	if item == nil {
		return nil
	}
	out := &SharePostSummary{
		ShareID:       item.ID,
		Status:        item.Status,
		Visibility:    item.Visibility,
		ShareURL:      item.ShareURL,
		ViewCount:     item.ViewCount,
		LikeCount:     item.LikeCount,
		FavoriteCount: item.FavoriteCount,
		Metadata:      decodeMap(item.Metadata),
	}
	if item.PublishedAt != nil {
		value := item.PublishedAt.UTC().Format(time.RFC3339)
		out.PublishedAt = &value
	}
	return out
}

func clampProgress(progress int, status string) int {
	if status == "completed" {
		return 100
	}
	if progress < 0 {
		return 0
	}
	if progress > 100 {
		return 100
	}
	return progress
}

func normalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]struct{}{}
	for _, item := range tags {
		value := strings.TrimSpace(item)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func mustEncodeJSON(value any) string {
	if value == nil {
		return ""
	}
	data, _ := json.Marshal(value)
	return string(data)
}

func decodeStringSlice(raw string) []string {
	if raw == "" {
		return []string{}
	}
	out := []string{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func decodeDimensions(raw string) []StyleDimension {
	if raw == "" {
		return []StyleDimension{}
	}
	out := []StyleDimension{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func decodeExecutionProfile(raw string) StyleExecutionProfile {
	if raw == "" {
		return StyleExecutionProfile{}
	}
	var out StyleExecutionProfile
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func decodeMap(raw string) map[string]any {
	if raw == "" {
		return map[string]any{}
	}
	out := map[string]any{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func mergeJSON(raw string, incoming map[string]any) string {
	if len(incoming) == 0 {
		return raw
	}
	current := decodeMap(raw)
	maps.Copy(current, incoming)
	return mustEncodeJSON(current)
}

func mergeMaps(base map[string]any, incoming map[string]any) map[string]any {
	out := map[string]any{}
	if len(base) > 0 {
		maps.Copy(out, base)
	}
	if len(incoming) > 0 {
		maps.Copy(out, incoming)
	}
	return out
}

func mapStatusToStage(status string) string {
	switch status {
	case "processing":
		return "running"
	case "completed":
		return "completed"
	case "failed":
		return "failed"
	case "canceled":
		return "canceled"
	default:
		return status
	}
}

func defaultStageMessage(stage, status string) string {
	switch stage {
	case "running":
		return "Provider is processing the job"
	case "completed":
		return "Job completed successfully"
	case "failed":
		return "Job failed"
	case "canceled":
		return "Job canceled"
	case "queued":
		return "Job is waiting in the queue"
	case "retry_scheduled":
		return "Job is scheduled for retry"
	default:
		return defaultString(status, "Job updated")
	}
}

func stringMapValue(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	if value, ok := values[key]; ok {
		if str, strOK := value.(string); strOK {
			return str
		}
	}
	return ""
}

func int64MapValue(values map[string]any, key string) int64 {
	if values == nil {
		return 0
	}
	value, ok := values[key]
	if !ok {
		return 0
	}
	switch typed := value.(type) {
	case int64:
		return typed
	case int:
		return int64(typed)
	case float64:
		return int64(typed)
	case float32:
		return int64(typed)
	default:
		return 0
	}
}

func optionalString(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func stringPtr(value string) *string {
	return &value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

var ErrInvalidVariantSelection = errors.New("invalid variant selection")
