package service

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/model"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/storage"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/telemetry"
)

type fakeCollector struct {
	snapshot model.MetricSnapshot
	info     model.SystemInfo
}

func (f *fakeCollector) CollectMetrics(_ context.Context) (model.MetricSnapshot, error) {
	return f.snapshot, nil
}

func (f *fakeCollector) GetSystemInfo(_ context.Context) (model.SystemInfo, error) {
	return f.info, nil
}

func TestGetCurrentAlerts(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	history, err := storage.NewHistoryStore(10, filepath.Join(tmpDir, "history.jsonl"))
	if err != nil {
		t.Fatalf("create history store: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewMetricsService(
		&fakeCollector{snapshot: model.MetricSnapshot{
			CPUUsagePercent:    92,
			MemoryUsagePercent: 88,
			DiskUsagePercent:   91,
			CollectedAt:        time.Now().UTC(),
		}},
		history,
		logger,
		telemetry.New(),
		AlertThresholds{CPU: 85, Memory: 85, Disk: 90},
	)

	alerts, err := service.GetCurrentAlerts(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentAlerts returned error: %v", err)
	}

	if !alerts.Triggered {
		t.Fatalf("expected triggered alerts")
	}
	if len(alerts.Alerts) != 3 {
		t.Fatalf("expected 3 alerts, got %d", len(alerts.Alerts))
	}
}
