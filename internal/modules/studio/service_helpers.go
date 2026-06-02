package studio

import (
	"encoding/json"
	"errors"
	"maps"
	"strings"

	"menu-service/internal/models"
	"menu-service/internal/platform"

	"gorm.io/gorm"
)

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

func decodeMapString(raw string) map[string]string {
	if raw == "" {
		return map[string]string{}
	}
	out := map[string]string{}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func decodeCreativeSource(raw string) *CreativeSourceSnapshot {
	metadata := decodeMap(raw)
	source, ok := metadata["creative_source"].(map[string]any)
	if !ok || len(source) == 0 {
		return nil
	}
	out := &CreativeSourceSnapshot{
		SourceType:        stringMapValue(source, "source_type"),
		SourceID:          firstNonEmpty(stringMapValue(source, "source_id"), stringMapValue(source, "template_id"), stringMapValue(source, "style_preset_id")),
		Title:             stringMapValue(source, "title"),
		PlanRequired:      stringMapValue(source, "plan_required"),
		CreditsCost:       int64MapValue(source, "credits_cost"),
		TargetPlatform:    stringMapValue(source, "target_platform"),
		TemplateID:        stringMapValue(source, "template_id"),
		TemplateVersionID: stringMapValue(source, "template_version_id"),
		StylePresetID:     stringMapValue(source, "style_preset_id"),
	}
	if out.SourceType == "" && out.SourceID == "" {
		return nil
	}
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

func normalizeCreativeSourceMetadata(input map[string]any) map[string]any {
	metadata := mergeMaps(nil, input)
	source, ok := metadata["creative_source"].(map[string]any)
	if !ok || len(source) == 0 {
		return metadata
	}
	if templateID := stringMapValue(source, "template_id"); templateID != "" {
		metadata["template_catalog_id"] = templateID
	}
	if templateVersionID := stringMapValue(source, "template_version_id"); templateVersionID != "" {
		metadata["template_version_id"] = templateVersionID
	}
	if stylePresetID := stringMapValue(source, "style_preset_id"); stylePresetID != "" {
		metadata["style_preset_id"] = stylePresetID
	}
	if targetPlatform := stringMapValue(source, "target_platform"); targetPlatform != "" {
		metadata["target_platform"] = targetPlatform
	}
	return metadata
}

func (s *Service) resolvePromptSnapshot(orgID string, input CreateGenerationJobInput, metadata map[string]any) (StyleExecutionProfile, error) {
	profile := StyleExecutionProfile{}
	language := defaultString(stringMapValue(input.Params, "language"), "en")
	templateProfile, err := s.resolveTemplatePromptProfile(metadata, language)
	if err != nil {
		return StyleExecutionProfile{}, err
	}
	profile = mergeExecutionProfiles(profile, templateProfile)
	stylePresetID := firstNonEmpty(input.StylePresetID, stringMapValue(metadata, "style_preset_id"))
	if stylePresetID != "" {
		style, err := s.repo.FindStylePresetByID(orgID, stylePresetID)
		if err != nil {
			return StyleExecutionProfile{}, err
		}
		profile = mergeExecutionProfiles(profile, sanitizeFrontendExecutionProfile(decodeExecutionProfile(style.ExecutionProfile)))
	}
	profile = mergeExecutionProfiles(profile, buildPromptSnapshotFromInput(input, metadata))
	profile.Provider = firstNonEmpty(input.Provider, profile.Provider)
	return finalizePromptSnapshot(profile, strings.TrimSpace(input.Prompt)), nil
}

func (s *Service) resolveTemplatePromptProfile(metadata map[string]any, language string) (StyleExecutionProfile, error) {
	templateVersionID := stringMapValue(metadata, "template_version_id")
	templateID := firstNonEmpty(stringMapValue(metadata, "template_catalog_id"), stringMapValue(metadata, "template_id"))
	if templateVersionID == "" && templateID == "" {
		return StyleExecutionProfile{}, nil
	}
	if s.templateRepo != nil && templateVersionID != "" {
		version, err := s.templateRepo.FindCatalogVersionByID(templateVersionID)
		if err == nil {
			var catalog *models.TemplateCatalog
			if s.templateRepo != nil {
				catalog, _ = s.templateRepo.FindCatalogByID(version.TemplateCatalogID)
			}
			return profileFromTemplateVersion(catalog, version, language), nil
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return StyleExecutionProfile{}, err
		}
	}
	if s.platform != nil && templateID != "" {
		detail, err := s.platform.InternalTemplateCatalogDetail(normalizePlatformTemplateRef("menu", templateID))
		if err != nil {
			return StyleExecutionProfile{}, err
		}
		return profileFromPlatformTemplateDetail(detail, language), nil
	}
	return StyleExecutionProfile{}, nil
}

func buildPromptSnapshotFromInput(input CreateGenerationJobInput, metadata map[string]any) StyleExecutionProfile {
	profile := StyleExecutionProfile{
		Provider:   firstNonEmpty(input.Provider, stringMapValue(metadata, "provider")),
		UserPrompt: strings.TrimSpace(input.Prompt),
	}
	if executionProfile, ok := metadata["execution_profile"].(map[string]any); ok {
		profile.Provider = firstNonEmpty(profile.Provider, stringMapValue(executionProfile, "provider"))
		profile.Model = stringMapValue(executionProfile, "model")
		profile.StylePrompt = stringMapValue(executionProfile, "style_prompt")
		profile.UserPrompt = firstNonEmpty(profile.UserPrompt, stringMapValue(executionProfile, "user_prompt"))
		profile.NegativePromptTemplate = stringMapValue(executionProfile, "negative_prompt_template")
		if parameterProfile, ok := executionProfile["parameter_profile"].(map[string]any); ok {
			profile.ParameterProfile = mergeMaps(nil, parameterProfile)
		}
		if variables, ok := executionProfile["variables"].(map[string]any); ok {
			out := map[string]string{}
			for key, value := range variables {
				if str, ok := value.(string); ok {
					out[key] = str
				}
			}
			if len(out) > 0 {
				profile.Variables = out
			}
		}
	}
	profile = finalizePromptSnapshot(profile, strings.TrimSpace(input.Prompt))
	return profile
}

func sanitizeFrontendExecutionProfile(profile StyleExecutionProfile) StyleExecutionProfile {
	profile.SystemPrompt = ""
	profile.PromptTemplate = ""
	return profile
}

func mergeExecutionProfiles(base, overlay StyleExecutionProfile) StyleExecutionProfile {
	base.Provider = firstNonEmpty(base.Provider, overlay.Provider)
	base.Model = firstNonEmpty(base.Model, overlay.Model)
	base.SystemPrompt = firstNonEmpty(base.SystemPrompt, overlay.SystemPrompt)
	base.StylePrompt = firstNonEmpty(base.StylePrompt, overlay.StylePrompt)
	base.UserPrompt = firstNonEmpty(base.UserPrompt, overlay.UserPrompt)
	base.NegativePromptTemplate = firstNonEmpty(base.NegativePromptTemplate, overlay.NegativePromptTemplate)
	if len(base.ParameterProfile) == 0 && len(overlay.ParameterProfile) > 0 {
		base.ParameterProfile = mergeMaps(nil, overlay.ParameterProfile)
	}
	if len(base.Variables) == 0 && len(overlay.Variables) > 0 {
		base.Variables = maps.Clone(overlay.Variables)
	}
	return base
}

func profileFromTemplateVersion(catalog *models.TemplateCatalog, version *models.TemplateCatalogVersion, language string) StyleExecutionProfile {
	profile := decodeExecutionProfile(version.ExecutionProfileJSON)
	prompts := decodeMapString(version.PromptTemplatesJSON)
	profile.SystemPrompt = defaultString(prompts[language], profile.SystemPrompt)
	if profile.StylePrompt == "" && profile.PromptTemplate != "" && profile.PromptTemplate != profile.SystemPrompt {
		profile.StylePrompt = profile.PromptTemplate
	}
	if profile.SystemPrompt == "" && profile.StylePrompt == "" && catalog != nil {
		profile.StylePrompt = buildTemplateFallbackStylePrompt(
			catalog.Name,
			catalog.Description,
			catalog.Cuisine,
			catalog.DishType,
			decodeStringSlice(catalog.PlatformsJSON),
			decodeStringSlice(catalog.MoodsJSON),
			decodeStringSlice(catalog.TagsJSON),
			decodeMap(version.DesignSpecJSON),
			decodeMap(version.MetadataJSON),
		)
	}
	profile.PromptTemplate = ""
	return profile
}

func profileFromPlatformTemplateDetail(detail *platform.PlatformTemplateCatalogDetail, language string) StyleExecutionProfile {
	profile := decodeExecutionProfile(mustEncodeJSON(mapValue(detail.DetailRaw, "execution_profile")))
	prompts := mapStringFromAnyMap(mapValue(detail.DetailRaw, "prompt_templates"))
	profile.SystemPrompt = defaultString(prompts[language], profile.SystemPrompt)
	if profile.StylePrompt == "" && profile.PromptTemplate != "" && profile.PromptTemplate != profile.SystemPrompt {
		profile.StylePrompt = profile.PromptTemplate
	}
	if profile.SystemPrompt == "" && profile.StylePrompt == "" {
		profile.StylePrompt = buildTemplateFallbackStylePrompt(
			detail.Item.Name,
			detail.Item.Summary,
			stringMapValue(detail.Item.Raw, "cuisine"),
			stringMapValue(detail.Item.Raw, "dish_type"),
			detail.Item.Platforms,
			decodeAnyStringSlice(detail.Item.Raw["moods"]),
			detail.Item.Tags,
			mergeMaps(
				mapValue(detail.DetailRaw, "design_spec"),
				map[string]any{
					"layout":   stringMapValue(detail.DetailRaw, "layout"),
					"lighting": stringMapValue(detail.DetailRaw, "lighting"),
					"props":    decodeAnyStringSlice(detail.DetailRaw["props"]),
				},
			),
			mapValue(detail.DetailRaw, "metadata"),
		)
	}
	profile.PromptTemplate = ""
	return profile
}

func finalizePromptSnapshot(profile StyleExecutionProfile, rawInputPrompt string) StyleExecutionProfile {
	profile.SystemPrompt = strings.TrimSpace(profile.SystemPrompt)
	profile.StylePrompt = strings.TrimSpace(profile.StylePrompt)
	profile.UserPrompt = strings.TrimSpace(profile.UserPrompt)
	profile.PromptTemplate = strings.TrimSpace(profile.PromptTemplate)

	if profile.SystemPrompt == "" && profile.PromptTemplate != "" {
		profile.SystemPrompt = profile.PromptTemplate
	}
	if raw := strings.TrimSpace(rawInputPrompt); raw != "" {
		withoutUser := composePromptParts(profile.SystemPrompt, profile.StylePrompt)
		if raw != withoutUser && raw != profile.PromptTemplate {
			profile.UserPrompt = raw
		}
	}
	profile.PromptTemplate = composePromptParts(profile.SystemPrompt, profile.StylePrompt, profile.UserPrompt)
	return profile
}

func composePromptParts(parts ...string) string {
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

func mapValue(items map[string]any, key string) map[string]any {
	if items == nil {
		return map[string]any{}
	}
	if value, ok := items[key].(map[string]any); ok {
		return value
	}
	return map[string]any{}
}

func mapStringFromAnyMap(items map[string]any) map[string]string {
	out := map[string]string{}
	for key, value := range items {
		if text, ok := value.(string); ok {
			out[key] = text
		}
	}
	return out
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

func normalizePlatformTemplateRef(productCode, templateRef string) string {
	trimmed := strings.TrimSpace(templateRef)
	if trimmed == "" || strings.Contains(trimmed, ":") {
		return trimmed
	}
	return productCode + ":" + trimmed
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

func normalizeRuntimeProviderCode(provider string) string {
	trimmed := strings.TrimSpace(provider)
	switch strings.ToLower(trimmed) {
	case "", "default":
		return ""
	case "comfyui":
		return "comfyui_bridge"
	default:
		return trimmed
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
