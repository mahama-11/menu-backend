package studio

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
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
	repo         *repository.StudioRepository
	templateRepo *repository.TemplateCenterRepository
	shareRepo    *repository.ShareRepository
	userRepo     *repository.UserRepository
	audit        *audit.Service
	platform     *platform.Client
	appCfg       config.AppConfig
	cfg          config.StudioConfig
	security     config.SecurityConfig
}

func NewService(repo *repository.StudioRepository, templateRepo *repository.TemplateCenterRepository, shareRepo *repository.ShareRepository, userRepo *repository.UserRepository, auditService *audit.Service, platformClient *platform.Client, appCfg config.AppConfig, cfg config.StudioConfig, securityCfg config.SecurityConfig) *Service {
	return &Service{
		repo:         repo,
		templateRepo: templateRepo,
		shareRepo:    shareRepo,
		userRepo:     userRepo,
		audit:        auditService,
		platform:     platformClient,
		appCfg:       appCfg,
		cfg:          cfg,
		security:     securityCfg,
	}
}

func (s *Service) WithContext(ctx context.Context) *Service {
	if s == nil || ctx == nil {
		return s
	}
	clone := *s
	if s.platform != nil {
		clone.platform = s.platform.WithContext(ctx)
	}
	return &clone
}

func (s *Service) RegisterAsset(userID, orgID string, input RegisterAssetInput) (*AssetSummary, error) {
	normalizedInput := input
	if payload := firstNonEmpty(strings.TrimSpace(input.SourceURL), strings.TrimSpace(input.PreviewURL)); payload != "" && !strings.HasPrefix(payload, "http://") && !strings.HasPrefix(payload, "https://") {
		if s.platform == nil {
			return nil, fmt.Errorf("platform client is required for asset upload")
		}
		stored, err := s.platform.UploadAsset(platform.UploadAssetInput{
			ProductCode: s.cfg.ProductCode,
			Category:    "studio-assets",
			FileName:    input.FileName,
			MimeType:    input.MimeType,
			Payload:     payload,
		})
		if err != nil {
			return nil, err
		}
		normalizedInput.StorageKey = stored.StorageKey
		normalizedInput.SourceURL = ""
		normalizedInput.PreviewURL = ""
		normalizedInput.MimeType = firstNonEmpty(stored.MimeType, input.MimeType)
		normalizedInput.FileSize = stored.FileSize
	}
	item := &models.StudioAsset{
		UserID:         userID,
		OrganizationID: orgID,
		AssetType:      normalizedInput.AssetType,
		SourceType:     normalizedInput.SourceType,
		Status:         "ready",
		FileName:       normalizedInput.FileName,
		MimeType:       normalizedInput.MimeType,
		StorageKey:     normalizedInput.StorageKey,
		SourceURL:      normalizedInput.SourceURL,
		PreviewURL:     firstNonEmpty(normalizedInput.PreviewURL, normalizedInput.SourceURL),
		Width:          normalizedInput.Width,
		Height:         normalizedInput.Height,
		FileSize:       normalizedInput.FileSize,
		Metadata:       mustEncodeJSON(normalizedInput.Metadata),
	}
	if err := s.repo.CreateAsset(item); err != nil {
		return nil, err
	}
	_ = s.createActivity(userID, orgID, "studio.asset", "Register asset", "succeeded", 0, "", "")
	return s.mapAsset(item), nil
}

