package referral

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/platform"
)

func TestCreateCode_ReturnsInviteURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/incentives/referral-codes" {
			http.NotFound(w, r)
			return
		}
		writeReferralPlatformSuccess(w, map[string]any{
			"id":                   "code-1",
			"program_id":           "program-1",
			"product_code":         "menu",
			"code":                 "ABC123",
			"promoter_subject_type": "organization",
			"promoter_subject_id":  "org-1",
			"status":               "active",
			"metadata":             `{"source":"menu"}`,
			"created_at":           time.Now().UTC().Format(time.RFC3339),
			"updated_at":           time.Now().UTC().Format(time.RFC3339),
		})
	}))
	defer server.Close()

	service := NewService(platform.New(config.PlatformConfig{
		BaseURL:               server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-referral-test",
		InternalServiceSecret: "test-secret",
	}), config.AppConfig{
		FrontendBaseURL: "https://menu.example.com",
	})

	item, err := service.CreateCode("org-1", CreateCodeInput{ProgramCode: "menu_signup_default"})
	if err != nil {
		t.Fatalf("CreateCode() error = %v", err)
	}
	if item.InviteURL != "https://menu.example.com/signup?referral_code=ABC123" {
		t.Fatalf("invite url = %s", item.InviteURL)
	}
	if item.ShareText == "" {
		t.Fatalf("expected share text")
	}
}

func writeReferralPlatformSuccess(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":      0,
		"message":   "success",
		"timestamp": time.Now().UnixMilli(),
		"data":      data,
	})
}
