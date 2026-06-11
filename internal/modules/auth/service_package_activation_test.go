package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/platform"
)

func TestRegisterActivatesConfiguredSignupPackage(t *testing.T) {
	var activationRequests []platform.ActivatePackageInput
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/register":
			writeTestEnvelope(t, w, map[string]any{
				"access_token": "token-1",
				"user": map[string]any{
					"id": "user-1", "email": "new@example.com", "full_name": "New Restaurant",
					"org_id": "org-1", "org_role": "owner", "plan_id": "starter", "status": "active",
					"orgs": []map[string]any{{"id": "org-1", "name": "New Restaurant", "role": "owner"}},
				},
			})
		case "/internal/v1/controls/package-activations":
			var req platform.ActivatePackageInput
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode activation request: %v", err)
			}
			activationRequests = append(activationRequests, req)
			writeTestEnvelope(t, w, platform.PackageActivationResult{ProductCode: req.ProductCode, PackageCode: req.PackageCode, BillingSubjectType: req.BillingSubjectType, BillingSubjectID: req.BillingSubjectID, ReferenceID: req.ReferenceID, GrantedQuotaUnits: 5})
		case "/internal/v1/wallet/summary":
			writeTestEnvelope(t, w, platform.WalletSummary{BillingSubjectType: "organization", BillingSubjectID: "org-1", ProductCode: "menu"})
		default:
			t.Fatalf("unexpected platform path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := platform.New(config.PlatformConfig{BaseURL: server.URL, Timeout: time.Second, ServiceName: "menu-test", InternalServiceSecret: "secret"})
	service := NewService(client, nil, nil, config.AppConfig{SignupBonusCredits: 0, CreditsAssetCode: "MENU_CREDIT", SignupPackageCode: "menu.pkg.trial.signup"})

	if _, err := service.Register(RegisterInput{Email: "new@example.com", Password: "secret123", RestaurantName: "New Restaurant"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	if len(activationRequests) != 1 {
		t.Fatalf("activation requests = %d, want 1", len(activationRequests))
	}
	got := activationRequests[0]
	if got.ProductCode != "menu" || got.PackageCode != "menu.pkg.trial.signup" || got.BillingSubjectType != "organization" || got.BillingSubjectID != "org-1" {
		t.Fatalf("unexpected activation request: %+v", got)
	}
	if got.ActivationReason != "signup_trial" || got.ReferenceID != "menu:signup:org-1" {
		t.Fatalf("activation idempotency/reason mismatch: %+v", got)
	}
	if len(got.ReferenceID) > 64 {
		t.Fatalf("activation reference_id should fit platform storage limit, got len=%d", len(got.ReferenceID))
	}
}

func TestLoginActivatesConfiguredSignupPackageForRecovery(t *testing.T) {
	var activationRequests []platform.ActivatePackageInput
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			writeTestEnvelope(t, w, map[string]any{
				"access_token": "token-1",
				"user": map[string]any{
					"id": "user-1", "email": "new@example.com", "full_name": "New Restaurant",
					"org_id": "org-1", "org_role": "owner", "plan_id": "starter", "status": "active",
					"orgs": []map[string]any{{"id": "org-1", "name": "New Restaurant", "role": "owner"}},
				},
			})
		case "/internal/v1/controls/package-activations":
			var req platform.ActivatePackageInput
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode activation request: %v", err)
			}
			activationRequests = append(activationRequests, req)
			writeTestEnvelope(t, w, platform.PackageActivationResult{ProductCode: req.ProductCode, PackageCode: req.PackageCode, BillingSubjectType: req.BillingSubjectType, BillingSubjectID: req.BillingSubjectID, ReferenceID: req.ReferenceID, GrantedQuotaUnits: 5, Idempotent: true})
		case "/internal/v1/access/users/user-1/orgs/org-1":
			writeTestEnvelope(t, w, platform.PlatformAccessData{UserID: "user-1", OrgID: "org-1", OrgRole: "owner", Permissions: []string{"menu:studio:write"}})
		case "/internal/v1/wallet/summary":
			writeTestEnvelope(t, w, platform.WalletSummary{BillingSubjectType: "organization", BillingSubjectID: "org-1", ProductCode: "menu"})
		default:
			t.Fatalf("unexpected platform path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := platform.New(config.PlatformConfig{BaseURL: server.URL, Timeout: time.Second, ServiceName: "menu-test", InternalServiceSecret: "secret"})
	service := NewService(client, nil, nil, config.AppConfig{SignupBonusCredits: 0, CreditsAssetCode: "MENU_CREDIT", SignupPackageCode: "menu.pkg.trial.signup"})

	if _, err := service.Login(LoginInput{Email: "new@example.com", Password: "secret123"}); err != nil {
		t.Fatalf("Login: %v", err)
	}
	if len(activationRequests) != 1 {
		t.Fatalf("activation requests = %d, want 1", len(activationRequests))
	}
	if got := activationRequests[0]; got.ReferenceID != "menu:signup:org-1" || got.BillingSubjectID != "org-1" {
		t.Fatalf("unexpected recovery activation request: %+v", got)
	}
}

func TestLoginContinuesWhenSignupPackageRecoveryActivationFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/auth/login":
			writeTestEnvelope(t, w, map[string]any{
				"access_token": "token-1",
				"user": map[string]any{
					"id": "user-1", "email": "new@example.com", "full_name": "New Restaurant",
					"org_id": "org-1", "org_role": "owner", "plan_id": "starter", "status": "active",
					"orgs": []map[string]any{{"id": "org-1", "name": "New Restaurant", "role": "owner"}},
				},
			})
		case "/internal/v1/controls/package-activations":
			w.WriteHeader(http.StatusBadGateway)
			_ = json.NewEncoder(w).Encode(map[string]any{"code": 1001, "message": "activation temporarily unavailable"})
		case "/internal/v1/access/users/user-1/orgs/org-1":
			writeTestEnvelope(t, w, platform.PlatformAccessData{UserID: "user-1", OrgID: "org-1", OrgRole: "owner", Permissions: []string{"menu:studio:write"}})
		case "/internal/v1/wallet/summary":
			writeTestEnvelope(t, w, platform.WalletSummary{BillingSubjectType: "organization", BillingSubjectID: "org-1", ProductCode: "menu"})
		default:
			t.Fatalf("unexpected platform path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := platform.New(config.PlatformConfig{BaseURL: server.URL, Timeout: time.Second, ServiceName: "menu-test", InternalServiceSecret: "secret"})
	service := NewService(client, nil, nil, config.AppConfig{SignupBonusCredits: 0, CreditsAssetCode: "MENU_CREDIT", SignupPackageCode: "menu.pkg.trial.signup"})

	result, err := service.Login(LoginInput{Email: "new@example.com", Password: "secret123"})
	if err != nil {
		t.Fatalf("Login should continue on recovery activation failure: %v", err)
	}
	if result == nil || result.AccessToken != "token-1" {
		t.Fatalf("unexpected login result: %+v", result)
	}
}

func writeTestEnvelope(t *testing.T, w http.ResponseWriter, data any) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]any{"code": 0, "message": "success", "data": data}); err != nil {
		t.Fatalf("write envelope: %v", err)
	}
}
