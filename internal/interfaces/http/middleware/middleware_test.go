package middleware

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestCORS_AllowedOrigin(t *testing.T) {
	os.Setenv("CORS_ALLOW_ORIGIN", "https://example.com,https://app.example.com")
	defer os.Unsetenv("CORS_ALLOW_ORIGIN")

	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Allowed origin
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if got := rr.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("expected https://example.com, got %q", got)
	}
	if got := rr.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected credentials=true, got %q", got)
	}

	// Rejected origin
	req2 := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	req2.Header.Set("Origin", "https://evil.com")
	rr2 := httptest.NewRecorder()
	handler.ServeHTTP(rr2, req2)

	if got := rr2.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected empty origin for rejected, got %q", got)
	}
}

func TestCORS_Preflight(t *testing.T) {
	os.Setenv("CORS_ALLOW_ORIGIN", "https://example.com")
	defer os.Unsetenv("CORS_ALLOW_ORIGIN")

	handler := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/test", nil)
	req.Header.Set("Origin", "https://example.com")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Errorf("expected 204 for preflight, got %d", rr.Code)
	}
}

func TestTracing_AddsSpan(t *testing.T) {
	var called bool
	handler := Tracing(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("handler was not called")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

func TestAuditLog_SkipsNonSensitiveGET(t *testing.T) {
	if isSensitiveGET("/api/v1/health") {
		t.Error("/api/v1/health should not be sensitive")
	}
	if !isSensitiveGET("/api/v1/recordings/123") {
		t.Error("/api/v1/recordings/123 should be sensitive")
	}
	if !isSensitiveGET("/api/v1/customers") {
		t.Error("/api/v1/customers should be sensitive")
	}
}

func TestClientIP_XForwardedFor(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-For", "1.2.3.4, 5.6.7.8")
	if got := clientIP(req); got != "1.2.3.4" {
		t.Errorf("expected 1.2.3.4, got %s", got)
	}
}

func TestClientIP_XRealIP(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Real-IP", "10.0.0.1")
	if got := clientIP(req); got != "10.0.0.1" {
		t.Errorf("expected 10.0.0.1, got %s", got)
	}
}
