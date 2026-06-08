package templatecenter

import (
	"encoding/json"
	"strings"

	"menu-service/internal/models"
	studio "menu-service/internal/modules/studio"
)

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
	// Access should be driven by the business plan attached to the template.
	// Platform projection `scope=official` describes the catalog origin, not a
	// blanket requirement for the paid `official_templates` capability.
	switch strings.TrimSpace(planRequired) {
	case "pro":
		return "official_templates"
	case "growth", "max":
		return "all_templates"
	case "basic", "starter", "free":
		return "free_templates"
	}

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
	return "free_templates"
}

func templateScopeRank(scope string) int {
	switch strings.TrimSpace(scope) {
	case "all_templates":
		return 3
	case "official_templates":
		return 2
	case "free_templates":
		return 1
	default:
		return 0
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
	if strings.TrimSpace(orgID) == "" {
		return ""
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
