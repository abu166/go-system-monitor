package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/collector"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/model"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/storage"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/telemetry"
)

type AlertThresholds struct {
	CPU    float64
	Memory float64
	Disk   float64
}

type MetricsService interface {
	GetLatest(ctx context.Context) (model.MetricSnapshot, error)
	GetLatestWithAlerts(ctx context.Context) (model.MetricsStreamEvent, error)
	GetHistory(ctx context.Context) ([]model.MetricSnapshot, error)
	GetSystemInfo(ctx context.Context) (model.SystemInfo, error)
	GetCurrentAlerts(ctx context.Context) (model.AlertStatus, error)
}

type metricsService struct {
	collector   collector.Collector
	history     *storage.HistoryStore
	logger      *slog.Logger
	telemetry   *telemetry.Metrics
	timeout     time.Duration
	thresholds  AlertThresholds
}

func NewMetricsService(c collector.Collector, history *storage.HistoryStore, logger *slog.Logger, tm *telemetry.Metrics, thresholds AlertThresholds) MetricsService {
	return &metricsService{
		collector:  c,
		history:    history,
		logger:     logger,
		telemetry:  tm,
		timeout:    3 * time.Second,
		thresholds: thresholds,
	}
}

func (s *metricsService) GetLatest(ctx context.Context) (model.MetricSnapshot, error) {
	readCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	snapshot, err := s.collector.CollectMetrics(readCtx)
	if err != nil {
		s.telemetry.CollectErrors.WithLabelValues("metrics").Inc()
		s.logger.Error("failed to collect metrics", "error", err)
		return model.MetricSnapshot{}, fmt.Errorf("collect latest metrics: %w", err)
	}

	if err := s.history.Add(snapshot); err != nil {
		s.logger.Error("failed to persist metric snapshot", "error", err)
		return model.MetricSnapshot{}, fmt.Errorf("persist latest metrics: %w", err)
	}

	s.telemetry.ObserveSnapshot(snapshot)
	return snapshot, nil
}

func (s *metricsService) GetLatestWithAlerts(ctx context.Context) (model.MetricsStreamEvent, error) {
	snapshot, err := s.GetLatest(ctx)
	if err != nil {
		return model.MetricsStreamEvent{}, err
	}

	return model.MetricsStreamEvent{
		Snapshot: snapshot,
		Alerts:   s.evaluateAlerts(snapshot),
	}, nil
}

func (s *metricsService) GetCurrentAlerts(ctx context.Context) (model.AlertStatus, error) {
	snapshot, err := s.GetLatest(ctx)
	if err != nil {
		return model.AlertStatus{}, err
	}
	return s.evaluateAlerts(snapshot), nil
}

func (s *metricsService) GetHistory(ctx context.Context) ([]model.MetricSnapshot, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	return s.history.List(), nil
}

func (s *metricsService) GetSystemInfo(ctx context.Context) (model.SystemInfo, error) {
	readCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	info, err := s.collector.GetSystemInfo(readCtx)
	if err != nil {
		s.telemetry.CollectErrors.WithLabelValues("system_info").Inc()
		s.logger.Error("failed to collect system info", "error", err)
		return model.SystemInfo{}, fmt.Errorf("collect system info: %w", err)
	}

	return info, nil
}

func (s *metricsService) evaluateAlerts(snapshot model.MetricSnapshot) model.AlertStatus {
	alerts := make([]model.Alert, 0, 3)

	if snapshot.CPUUsagePercent >= s.thresholds.CPU {
		alerts = append(alerts, model.Alert{
			Resource:   "cpu",
			Value:      snapshot.CPUUsagePercent,
			Threshold:  s.thresholds.CPU,
			Message:    "CPU usage is above threshold",
			IsCritical: snapshot.CPUUsagePercent >= s.thresholds.CPU+10,
		})
	}

	if snapshot.MemoryUsagePercent >= s.thresholds.Memory {
		alerts = append(alerts, model.Alert{
			Resource:   "memory",
			Value:      snapshot.MemoryUsagePercent,
			Threshold:  s.thresholds.Memory,
			Message:    "Memory usage is above threshold",
			IsCritical: snapshot.MemoryUsagePercent >= s.thresholds.Memory+10,
		})
	}

	if snapshot.DiskUsagePercent >= s.thresholds.Disk {
		alerts = append(alerts, model.Alert{
			Resource:   "disk",
			Value:      snapshot.DiskUsagePercent,
			Threshold:  s.thresholds.Disk,
			Message:    "Disk usage is above threshold",
			IsCritical: snapshot.DiskUsagePercent >= s.thresholds.Disk+5,
		})
	}

	return model.AlertStatus{
		Triggered:  len(alerts) > 0,
		Alerts:     alerts,
		EvaluatedAt: time.Now().UTC(),
	}
}
