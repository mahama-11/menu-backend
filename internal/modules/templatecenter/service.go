package templatecenter

import (
	"encoding/json"
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
}

type TemplateCatalogSummary struct {
	TemplateID     string   `json:"template_id"`
	Slug           string   `json:"slug"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Cuisine        string   `json:"cuisine"`
	DishType       string   `json:"dish_type"`
	Platforms      []string `json:"platforms"`
	Moods          []string `json:"moods"`
	Tags           []string `json:"tags"`
	PlanRequired   string   `json:"plan_required"`
	CreditsCost    int64    `json:"credits_cost"`
	Locked         bool     `json:"locked"`
	IsFavorite     bool     `json:"is_favorite"`
	CoverAssetID   string   `json:"cover_asset_id,omitempty"`
	RecommendScore int      `json:"recommend_score"`
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
	if s.platform != nil {
		return nil
	}
	for index, seed := range defaultTemplateSeeds() {
		catalog, err := s.repo.FindCatalogByID(seed.ID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			catalog = seed.catalogModel()
			catalog.SortOrder = index + 1
			if err := s.repo.CreateCatalog(catalog); err != nil {
				return err
			}
		}
		versionID := seed.ID + "-v1"
		if _, err := s.repo.FindCatalogVersionByID(versionID); err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			if err := s.repo.CreateCatalogVersion(seed.versionModel()); err != nil {
				return err
			}
		}
		if err := s.repo.ReplaceCatalogExamples(versionID, seed.exampleModels(versionID)); err != nil {
			return err
		}
		if catalog.CurrentVersionID != versionID || catalog.SortOrder != index+1 {
			catalog.CurrentVersionID = versionID
			catalog.SortOrder = index + 1
			if err := s.repo.SaveCatalog(catalog); err != nil {
				return err
			}
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
	if s.platform != nil {
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
		out = append(out, mapCatalogSummary(&item, favoriteMap[item.ID], input.Plan, currentScope))
	}
	return out, nil
}

func (s *Service) GetCatalogDetail(userID, orgID, templateID, plan string) (*TemplateCatalogDetail, error) {
	if detail, err := s.platformCatalogDetail(userID, orgID, templateID, plan); err == nil && detail != nil {
		return detail, nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
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
	detail := &TemplateCatalogDetail{
		TemplateCatalogSummary: mapCatalogSummary(catalog, favorite != nil, plan, s.resolveTemplateScope(orgID, plan)),
		CurrentVersionID:       version.ID,
		PromptTemplates:        decodeMapString(version.PromptTemplatesJSON),
		CopyTemplates:          decodeMapAny(version.CopyTemplatesJSON),
		Hashtags:               decodeMapStringSlice(version.HashtagsJSON),
		DesignSpec:             decodeMapAny(version.DesignSpecJSON),
		ExportSpecs:            decodeMapAny(version.ExportSpecsJSON),
		InputSchema:            decodeMapAny(version.InputSchemaJSON),
		ExecutionProfile:       decodeExecutionProfile(version.ExecutionProfileJSON),
		Examples:               examples,
		Metadata:               decodeMapAny(version.MetadataJSON),
	}
	detail.ExecutionProfile = enrichTemplateDetailPromptProfile(detail, "")
	return detail, nil
}

func mapPlatformCatalogSummary(item platform.PlatformTemplateCatalogItem, plan string, currentScope string) TemplateCatalogSummary {
	raw := item.Raw
	cuisine, _ := raw["cuisine"].(string)
	dishType, _ := raw["dish_type"].(string)
	planRequired, _ := raw["plan_required"].(string)
	moods := decodeAnyStringSlice(raw["moods"])
	creditsCost := int64(numberValue(raw["credits_cost"]))
	return TemplateCatalogSummary{
		TemplateID:     item.TemplateID,
		Slug:           item.Slug,
		Name:           item.Name,
		Description:    item.Summary,
		Cuisine:        cuisine,
		DishType:       dishType,
		Platforms:      item.Platforms,
		Moods:          moods,
		Tags:           item.Tags,
		PlanRequired:   planRequired,
		CreditsCost:    creditsCost,
		Locked:         !hasTemplateScopeAccess(currentScope, requiredTemplateScope(item.Scope, planRequired)),
		IsFavorite:     false,
		CoverAssetID:   item.CoverAssetID,
		RecommendScore: item.RecommendScore,
	}
}

func mapPlatformCatalogDetail(detail *platform.PlatformTemplateCatalogDetail, plan string, currentScope string) *TemplateCatalogDetail {
	raw := detail.DetailRaw
	promptTemplates := map[string]string{}
	if prompts, ok := raw["prompt_templates"].(map[string]any); ok {
		for key, value := range prompts {
			if str, ok := value.(string); ok {
				promptTemplates[key] = str
			}
		}
	}
	currentVersionID := stringMapValue(raw, "current_version_id")
	if currentVersionID == "" {
		currentVersionID = detail.Item.TemplateID + "-platform"
	}
	designSpec := mapAnyValue(raw, "design_spec")
	if layout := stringMapValue(raw, "layout"); layout != "" {
		designSpec["layout"] = layout
	}
	if lighting := stringMapValue(raw, "lighting"); lighting != "" {
		designSpec["lighting"] = lighting
	}
	if props := decodeAnyStringSlice(raw["props"]); len(props) > 0 {
		designSpec["props"] = props
	}
	if moods := decodeAnyStringSlice(raw["moods"]); len(moods) > 0 {
		designSpec["moods"] = moods
	}
	result := &TemplateCatalogDetail{
		TemplateCatalogSummary: mapPlatformCatalogSummary(detail.Item, plan, currentScope),
		CurrentVersionID:       currentVersionID,
		PromptTemplates:        promptTemplates,
		CopyTemplates:          mapAnyValue(raw, "copy_templates"),
		Hashtags:               mapStringSliceValue(raw, "hashtags"),
		DesignSpec:             designSpec,
		ExportSpecs:            mapAnyValue(raw, "export_specs"),
		InputSchema:            mapAnyValue(raw, "input_schema"),
		ExecutionProfile:       decodeExecutionProfileFromAny(raw["execution_profile"]),
		Examples:               decodePlatformExamples(currentVersionID, raw["examples"]),
		Metadata:               mapAnyValue(raw, "metadata"),
	}
	result.ExecutionProfile = enrichTemplateDetailPromptProfile(result, "")
	return result
}

func decodeAnyStringSlice(value any) []string {
	items, ok := value.([]any)
	if !ok {
		if stringsValue, ok := value.([]string); ok {
			return stringsValue
		}
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		if str, ok := item.(string); ok {
			result = append(result, str)
		}
	}
	return result
}

func stringMapValue(input map[string]any, key string) string {
	value, _ := input[key].(string)
	return value
}

func mapAnyValue(input map[string]any, key string) map[string]any {
	value, _ := input[key].(map[string]any)
	if value == nil {
		return map[string]any{}
	}
	return value
}

func mapStringSliceValue(input map[string]any, key string) map[string][]string {
	value, _ := input[key].(map[string]any)
	if len(value) == 0 {
		return nil
	}
	result := map[string][]string{}
	for k, item := range value {
		result[k] = decodeAnyStringSlice(item)
	}
	return result
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
	profile := decodeExecutionProfile(version.ExecutionProfileJSON)
	profile = enrichCatalogVersionPromptProfile(profile, catalog, version, language)
	jobInput := studio.CreateGenerationJobInput{
		Mode:              "single",
		Provider:          profile.Provider,
		Prompt:            "",
		RequestedVariants: 1,
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
		TemplateContext: map[string]any{
			"name":        catalog.Name,
			"platforms":   platforms,
			"moods":       decodeStringSlice(catalog.MoodsJSON),
			"export_spec": exportSpec,
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

func (s *Service) platformCatalogDetail(userID, orgID, templateID, plan string) (*TemplateCatalogDetail, error) {
	if s.platform == nil {
		return nil, gorm.ErrRecordNotFound
	}
	result, err := s.platform.InternalTemplateCatalogDetail("menu:" + templateID)
	if err != nil || result == nil {
		if err != nil {
			return nil, err
		}
		return nil, gorm.ErrRecordNotFound
	}
	detail := mapPlatformCatalogDetail(result, plan, s.resolveTemplateScope(orgID, plan))
	favorite, favoriteErr := s.repo.FindFavorite(templateID, userID, orgID)
	if favoriteErr != nil && !errors.Is(favoriteErr, gorm.ErrRecordNotFound) {
		return nil, favoriteErr
	}
	detail.IsFavorite = favorite != nil
	return detail, nil
}

func (s *Service) usePlatformTemplate(userID, orgID, templateID, plan string, input UseTemplateInput, detail *TemplateCatalogDetail) (*UseTemplateResult, error) {
	if detail.Locked {
		return nil, fmt.Errorf("template requires %s scope", requiredTemplateScope("", detail.PlanRequired))
	}
	if !containsFold(detail.Platforms, input.TargetPlatform) {
		return nil, fmt.Errorf("template does not support target platform %s", input.TargetPlatform)
	}
	exportSpec := resolveTemplateExportSpec(detail.Platforms, detail.ExportSpecs, input.TargetPlatform)
	language := defaultString(input.Language, "en")
	profile := detail.ExecutionProfile
	profile = enrichTemplateDetailPromptProfile(detail, language)
	jobInput := studio.CreateGenerationJobInput{
		Mode:              "single",
		Provider:          profile.Provider,
		Prompt:            "",
		RequestedVariants: 1,
		Params: map[string]any{
			"target_platform":  input.TargetPlatform,
			"language":         language,
			"custom_copy":      input.CustomCopy,
			"upload_image_url": input.UploadImageURL,
		},
		Metadata: map[string]any{
			"template_catalog_id": detail.TemplateID,
			"template_version_id": detail.CurrentVersionID,
			"cuisine":             detail.Cuisine,
			"dish_type":           detail.DishType,
			"creative_source": map[string]any{
				"source_type":         "template",
				"source_id":           detail.TemplateID,
				"title":               detail.Name,
				"plan_required":       detail.PlanRequired,
				"credits_cost":        detail.CreditsCost,
				"target_platform":     input.TargetPlatform,
				"template_id":         detail.TemplateID,
				"template_version_id": detail.CurrentVersionID,
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
	if err := s.recordUsageEvent(templateID, detail.CurrentVersionID, userID, orgID, "use", "prepared", "", "", map[string]any{
		"target_platform": input.TargetPlatform,
		"language":        language,
		"managed_source":  "platform_projection",
	}); err != nil {
		return nil, err
	}
	return &UseTemplateResult{
		TemplateID:        detail.TemplateID,
		TemplateVersionID: detail.CurrentVersionID,
		TargetRoute:       "/api/v1/menu/studio/jobs",
		TargetMethod:      "POST",
		CreditsCost:       detail.CreditsCost,
		PlanRequired:      detail.PlanRequired,
		PrefilledJob:      jobInput,
		TemplateContext: map[string]any{
			"name":        detail.Name,
			"platforms":   detail.Platforms,
			"moods":       detail.Moods,
			"export_spec": exportSpec,
		},
	}, nil
}

func (s *Service) copyPlatformTemplate(userID, orgID, templateID string, input CopyTemplateInput, detail *TemplateCatalogDetail) (*CopiedTemplateResult, error) {
	item := &models.StylePreset{
		OrganizationID:   orgID,
		CreatedByUserID:  userID,
		SourceType:       "template_catalog",
		SourceCatalogID:  detail.TemplateID,
		SourceVersionID:  detail.CurrentVersionID,
		Name:             firstNonEmpty(strings.TrimSpace(input.Name), detail.Name+" Copy"),
		Description:      detail.Description,
		Visibility:       firstNonEmpty(input.Visibility, "private"),
		Status:           "active",
		Version:          1,
		PreviewAssetID:   detail.CoverAssetID,
		DimensionsJSON:   mustEncodeJSON(buildStyleDimensions(detail.Platforms)),
		TagsJSON:         mustEncodeJSON(detail.Tags),
		ExecutionProfile: mustEncodeJSON(detail.ExecutionProfile),
		Metadata:         mustEncodeJSON(map[string]any{"template_catalog_id": detail.TemplateID, "template_version_id": detail.CurrentVersionID, "managed_source": "platform_projection"}),
	}
	if err := s.studioRepo.CreateStylePreset(item); err != nil {
		return nil, err
	}
	if err := s.recordUsageEvent(templateID, detail.CurrentVersionID, userID, orgID, "copy", "recorded", item.ID, "", map[string]any{
		"visibility":     item.Visibility,
		"managed_source": "platform_projection",
	}); err != nil {
		return nil, err
	}
	return &CopiedTemplateResult{
		StyleID:         item.ID,
		Name:            item.Name,
		Visibility:      item.Visibility,
		SourceCatalogID: detail.TemplateID,
		SourceVersionID: detail.CurrentVersionID,
	}, nil
}

func matchesCatalogFilters(item TemplateCatalogSummary, input ListCatalogInput) bool {
	if input.Cuisine != "" && !strings.EqualFold(item.Cuisine, input.Cuisine) {
		return false
	}
	if input.DishType != "" && !strings.EqualFold(item.DishType, input.DishType) {
		return false
	}
	if input.Platform != "" && !containsFold(item.Platforms, input.Platform) {
		return false
	}
	if input.Mood != "" && !containsFold(item.Moods, input.Mood) {
		return false
	}
	query := strings.TrimSpace(strings.ToLower(input.Query))
	if query == "" {
		return true
	}
	return strings.Contains(strings.ToLower(item.Name), query) ||
		strings.Contains(strings.ToLower(item.Description), query) ||
		strings.Contains(strings.ToLower(item.TemplateID), query)
}

func containsFold(items []string, target string) bool {
	for _, item := range items {
		if strings.EqualFold(item, target) {
			return true
		}
	}
	return false
}

func finalizeTemplatePromptProfile(profile studio.StyleExecutionProfile) studio.StyleExecutionProfile {
	profile.SystemPrompt = strings.TrimSpace(profile.SystemPrompt)
	profile.StylePrompt = strings.TrimSpace(profile.StylePrompt)
	profile.UserPrompt = strings.TrimSpace(profile.UserPrompt)
	profile.PromptTemplate = strings.TrimSpace(profile.PromptTemplate)
	if profile.SystemPrompt == "" && profile.PromptTemplate != "" {
		profile.SystemPrompt = profile.PromptTemplate
	}
	profile.PromptTemplate = composeTemplatePromptParts(profile.SystemPrompt, profile.StylePrompt, profile.UserPrompt)
	return profile
}

func enrichCatalogVersionPromptProfile(profile studio.StyleExecutionProfile, catalog *models.TemplateCatalog, version *models.TemplateCatalogVersion, language string) studio.StyleExecutionProfile {
	prompts := decodeMapString(version.PromptTemplatesJSON)
	profile.SystemPrompt = defaultString(prompts[language], firstNonEmpty(profile.SystemPrompt, profile.PromptTemplate))
	if profile.StylePrompt == "" && profile.PromptTemplate != "" && profile.PromptTemplate != profile.SystemPrompt {
		profile.StylePrompt = profile.PromptTemplate
	}
	if profile.SystemPrompt == "" && profile.StylePrompt == "" {
		profile.StylePrompt = buildTemplateFallbackStylePrompt(
			catalog.Name,
			catalog.Description,
			catalog.Cuisine,
			catalog.DishType,
			decodeStringSlice(catalog.PlatformsJSON),
			decodeStringSlice(catalog.MoodsJSON),
			decodeStringSlice(catalog.TagsJSON),
			decodeMapAny(version.DesignSpecJSON),
			decodeMapAny(version.MetadataJSON),
		)
	}
	return finalizeTemplatePromptProfile(profile)
}

func enrichTemplateDetailPromptProfile(detail *TemplateCatalogDetail, language string) studio.StyleExecutionProfile {
	profile := detail.ExecutionProfile
	profile.SystemPrompt = defaultString(detail.PromptTemplates[language], firstNonEmpty(profile.SystemPrompt, profile.PromptTemplate))
	if profile.StylePrompt == "" && profile.PromptTemplate != "" && profile.PromptTemplate != profile.SystemPrompt {
		profile.StylePrompt = profile.PromptTemplate
	}
	if profile.SystemPrompt == "" && profile.StylePrompt == "" {
		profile.StylePrompt = buildTemplateFallbackStylePrompt(
			detail.Name,
			detail.Description,
			detail.Cuisine,
			detail.DishType,
			detail.Platforms,
			detail.Moods,
			detail.Tags,
			detail.DesignSpec,
			detail.Metadata,
		)
	}
	return finalizeTemplatePromptProfile(profile)
}

func buildTemplateFallbackStylePrompt(name, description, cuisine, dishType string, platforms, moods, tags []string, designSpec, metadata map[string]any) string {
	parts := make([]string, 0, 10)
	appendLabeled := func(label, value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		parts = append(parts, label+": "+trimmed)
	}
	appendList := func(label string, values []string) {
		filtered := make([]string, 0, len(values))
		for _, item := range values {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				filtered = append(filtered, trimmed)
			}
		}
		if len(filtered) == 0 {
			return
		}
		parts = append(parts, label+": "+strings.Join(filtered, ", "))
	}
	appendLabeled("Template", name)
	appendLabeled("Description", description)
	appendLabeled("Cuisine", cuisine)
	appendLabeled("Dish Type", dishType)
	appendList("Platforms", platforms)
	appendList("Moods", moods)
	appendList("Tags", tags)
	appendLabeled("Layout", firstNonEmpty(stringMapValue(designSpec, "layout"), stringMapValue(metadata, "layout"), stringMapValue(metadata, "cover_layout")))
	appendLabeled("Lighting", firstNonEmpty(stringMapValue(designSpec, "lighting"), stringMapValue(metadata, "lighting")))
	appendList("Props", firstNonEmptyStringSlice(
		decodeAnyStringSlice(designSpec["props"]),
		decodeAnyStringSlice(metadata["props"]),
	))
	return strings.Join(parts, "\n")
}

func firstNonEmptyStringSlice(values ...[]string) []string {
	for _, items := range values {
		if len(items) > 0 {
			return items
		}
	}
	return nil
}

func composeTemplatePromptParts(parts ...string) string {
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return strings.Join(out, "\n\n")
}

func resolveTemplateExportSpec(platforms []string, exportSpecs map[string]any, targetPlatform string) map[string]any {
	if spec, ok := exportSpecs[targetPlatform].(map[string]any); ok && len(spec) > 0 {
		return spec
	}
	if !containsFold(platforms, targetPlatform) {
		return nil
	}
	if option, ok := templatePlatforms[targetPlatform]; ok {
		return map[string]any{
			"id":     option.ID,
			"label":  option.Label,
			"width":  option.Width,
			"height": option.Height,
			"ratio":  option.Ratio,
			"format": option.Format,
		}
	}
	return map[string]any{"platform": targetPlatform}
}

func decodePlatformExamples(versionID string, value any) []models.TemplateCatalogExample {
	rawExamples, ok := value.([]any)
	if !ok {
		return nil
	}
	items := make([]models.TemplateCatalogExample, 0, len(rawExamples))
	for idx, rawExample := range rawExamples {
		example, ok := rawExample.(map[string]any)
		if !ok {
			continue
		}
		items = append(items, models.TemplateCatalogExample{
			ID:                firstNonEmpty(stringMapValue(example, "id"), fmt.Sprintf("%s-example-%d", versionID, idx+1)),
			TemplateVersionID: versionID,
			ExampleType:       defaultString(stringMapValue(example, "exampleType"), "preview"),
			Title:             stringMapValue(example, "title"),
			Description:       stringMapValue(example, "description"),
			SourceRef:         stringMapValue(example, "sourceRef"),
			StorageKey:        stringMapValue(example, "storageKey"),
			AssetID:           stringMapValue(example, "assetId"),
			PreviewURL:        firstNonEmpty(stringMapValue(example, "preview_url"), stringMapValue(example, "previewAssetUrl")),
			InputAssetURL:     stringMapValue(example, "input_asset_url"),
			OutputAssetURL:    stringMapValue(example, "output_asset_url"),
			MetadataJSON:      mustEncodeJSON(mapAnyValue(example, "metadata")),
			SortOrder:         idx + 1,
		})
	}
	return items
}

func decodeExecutionProfileFromAny(value any) studio.StyleExecutionProfile {
	switch typed := value.(type) {
	case map[string]any:
		return decodeExecutionProfile(mustEncodeJSON(typed))
	case string:
		return decodeExecutionProfile(typed)
	default:
		return studio.StyleExecutionProfile{}
	}
}

func numberValue(value any) int {
	switch typed := value.(type) {
	case float64:
		return int(typed)
	case float32:
		return int(typed)
	case int:
		return typed
	case int64:
		return int(typed)
	default:
		return 0
	}
}

func mapCatalogSummary(item *models.TemplateCatalog, isFavorite bool, plan string, currentScope string) TemplateCatalogSummary {
	return TemplateCatalogSummary{
		TemplateID:     item.ID,
		Slug:           item.Slug,
		Name:           item.Name,
		Description:    item.Description,
		Cuisine:        item.Cuisine,
		DishType:       item.DishType,
		Platforms:      decodeStringSlice(item.PlatformsJSON),
		Moods:          decodeStringSlice(item.MoodsJSON),
		Tags:           decodeStringSlice(item.TagsJSON),
		PlanRequired:   item.PlanRequired,
		CreditsCost:    item.CreditsCost,
		Locked:         !hasTemplateScopeAccess(currentScope, requiredTemplateScope("", item.PlanRequired)),
		IsFavorite:     isFavorite,
		CoverAssetID:   item.CoverAssetID,
		RecommendScore: item.RecommendScore,
	}
}

func planRank(plan string) int {
	switch plan {
	case "growth":
		return 3
	case "pro":
		return 2
	default:
		return 1
	}
}

func requiredTemplateScope(scope, planRequired string) string {
	normalizedScope := strings.TrimSpace(scope)
	if normalizedScope != "" {
		switch normalizedScope {
		case "public":
			return "free_templates"
		case "official":
			return "official_templates"
		case "all", "all_templates":
			return "all_templates"
		}
	}
	switch strings.TrimSpace(planRequired) {
	case "pro":
		return "official_templates"
	case "growth", "max":
		return "all_templates"
	default:
		return "free_templates"
	}
}

func templateScopeRank(scope string) int {
	switch strings.TrimSpace(scope) {
	case "all_templates":
		return 3
	case "official_templates":
		return 2
	default:
		return 1
	}
}

func hasTemplateScopeAccess(currentScope, requiredScope string) bool {
	return templateScopeRank(currentScope) >= templateScopeRank(requiredScope)
}

func planTemplateScope(plan string) string {
	switch plan {
	case "pro":
		return "official_templates"
	case "growth", "max":
		return "all_templates"
	default:
		return "free_templates"
	}
}

func (s *Service) resolveTemplateScope(orgID, plan string) string {
	if s.platform != nil && orgID != "" {
		result, err := s.platform.ResolveCapability("menu", "organization", orgID, "template_scope")
		if err == nil && result != nil && strings.TrimSpace(result.GrantValue) != "" {
			return result.GrantValue
		}
	}
	return planTemplateScope(plan)
}

func buildStyleDimensions(platforms []string) []map[string]string {
	out := make([]map[string]string, 0, len(platforms))
	for _, platformID := range platforms {
		out = append(out, map[string]string{
			"type":  "platform",
			"key":   platformID,
			"label": strings.ReplaceAll(platformID, "_", " "),
		})
	}
	return out
}

func decodeStringSlice(raw string) []string {
	var items []string
	_ = json.Unmarshal([]byte(raw), &items)
	return items
}

func decodeMapString(raw string) map[string]string {
	var items map[string]string
	_ = json.Unmarshal([]byte(raw), &items)
	if items == nil {
		return map[string]string{}
	}
	return items
}

func decodeMapStringSlice(raw string) map[string][]string {
	var items map[string][]string
	_ = json.Unmarshal([]byte(raw), &items)
	if items == nil {
		return map[string][]string{}
	}
	return items
}

func decodeMapAny(raw string) map[string]any {
	var items map[string]any
	_ = json.Unmarshal([]byte(raw), &items)
	if items == nil {
		return map[string]any{}
	}
	return items
}

func decodeExecutionProfile(raw string) studio.StyleExecutionProfile {
	var item studio.StyleExecutionProfile
	_ = json.Unmarshal([]byte(raw), &item)
	return item
}

func mustEncodeJSON(value any) string {
	payload, _ := json.Marshal(value)
	return string(payload)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
