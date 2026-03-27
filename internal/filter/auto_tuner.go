package filter

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// AutoTuner learns optimal layer combinations per content type.
// Replaces static presets with data-driven layer selection.
//
// Mechanism:
// 1. For each content type, track which layers produce the best compression
// 2. Weight layers by their historical contribution to total compression
// 3. Dynamically enable/disable layers based on learned weights
// 4. Persist learned weights to disk for cross-session learning
type AutoTuner struct {
	config   AutoTunerConfig
	weights  map[string]map[string]float64 // contentType -> layerName -> weight
	history  []TuningRecord
	mu       sync.RWMutex
	dataPath string
}

// AutoTunerConfig holds configuration for the auto-tuner
type AutoTunerConfig struct {
	// Enabled controls whether auto-tuning is active
	Enabled bool

	// LearningRate controls how fast weights adjust (0-1)
	LearningRate float64

	// MinSamples before applying learned weights
	MinSamples int

	// DataPath for persisting learned weights
	DataPath string

	// MaxHistory records to keep
	MaxHistory int
}

// TuningRecord captures a single compression result for learning
type TuningRecord struct {
	ContentType string         `json:"content_type"`
	InputSize   int            `json:"input_size"`
	OutputSize  int            `json:"output_size"`
	Saved       int            `json:"saved"`
	LayerStats  map[string]int `json:"layer_stats"` // layer -> tokens saved
	Latency     int64          `json:"latency_us"`
}

// DefaultAutoTunerConfig returns default configuration
func DefaultAutoTunerConfig() AutoTunerConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}
	return AutoTunerConfig{
		Enabled:      true,
		LearningRate: 0.1,
		MinSamples:   5,
		DataPath:     filepath.Join(homeDir, ".config", "tokman", "autotuner.json"),
		MaxHistory:   1000,
	}
}

// NewAutoTuner creates a new auto-tuner
func NewAutoTuner() *AutoTuner {
	return NewAutoTunerWithConfig(DefaultAutoTunerConfig())
}

// NewAutoTunerWithConfig creates an auto-tuner with custom config
func NewAutoTunerWithConfig(cfg AutoTunerConfig) *AutoTuner {
	t := &AutoTuner{
		config:   cfg,
		weights:  make(map[string]map[string]float64),
		history:  make([]TuningRecord, 0, cfg.MaxHistory),
		dataPath: cfg.DataPath,
	}

	// Load persisted weights
	t.loadWeights()

	// Initialize default weights
	t.initDefaultWeights()

	return t
}

// initDefaultWeights sets initial uniform weights
func (t *AutoTuner) initDefaultWeights() {
	contentTypes := []string{"code", "logs", "text", "mixed", "git", "docker", "test"}
	layers := []string{
		"0_tfidf", "1_entropy", "2_perplexity", "3_goal_driven", "4_ast_preserve",
		"5_contrastive", "6_ngram", "7_evaluator", "8_gist", "9_hierarchical",
		"11_compaction", "12_attribution", "13_h2o", "14_attention_sink",
		"15_meta_token", "16_semantic_chunk", "17_sketch_store", "18_lazy_pruner",
		"19_semantic_anchor", "20_agent_memory",
		"21_reasoning_trace", "22_symbolic_compress", "23_phrase_grouping",
		"24_numerical_quant", "25_dynamic_ratio",
		"26_hypernym", "27_semantic_cache", "28_scope", "29_kvzip",
	}

	for _, ct := range contentTypes {
		if t.weights[ct] == nil {
			t.weights[ct] = make(map[string]float64)
		}
		// Start with uniform weights
		for _, layer := range layers {
			if _, exists := t.weights[ct][layer]; !exists {
				t.weights[ct][layer] = 1.0
			}
		}
	}
}

