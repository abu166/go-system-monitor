package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/abukhassymkhydyrbayev/go-system-monitor/backend/internal/model"
)

type HistoryStore struct {
	mu       sync.RWMutex
	limit    int
	records  []model.MetricSnapshot
	filePath string
}

func NewHistoryStore(limit int, filePath string) (*HistoryStore, error) {
	store := &HistoryStore{limit: limit, records: make([]model.MetricSnapshot, 0, limit), filePath: filePath}

	if err := store.ensureDataFile(); err != nil {
		return nil, err
	}
	if err := store.loadFromFile(); err != nil {
		return nil, err
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

func (s *HistoryStore) loadFromFile() error {
	f, err := os.Open(s.filePath)
	if err != nil {
		return fmt.Errorf("open history file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var snapshot model.MetricSnapshot
		if err := json.Unmarshal(line, &snapshot); err != nil {
			continue
		}
		s.addInMemory(snapshot)
	}

	if err := scanner.Err(); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("scan history file: %w", err)
	}

	return nil
}

func (s *HistoryStore) Add(snapshot model.MetricSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.addInMemory(snapshot)

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

	return nil
}

func (s *HistoryStore) addInMemory(snapshot model.MetricSnapshot) {
	if len(s.records) >= s.limit {
		s.records = s.records[1:]
	}
	s.records = append(s.records, snapshot)
}

func (s *HistoryStore) List() []model.MetricSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]model.MetricSnapshot, len(s.records))
	copy(out, s.records)
	return out
}
