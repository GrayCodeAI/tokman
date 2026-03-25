package filter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CrossSessionLearner persists learned compression patterns across sessions.
// Unlike the in-memory AutoTuner, this saves weights to disk and loads them
// on startup, enabling cumulative learning over time.
//
// Storage: ~/.local/share/tokman/learned_weights.json
type CrossSessionLearner struct {
	config   LearnerConfig
	data     *LearnedData
	mu       sync.RWMutex
	dataPath string
}

// LearnerConfig holds configuration for cross-session learning
type LearnerConfig struct {
	Enabled      bool
	DataDir      string
	SaveInterval time.Duration
	MaxRecords   int
}

// LearnedData persists across sessions
type LearnedData struct {
	Version      int                           `json:"version"`
	Weights      map[string]map[string]float64 `json:"weights"` // contentType -> layer -> weight
	History      []LearnerRecord               `json:"history"`
	TotalRuns    int64                         `json:"total_runs"`
	TotalSaved   int64                         `json:"total_saved"`
	LastUpdated  time.Time                     `json:"last_updated"`
	ContentTypes map[string]int64              `json:"content_types"` // contentType -> count
}

// LearnerRecord captures a compression result
type LearnerRecord struct {
	ContentType string         `json:"content_type"`
	Saved       int            `json:"saved"`
	LatencyMs   float64        `json:"latency_ms"`
	LayerStats  map[string]int `json:"layer_stats"`
	Timestamp   time.Time      `json:"timestamp"`
}

// DefaultLearnerConfig returns default configuration
func DefaultLearnerConfig() LearnerConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}
	return LearnerConfig{
		Enabled:      true,
		DataDir:      filepath.Join(homeDir, ".local", "share", "tokman"),
		SaveInterval: 5 * time.Minute,
		MaxRecords:   5000,
	}
}

// NewCrossSessionLearner creates a new cross-session learner
func NewCrossSessionLearner() *CrossSessionLearner {
	return NewCrossSessionLearnerWithConfig(DefaultLearnerConfig())
}

// NewCrossSessionLearnerWithConfig creates a learner with custom config
func NewCrossSessionLearnerWithConfig(cfg LearnerConfig) *CrossSessionLearner {
	l := &CrossSessionLearner{
		config:   cfg,
		dataPath: filepath.Join(cfg.DataDir, "learned_weights.json"),
		data: &LearnedData{
			Version:      1,
			Weights:      make(map[string]map[string]float64),
			History:      make([]LearnerRecord, 0, cfg.MaxRecords),
			ContentTypes: make(map[string]int64),
			LastUpdated:  time.Now(),
		},
	}

	if cfg.Enabled {
		l.load()
	}

	return l
}

// Record captures a compression result for cross-session learning
func (l *CrossSessionLearner) Record(contentType string, layerStats map[string]int, latencyMs float64) {
	if !l.config.Enabled {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	totalSaved := 0
	for _, saved := range layerStats {
		totalSaved += saved
	}

	record := LearnerRecord{
		ContentType: contentType,
		Saved:       totalSaved,
		LatencyMs:   latencyMs,
		LayerStats:  layerStats,
		Timestamp:   time.Now(),
	}

	l.data.History = append(l.data.History, record)
	if len(l.data.History) > l.config.MaxRecords {
		l.data.History = l.data.History[len(l.data.History)-l.config.MaxRecords:]
	}

	l.data.TotalRuns++
	l.data.TotalSaved += int64(totalSaved)
	l.data.ContentTypes[contentType]++
	l.data.LastUpdated = time.Now()

	// Update weights
	l.updateWeights(contentType, layerStats, totalSaved)
}

// updateWeights adjusts layer weights based on contribution
func (l *CrossSessionLearner) updateWeights(contentType string, layerStats map[string]int, totalSaved int) {
	if l.data.Weights[contentType] == nil {
		l.data.Weights[contentType] = make(map[string]float64)
	}

	if totalSaved == 0 {
		return
	}

	lr := 0.05 // Conservative learning rate for cross-session
	for layer, saved := range layerStats {
		contribution := float64(saved) / float64(totalSaved)
		current := l.data.Weights[contentType][layer]
		if current == 0 {
			current = 1.0
		}
		l.data.Weights[contentType][layer] = current*(1-lr) + contribution*lr
	}
}

// GetOptimalLayers returns the best layers for a content type based on all history
func (l *CrossSessionLearner) GetOptimalLayers(contentType string) map[string]float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if w, ok := l.data.Weights[contentType]; ok {
		result := make(map[string]float64, len(w))
		for k, v := range w {
			result[k] = v
		}
		return result
	}
	return nil
}

// GetStats returns learning statistics
func (l *CrossSessionLearner) GetStats() LearnerStats {
	l.mu.RLock()
	defer l.mu.RUnlock()

	return LearnerStats{
		TotalRuns:    l.data.TotalRuns,
		TotalSaved:   l.data.TotalSaved,
		ContentTypes: l.data.ContentTypes,
		LastUpdated:  l.data.LastUpdated,
		HistorySize:  len(l.data.History),
	}
}

// LearnerStats holds learning statistics
type LearnerStats struct {
	TotalRuns    int64            `json:"total_runs"`
	TotalSaved   int64            `json:"total_saved"`
	ContentTypes map[string]int64 `json:"content_types"`
	LastUpdated  time.Time        `json:"last_updated"`
	HistorySize  int              `json:"history_size"`
}

// Save persists learned data to disk
func (l *CrossSessionLearner) Save() error {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.dataPath == "" {
		return nil
	}

	os.MkdirAll(filepath.Dir(l.dataPath), 0755)

	data, err := json.MarshalIndent(l.data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(l.dataPath, data, 0644)
}

// load reads persisted data from disk
func (l *CrossSessionLearner) load() {
	if l.dataPath == "" {
		return
	}

	data, err := os.ReadFile(l.dataPath)
	if err != nil {
		return
	}

	var loaded LearnedData
	if err := json.Unmarshal(data, &loaded); err != nil {
		return
	}

	if loaded.Version == l.data.Version {
		l.data = &loaded
	}
}

// Reset clears all learned data
func (l *CrossSessionLearner) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.data = &LearnedData{
		Version:      1,
		Weights:      make(map[string]map[string]float64),
		History:      make([]LearnerRecord, 0, l.config.MaxRecords),
		ContentTypes: make(map[string]int64),
		LastUpdated:  time.Now(),
	}
}