func (s *Service) ListAssets(userID, orgID, assetType, status string) ([]AssetSummary, error) {
	items, err := s.repo.ListAssets(orgID, "", assetType, status)
	if err != nil {
		return nil, err
	}
	out := make([]AssetSummary, 0, len(items))
	for _, item := range items {
		out = append(out, *s.mapAsset(&item))
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
	input, err := normalizeGenerationJobInput(input)
	if err != nil {
		return nil, err
	}
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
	normalizedMetadata := normalizeCreativeSourceMetadata(input.Metadata)
	promptSnapshot, err := s.resolvePromptSnapshot(orgID, input, normalizedMetadata)
	if err != nil {
		return nil, err
	}
	providerName, err := resolveStudioProvider([]string{input.Provider, promptSnapshot.Provider, stringMapValue(normalizedMetadata, "provider")}, input.InputMode, s.cfg.DefaultProvider)
	if err != nil {
		return nil, err
	}
	requestedVariants := input.RequestedVariants
	if requestedVariants <= 0 {
		requestedVariants = s.cfg.DefaultVariantCount
	}
	effectivePrompt := promptSnapshot.PromptTemplate
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
			Prompt:            effectivePrompt,
			PromptSnapshot:    mustEncodeJSON(promptSnapshot),
			ParamsSnapshot:    mustEncodeJSON(input.Params),
			Metadata:          mustEncodeJSON(normalizedMetadata),
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
				Prompt:            effectivePrompt,
				PromptSnapshot:    mustEncodeJSON(promptSnapshot),
				ParamsSnapshot:    mustEncodeJSON(mergeMaps(input.Params, map[string]any{"batch_index": index})),
				Metadata:          mustEncodeJSON(normalizedMetadata),
				MaxAttempts:       s.cfg.MaxAttempts,
			}
			if err := s.repo.CreateGenerationJob(child); err != nil {
				return nil, err
			}
			if err := s.createChargeIntentForJob(child); err != nil {
				return nil, err
			}
			if err := s.createPlatformRuntimeJob(child); err != nil {
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
		Prompt:            effectivePrompt,
		QueuePosition:     0,
		EtaSeconds:        0,
		PromptSnapshot:    mustEncodeJSON(promptSnapshot),
		ParamsSnapshot:    mustEncodeJSON(input.Params),
		SelectedVariantID: "",
		Metadata:          mustEncodeJSON(normalizedMetadata),
		MaxAttempts:       s.cfg.MaxAttempts,
	}
	if err := s.repo.CreateGenerationJob(item); err != nil {
		return nil, err
	}
	if err := s.createChargeIntentForJob(item); err != nil {
		return nil, err
	}
	if err := s.createPlatformRuntimeJob(item); err != nil {
		return nil, err
	}
	_ = s.createActivity(userID, orgID, "studio.job", "Create generation job", "queued", 0, "", item.ID)
	return s.GetGenerationJob(orgID, item.ID)
}

