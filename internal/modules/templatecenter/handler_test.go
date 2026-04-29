package templatecenter

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestTemplateCenterHandlerFlows(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _, _ := newTemplateCenterTestService(t)
	handler := NewHandler(service, nil)

	metaResp := performTemplateRequest(t, handler.Meta, http.MethodGet, "/meta", nil, nil, nil)
	if metaResp.Code != http.StatusOK {
		t.Fatalf("Meta status=%d body=%s", metaResp.Code, metaResp.Body.String())
	}

	listResp := performTemplateRequest(t, handler.ListCatalog, http.MethodGet, "/catalog?plan=basic", nil, nil, map[string]string{"userID": "user-1", "orgID": "org-1"})
	if listResp.Code != http.StatusOK {
		t.Fatalf("ListCatalog status=%d body=%s", listResp.Code, listResp.Body.String())
	}

	detailResp := performTemplateRequest(t, handler.Detail, http.MethodGet, "/catalog/TPL-TH-001?plan=basic", nil, gin.Params{{Key: "templateID", Value: "TPL-TH-001"}}, map[string]string{"userID": "user-1", "orgID": "org-1"})
	if detailResp.Code != http.StatusOK {
		t.Fatalf("Detail status=%d body=%s", detailResp.Code, detailResp.Body.String())
	}

	favoriteResp := performTemplateRequest(t, handler.SetFavorite, http.MethodPost, "/favorites/TPL-TH-001", nil, gin.Params{{Key: "templateID", Value: "TPL-TH-001"}}, map[string]string{"userID": "user-1", "orgID": "org-1"})
	if favoriteResp.Code != http.StatusOK {
		t.Fatalf("SetFavorite status=%d body=%s", favoriteResp.Code, favoriteResp.Body.String())
	}
	listFavoriteResp := performTemplateRequest(t, handler.ListFavorites, http.MethodGet, "/favorites", nil, nil, map[string]string{"userID": "user-1", "orgID": "org-1"})
	if listFavoriteResp.Code != http.StatusOK {
		t.Fatalf("ListFavorites status=%d body=%s", listFavoriteResp.Code, listFavoriteResp.Body.String())
	}

	useResp := performTemplateRequest(t, handler.Use, http.MethodPost, "/catalog/TPL-TH-001/use?plan=basic", UseTemplateInput{
		TargetPlatform: "instagram_feed",
		Language:       "en",
	}, gin.Params{{Key: "templateID", Value: "TPL-TH-001"}}, map[string]string{"userID": "user-1", "orgID": "org-1"})
	if useResp.Code != http.StatusOK {
		t.Fatalf("Use status=%d body=%s", useResp.Code, useResp.Body.String())
	}

	copyResp := performTemplateRequest(t, handler.CopyToMyTemplates, http.MethodPost, "/catalog/TPL-TH-001/copy", CopyTemplateInput{
		Name:       "My Template",
		Visibility: "private",
	}, gin.Params{{Key: "templateID", Value: "TPL-TH-001"}}, map[string]string{"userID": "user-1", "orgID": "org-1"})
	if copyResp.Code != http.StatusCreated {
		t.Fatalf("CopyToMyTemplates status=%d body=%s", copyResp.Code, copyResp.Body.String())
	}
}

func TestTemplateCenterHandlerErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service, _, _ := newTemplateCenterTestService(t)
	handler := NewHandler(service, nil)

	resp := performTemplateRaw(t, handler.Use, http.MethodPost, "/catalog/TPL-TH-001/use", []byte("{bad"), gin.Params{{Key: "templateID", Value: "TPL-TH-001"}}, map[string]string{"userID": "user-1", "orgID": "org-1"})
	if resp.Code == http.StatusOK {
		t.Fatalf("expected bind error")
	}
	resp = performTemplateRequest(t, handler.Detail, http.MethodGet, "/catalog/missing", nil, gin.Params{{Key: "templateID", Value: "missing"}}, map[string]string{"userID": "user-1", "orgID": "org-1"})
	if resp.Code == http.StatusOK {
		t.Fatalf("expected missing detail error")
	}
	resp = performTemplateRequest(t, handler.Use, http.MethodPost, "/catalog/TPL-JP-001/use?plan=basic", UseTemplateInput{
		TargetPlatform: "instagram_feed",
	}, gin.Params{{Key: "templateID", Value: "TPL-JP-001"}}, map[string]string{"userID": "user-1", "orgID": "org-1"})
	if resp.Code == http.StatusOK {
		t.Fatalf("expected plan conflict")
	}
}

func performTemplateRequest(t *testing.T, fn func(*gin.Context), method, path string, body any, params gin.Params, values map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var payload []byte
	if body != nil {
		payload, _ = json.Marshal(body)
	}
	return performTemplateRaw(t, fn, method, path, payload, params, values)
}

func performTemplateRaw(t *testing.T, fn func(*gin.Context), method, path string, body []byte, params gin.Params, values map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c.Request = req
	c.Params = params
	for key, value := range values {
		c.Set(key, value)
	}
	fn(c)
	if w.Code >= 500 {
		t.Fatalf("unexpected template handler failure for %s %s: status=%d body=%s", method, path, w.Code, w.Body.String())
	}
	return w
}
