package filter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// PersistedPipelineState is a serializable snapshot of pipeline state
// that can survive process restarts.
// Task #185: Pipeline state persistence across restarts.
type PersistedPipelineState struct {
	Version          int       `json:"version"`
	SavedAt          time.Time `json:"saved_at"`
	SessionID        string    `json:"session_id"`
	TotalInputTokens int       `json:"total_input_tokens"`
	TotalSaved       int       `json:"total_saved"`
	FilterStats      map[string]int `json:"filter_stats"` // filterName → totalSaved
	LastMode         string    `json:"last_mode"`
	LastContentType  string    `json:"last_content_type"`
	RunCount         int       `json:"run_count"`
}

// PipelineStateStore persists and restores pipeline state across process restarts.
type PipelineStateStore struct {
	mu      sync.Mutex
	path    string
	current *PersistedPipelineState
}

const pipelineStateVersion = 1

// NewPipelineStateStore creates a store backed by the given file path.
func NewPipelineStateStore(path string) *PipelineStateStore {
	return &PipelineStateStore{path: path}
}

// DefaultPipelineStateStore returns a store at the default XDG path.
func DefaultPipelineStateStore() *PipelineStateStore {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".local", "share", "tokman", "pipeline_state.json")
	return NewPipelineStateStore(path)
}

// Load reads persisted state from disk. Returns empty state if file doesn't exist.
func (s *PipelineStateStore) Load() (*PersistedPipelineState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		s.current = &PersistedPipelineState{
			Version:     pipelineStateVersion,
			FilterStats: make(map[string]int),
		}
		return s.current, nil
	}
	if err != nil {
		return nil, fmt.Errorf("pipeline state: read: %w", err)
	}

	var state PersistedPipelineState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("pipeline state: parse: %w", err)
	}

	if state.FilterStats == nil {
		state.FilterStats = make(map[string]int)
	}
	s.current = &state
	return s.current, nil
}

// Save persists current state to disk atomically.
func (s *PipelineStateStore) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		return nil
	}
	s.current.SavedAt = time.Now()

	data, err := json.MarshalIndent(s.current, "", "  ")
	if err != nil {
		return fmt.Errorf("pipeline state: marshal: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(s.path), 0700); err != nil {
		return fmt.Errorf("pipeline state: mkdir: %w", err)
	}

	// Atomic write via temp file + rename
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return fmt.Errorf("pipeline state: write tmp: %w", err)
	}
	return os.Rename(tmp, s.path)
}

// Record updates the in-memory state with a compression result.
func (s *PipelineStateStore) Record(filterName string, mode Mode, contentType string, inputTokens, saved int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil {
		s.current = &PersistedPipelineState{
			Version:     pipelineStateVersion,
			FilterStats: make(map[string]int),
		}
	}

	s.current.TotalInputTokens += inputTokens
	s.current.TotalSaved += saved
	s.current.FilterStats[filterName] += saved
	s.current.LastMode = string(mode)
	s.current.LastContentType = contentType
	s.current.RunCount++
}

// Summary returns a human-readable summary of the persisted state.
func (s *PipelineStateStore) Summary() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.current == nil || s.current.RunCount == 0 {
		return "No pipeline state persisted."
	}

	c := s.current
	var avgReduction float64
	if c.TotalInputTokens > 0 {
		avgReduction = float64(c.TotalSaved) / float64(c.TotalInputTokens) * 100
	}

	return fmt.Sprintf("Pipeline state: %d runs, %d tokens in, %d saved (%.1f%% avg reduction)\n",
		c.RunCount, c.TotalInputTokens, c.TotalSaved, avgReduction)
}

// EstimateTokens exposes core.EstimateTokens for use in pipeline state calculations.
var _ = core.EstimateTokens // ensure import is used
