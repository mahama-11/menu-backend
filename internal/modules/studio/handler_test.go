package studio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"menu-service/internal/config"
	"menu-service/internal/platform"

	"github.com/gin-gonic/gin"
)

func TestWriteRegisterAssetError_InvalidPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	err := newPlatformUploadErrorForTest(t, http.StatusBadRequest, map[string]any{
		"code":       1000,
		"message":    "Invalid parameter",
		"error":      "invalid asset upload payload",
		"error_code": "STORAGE_ASSET_PAYLOAD_INVALID",
		"error_hint": "Send a valid data URL or base64-encoded image payload.",
		"request_id": "platform-test-invalid-payload",
		"timestamp":  time.Now().UnixMilli(),
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	writeRegisterAssetError(ctx, err)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if jsonErr := json.Unmarshal(recorder.Body.Bytes(), &body); jsonErr != nil {
		t.Fatalf("json.Unmarshal: %v", jsonErr)
	}
	if body["error_code"] != "STUDIO_ASSET_PAYLOAD_INVALID" {
		t.Fatalf("unexpected error response: %+v", body)
	}
}

func TestWriteRegisterAssetError_UpstreamFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	err := newPlatformUploadErrorForTest(t, http.StatusInternalServerError, map[string]any{
		"code":       5000,
		"message":    "Internal server error",
		"error":      "failed to upload asset",
		"error_code": "ASSET_STORAGE_UPLOAD_FAILED",
		"error_hint": "Check platform logs and retry.",
		"request_id": "platform-test-upstream",
		"timestamp":  time.Now().UnixMilli(),
	})

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	writeRegisterAssetError(ctx, err)

	if recorder.Code != http.StatusBadGateway {
		t.Fatalf("expected 502, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if jsonErr := json.Unmarshal(recorder.Body.Bytes(), &body); jsonErr != nil {
		t.Fatalf("json.Unmarshal: %v", jsonErr)
	}
	if body["error_code"] != "STUDIO_ASSET_UPSTREAM_FAILED" {
		t.Fatalf("unexpected error response: %+v", body)
	}
}

func newPlatformUploadErrorForTest(t *testing.T, status int, payload map[string]any) error {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/v1/storage/assets" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer server.Close()

	client := platform.New(config.PlatformConfig{
		BaseURL:               server.URL,
		Timeout:               time.Second,
		ServiceName:           "menu-test",
		InternalServiceSecret: "test-secret",
	})
	_, err := client.UploadAsset(platform.UploadAssetInput{
		ProductCode: "menu",
		Category:    "studio-assets",
		FileName:    "probe.png",
		MimeType:    "image/png",
		Payload:     "data:image/png;base64,aGVsbG8=",
	})
	if err == nil {
		t.Fatal("expected platform upload error")
	}
	return err
}
