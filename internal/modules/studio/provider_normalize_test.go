package studio

import "testing"

func TestNormalizeRuntimeProviderCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "default", input: "default", expected: ""},
		{name: "default mixed case", input: " Default ", expected: ""},
		{name: "explicit provider", input: "comfyui_bridge", expected: "comfyui_bridge"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeRuntimeProviderCode(tc.input); got != tc.expected {
				t.Fatalf("normalizeRuntimeProviderCode(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}
