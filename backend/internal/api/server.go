package api

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/telemetry"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

type ServerConfig struct {
	Port              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
	CORSAllowedOrigins []string
	LogSampleRate      int
}

func NewServer(cfg ServerConfig, logger *slog.Logger, handler *Handler, metrics *telemetry.Metrics) *Server {
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	mux.Handle("/metrics", promhttp.Handler())

	sampler := NewPathSampler(cfg.LogSampleRate)
	chain := RecoverMiddleware(
		logger,
		RequestIDMiddleware(
			SecurityHeadersMiddleware(
				CORSMiddleware(CORSConfig{AllowedOrigins: cfg.CORSAllowedOrigins},
					LoggingMiddleware(logger, metrics, sampler, mux),
				),
			),
		),
	)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           chain,
		ReadTimeout:       cfg.ReadTimeout,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    cfg.MaxHeaderBytes,
	}

	return &Server{httpServer: httpServer, logger: logger}
}

func (s *Server) Start() error {
	s.logger.Info("starting HTTP server", "addr", s.httpServer.Addr)
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down HTTP server")
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}
	return nil
}
