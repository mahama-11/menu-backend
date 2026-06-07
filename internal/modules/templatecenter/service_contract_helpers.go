package templatecenter

import (
	"encoding/json"
	"strings"
)

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

func templateContractMapFromRaw(raw map[string]any) map[string]any {
	contract := map[string]any{}
	for key, value := range mapAnyValue(raw, "input_schema") {
		contract[key] = value
	}
	for _, key := range []string{"business_goal", "input_slots", "target_outputs", "strategy_policy"} {
		if value, ok := raw[key]; ok && value != nil {
			contract[key] = value
		}
	}
	return contract
}

func applyTemplateContract(summary *TemplateCatalogSummary, inputSchema map[string]any) {
	if summary == nil || len(inputSchema) == 0 {
		return
	}
	if businessGoal := stringMapValue(inputSchema, "business_goal"); businessGoal != "" {
		summary.BusinessGoal = businessGoal
	}
	if slots := decodeTemplateInputSlots(inputSchema["input_slots"]); len(slots) > 0 {
		summary.InputSlots = slots
	}
	if outputs := decodeTemplateTargetOutputs(inputSchema["target_outputs"]); len(outputs) > 0 {
		summary.TargetOutputs = outputs
	}
	if strategy := mapAnyValue(inputSchema, "strategy_policy"); len(strategy) > 0 {
		summary.StrategyPolicy = strategy
	}
}

func decodeTemplateInputSlots(value any) []TemplateInputSlot {
	if value == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var slots []TemplateInputSlot
	if err := json.Unmarshal(payload, &slots); err != nil {
		return nil
	}
	return slots
}

func decodeTemplateTargetOutputs(value any) []TemplateTargetOutput {
	if value == nil {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var outputs []TemplateTargetOutput
	if err := json.Unmarshal(payload, &outputs); err != nil {
		return nil
	}
	return outputs
}

func templateStrategyValue(strategy map[string]any, key, fallback string) string {
	if value, ok := strategy[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}
