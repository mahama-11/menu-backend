package templatecenter

import (
	"errors"
	"fmt"
	"strings"

	"menu-service/internal/models"
	studio "menu-service/internal/modules/studio"
	"menu-service/internal/platform"

	"gorm.io/gorm"
)

func mapPlatformCatalogSummary(item platform.PlatformTemplateCatalogItem, plan string, currentScope string) TemplateCatalogSummary {
	raw := item.Raw
	cuisine, _ := raw["cuisine"].(string)
	dishType, _ := raw["dish_type"].(string)
	planRequired, _ := raw["plan_required"].(string)
	moods := decodeAnyStringSlice(raw["moods"])
	creditsCost := int64(numberValue(raw["credits_cost"]))
	summary := TemplateCatalogSummary{
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
	applyTemplateContract(&summary, templateContractMapFromRaw(raw))
	return summary
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
	inputSchema := templateContractMapFromRaw(raw)
	result := &TemplateCatalogDetail{
		TemplateCatalogSummary: mapPlatformCatalogSummary(detail.Item, plan, currentScope),
		CurrentVersionID:       currentVersionID,
		PromptTemplates:        promptTemplates,
		CopyTemplates:          mapAnyValue(raw, "copy_templates"),
		Hashtags:               mapStringSliceValue(raw, "hashtags"),
		DesignSpec:             designSpec,
		ExportSpecs:            mapAnyValue(raw, "export_specs"),
		InputSchema:            inputSchema,
		ExecutionProfile:       decodeExecutionProfileFromAny(raw["execution_profile"]),
		Examples:               decodePlatformExamples(currentVersionID, raw["examples"]),
		Metadata:               mapAnyValue(raw, "metadata"),
	}
	applyTemplateContract(&result.TemplateCatalogSummary, inputSchema)
	result.ExecutionProfile = enrichTemplateDetailPromptProfile(result, "")
	return result
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
	inputMode := templateStrategyValue(detail.StrategyPolicy, "input_mode", "image_to_image")
	generationStrategy := templateStrategyValue(detail.StrategyPolicy, "generation_strategy", inputMode)
	provider := templateStrategyValue(detail.StrategyPolicy, "provider", profile.Provider)
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
		InputSlots:        detail.InputSlots,
		TargetOutputs:     detail.TargetOutputs,
		ResolvedStrategy:  detail.StrategyPolicy,
		TemplateContext: map[string]any{
			"name":              detail.Name,
			"platforms":         detail.Platforms,
			"moods":             detail.Moods,
			"export_spec":       exportSpec,
			"input_slots":       detail.InputSlots,
			"target_outputs":    detail.TargetOutputs,
			"resolved_strategy": detail.StrategyPolicy,
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
