package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPPort               string
	ReadTimeout            time.Duration
	WriteTimeout           time.Duration
	ShutdownTimeout        time.Duration
	HistoryLimit           int
	DiskPath               string
	StreamInterval         time.Duration
	PersistentHistoryPath  string
	CPUAlertThreshold      float64
	MemoryAlertThreshold   float64
	DiskAlertThreshold     float64
}

func Load() (*Config, error) {
	cfg := &Config{
		HTTPPort:              getEnv("BACKEND_PORT", "8080"),
		ReadTimeout:           mustDuration("BACKEND_READ_TIMEOUT", 5*time.Second),
		WriteTimeout:          mustDuration("BACKEND_WRITE_TIMEOUT", 5*time.Second),
		ShutdownTimeout:       mustDuration("BACKEND_SHUTDOWN_TIMEOUT", 10*time.Second),
		HistoryLimit:          mustInt("BACKEND_HISTORY_LIMIT", 100),
		DiskPath:              getEnv("BACKEND_DISK_PATH", "/"),
		StreamInterval:        mustDuration("BACKEND_STREAM_INTERVAL", 3*time.Second),
		PersistentHistoryPath: getEnv("BACKEND_HISTORY_FILE", "data/metrics-history.jsonl"),
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
	if cfg.DiskPath == "" {
		return nil, errors.New("BACKEND_DISK_PATH must not be empty")
	}
	if cfg.PersistentHistoryPath == "" {
		return nil, errors.New("BACKEND_HISTORY_FILE must not be empty")
	}
	if cfg.StreamInterval <= 0 {
		return nil, fmt.Errorf("BACKEND_STREAM_INTERVAL must be > 0, got %s", cfg.StreamInterval)
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
