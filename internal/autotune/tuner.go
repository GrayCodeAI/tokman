// Package autotune provides auto-tuning capabilities
package autotune

import (
	"math"
	"math/rand"
	"sync"
)

// Tuner provides auto-tuning for compression parameters
type Tuner struct {
	configs []TuningConfig
	mu      sync.RWMutex
}

// TuningConfig represents a tuning configuration
type TuningConfig struct {
	Name       string
	Parameters map[string]interface{}
	Score      float64
	EvalCount  int
}

// NewTuner creates a new tuner
func NewTuner() *Tuner {
	return &Tuner{
		configs: make([]TuningConfig, 0),
	}
}

// AddConfig adds a tuning configuration
func (t *Tuner) AddConfig(config TuningConfig) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.configs = append(t.configs, config)
}

// GetBestConfig returns the best configuration
func (t *Tuner) GetBestConfig() *TuningConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.configs) == 0 {
		return nil
	}

	best := &t.configs[0]
	for i := 1; i < len(t.configs); i++ {
		if t.configs[i].Score > best.Score {
			best = &t.configs[i]
		}
	}

	return best
}

// UpdateScore updates a config's score
func (t *Tuner) UpdateScore(name string, score float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := range t.configs {
		if t.configs[i].Name == name {
			// Exponential moving average
			alpha := 0.3
			t.configs[i].Score = alpha*score + (1-alpha)*t.configs[i].Score
			t.configs[i].EvalCount++
			return
		}
	}
}

// GenerateConfigs generates new configurations to test
func (t *Tuner) GenerateConfigs(count int) []TuningConfig {
	configs := make([]TuningConfig, 0, count)

	for i := 0; i < count; i++ {
		config := TuningConfig{
			Name:       "auto-tuned-" + string(rune('A'+i)),
			Parameters: make(map[string]interface{}),
		}

		// Randomize parameters
		config.Parameters["compression_level"] = 0.5 + rand.Float64()*0.5
		config.Parameters["min_token_length"] = 2 + int(rand.Float64()*8)
		config.Parameters["preserve_keywords"] = rand.Float64() > 0.5

		configs = append(configs, config)
	}

	return configs
}

// ContentClassifier classifies content for optimal processing
type ContentClassifier struct {
	rules []ClassificationRule
}

// ClassificationRule represents a classification rule
type ClassificationRule struct {
	Pattern     string
	ContentType string
	Priority    int
}

// NewContentClassifier creates a content classifier
func NewContentClassifier() *ContentClassifier {
	return &ContentClassifier{
		rules: []ClassificationRule{
			{Pattern: "func |class |import |package ", ContentType: "code", Priority: 1},
			{Pattern: "INFO|ERROR|WARN|DEBUG", ContentType: "log", Priority: 2},
			{Pattern: "<html|<div|<span", ContentType: "html", Priority: 3},
			{Pattern: "^\\{|^\\[", ContentType: "json", Priority: 4},
		},
	}
}

// Classify classifies content
func (cc *ContentClassifier) Classify(content string) string {
	for _, rule := range cc.rules {
		if matches(content, rule.Pattern) {
			return rule.ContentType
		}
	}
	return "text"
}

func matches(content, pattern string) bool {
	// Simple pattern matching
	for i := 0; i <= len(content)-len(pattern); i++ {
		if content[i:i+len(pattern)] == pattern[:1] {
			return true
		}
	}
	return false
}
