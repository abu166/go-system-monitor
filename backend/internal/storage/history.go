package storage

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/model"
)

type Config struct {
	Limit            int
	FilePath         string
	MaxAge           time.Duration
	MaxFileSizeBytes int64
	InMemoryOnly     bool
}

type HistoryStore struct {
	mu               sync.RWMutex
	limit            int
	records          []model.MetricSnapshot
	filePath         string
	maxAge           time.Duration
	maxFileSizeBytes int64
	persist          bool
}

func NewHistoryStore(limit int, filePath string) (*HistoryStore, error) {
	return NewHistoryStoreWithConfig(Config{
		Limit:            limit,
		FilePath:         filePath,
		MaxAge:           24 * time.Hour,
		MaxFileSizeBytes: 10 << 20,
	})
}

func NewHistoryStoreWithConfig(cfg Config) (*HistoryStore, error) {
	if cfg.Limit <= 0 {
		return nil, fmt.Errorf("history limit must be > 0, got %d", cfg.Limit)
	}
	if cfg.MaxAge <= 0 {
		cfg.MaxAge = 24 * time.Hour
	}
	if cfg.MaxFileSizeBytes <= 0 {
		cfg.MaxFileSizeBytes = 10 << 20
	}

	store := &HistoryStore{
		limit:            cfg.Limit,
		records:          make([]model.MetricSnapshot, 0, cfg.Limit),
		filePath:         cfg.FilePath,
		maxAge:           cfg.MaxAge,
		maxFileSizeBytes: cfg.MaxFileSizeBytes,
		persist:          !cfg.InMemoryOnly && cfg.FilePath != "",
	}

	if !store.persist {
		return store, nil
	}

	if err := store.ensureDataFile(); err != nil {
		return nil, err
	}
	needsCompact, err := store.loadFromFile()
	if err != nil {
		return nil, err
	}
	if needsCompact {
		if err := store.compactLocked(); err != nil {
			return nil, err
		}
	}

	return store, nil
}

func (s *HistoryStore) ensureDataFile() error {
	dir := filepath.Dir(s.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create history directory: %w", err)
	}

	f, err := os.OpenFile(s.filePath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("create history file: %w", err)
	}
	_ = f.Close()
	return nil
}

func (s *HistoryStore) loadFromFile() (bool, error) {
	f, err := os.Open(s.filePath)
	if err != nil {
		return false, fmt.Errorf("open history file: %w", err)
	}
	defer f.Close()

	needsCompact := false
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var snapshot model.MetricSnapshot
		if err := json.Unmarshal(line, &snapshot); err != nil {
			needsCompact = true
			continue
		}
		s.addInMemoryLocked(snapshot)
	}

	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("scan history file: %w", err)
	}

	if s.applyRetentionLocked(time.Now().UTC()) {
		needsCompact = true
	}
	return needsCompact, nil
}

func (s *HistoryStore) Add(snapshot model.MetricSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.addInMemoryLocked(snapshot)
	retentionTrimmed := s.applyRetentionLocked(time.Now().UTC())
	if !s.persist {
		return nil
	}

	payload, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("encode snapshot: %w", err)
	}
	payload = append(payload, '\n')

	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open history file for append: %w", err)
	}
	defer f.Close()

	if _, err := f.Write(payload); err != nil {
		return fmt.Errorf("append snapshot to history file: %w", err)
	}

	if retentionTrimmed {
		if err := s.compactLocked(); err != nil {
			return err
		}
		return nil
	}

	stat, err := f.Stat()
	if err == nil && stat.Size() > s.maxFileSizeBytes {
		if err := s.compactLocked(); err != nil {
			return err
		}
	}

	return nil
}

func (s *HistoryStore) addInMemoryLocked(snapshot model.MetricSnapshot) {
	if len(s.records) >= s.limit {
		s.records = s.records[1:]
	}
	s.records = append(s.records, snapshot)
}

func (s *HistoryStore) applyRetentionLocked(now time.Time) bool {
	trimmed := false
	if s.maxAge > 0 {
		cutoff := now.Add(-s.maxAge)
		kept := s.records[:0]
		for _, rec := range s.records {
			if rec.CollectedAt.IsZero() || rec.CollectedAt.After(cutoff) || rec.CollectedAt.Equal(cutoff) {
				kept = append(kept, rec)
			} else {
				trimmed = true
			}
		}
		s.records = kept
	}
	for len(s.records) > s.limit {
		s.records = s.records[1:]
		trimmed = true
	}
	return trimmed
}

func (s *HistoryStore) compactLocked() error {
	if !s.persist {
		return nil
	}

	dir := filepath.Dir(s.filePath)
	tmp, err := os.CreateTemp(dir, "history-*.jsonl")
	if err != nil {
		return fmt.Errorf("create compact temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
	}()

	writer := bufio.NewWriter(tmp)
	for _, rec := range s.records {
		payload, err := json.Marshal(rec)
		if err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("encode snapshot during compact: %w", err)
		}
		if _, err := writer.Write(append(payload, '\n')); err != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("write snapshot during compact: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("flush compact file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close compact file: %w", err)
	}
	if err := os.Rename(tmpPath, s.filePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace history file with compacted file: %w", err)
	}
	return nil
}

func (s *HistoryStore) List() []model.MetricSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]model.MetricSnapshot, len(s.records))
	copy(out, s.records)
	return out
}

func (s *HistoryStore) Latest() (model.MetricSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.records) == 0 {
		return model.MetricSnapshot{}, false
	}
	return s.records[len(s.records)-1], true
}
