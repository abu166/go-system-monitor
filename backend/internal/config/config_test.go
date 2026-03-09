package config

import (
	"strings"
	"testing"
)

func TestLoadRejectsInvalidThreshold(t *testing.T) {
	t.Setenv("BACKEND_CPU_ALERT_THRESHOLD", "101")

	_, err := Load()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "BACKEND_CPU_ALERT_THRESHOLD") {
		t.Fatalf("expected threshold error, got: %v", err)
	}
}

func TestLoadRejectsInvalidServerLimits(t *testing.T) {
	t.Setenv("BACKEND_MAX_HEADER_BYTES", "0")

	_, err := Load()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "BACKEND_MAX_HEADER_BYTES") {
		t.Fatalf("expected max header bytes error, got: %v", err)
	}
}

func TestLoadRejectsEmptyCORSOrigins(t *testing.T) {
	t.Setenv("BACKEND_CORS_ALLOWED_ORIGINS", " , ")

	_, err := Load()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "BACKEND_CORS_ALLOWED_ORIGINS") {
		t.Fatalf("expected CORS validation error, got: %v", err)
	}
}
