package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/telemetry"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Flush() {
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

type requestIDContextKey string

const requestIDKey requestIDContextKey = "request_id"

type CORSConfig struct {
	AllowedOrigins []string
}

type PathSampler struct {
	everyN int
	mu     sync.Mutex
	counts map[string]int
}

func NewPathSampler(everyN int) *PathSampler {
	if everyN <= 0 {
		everyN = 1
	}
	return &PathSampler{
		everyN: everyN,
		counts: make(map[string]int),
	}
}

func (s *PathSampler) ShouldLog(path string, status int) bool {
	if status >= http.StatusInternalServerError {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counts[path]++
	return (s.counts[path]-1)%s.everyN == 0
}

func RequestIDFromContext(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = generateRequestID()
		}

		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), requestIDKey, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func generateRequestID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("rid-%d", time.Now().UnixNano())
	}
	return "rid-" + hex.EncodeToString(buf)
}

func LoggingMiddleware(logger *slog.Logger, metrics *telemetry.Metrics, sampler *PathSampler, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)

		if metrics != nil {
			metrics.APIRequests.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", sw.status)).Inc()
			metrics.APIRequestDuration.WithLabelValues(r.Method, r.URL.Path, fmt.Sprintf("%d", sw.status)).Observe(time.Since(started).Seconds())
		}
		if sampler != nil && !sampler.ShouldLog(r.URL.Path, sw.status) {
			return
		}
		logger.Info("http request",
			"request_id", RequestIDFromContext(r.Context()),
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
			"status", sw.status,
			"duration_ms", time.Since(started).Milliseconds(),
		)
	})
}

func RecoverMiddleware(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered", "panic", rec)
				writeError(w, http.StatusInternalServerError, "internal server error")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers := w.Header()
		headers.Set("X-Content-Type-Options", "nosniff")
		headers.Set("X-Frame-Options", "DENY")
		headers.Set("Referrer-Policy", "no-referrer")
		headers.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'; base-uri 'none'")

		next.ServeHTTP(w, r)
	})
}

func CORSMiddleware(cfg CORSConfig, next http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			allowed[trimmed] = struct{}{}
		}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if _, ok := allowed["*"]; ok {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			} else if _, ok := allowed[origin]; ok {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
