package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort               string
	ReadTimeout            time.Duration
	ReadHeaderTimeout      time.Duration
	WriteTimeout           time.Duration
	IdleTimeout            time.Duration
	MaxHeaderBytes         int
	ShutdownTimeout        time.Duration
	HistoryLimit           int
	HistoryMaxAge          time.Duration
	HistoryMaxFileSizeBytes int64
	HistoryInMemoryOnly    bool
	HistoryFallbackToMemory bool
	DiskPath               string
	StreamInterval         time.Duration
	PersistentHistoryPath  string
	CORSAllowedOrigins     []string
	LogSampleRate          int
	CPUAlertThreshold      float64
	MemoryAlertThreshold   float64
	DiskAlertThreshold     float64
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:              getEnv("BACKEND_PORT", "8080"),
		ReadTimeout:           mustDuration("BACKEND_READ_TIMEOUT", 5*time.Second),
		ReadHeaderTimeout:     mustDuration("BACKEND_READ_HEADER_TIMEOUT", 2*time.Second),
		WriteTimeout:          mustDuration("BACKEND_WRITE_TIMEOUT", 5*time.Second),
		IdleTimeout:           mustDuration("BACKEND_IDLE_TIMEOUT", 30*time.Second),
		MaxHeaderBytes:        mustInt("BACKEND_MAX_HEADER_BYTES", 1<<20),
		ShutdownTimeout:       mustDuration("BACKEND_SHUTDOWN_TIMEOUT", 10*time.Second),
		HistoryLimit:          mustInt("BACKEND_HISTORY_LIMIT", 100),
		HistoryMaxAge:         mustDuration("BACKEND_HISTORY_MAX_AGE", 24*time.Hour),
		HistoryMaxFileSizeBytes: mustInt64("BACKEND_HISTORY_MAX_FILE_SIZE_BYTES", 10<<20),
		HistoryInMemoryOnly:   mustBool("BACKEND_HISTORY_IN_MEMORY_ONLY", false),
		HistoryFallbackToMemory: mustBool("BACKEND_HISTORY_FALLBACK_TO_MEMORY", true),
		DiskPath:              getEnv("BACKEND_DISK_PATH", "/"),
		StreamInterval:        mustDuration("BACKEND_STREAM_INTERVAL", 3*time.Second),
		PersistentHistoryPath: getEnv("BACKEND_HISTORY_FILE", "data/metrics-history.jsonl"),
		CORSAllowedOrigins:    mustCSV("BACKEND_CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost"),
		LogSampleRate:         mustInt("BACKEND_LOG_SAMPLE_RATE", 5),
		CPUAlertThreshold:     mustFloat("BACKEND_CPU_ALERT_THRESHOLD", 85),
		MemoryAlertThreshold:  mustFloat("BACKEND_MEMORY_ALERT_THRESHOLD", 85),
		DiskAlertThreshold:    mustFloat("BACKEND_DISK_ALERT_THRESHOLD", 90),
	}

	if cfg.HTTPPort == "" {
		return nil, errors.New("BACKEND_PORT must not be empty")
	}
	if cfg.HistoryLimit <= 0 {
		return nil, fmt.Errorf("BACKEND_HISTORY_LIMIT must be > 0, got %d", cfg.HistoryLimit)
	}
	if cfg.HistoryMaxAge <= 0 {
		return nil, fmt.Errorf("BACKEND_HISTORY_MAX_AGE must be > 0, got %s", cfg.HistoryMaxAge)
	}
	if cfg.HistoryMaxFileSizeBytes <= 0 {
		return nil, fmt.Errorf("BACKEND_HISTORY_MAX_FILE_SIZE_BYTES must be > 0, got %d", cfg.HistoryMaxFileSizeBytes)
	}
	if cfg.ReadTimeout <= 0 {
		return nil, fmt.Errorf("BACKEND_READ_TIMEOUT must be > 0, got %s", cfg.ReadTimeout)
	}
	if cfg.ReadHeaderTimeout <= 0 {
		return nil, fmt.Errorf("BACKEND_READ_HEADER_TIMEOUT must be > 0, got %s", cfg.ReadHeaderTimeout)
	}
	if cfg.WriteTimeout <= 0 {
		return nil, fmt.Errorf("BACKEND_WRITE_TIMEOUT must be > 0, got %s", cfg.WriteTimeout)
	}
	if cfg.IdleTimeout <= 0 {
		return nil, fmt.Errorf("BACKEND_IDLE_TIMEOUT must be > 0, got %s", cfg.IdleTimeout)
	}
	if cfg.MaxHeaderBytes <= 0 {
		return nil, fmt.Errorf("BACKEND_MAX_HEADER_BYTES must be > 0, got %d", cfg.MaxHeaderBytes)
	}
	if cfg.DiskPath == "" {
		return nil, errors.New("BACKEND_DISK_PATH must not be empty")
	}
	if cfg.PersistentHistoryPath == "" {
		return nil, errors.New("BACKEND_HISTORY_FILE must not be empty")
	}
	if cfg.StreamInterval <= 0 {
		return nil, fmt.Errorf("BACKEND_STREAM_INTERVAL must be > 0, got %s", cfg.StreamInterval)
	}
	if cfg.LogSampleRate <= 0 {
		return nil, fmt.Errorf("BACKEND_LOG_SAMPLE_RATE must be > 0, got %d", cfg.LogSampleRate)
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		return nil, errors.New("BACKEND_CORS_ALLOWED_ORIGINS must not be empty")
	}
	if err := validateThreshold("BACKEND_CPU_ALERT_THRESHOLD", cfg.CPUAlertThreshold); err != nil {
		return nil, err
	}
	if err := validateThreshold("BACKEND_MEMORY_ALERT_THRESHOLD", cfg.MemoryAlertThreshold); err != nil {
		return nil, err
	}
	if err := validateThreshold("BACKEND_DISK_ALERT_THRESHOLD", cfg.DiskAlertThreshold); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validateThreshold(name string, value float64) error {
	if value < 0 || value > 100 {
		return fmt.Errorf("%s must be between 0 and 100, got %.2f", name, value)
	}
	return nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func mustDuration(key string, fallback time.Duration) time.Duration {
	value := getEnv(key, fallback.String())
	duration, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return duration
}

func mustInt(key string, fallback int) int {
	value := getEnv(key, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func mustFloat(key string, fallback float64) float64 {
	value := getEnv(key, strconv.FormatFloat(fallback, 'f', -1, 64))
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func mustInt64(key string, fallback int64) int64 {
	value := getEnv(key, strconv.FormatInt(fallback, 10))
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func mustBool(key string, fallback bool) bool {
	value := getEnv(key, strconv.FormatBool(fallback))
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func mustCSV(key, fallback string) []string {
	value := getEnv(key, fallback)
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}