func (s *Service) createPlatformRuntimeJob(item *models.GenerationJob) error {
	if s.platform == nil {
		return fmt.Errorf("platform client is required")
	}
	sourceAssetIDs := decodeStringSlice(item.SourceAssetIDs)
	metadata := decodeMap(item.Metadata)
	paramsSnapshot := decodeMap(item.ParamsSnapshot)
	inputMode := firstNonEmpty(stringMapValue(paramsSnapshot, "input_mode"), stringMapValue(metadata, "input_mode"), inferInputModeFromAssetCount(len(sourceAssetIDs)))
	generationStrategy := firstNonEmpty(stringMapValue(paramsSnapshot, "generation_strategy"), stringMapValue(metadata, "generation_strategy"), inputMode)
	rolesByAssetID := roleByAssetIDFromMetadata(metadata)
	sourceAssets := make([]map[string]any, 0, len(sourceAssetIDs))
	for _, assetID := range sourceAssetIDs {
		asset, err := s.repo.FindAssetByIDGlobal(assetID)
		if err != nil {
			return fmt.Errorf("load source asset %s: %w", assetID, err)
		}
		entry := map[string]any{
			"id":          asset.ID,
			"asset_id":    asset.ID,
			"storage_key": asset.StorageKey,
			"mime_type":   asset.MimeType,
			"width":       asset.Width,
			"height":      asset.Height,
		}
		if role := rolesByAssetID[asset.ID]; role != "" {
			entry["role"] = role
		}
		sourceAssets = append(sourceAssets, entry)
	}
	inputManifest := mustEncodeJSON(map[string]any{
		"mode":                item.Mode,
		"input_mode":          inputMode,
		"generation_strategy": generationStrategy,
		"prompt":              item.Prompt,
		"prompt_snapshot":     decodeExecutionProfile(item.PromptSnapshot),
		"params_snapshot":     paramsSnapshot,
		"source_asset_ids":    sourceAssetIDs,
		"source_assets":       sourceAssets,
		"requested_variants":  item.RequestedVariants,
	})
	routeSnapshot := mustEncodeJSON(map[string]any{
		"provider":            item.Provider,
		"input_mode":          inputMode,
		"generation_strategy": generationStrategy,
	})
	runtimeMetadata := map[string]any{
		"menu_job_id":         item.ID,
		"creative_source":     decodeCreativeSource(item.Metadata),
		"studio_metadata":     metadata,
		"input_mode":          inputMode,
		"generation_strategy": generationStrategy,
		"target_platform":     stringMapValue(paramsSnapshot, "target_platform"),
		"requested_variants":  item.RequestedVariants,
	}
	idempotencyKey := firstNonEmpty(derefString(item.IdempotencyKey), fmt.Sprintf("menu:%s:create_runtime", item.ID))
	runtimeJob, err := s.platform.CreateRuntimeJob(platform.CreateRuntimeJobInput{
		ProductCode:     s.cfg.ProductCode,
		TaskType:        "image_generation",
		ProviderCode:    normalizeRuntimeProviderCode(item.Provider),
		ProviderMode:    "async",
		OrganizationID:  item.OrganizationID,
		UserID:          item.UserID,
		SourceType:      "menu_generation_job",
		SourceID:        item.ID,
		IdempotencyKey:  idempotencyKey,
		ChargeSessionID: item.ChargeSessionID,
		InputManifest:   inputManifest,
		RouteSnapshot:   routeSnapshot,
		Metadata:        mustEncodeJSON(runtimeMetadata),
		Priority:        100,
		MaxAttempts:     item.MaxAttempts,
		TimeoutSeconds:  600,
	})
	if err != nil {
		_ = s.releaseChargeIntent(item)
		now := time.Now()
		item.Status = "failed"
		item.Stage = "runtime_create_failed"
		item.StageMessage = "Failed to create runtime job"
		item.ErrorCode = "RUNTIME_CREATE_FAILED"
		item.ErrorMessage = err.Error()
		item.CompletedAt = &now
		item.TimeoutAt = nil
		item.HeartbeatAt = nil
		item.NextRetryAt = nil
		_ = s.repo.SaveGenerationJob(item)
		return err
	}
	item.RuntimeJobID = runtimeJob.ID
	item.ProviderJobID = runtimeJob.ProviderJobID
	item.Status = firstNonEmpty(runtimeJob.Status, "queued")
	item.Stage = firstNonEmpty(runtimeJob.Stage, "queued")
	item.StageMessage = firstNonEmpty(runtimeJob.StageMessage, "Runtime job queued")
	item.ErrorCode = ""
	item.ErrorMessage = ""
	if err := s.repo.SaveGenerationJob(item); err != nil {
		return err
	}
	return nil
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

func (s *Service) GetAssetContent(orgID, assetID string) (*models.StudioAsset, io.ReadCloser, http.Header, error) {
	var (
		item *models.StudioAsset
		err  error
	)
	if orgID != "" {
		item, err = s.repo.FindAssetByID(orgID, assetID)
	} else {
		item, err = s.repo.FindAssetByIDGlobal(assetID)
	}
	if err != nil {
		return nil, nil, nil, err
	}
	if item.StorageKey == "" {
		return item, nil, nil, fmt.Errorf("asset storage key is empty")
	}
	if s.platform == nil {
		return item, nil, nil, fmt.Errorf("platform client is required")
	}
	body, headers, err := s.platform.DownloadAsset(item.StorageKey)
	if err != nil {
		return item, nil, nil, err
	}
	return item, body, headers, nil
}

func (s *Service) BuildSignedAssetContentURL(assetID string) string {
	expiresAt := time.Now().Add(15 * time.Minute).Unix()
	sig := s.signAssetAccess(assetID, expiresAt)
	values := url.Values{}
	values.Set("expires", strconv.FormatInt(expiresAt, 10))
	values.Set("sig", sig)
	return fmt.Sprintf("/api/v1/menu/studio/assets/%s/content?%s", assetID, values.Encode())
}

func (s *Service) ValidateAssetAccessSignature(assetID string, expiresAt int64, sig string) bool {
	if assetID == "" || sig == "" || expiresAt <= 0 || time.Now().Unix() > expiresAt {
		return false
	}
	expected := s.signAssetAccess(assetID, expiresAt)
	return hmac.Equal([]byte(expected), []byte(sig))
}

func (s *Service) signAssetAccess(assetID string, expiresAt int64) string {
	secret := strings.TrimSpace(s.security.EncryptionKey)
	if secret == "" {
		secret = strings.TrimSpace(s.security.JWTSecret)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(assetID))
	mac.Write([]byte(":"))
	mac.Write([]byte(strconv.FormatInt(expiresAt, 10)))
	return hex.EncodeToString(mac.Sum(nil))
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
		if assetID != "" {
			metadata := decodeMap(variant.Metadata)
			if metadata == nil {
				metadata = map[string]any{}
			}
			if previewURL := stringMapValue(variantInput.Asset.Metadata, "preview_url"); previewURL != "" {
				metadata["preview_url"] = previewURL
			}
			if metadata["preview_url"] == nil && variantInput.Asset.SourceURL != "" {
				metadata["preview_url"] = variantInput.Asset.SourceURL
			}
			variant.Metadata = mustEncodeJSON(metadata)
		}
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