// Record captures a compression result for learning
func (t *AutoTuner) Record(contentType string, input string, output string, layerStats map[string]int, latencyUs int64) {
	if !t.config.Enabled {
		return
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	record := TuningRecord{
		ContentType: contentType,
		InputSize:   len(input),
		OutputSize:  len(output),
		Saved:       core.EstimateTokens(input) - core.EstimateTokens(output),
		LayerStats:  layerStats,
		Latency:     latencyUs,
	}

	t.history = append(t.history, record)
	if len(t.history) > t.config.MaxHistory {
		t.history = t.history[len(t.history)-t.config.MaxHistory:]
	}

	// Update weights based on this record
	t.updateWeights(contentType, layerStats)
}

// updateWeights adjusts layer weights based on their contribution
func (t *AutoTuner) updateWeights(contentType string, layerStats map[string]int) {
	if t.weights[contentType] == nil {
		t.weights[contentType] = make(map[string]float64)
	}

	totalSaved := 0
	for _, saved := range layerStats {
		totalSaved += saved
	}

	if totalSaved == 0 {
		return
	}

	// Update weights: layers that saved more get higher weight
	lr := t.config.LearningRate
	for layer, saved := range layerStats {
		contribution := float64(saved) / float64(totalSaved)
		currentWeight := t.weights[contentType][layer]
		if currentWeight == 0 {
			currentWeight = 1.0
		}

		// Exponential moving average
		newWeight := currentWeight*(1-lr) + contribution*lr
		t.weights[contentType][layer] = newWeight
	}
}

// GetOptimalConfig returns the best pipeline config for a content type
func (t *AutoTuner) GetOptimalConfig(contentType string, baseMode Mode) PipelineConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()

	cfg := PipelineConfig{
		Mode:            baseMode,
		SessionTracking: true,
	}

	// Check if we have enough samples
	sampleCount := t.countSamples(contentType)
	if sampleCount < t.config.MinSamples {
		// Not enough data - use full preset
		fullCfg := PresetConfig(PresetFull, baseMode)
		return fullCfg
	}

	weights := t.weights[contentType]
	if weights == nil {
		return PresetConfig(PresetFull, baseMode)
	}

	// Enable layers with weight > threshold
	threshold := 0.3
	for layer, weight := range weights {
		if weight > threshold {
			t.enableLayer(&cfg, layer)
		}
	}

	return cfg
}

// enableLayer enables a specific layer in the config
func (t *AutoTuner) enableLayer(cfg *PipelineConfig, layer string) {
	switch layer {
	case "0_tfidf":
		cfg.EnableTFIDF = true
	case "1_entropy":
		cfg.EnableEntropy = true
	case "2_perplexity":
		cfg.EnablePerplexity = true
	case "3_goal_driven":
		cfg.EnableGoalDriven = true
	case "4_ast_preserve":
		cfg.EnableAST = true
	case "5_contrastive":
		cfg.EnableContrastive = true
	case "6_ngram":
		cfg.NgramEnabled = true
	case "7_evaluator":
		cfg.EnableEvaluator = true
	case "8_gist":
		cfg.EnableGist = true
	case "9_hierarchical":
		cfg.EnableHierarchical = true
	case "11_compaction":
		cfg.EnableCompaction = true
	case "12_attribution":
		cfg.EnableAttribution = true
	case "13_h2o":
		cfg.EnableH2O = true
	case "14_attention_sink":
		cfg.EnableAttentionSink = true
	case "15_meta_token":
		cfg.EnableMetaToken = true
	case "16_semantic_chunk":
		cfg.EnableSemanticChunk = true
	case "17_sketch_store":
		cfg.EnableSketchStore = true
	case "18_lazy_pruner":
		cfg.EnableLazyPruner = true
	case "19_semantic_anchor":
		cfg.EnableSemanticAnchor = true
	case "20_agent_memory":
		cfg.EnableAgentMemory = true
	case "21_reasoning_trace":
		cfg.EnableReasoningTrace = true
	case "22_symbolic_compress":
		cfg.EnableSymbolicCompress = true
	case "23_phrase_grouping":
		cfg.EnablePhraseGrouping = true
	case "24_numerical_quant":
		cfg.EnableNumericalQuant = true
	case "25_dynamic_ratio":
		cfg.EnableDynamicRatio = true
	case "26_hypernym":
		cfg.EnableHypernym = true
	case "27_semantic_cache":
		cfg.EnableSemanticCache = true
	case "28_scope":
		cfg.EnableScope = true
	case "29_kvzip":
		cfg.EnableKVzip = true
	}
}

