package api

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/telemetry"
)

func TestRecoverMiddlewareReturnsInternalServerError(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handler := RecoverMiddleware(logger, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"success":false`) {
		t.Fatalf("expected error response body, got: %s", rec.Body.String())
	}
}

func TestSecurityHeadersMiddlewareSetsHeaders(t *testing.T) {
	t.Parallel()

	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeSuccess(w, http.StatusOK, map[string]string{"status": "ok"})
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	tests := []struct {
		name string
		key  string
		want string
	}{
		{name: "nosniff", key: "X-Content-Type-Options", want: "nosniff"},
		{name: "frame-options", key: "X-Frame-Options", want: "DENY"},
		{name: "referrer-policy", key: "Referrer-Policy", want: "no-referrer"},
	}

	for _, tc := range tests {
		if got := rec.Header().Get(tc.key); got != tc.want {
			t.Fatalf("%s header mismatch: expected %q got %q", tc.name, tc.want, got)
		}
	}

	csp := rec.Header().Get("Content-Security-Policy")
	if !strings.Contains(csp, "default-src 'none'") {
		t.Fatalf("expected restrictive CSP header, got %q", csp)
	}
}

func TestLoggingMiddlewareDoesNotBreakWithoutPanic(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	handler := LoggingMiddleware(logger, telemetry.New(), NewPathSampler(1), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected status %d got %d", http.StatusAccepted, rec.Code)
	}
}
