package service

import (
	"context"
	"encoding/json"
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

func TestGetCurrentAlertsDoesNotPersistHistoryAgain(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	history, err := storage.NewHistoryStore(10, filepath.Join(tmpDir, "history.jsonl"))
	if err != nil {
		t.Fatalf("create history store: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewMetricsService(
		&fakeCollector{snapshot: model.MetricSnapshot{
			CPUUsagePercent:    70,
			MemoryUsagePercent: 50,
			DiskUsagePercent:   40,
			CollectedAt:        time.Now().UTC(),
		}},
		history,
		logger,
		telemetry.New(),
		AlertThresholds{CPU: 85, Memory: 85, Disk: 90},
	)

	if _, err := svc.GetLatest(context.Background()); err != nil {
		t.Fatalf("GetLatest returned error: %v", err)
	}
	before := len(history.List())

	if _, err := svc.GetCurrentAlerts(context.Background()); err != nil {
		t.Fatalf("GetCurrentAlerts returned error: %v", err)
	}
	after := len(history.List())

	if before != after {
		t.Fatalf("expected history length unchanged, before=%d after=%d", before, after)
	}

	raw, readErr := os.ReadFile(filepath.Join(tmpDir, "history.jsonl"))
	if readErr != nil {
		t.Fatalf("read history file: %v", readErr)
	}
	lines := 0
	for _, b := range raw {
		if b == '\n' {
			lines++
		}
	}
	if lines != 1 {
		t.Fatalf("expected exactly 1 persisted snapshot, got %d lines", lines)
	}
}

func TestGetCurrentAlertsUsesLatestSnapshotFromHistory(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "history.jsonl")
	snapshot := model.MetricSnapshot{
		CPUUsagePercent:    91,
		MemoryUsagePercent: 89,
		DiskUsagePercent:   95,
		CollectedAt:        time.Now().UTC(),
	}
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatalf("marshal snapshot: %v", err)
	}
	if err := os.WriteFile(filePath, append(encoded, '\n'), 0o644); err != nil {
		t.Fatalf("seed history: %v", err)
	}

	history, err := storage.NewHistoryStore(10, filePath)
	if err != nil {
		t.Fatalf("create history store: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	svc := NewMetricsService(
		&fakeCollector{snapshot: model.MetricSnapshot{
			CPUUsagePercent:    10,
			MemoryUsagePercent: 10,
			DiskUsagePercent:   10,
			CollectedAt:        time.Now().UTC(),
		}},
		history,
		logger,
		telemetry.New(),
		AlertThresholds{CPU: 85, Memory: 85, Disk: 90},
	)

	alerts, err := svc.GetCurrentAlerts(context.Background())
	if err != nil {
		t.Fatalf("GetCurrentAlerts returned error: %v", err)
	}

	if !alerts.Triggered {
		t.Fatalf("expected alerts from latest persisted snapshot")
	}
}
