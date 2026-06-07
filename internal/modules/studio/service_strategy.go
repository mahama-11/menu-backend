package studio

import (
	"errors"
	"fmt"
	"maps"
	"strings"
)

const (
	studioInputModeTextToImage  = "text_to_image"
	studioInputModeImageToImage = "image_to_image"
	studioInputModeMultiImage   = "multi_image"
	studioInputModeImageEdit    = "image_edit"
	studioInputModeAskInput     = "ask_for_required_input"

	studioGenerationStrategyTextToImage  = "text_to_image"
	studioGenerationStrategyImageToImage = "image_to_image"
	studioGenerationStrategyMultiImage   = "multi_image"

	studioSourceAssetLimit = 4
)

var multiImageCapableProviders = map[string]struct{}{
	"comfyui_bridge": {},
}

// StudioSourceAssetInput is the role-aware asset contract used by Template Center -> Studio handoff.
// AssetID is the canonical JSON field used by the frontend; ID is accepted as a compatibility alias.
type StudioSourceAssetInput struct {
	AssetID  string         `json:"asset_id,omitempty"`
	ID       string         `json:"id,omitempty"`
	Role     string         `json:"role,omitempty"`
	Label    string         `json:"label,omitempty"`
	Required bool           `json:"required,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type studioDomainError struct {
	Code       string
	Message    string
	Hint       string
	HTTPStatus int
}

func (e studioDomainError) Error() string { return e.Message }

func newStudioBadRequest(code, message, hint string) error {
	return studioDomainError{Code: code, Message: message, Hint: hint, HTTPStatus: 400}
}

func asStudioDomainError(err error) (studioDomainError, bool) {
	var out studioDomainError
	if errors.As(err, &out) {
		return out, true
	}
	return studioDomainError{}, false
}

func normalizeGenerationJobInput(input CreateGenerationJobInput) (CreateGenerationJobInput, error) {
	normalized := input
	normalized.SourceAssets = normalizeSourceAssetInputs(input.SourceAssets, input.SourceAssetIDs)
	if len(normalized.SourceAssets) > studioSourceAssetLimit {
		return CreateGenerationJobInput{}, newStudioBadRequest(
			"STUDIO_SOURCE_ASSETS_LIMIT_EXCEEDED",
			"Too many source images for this generation",
			"Use up to four required materials for one generation.",
		)
	}
	normalized.SourceAssetIDs = sourceAssetIDsFromInputs(normalized.SourceAssets)
	if len(normalized.SourceAssetIDs) == 0 {
		normalized.SourceAssetIDs = dedupeTrimmedStrings(input.SourceAssetIDs)
	}
	strategyAssetCount := len(normalized.SourceAssetIDs)
	if strings.EqualFold(strings.TrimSpace(normalized.Mode), "batch") && strategyAssetCount > 1 {
		// Batch mode fans out into one image-to-image child per source asset.
		// Do not classify the root request as multi_image, otherwise provider routing
		// and child runtime manifests claim multi-image while each child has one input.
		strategyAssetCount = 1
	}
	inputMode, generationStrategy, err := resolveExecutableGenerationStrategy(normalized.InputMode, normalized.GenerationStrategy, strategyAssetCount)
	if err != nil {
		return CreateGenerationJobInput{}, err
	}
	normalized.InputMode = inputMode
	normalized.GenerationStrategy = generationStrategy
	params := mergeMaps(normalized.Params, map[string]any{
		"input_mode":          inputMode,
		"generation_strategy": generationStrategy,
	})
	metadata := mergeMaps(normalized.Metadata, map[string]any{
		"input_mode":          inputMode,
		"generation_strategy": generationStrategy,
		"source_assets":       sourceAssetInputsForMetadata(normalized.SourceAssets),
	})
	normalized.Params = params
	normalized.Metadata = metadata
	return normalized, nil
}

func normalizeSourceAssetInputs(inputs []StudioSourceAssetInput, legacyIDs []string) []StudioSourceAssetInput {
	out := make([]StudioSourceAssetInput, 0, len(inputs)+len(legacyIDs))
	seen := map[string]struct{}{}
	appendOne := func(item StudioSourceAssetInput) {
		assetID := strings.TrimSpace(firstNonEmpty(item.AssetID, item.ID))
		if assetID == "" {
			return
		}
		if _, exists := seen[assetID]; exists {
			return
		}
		seen[assetID] = struct{}{}
		item.AssetID = assetID
		item.ID = ""
		item.Role = strings.TrimSpace(item.Role)
		item.Label = strings.TrimSpace(item.Label)
		out = append(out, item)
	}
	for _, item := range inputs {
		appendOne(item)
	}
	for _, assetID := range legacyIDs {
		appendOne(StudioSourceAssetInput{AssetID: assetID})
	}
	return out
}

func sourceAssetIDsFromInputs(inputs []StudioSourceAssetInput) []string {
	out := make([]string, 0, len(inputs))
	for _, item := range inputs {
		assetID := strings.TrimSpace(firstNonEmpty(item.AssetID, item.ID))
		if assetID != "" {
			out = append(out, assetID)
		}
	}
	return out
}

func dedupeTrimmedStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func resolveExecutableGenerationStrategy(inputMode, generationStrategy string, sourceAssetCount int) (string, string, error) {
	mode := normalizeGenerationModeValue(firstNonEmpty(inputMode, generationStrategy))
	strategy := normalizeGenerationModeValue(firstNonEmpty(generationStrategy, inputMode))
	if mode == "" || mode == studioInputModeAskInput {
		mode = inferInputModeFromAssetCount(sourceAssetCount)
	}
	if strategy == "" || strategy == studioInputModeAskInput {
		strategy = mode
	}
	if mode == studioInputModeAskInput || strategy == studioInputModeAskInput {
		return "", "", newStudioBadRequest(
			"STUDIO_TEMPLATE_INPUT_REQUIRED",
			"Required creative materials are missing",
			"Upload the required materials before generating.",
		)
	}
	if mode == studioInputModeMultiImage && sourceAssetCount < 2 {
		return "", "", newStudioBadRequest(
			"STUDIO_MULTI_IMAGE_SOURCE_ASSETS_REQUIRED",
			"Multi-image generation requires at least two source images",
			"Add the required material slots before generating.",
		)
	}
	if mode != studioInputModeMultiImage && sourceAssetCount > 1 {
		mode = studioInputModeMultiImage
		strategy = studioGenerationStrategyMultiImage
	}
	return mode, strategy, nil
}

func normalizeGenerationModeValue(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "default":
		return ""
	case studioInputModeAskInput:
		return studioInputModeAskInput
	case studioInputModeTextToImage, "text", "txt2img":
		return studioInputModeTextToImage
	case studioInputModeImageToImage, "image", "img2img":
		return studioInputModeImageToImage
	case studioInputModeMultiImage, "multi", "multi-image", "multi_image_reference":
		return studioInputModeMultiImage
	case studioInputModeImageEdit, "edit", "image_editing":
		return studioInputModeImageEdit
	default:
		return strings.TrimSpace(value)
	}
}

func inferInputModeFromAssetCount(sourceAssetCount int) string {
	switch {
	case sourceAssetCount == 0:
		return studioInputModeTextToImage
	case sourceAssetCount == 1:
		return studioInputModeImageToImage
	default:
		return studioInputModeMultiImage
	}
}

func sourceAssetInputsForMetadata(inputs []StudioSourceAssetInput) []map[string]any {
	out := make([]map[string]any, 0, len(inputs))
	for _, item := range inputs {
		entry := map[string]any{
			"asset_id": firstNonEmpty(item.AssetID, item.ID),
		}
		if item.Role != "" {
			entry["role"] = item.Role
		}
		if item.Label != "" {
			entry["label"] = item.Label
		}
		if item.Required {
			entry["required"] = true
		}
		if len(item.Metadata) > 0 {
			entry["metadata"] = item.Metadata
		}
		out = append(out, entry)
	}
	return out
}

func resolveStudioProvider(providerCandidates []string, inputMode string, defaultProvider string) (string, error) {
	candidate := ""
	for _, item := range providerCandidates {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || strings.EqualFold(trimmed, "default") {
			continue
		}
		candidate = trimmed
		break
	}
	if inputMode == studioInputModeMultiImage {
		if candidate == "" {
			return "comfyui_bridge", nil
		}
		if _, ok := multiImageCapableProviders[normalizeRuntimeProviderCode(candidate)]; !ok {
			return "", newStudioBadRequest(
				"STUDIO_MULTI_IMAGE_PROVIDER_UNSUPPORTED",
				fmt.Sprintf("Provider %s does not support multi-image generation", candidate),
				"Choose a multi-image capable template/provider and try again.",
			)
		}
	}
	if candidate != "" {
		return candidate, nil
	}
	if inputMode == studioInputModeMultiImage {
		return "comfyui_bridge", nil
	}
	return defaultProvider, nil
}

func roleByAssetIDFromMetadata(metadata map[string]any) map[string]string {
	out := map[string]string{}
	if metadata == nil {
		return out
	}
	items, ok := metadata["source_assets"].([]map[string]any)
	if ok {
		for _, item := range items {
			assetID := stringMapValue(item, "asset_id")
			if assetID != "" {
				out[assetID] = stringMapValue(item, "role")
			}
		}
		return out
	}
	genericItems, ok := metadata["source_assets"].([]any)
	if !ok {
		return out
	}
	for _, raw := range genericItems {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		assetID := stringMapValue(item, "asset_id")
		if assetID != "" {
			out[assetID] = stringMapValue(item, "role")
		}
	}
	return out
}

func cloneMap(values map[string]any) map[string]any {
	out := map[string]any{}
	if len(values) > 0 {
		maps.Copy(out, values)
	}
	return out
}
