package templatecenter

import (
	"errors"
	"fmt"
	"strings"

	"menu-service/internal/models"
	audit "menu-service/internal/modules/audit"
	studio "menu-service/internal/modules/studio"
	"menu-service/internal/platform"
	"menu-service/internal/repository"

	"gorm.io/gorm"
)

type Service struct {
	repo       *repository.TemplateCenterRepository
	studioRepo *repository.StudioRepository
	audit      *audit.Service
	platform   *platform.Client
}

type ListCatalogInput struct {
	Cuisine  string
	DishType string
	Platform string
	Mood     string
	Query    string
	Plan     string
	Source   string
}

type TemplateInputSlot struct {
	Role        string `json:"role"`
	Label       string `json:"label"`
	Required    bool   `json:"required"`
	Accept      string `json:"accept,omitempty"`
	Description string `json:"description,omitempty"`
	MaxCount    int    `json:"max_count,omitempty"`
}

type TemplateTargetOutput map[string]any

type TemplateCatalogSummary struct {
	TemplateID     string                 `json:"template_id"`
	Slug           string                 `json:"slug"`
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Cuisine        string                 `json:"cuisine"`
	DishType       string                 `json:"dish_type"`
	Platforms      []string               `json:"platforms"`
	Moods          []string               `json:"moods"`
	Tags           []string               `json:"tags"`
	PlanRequired   string                 `json:"plan_required"`
	CreditsCost    int64                  `json:"credits_cost"`
	Locked         bool                   `json:"locked"`
	IsFavorite     bool                   `json:"is_favorite"`
	CoverAssetID   string                 `json:"cover_asset_id,omitempty"`
	RecommendScore int                    `json:"recommend_score"`
	BusinessGoal   string                 `json:"business_goal,omitempty"`
	InputSlots     []TemplateInputSlot    `json:"input_slots,omitempty"`
	TargetOutputs  []TemplateTargetOutput `json:"target_outputs,omitempty"`
	StrategyPolicy map[string]any         `json:"strategy_policy,omitempty"`
}

type TemplateCatalogDetail struct {
	TemplateCatalogSummary
	CurrentVersionID string                          `json:"current_version_id"`
	PromptTemplates  map[string]string               `json:"prompt_templates"`
	CopyTemplates    map[string]any                  `json:"copy_templates"`
	Hashtags         map[string][]string             `json:"hashtags"`
	DesignSpec       map[string]any                  `json:"design_spec"`
	ExportSpecs      map[string]any                  `json:"export_specs"`
	InputSchema      map[string]any                  `json:"input_schema"`
	ExecutionProfile studio.StyleExecutionProfile    `json:"execution_profile"`
	Examples         []models.TemplateCatalogExample `json:"examples"`
	Metadata         map[string]any                  `json:"metadata,omitempty"`
}

type UseTemplateInput struct {
	TargetPlatform string            `json:"target_platform" binding:"required"`
	Language       string            `json:"language"`
	UploadImageURL string            `json:"upload_image_url"`
	CustomCopy     map[string]string `json:"custom_copy"`
}

type UseTemplateResult struct {
	TemplateID        string                          `json:"template_id"`
	TemplateVersionID string                          `json:"template_version_id"`
	TargetRoute       string                          `json:"target_route"`
	TargetMethod      string                          `json:"target_method"`
	CreditsCost       int64                           `json:"credits_cost"`
	PlanRequired      string                          `json:"plan_required"`
	PrefilledJob      studio.CreateGenerationJobInput `json:"prefilled_job"`
	InputSlots        []TemplateInputSlot             `json:"input_slots,omitempty"`
	TargetOutputs     []TemplateTargetOutput          `json:"target_outputs,omitempty"`
	ResolvedStrategy  map[string]any                  `json:"resolved_strategy,omitempty"`
	TemplateContext   map[string]any                  `json:"template_context"`
}

type CopyTemplateInput struct {
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}

type CopiedTemplateResult struct {
	StyleID         string `json:"style_id"`
	Name            string `json:"name"`
	Visibility      string `json:"visibility"`
	SourceCatalogID string `json:"source_catalog_id"`
	SourceVersionID string `json:"source_version_id"`
}

