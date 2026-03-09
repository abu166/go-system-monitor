package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestIDMiddlewareGeneratesIDWhenMissing(t *testing.T) {
	t.Parallel()

	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) == "" {
			t.Fatalf("expected request id in context")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatalf("expected X-Request-ID response header")
	}
}

func TestRequestIDMiddlewarePropagatesIncomingID(t *testing.T) {
	t.Parallel()

	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := RequestIDFromContext(r.Context()); got != "rid-123" {
			t.Fatalf("unexpected context request id: %q", got)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-ID", "rid-123")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-ID"); got != "rid-123" {
		t.Fatalf("unexpected response request id: %q", got)
	}
}

func TestShouldLogRequestSampling(t *testing.T) {
	t.Parallel()

	sampler := NewPathSampler(3)
	path := "/health"

	if !sampler.ShouldLog(path, 500) {
		t.Fatalf("expected always log for 5xx status")
	}

	if !sampler.ShouldLog(path, 200) {
		t.Fatalf("expected first request to be logged")
	}
	if sampler.ShouldLog(path, 200) {
		t.Fatalf("expected sampled request to be skipped")
	}
	if sampler.ShouldLog(path, 200) {
		t.Fatalf("expected sampled request to be skipped")
	}
	if !sampler.ShouldLog(path, 200) {
		t.Fatalf("expected sampled request to be logged every 3 requests")
	}
}
