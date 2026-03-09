package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/model"
)

func TestHistoryStoreLoadsValidLinesFromCorruptedFile(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "history.jsonl")

	validOld := model.MetricSnapshot{CPUUsagePercent: 10, CollectedAt: time.Now().UTC().Add(-2 * time.Hour)}
	validNew := model.MetricSnapshot{CPUUsagePercent: 20, CollectedAt: time.Now().UTC()}
	oldRaw, _ := json.Marshal(validOld)
	newRaw, _ := json.Marshal(validNew)
	content := string(oldRaw) + "\n" + "not-json\n" + string(newRaw) + "\n"
	if err := os.WriteFile(filePath, []byte(content), 0o644); err != nil {
		t.Fatalf("seed corrupted file: %v", err)
	}

	store, err := NewHistoryStoreWithConfig(Config{
		Limit:            100,
		FilePath:         filePath,
		MaxAge:           24 * time.Hour,
		MaxFileSizeBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	got := store.List()
	if len(got) != 2 {
		t.Fatalf("expected 2 valid snapshots loaded, got %d", len(got))
	}
}

func TestHistoryStoreRetentionByAgeAndCount(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "history.jsonl")
	store, err := NewHistoryStoreWithConfig(Config{
		Limit:            2,
		FilePath:         filePath,
		MaxAge:           30 * time.Minute,
		MaxFileSizeBytes: 1 << 20,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	now := time.Now().UTC()
	if err := store.Add(model.MetricSnapshot{CPUUsagePercent: 10, CollectedAt: now.Add(-2 * time.Hour)}); err != nil {
		t.Fatalf("add old snapshot: %v", err)
	}
	if err := store.Add(model.MetricSnapshot{CPUUsagePercent: 20, CollectedAt: now.Add(-10 * time.Minute)}); err != nil {
		t.Fatalf("add new snapshot: %v", err)
	}
	if err := store.Add(model.MetricSnapshot{CPUUsagePercent: 30, CollectedAt: now}); err != nil {
		t.Fatalf("add latest snapshot: %v", err)
	}

	got := store.List()
	if len(got) != 2 {
		t.Fatalf("expected 2 snapshots after retention, got %d", len(got))
	}
	if got[0].CPUUsagePercent != 20 || got[1].CPUUsagePercent != 30 {
		t.Fatalf("unexpected retained snapshots: %+v", got)
	}
}

func TestHistoryStoreCompactsWhenMaxFileSizeExceeded(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "history.jsonl")
	store, err := NewHistoryStoreWithConfig(Config{
		Limit:            3,
		FilePath:         filePath,
		MaxAge:           24 * time.Hour,
		MaxFileSizeBytes: 600,
	})
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	now := time.Now().UTC()
	for i := 0; i < 6; i++ {
		if err := store.Add(model.MetricSnapshot{
			CPUUsagePercent: float64(i),
			CollectedAt:     now.Add(time.Duration(i) * time.Second),
		}); err != nil {
			t.Fatalf("add snapshot %d: %v", i, err)
		}
	}

	stat, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("stat history file: %v", err)
	}
	if stat.Size() <= 0 {
		t.Fatalf("expected compacted file to be non-empty")
	}

	raw, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read history file: %v", err)
	}
	lineCount := 0
	for _, b := range raw {
		if b == '\n' {
			lineCount++
		}
	}
	if lineCount > 3 {
		t.Fatalf("expected compacted file to keep at most 3 records, got %d lines", lineCount)
	}

	got := store.List()
	if len(got) != 3 {
		t.Fatalf("expected in-memory limit 3, got %d", len(got))
	}
}

func TestHistoryStoreInMemoryOnlyMode(t *testing.T) {
	t.Parallel()

	store, err := NewHistoryStoreWithConfig(Config{
		Limit:        10,
		InMemoryOnly: true,
		MaxAge:       24 * time.Hour,
	})
	if err != nil {
		t.Fatalf("new in-memory store: %v", err)
	}

	if err := store.Add(model.MetricSnapshot{CPUUsagePercent: 50, CollectedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("add snapshot: %v", err)
	}
	if len(store.List()) != 1 {
		t.Fatalf("expected one in-memory snapshot")
	}
}