func NewService(repo *repository.TemplateCenterRepository, studioRepo *repository.StudioRepository, auditService *audit.Service, platformClient *platform.Client) *Service {
	return &Service{repo: repo, studioRepo: studioRepo, audit: auditService, platform: platformClient}
}

func (s *Service) Bootstrap() error {
	for index, seed := range defaultTemplateSeeds() {
		desiredCatalog := seed.catalogModel()
		desiredCatalog.SortOrder = index + 1
		catalog, err := s.repo.FindCatalogByID(seed.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			catalog = desiredCatalog
			if err := s.repo.CreateCatalog(catalog); err != nil {
				return err
			}
		} else {
			catalog.Slug = desiredCatalog.Slug
			catalog.Name = desiredCatalog.Name
			catalog.Description = desiredCatalog.Description
			catalog.Status = desiredCatalog.Status
			catalog.Scope = desiredCatalog.Scope
			catalog.Cuisine = desiredCatalog.Cuisine
			catalog.DishType = desiredCatalog.DishType
			catalog.PlanRequired = desiredCatalog.PlanRequired
			catalog.CreditsCost = desiredCatalog.CreditsCost
			catalog.PlatformsJSON = desiredCatalog.PlatformsJSON
			catalog.MoodsJSON = desiredCatalog.MoodsJSON
			catalog.TagsJSON = desiredCatalog.TagsJSON
			catalog.MetadataJSON = desiredCatalog.MetadataJSON
			catalog.RecommendScore = desiredCatalog.RecommendScore
			catalog.SortOrder = desiredCatalog.SortOrder
		}
		versionID := seed.ID + "-v1"
		desiredVersion := seed.versionModel()
		version, err := s.repo.FindCatalogVersionByID(versionID)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := s.repo.CreateCatalogVersion(desiredVersion); err != nil {
				return err
			}
		} else {
			version.Status = desiredVersion.Status
			version.Name = desiredVersion.Name
			version.Summary = desiredVersion.Summary
			version.PromptTemplatesJSON = desiredVersion.PromptTemplatesJSON
			version.CopyTemplatesJSON = desiredVersion.CopyTemplatesJSON
			version.HashtagsJSON = desiredVersion.HashtagsJSON
			version.DesignSpecJSON = desiredVersion.DesignSpecJSON
			version.ExportSpecsJSON = desiredVersion.ExportSpecsJSON
			version.InputSchemaJSON = desiredVersion.InputSchemaJSON
			version.ExecutionProfileJSON = desiredVersion.ExecutionProfileJSON
			version.MetadataJSON = desiredVersion.MetadataJSON
			if err := s.repo.SaveCatalogVersion(version); err != nil {
				return err
			}
		}
		if err := s.repo.ReplaceCatalogExamples(versionID, seed.exampleModels(versionID)); err != nil {
			return err
		}
		if catalog.CurrentVersionID != versionID || catalog.SortOrder != index+1 {
			catalog.CurrentVersionID = versionID
		}
		if err := s.repo.SaveCatalog(catalog); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Meta() TemplateMetaResult {
	return defaultTemplateMeta()
}

func (s *Service) ListCatalogs(userID, orgID string, input ListCatalogInput) ([]TemplateCatalogSummary, error) {
	favoriteMap, err := s.favoriteMap(userID, orgID)
	if err != nil {
		return nil, err
	}
	currentScope := s.resolveTemplateScope(orgID, input.Plan)
	if s.platform != nil && !strings.EqualFold(strings.TrimSpace(input.Source), "local") {
		if result, err := s.platform.InternalTemplateCatalog("menu"); err == nil && result != nil && len(result.Items) > 0 {
			out := make([]TemplateCatalogSummary, 0, len(result.Items))
			for _, item := range result.Items {
				summary := mapPlatformCatalogSummary(item, input.Plan, currentScope)
				summary.IsFavorite = favoriteMap[item.TemplateID]
				if matchesCatalogFilters(summary, input) {
					out = append(out, summary)
				}
			}
			return out, nil
		}
	}
	items, err := s.repo.ListCatalogs(repository.TemplateCatalogListFilter{
		Cuisine:  input.Cuisine,
		DishType: input.DishType,
		Platform: input.Platform,
		Mood:     input.Mood,
		Query:    input.Query,
		Status:   "active",
	})
	if err != nil {
		return nil, err
	}
	out := make([]TemplateCatalogSummary, 0, len(items))
	for _, item := range items {
		summary := mapCatalogSummary(&item, favoriteMap[item.ID], input.Plan, currentScope)
		if item.CurrentVersionID != "" {
			if version, versionErr := s.repo.FindCatalogVersionByID(item.CurrentVersionID); versionErr == nil {
				applyTemplateContract(&summary, decodeMapAny(version.InputSchemaJSON))
			}
		}
		out = append(out, summary)
	}
	return out, nil
}

func (s *Service) GetCatalogDetail(userID, orgID, templateID, plan string) (*TemplateCatalogDetail, error) {
	if detail, err := s.platformCatalogDetail(userID, orgID, templateID, plan); err == nil && detail != nil {
		return detail, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return s.GetLocalCatalogDetail(userID, orgID, templateID, plan)
}

func (s *Service) GetLocalCatalogDetail(userID, orgID, templateID, plan string) (*TemplateCatalogDetail, error) {
	catalog, version, err := s.loadCatalogAndVersion(templateID)
	if err != nil {
		return nil, err
	}
	examples, err := s.repo.ListCatalogExamples(version.ID)
	if err != nil {
		return nil, err
	}
	favorite, err := s.repo.FindFavorite(templateID, userID, orgID)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	inputSchema := decodeMapAny(version.InputSchemaJSON)
	detail := &TemplateCatalogDetail{
		TemplateCatalogSummary: mapCatalogSummary(catalog, favorite != nil, plan, s.resolveTemplateScope(orgID, plan)),
		CurrentVersionID:       version.ID,
		PromptTemplates:        decodeMapString(version.PromptTemplatesJSON),
		CopyTemplates:          decodeMapAny(version.CopyTemplatesJSON),
		Hashtags:               decodeMapStringSlice(version.HashtagsJSON),
		DesignSpec:             decodeMapAny(version.DesignSpecJSON),
		ExportSpecs:            decodeMapAny(version.ExportSpecsJSON),
		InputSchema:            inputSchema,
		ExecutionProfile:       decodeExecutionProfile(version.ExecutionProfileJSON),
		Examples:               examples,
		Metadata:               decodeMapAny(version.MetadataJSON),
	}
	applyTemplateContract(&detail.TemplateCatalogSummary, inputSchema)
	detail.ExecutionProfile = enrichTemplateDetailPromptProfile(detail, "")
	return detail, nil
}

func (s *Service) ListFavorites(userID, orgID, plan string) ([]TemplateCatalogSummary, error) {
	favorites, err := s.repo.ListFavorites(userID, orgID)
	if err != nil {
		return nil, err
	}
	out := make([]TemplateCatalogSummary, 0, len(favorites))
	for _, favorite := range favorites {
		catalog, err := s.repo.FindCatalogByID(favorite.TemplateCatalogID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				if detail, platformErr := s.platformCatalogDetail(userID, orgID, favorite.TemplateCatalogID, plan); platformErr == nil && detail != nil {
					summary := detail.TemplateCatalogSummary
					summary.IsFavorite = true
					out = append(out, summary)
					continue
				}
				continue
			}
			return nil, err
		}
		out = append(out, mapCatalogSummary(catalog, true, plan, s.resolveTemplateScope(orgID, plan)))
	}
	return out, nil
}

func (s *Service) SetFavorite(userID, orgID, templateID string) error {
	if _, _, err := s.loadCatalogAndVersion(templateID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if detail, platformErr := s.platformCatalogDetail(userID, orgID, templateID, ""); platformErr != nil || detail == nil {
				if platformErr != nil {
					return platformErr
				}
				return err
			}
		} else {
			return err
		}
	}
	if _, err := s.repo.FindFavorite(templateID, userID, orgID); err == nil {
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if err := s.repo.CreateFavorite(&models.TemplateFavorite{
		TemplateCatalogID: templateID,
		UserID:            userID,
		OrganizationID:    orgID,
	}); err != nil {
		return err
	}
	return s.recordUsageEvent(templateID, "", userID, orgID, "favorite", "recorded", "", "", map[string]any{})
}

func (s *Service) RemoveFavorite(userID, orgID, templateID string) error {
	return s.repo.DeleteFavorite(templateID, userID, orgID)
}

func (s *Service) UseTemplate(userID, orgID, templateID, plan string, input UseTemplateInput) (*UseTemplateResult, error) {
	if detail, err := s.platformCatalogDetail(userID, orgID, templateID, plan); err == nil && detail != nil {
		return s.usePlatformTemplate(userID, orgID, templateID, plan, input, detail)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	catalog, version, err := s.loadCatalogAndVersion(templateID)
	if err != nil {
		return nil, err
	}
	currentScope := s.resolveTemplateScope(orgID, plan)
	if !hasTemplateScopeAccess(currentScope, requiredTemplateScope("", catalog.PlanRequired)) {
		return nil, fmt.Errorf("template requires %s scope", requiredTemplateScope("", catalog.PlanRequired))
	}
	platforms := decodeStringSlice(catalog.PlatformsJSON)
	if !containsFold(platforms, input.TargetPlatform) {
		return nil, fmt.Errorf("template does not support target platform %s", input.TargetPlatform)
	}
	exportSpec := resolveTemplateExportSpec(platforms, decodeMapAny(version.ExportSpecsJSON), input.TargetPlatform)
	language := defaultString(input.Language, "en")
	inputSchema := decodeMapAny(version.InputSchemaJSON)
	contractSummary := TemplateCatalogSummary{}
	applyTemplateContract(&contractSummary, inputSchema)
	profile := decodeExecutionProfile(version.ExecutionProfileJSON)
	profile = enrichCatalogVersionPromptProfile(profile, catalog, version, language)
	inputMode := templateStrategyValue(contractSummary.StrategyPolicy, "input_mode", "image_to_image")
	generationStrategy := templateStrategyValue(contractSummary.StrategyPolicy, "generation_strategy", inputMode)
	provider := templateStrategyValue(contractSummary.StrategyPolicy, "provider", profile.Provider)
	jobInput := studio.CreateGenerationJobInput{
		Mode:               "single",
		InputMode:          inputMode,
		GenerationStrategy: generationStrategy,
		Provider:           provider,
		Prompt:             "",
		RequestedVariants:  1,
		Params: map[string]any{
			"target_platform":  input.TargetPlatform,
			"language":         language,
			"custom_copy":      input.CustomCopy,
			"upload_image_url": input.UploadImageURL,
		},
		Metadata: map[string]any{
			"template_catalog_id": catalog.ID,
			"template_version_id": version.ID,
			"cuisine":             catalog.Cuisine,
			"dish_type":           catalog.DishType,
			"creative_source": map[string]any{
				"source_type":         "template",
				"source_id":           catalog.ID,
				"title":               catalog.Name,
				"plan_required":       catalog.PlanRequired,
				"credits_cost":        catalog.CreditsCost,
				"target_platform":     input.TargetPlatform,
				"template_id":         catalog.ID,
				"template_version_id": version.ID,
			},
			"execution_profile": map[string]any{
				"provider":                 profile.Provider,
				"model":                    profile.Model,
				"style_prompt":             profile.StylePrompt,
				"negative_prompt_template": profile.NegativePromptTemplate,
				"parameter_profile":        profile.ParameterProfile,
				"variables":                profile.Variables,
			},
		},
	}
	if err := s.recordUsageEvent(templateID, version.ID, userID, orgID, "use", "prepared", "", "", map[string]any{
		"target_platform": input.TargetPlatform,
		"language":        language,
	}); err != nil {
		return nil, err
	}
	return &UseTemplateResult{
		TemplateID:        catalog.ID,
		TemplateVersionID: version.ID,
		TargetRoute:       "/api/v1/menu/studio/jobs",
		TargetMethod:      "POST",
		CreditsCost:       catalog.CreditsCost,
		PlanRequired:      catalog.PlanRequired,
		PrefilledJob:      jobInput,
		InputSlots:        contractSummary.InputSlots,
		TargetOutputs:     contractSummary.TargetOutputs,
		ResolvedStrategy:  contractSummary.StrategyPolicy,
		TemplateContext: map[string]any{
			"name":              catalog.Name,
			"platforms":         platforms,
			"moods":             decodeStringSlice(catalog.MoodsJSON),
			"export_spec":       exportSpec,
			"input_slots":       contractSummary.InputSlots,
			"target_outputs":    contractSummary.TargetOutputs,
			"resolved_strategy": contractSummary.StrategyPolicy,
		},
	}, nil
}

func (s *Service) CopyToMyTemplates(userID, orgID, templateID string, input CopyTemplateInput) (*CopiedTemplateResult, error) {
	if detail, err := s.platformCatalogDetail(userID, orgID, templateID, ""); err == nil && detail != nil {
		return s.copyPlatformTemplate(userID, orgID, templateID, input, detail)
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	catalog, version, err := s.loadCatalogAndVersion(templateID)
	if err != nil {
		return nil, err
	}
	item := &models.StylePreset{
		OrganizationID:   orgID,
		CreatedByUserID:  userID,
		SourceType:       "template_catalog",
		SourceCatalogID:  catalog.ID,
		SourceVersionID:  version.ID,
		Name:             firstNonEmpty(strings.TrimSpace(input.Name), catalog.Name+" Copy"),
		Description:      catalog.Description,
		Visibility:       firstNonEmpty(input.Visibility, "private"),
		Status:           "active",
		Version:          1,
		PreviewAssetID:   catalog.CoverAssetID,
		DimensionsJSON:   mustEncodeJSON(buildStyleDimensions(decodeStringSlice(catalog.PlatformsJSON))),
		TagsJSON:         mustEncodeJSON(decodeStringSlice(catalog.TagsJSON)),
		ExecutionProfile: version.ExecutionProfileJSON,
		Metadata:         mustEncodeJSON(map[string]any{"template_catalog_id": catalog.ID, "template_version_id": version.ID}),
	}
	if err := s.studioRepo.CreateStylePreset(item); err != nil {
		return nil, err
	}
	if err := s.recordUsageEvent(templateID, version.ID, userID, orgID, "copy", "recorded", item.ID, "", map[string]any{"visibility": item.Visibility}); err != nil {
		return nil, err
	}
	return &CopiedTemplateResult{
		StyleID:         item.ID,
		Name:            item.Name,
		Visibility:      item.Visibility,
		SourceCatalogID: catalog.ID,
		SourceVersionID: version.ID,
	}, nil
}

func (s *Service) loadCatalogAndVersion(templateID string) (*models.TemplateCatalog, *models.TemplateCatalogVersion, error) {
	catalog, err := s.repo.FindCatalogByID(templateID)
	if err != nil {
		return nil, nil, err
	}
	versionID := catalog.CurrentVersionID
	if versionID == "" {
		versions, err := s.repo.ListCatalogVersions(catalog.ID)
		if err != nil {
			return nil, nil, err
		}
		if len(versions) == 0 {
			return nil, nil, gorm.ErrRecordNotFound
		}
		versionID = versions[0].ID
	}
	version, err := s.repo.FindCatalogVersionByID(versionID)
	if err != nil {
		return nil, nil, err
	}
	return catalog, version, nil
}

func (s *Service) favoriteMap(userID, orgID string) (map[string]bool, error) {
	items, err := s.repo.ListFavorites(userID, orgID)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, item := range items {
		out[item.TemplateCatalogID] = true
	}
	return out, nil
}

func (s *Service) recordUsageEvent(templateID, versionID, userID, orgID, eventType, status, stylePresetID, jobID string, payload map[string]any) error {
	return s.repo.CreateUsageEvent(&models.TemplateUsageEvent{
		TemplateCatalogID: templateID,
		TemplateVersionID: versionID,
		UserID:            userID,
		OrganizationID:    orgID,
		EventType:         eventType,
		Status:            status,
		StylePresetID:     stylePresetID,
		JobID:             jobID,
		PayloadJSON:       mustEncodeJSON(payload),
	})
}
