package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/service"
)

type Handler struct {
	service        service.MetricsService
	logger         *slog.Logger
	streamInterval time.Duration
}

func NewHandler(svc service.MetricsService, logger *slog.Logger, streamInterval time.Duration) *Handler {
	return &Handler{service: svc, logger: logger, streamInterval: streamInterval}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("GET /live", h.live)
	mux.HandleFunc("GET /ready", h.ready)
	mux.HandleFunc("GET /api/metrics/latest", h.latestMetrics)
	mux.HandleFunc("GET /api/metrics/history", h.metricsHistory)
	mux.HandleFunc("GET /api/metrics/stream", h.streamMetrics)
	mux.HandleFunc("GET /api/system/info", h.systemInfo)
	mux.HandleFunc("GET /api/alerts/current", h.currentAlerts)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) live(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, http.StatusOK, map[string]string{"status": "live"})
}

func (h *Handler) ready(w http.ResponseWriter, r *http.Request) {
	if _, err := h.service.GetSystemInfo(r.Context()); err != nil {
		h.logger.Error("readiness check failed", "error", err)
		writeError(w, http.StatusServiceUnavailable, "not ready")
		return
	}
	writeSuccess(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (h *Handler) latestMetrics(w http.ResponseWriter, r *http.Request) {
	event, err := h.service.GetLatestWithAlerts(r.Context())
	if err != nil {
		h.logger.Error("latestMetrics failed", "error", err)
		writeError(w, http.StatusServiceUnavailable, "failed to read latest metrics")
		return
	}

	writeSuccess(w, http.StatusOK, event.Snapshot)
}

func (h *Handler) metricsHistory(w http.ResponseWriter, r *http.Request) {
	history, err := h.service.GetHistory(r.Context())
	if err != nil {
		h.logger.Error("metricsHistory failed", "error", err)
		if errors.Is(err, r.Context().Err()) {
			writeError(w, http.StatusRequestTimeout, "request cancelled")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to read metrics history")
		return
	}

	writeSuccess(w, http.StatusOK, history)
}

func (h *Handler) streamMetrics(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported")
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(h.streamInterval)
	defer ticker.Stop()

	sendSnapshot := func(ctx context.Context) error {
		event, err := h.service.GetLatestWithAlerts(ctx)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(event)
		if err != nil {
			return fmt.Errorf("encode stream payload: %w", err)
		}
		if _, err := fmt.Fprintf(w, "event: metrics\ndata: %s\n\n", payload); err != nil {
			return fmt.Errorf("write stream payload: %w", err)
		}
		flusher.Flush()
		return nil
	}

	if err := sendSnapshot(r.Context()); err != nil {
		h.logger.Error("failed initial stream send", "error", err)
		return
	}

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if err := sendSnapshot(r.Context()); err != nil {
				h.logger.Error("failed stream send", "error", err)
				return
			}
		}
	}
}

func (h *Handler) systemInfo(w http.ResponseWriter, r *http.Request) {
	info, err := h.service.GetSystemInfo(r.Context())
	if err != nil {
		h.logger.Error("systemInfo failed", "error", err)
		writeError(w, http.StatusServiceUnavailable, "failed to read system information")
		return
	}

	writeSuccess(w, http.StatusOK, info)
}

func (h *Handler) currentAlerts(w http.ResponseWriter, r *http.Request) {
	alerts, err := h.service.GetCurrentAlerts(r.Context())
	if err != nil {
		h.logger.Error("currentAlerts failed", "error", err)
		writeError(w, http.StatusServiceUnavailable, "failed to evaluate alerts")
		return
	}

	writeSuccess(w, http.StatusOK, alerts)
}