// countSamples counts records for a content type
func (t *AutoTuner) countSamples(contentType string) int {
	count := 0
	for _, r := range t.history {
		if r.ContentType == contentType {
			count++
		}
	}
	return count
}

// GetWeights returns current weights for a content type
func (t *AutoTuner) GetWeights(contentType string) map[string]float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if w, ok := t.weights[contentType]; ok {
		result := make(map[string]float64, len(w))
		for k, v := range w {
			result[k] = v
		}
		return result
	}
	return nil
}

// Save persists learned weights to disk
func (t *AutoTuner) Save() error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if t.dataPath == "" {
		return nil
	}

	data := struct {
		Weights map[string]map[string]float64 `json:"weights"`
		History []TuningRecord                `json:"history"`
	}{
		Weights: t.weights,
		History: t.history,
	}

	// Create directory if needed
	dir := filepath.Dir(t.dataPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(t.dataPath, bytes, 0600)
}

// loadWeights loads persisted weights from disk
func (t *AutoTuner) loadWeights() {
	if t.dataPath == "" {
		return
	}

	bytes, err := os.ReadFile(t.dataPath)
	if err != nil {
		return
	}

	var data struct {
		Weights map[string]map[string]float64 `json:"weights"`
		History []TuningRecord                `json:"history"`
	}

	if err := json.Unmarshal(bytes, &data); err != nil {
		return
	}

	if data.Weights != nil {
		t.weights = data.Weights
	}
	if data.History != nil {
		t.history = data.History
	}
}

// DetectContentType detects content type from input
func DetectContentTypeForTuner(input string) string {
	lines := splitLinesZeroCopy(input, nil)
	if len(lines) == 0 {
		return "unknown"
	}

	codeScore := 0
	logScore := 0
	gitScore := 0
	dockerScore := 0

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Code indicators
		if len(line) >= 4 {
			prefix := line[:min(4, len(line))]
			switch {
			case prefix == "func" || prefix == "def " || prefix == "clas" || prefix == "impo":
				codeScore++
			case (len(prefix) == 4 && prefix[0] == '2' && prefix[1] == '0' && prefix[2] >= '0' && prefix[2] <= '9' && prefix[3] >= '0' && prefix[3] <= '9') || prefix == "ERRO" || prefix == "WARN":
				logScore++
			case prefix == "diff" || prefix == "comm" || prefix == "Auth" || prefix == "inde":
				gitScore++
			case prefix == "CONT" || prefix == "IMAG" || prefix == "NAM ":
				dockerScore++
			}
		}
	}

	// Find dominant type
	maxScore := codeScore
	contentType := "code"
	if logScore > maxScore {
		maxScore = logScore
		contentType = "logs"
	}
	if gitScore > maxScore {
		maxScore = gitScore
		contentType = "git"
	}
	if dockerScore > maxScore {
		contentType = "docker"
	}

	return contentType
}

// PredictOptimalLayers returns which layers to use based on learned weights
func (t *AutoTuner) PredictOptimalLayers(contentType string, maxLayers int) []string {
	weights := t.GetWeights(contentType)
	if len(weights) == 0 {
		return nil
	}

	// Sort layers by weight (descending)
	type layerWeight struct {
		layer  string
		weight float64
	}

	var sorted []layerWeight
	for layer, weight := range weights {
		sorted = append(sorted, layerWeight{layer, weight})
	}

	// Simple selection sort
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].weight > sorted[i].weight {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	// Return top N
	n := int(math.Min(float64(maxLayers), float64(len(sorted))))
	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = sorted[i].layer
	}

	return result
}
