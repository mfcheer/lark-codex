package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type State struct {
	Cursor       string           `json:"cursor"`
	LastPollUnix int64            `json:"last_poll_unix"`
	Processed    map[string]int64 `json:"processed"`
}

type JSONStore struct {
	path string
	mu   sync.Mutex
}

func NewJSONStore(path string) *JSONStore {
	return &JSONStore{path: path}
}

func (s *JSONStore) Load() (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := State{Processed: map[string]int64{}}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return State{}, fmt.Errorf("read state: %w", err)
	}
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("parse state: %w", err)
	}
	if state.Processed == nil {
		state.Processed = map[string]int64{}
	}
	return state, nil
}

func (s *JSONStore) Save(state State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if state.Processed == nil {
		state.Processed = map[string]int64{}
	}
	prune(state.Processed, time.Now().Add(-7*24*time.Hour).Unix())
	data, _ := json.MarshalIndent(state, "", "  ")
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func prune(m map[string]int64, threshold int64) {
	for k, ts := range m {
		if ts < threshold {
			delete(m, k)
		}
	}
}
