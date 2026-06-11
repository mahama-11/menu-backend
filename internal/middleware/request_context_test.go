package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func TestRequestContextPrefersTraceparentOverLegacyTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(otelgin.Middleware("menu-request-context-test"), RequestContext())
	r.GET("/trace", func(c *gin.Context) {
		if got := c.GetString("requestID"); got != "req-menu-entry" {
			t.Fatalf("requestID=%q", got)
		}
		if got := c.GetString("traceID"); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
			t.Fatalf("traceID=%q", got)
		}
		if got, _ := c.Request.Context().Value("request_id").(string); got != "req-menu-entry" {
			t.Fatalf("context request_id=%q", got)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/trace", nil)
	req.Header.Set("X-Request-ID", "req-menu-entry")
	req.Header.Set("X-Trace-ID", "legacy-trace-id")
	req.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if got := w.Header().Get("X-Trace-ID"); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("response X-Trace-ID=%q", got)
	}
}
