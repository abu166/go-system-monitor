package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/api"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/collector"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/config"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/service"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/storage"
	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/telemetry"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	metrics := telemetry.New()
	collectorSvc := collector.NewSystemCollector(cfg.DiskPath)
	historyStore, err := storage.NewHistoryStore(cfg.HistoryLimit, cfg.PersistentHistoryPath)
	if err != nil {
		logger.Error("failed to initialize history storage", "error", err)
		os.Exit(1)
	}

	metricsService := service.NewMetricsService(collectorSvc, historyStore, logger, metrics, service.AlertThresholds{
		CPU:    cfg.CPUAlertThreshold,
		Memory: cfg.MemoryAlertThreshold,
		Disk:   cfg.DiskAlertThreshold,
	})
	handler := api.NewHandler(metricsService, logger, cfg.StreamInterval)
	server := api.NewServer(api.ServerConfig{
		Port:         cfg.HTTPPort,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}, logger, handler, metrics)

	go func() {
		if startErr := server.Start(); startErr != nil {
			logger.Error("server failed", "error", startErr)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		if errors.Is(err, context.DeadlineExceeded) {
			logger.Error("shutdown timed out")
		}
		os.Exit(1)
	}

	logger.Info("server exited cleanly", "timestamp", time.Now().UTC())
}
