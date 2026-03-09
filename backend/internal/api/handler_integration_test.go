package api

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/model"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/service"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/storage"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/telemetry"
)

type integrationCollector struct {
	snapshot model.MetricSnapshot
	info     model.SystemInfo
}

func (f *integrationCollector) CollectMetrics(_ context.Context) (model.MetricSnapshot, error) {
	return f.snapshot, nil
}

func (f *integrationCollector) GetSystemInfo(_ context.Context) (model.SystemInfo, error) {
	return f.info, nil
}

func setupIntegrationServer(t *testing.T) *httptest.Server {
	t.Helper()

	tmpDir := t.TempDir()
	history, err := storage.NewHistoryStore(100, filepath.Join(tmpDir, "history.jsonl"))
	if err != nil {
		t.Fatalf("create history store: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.NewMetricsService(
		&integrationCollector{
			snapshot: model.MetricSnapshot{
				CPUUsagePercent:    30,
				MemoryUsagePercent: 40,
				DiskUsagePercent:   50,
				CollectedAt:        time.Now().UTC(),
			},
			info: model.SystemInfo{
				Hostname: "test-host",
				OS:       "linux",
				CPUCores: 8,
			},
		},
		history,
		logger,
		telemetry.New(),
		service.AlertThresholds{CPU: 85, Memory: 85, Disk: 90},
	)

	handler := NewHandler(svc, logger, 20*time.Millisecond)
	server := NewServer(ServerConfig{
		Port:              "0",
		ReadTimeout:       2 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
		WriteTimeout:      2 * time.Second,
		IdleTimeout:       2 * time.Second,
		MaxHeaderBytes:    1 << 20,
		CORSAllowedOrigins: []string{"http://localhost:3000"},
		LogSampleRate:      1,
	}, logger, handler, telemetry.New())

	return httptest.NewServer(server.httpServer.Handler)
}

func TestAPIEndpointsJSONContracts(t *testing.T) {
	t.Parallel()
	ts := setupIntegrationServer(t)
	defer ts.Close()

	cases := []struct {
		path string
		key  string
	}{
		{path: "/health", key: `"success":true`},
		{path: "/api/metrics/latest", key: `"cpu_usage_percent"`},
		{path: "/api/metrics/history", key: `"success":true`},
		{path: "/api/system/info", key: `"hostname"`},
		{path: "/api/alerts/current", key: `"triggered"`},
		{path: "/live", key: `"status":"live"`},
		{path: "/ready", key: `"status":"ready"`},
	}

	for _, tc := range cases {
		resp, err := http.Get(ts.URL + tc.path)
		if err != nil {
			t.Fatalf("get %s: %v", tc.path, err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Fatalf("%s expected status 200 got %d body=%s", tc.path, resp.StatusCode, string(body))
		}
		if !strings.Contains(string(body), tc.key) {
			t.Fatalf("%s expected key %q body=%s", tc.path, tc.key, string(body))
		}
	}
}

func TestMetricsStreamSSEContract(t *testing.T) {
	t.Parallel()
	ts := setupIntegrationServer(t)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/metrics/stream", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.Contains(ct, "text/event-stream") {
		t.Fatalf("expected text/event-stream content type, got %q", ct)
	}

	reader := bufio.NewReader(resp.Body)
	line1, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read event line: %v", err)
	}
	if strings.TrimSpace(line1) != "event: metrics" {
		t.Fatalf("unexpected event line: %q", line1)
	}

	line2, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read data line: %v", err)
	}
	if !strings.HasPrefix(line2, "data: ") {
		t.Fatalf("unexpected data line: %q", line2)
	}

	payload := strings.TrimSpace(strings.TrimPrefix(line2, "data: "))
	var event model.MetricsStreamEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		t.Fatalf("decode stream payload: %v", err)
	}
	if event.Snapshot.CollectedAt.IsZero() {
		t.Fatalf("expected collected_at in stream payload")
	}
}

func TestRequestIDAndCORSHeaders(t *testing.T) {
	t.Parallel()
	ts := setupIntegrationServer(t)
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/health", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	req.Header.Set("X-Request-ID", "test-rid-123")
	req.Header.Set("Origin", "http://localhost:3000")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("perform request: %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("X-Request-ID"); got != "test-rid-123" {
		t.Fatalf("expected request id propagation, got %q", got)
	}

	allowOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	if allowOrigin != "http://localhost:3000" {
		t.Fatalf("expected CORS allow origin header, got %q", allowOrigin)
	}
}
