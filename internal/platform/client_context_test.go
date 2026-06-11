package platform

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"menu-service/internal/config"

	"go.opentelemetry.io/otel/trace"
)

func TestClientWithContextPropagatesCorrelationAndTraceparent(t *testing.T) {
	seen := map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen["request_id"] = r.Header.Get("X-Request-ID")
		seen["trace_id"] = r.Header.Get("X-Trace-ID")
		seen["traceparent"] = r.Header.Get("traceparent")
		seen["internal_service"] = r.Header.Get("X-Internal-Service")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":       0,
			"message":    "ok",
			"request_id": "platform-req",
			"trace_id":   "platform-trace",
			"data": map[string]any{
				"billing_subject_type": "organization",
				"billing_subject_id":   "org-ctx",
				"product_code":         "menu",
				"total_balance":        1,
			},
		})
	}))
	defer server.Close()

	ctx, traceID := outboundTraceContext("req-menu-outbound")
	client := New(config.PlatformConfig{BaseURL: server.URL, Timeout: DefaultTimeout(), ServiceName: "v-menu-backend-test", InternalServiceSecret: "secret"})
	if _, err := client.WithContext(ctx).GetWalletSummary("organization", "org-ctx", "menu"); err != nil {
		t.Fatalf("GetWalletSummary: %v", err)
	}
	assertCorrelationHeaders(t, seen, "req-menu-outbound", traceID.String(), "v-menu-backend-test")
}

func TestClientWithContextPropagatesCorrelationToPublicPlatformCalls(t *testing.T) {
	seen := map[string]string{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen["request_id"] = r.Header.Get("X-Request-ID")
		seen["trace_id"] = r.Header.Get("X-Trace-ID")
		seen["traceparent"] = r.Header.Get("traceparent")
		seen["internal_service"] = r.Header.Get("X-Internal-Service")
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"code":       0,
			"message":    "ok",
			"request_id": "platform-req",
			"trace_id":   "platform-trace",
			"data": map[string]any{
				"access_token": "token",
				"user": map[string]any{
					"id": "user-ctx", "email": "owner@example.test", "org_id": "org-ctx", "status": "active",
				},
			},
		})
	}))
	defer server.Close()

	ctx, traceID := outboundTraceContext("req-menu-public")
	client := New(config.PlatformConfig{BaseURL: server.URL, Timeout: DefaultTimeout(), ServiceName: "v-menu-backend-test", InternalServiceSecret: "secret"})
	if _, err := client.WithContext(ctx).Register(AuthRegisterInput{FullName: "Owner", Email: "owner@example.test", Password: "change-me", Company: "Kitchen"}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	assertCorrelationHeaders(t, seen, "req-menu-public", traceID.String(), "")
}

func outboundTraceContext(requestID string) (context.Context, trace.TraceID) {
	traceID := trace.TraceID{0x4b, 0xf9, 0x2f, 0x35, 0x77, 0xb3, 0x4d, 0xa6, 0xa3, 0xce, 0x92, 0x9d, 0x0e, 0x0e, 0x47, 0x36}
	spanID := trace.SpanID{0x00, 0xf0, 0x67, 0xaa, 0x0b, 0xa9, 0x02, 0xb7}
	spanCtx := trace.NewSpanContext(trace.SpanContextConfig{TraceID: traceID, SpanID: spanID, TraceFlags: trace.FlagsSampled, Remote: true})
	ctx := trace.ContextWithSpanContext(context.Background(), spanCtx)
	ctx = context.WithValue(ctx, "request_id", requestID)
	ctx = context.WithValue(ctx, "trace_id", traceID.String())
	return ctx, traceID
}

func assertCorrelationHeaders(t *testing.T, seen map[string]string, requestID, traceID, internalService string) {
	t.Helper()
	t.Logf("observed outbound correlation request_id=%s trace_id=%s traceparent=%s internal_service=%s", seen["request_id"], seen["trace_id"], seen["traceparent"], seen["internal_service"])
	if seen["request_id"] != requestID {
		t.Fatalf("X-Request-ID=%q", seen["request_id"])
	}
	if seen["trace_id"] != traceID {
		t.Fatalf("X-Trace-ID=%q", seen["trace_id"])
	}
	if seen["traceparent"] != "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01" {
		t.Fatalf("traceparent=%q", seen["traceparent"])
	}
	if seen["internal_service"] != internalService {
		t.Fatalf("X-Internal-Service=%q", seen["internal_service"])
	}
}
