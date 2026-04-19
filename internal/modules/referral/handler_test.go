package referral

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/platform"

	"github.com/gin-gonic/gin"
)

func TestRedeemCommissions_AllowsEmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/incentives/commissions/redeem" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if payload["beneficiary_subject_id"] != "org-1" {
			t.Fatalf("unexpected org id: %#v", payload["beneficiary_subject_id"])
		}
		if payload["asset_code"] != "MENU_PROMO_CREDIT" {
			t.Fatalf("unexpected asset code: %#v", payload["asset_code"])
		}

		writeReferralPlatformSuccess(w, map[string]any{
			"reward_ledger_id": "reward-1",
			"asset_code":       "MENU_PROMO_CREDIT",
			"total_amount":     20,
			"commissions":      []any{},
		})
	}))
	defer server.Close()

	service := NewService(platform.New(config.PlatformConfig{
		BaseURL:               server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-referral-handler-test",
		InternalServiceSecret: "test-secret",
	}), config.AppConfig{
		RewardAssetCode: "MENU_PROMO_CREDIT",
	})

	handler := NewHandler(service, nil)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/v1/menu/referrals/me/commissions/redeem", bytes.NewBuffer(nil))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("orgID", "org-1")

	handler.RedeemCommissions(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", recorder.Code, recorder.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			RewardLedgerID string `json:"reward_ledger_id"`
			AssetCode      string `json:"asset_code"`
			TotalAmount    int64  `json:"total_amount"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("expected success code 0, got %d", resp.Code)
	}
	if resp.Data.RewardLedgerID != "reward-1" {
		t.Fatalf("unexpected reward ledger id: %s", resp.Data.RewardLedgerID)
	}
}
