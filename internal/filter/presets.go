package filter

// PipelinePreset defines a compression pipeline mode with specific layers enabled.
// T90: Provide fast/balanced/full presets for different use cases.
type PipelinePreset string

const (
	// PresetFast runs only layers 1, 3, 10 (entropy, goal-driven, budget).
	// ~3x faster than full, ~60% of the compression.
	PresetFast PipelinePreset = "fast"

	// PresetBalanced runs layers 1-6, 10, 14 (entropy through ngram + budget + attention sink).
	// ~1.5x faster than full, ~85% of the compression.
	PresetBalanced PipelinePreset = "balanced"

	// PresetFull runs all 14 layers for maximum compression.
	PresetFull PipelinePreset = "full"
)

// PresetConfig returns a PipelineConfig for the given preset.
func PresetConfig(preset PipelinePreset, baseMode Mode) PipelineConfig {
	cfg := PipelineConfig{
		Mode:            baseMode,
		SessionTracking: true,
	}

	switch preset {
	case PresetFast:
		cfg.EnableEntropy = true
		cfg.EnablePerplexity = false
		cfg.EnableGoalDriven = true
		cfg.EnableAST = false
		cfg.EnableContrastive = false
		cfg.NgramEnabled = false
		cfg.EnableEvaluator = false
		cfg.EnableGist = false
		cfg.EnableHierarchical = false
		cfg.EnableCompaction = false
		cfg.EnableAttribution = false
		cfg.EnableH2O = false
		cfg.EnableAttentionSink = false

	case PresetBalanced:
		cfg.EnableEntropy = true
		cfg.EnablePerplexity = true
		cfg.EnableGoalDriven = true
		cfg.EnableAST = true
		cfg.EnableContrastive = true
		cfg.NgramEnabled = true
		cfg.EnableEvaluator = false
		cfg.EnableGist = false
		cfg.EnableHierarchical = false
		cfg.EnableCompaction = false
		cfg.EnableAttribution = false
		cfg.EnableH2O = false
		cfg.EnableAttentionSink = true

	default: // PresetFull
		cfg.EnableEntropy = true
		cfg.EnablePerplexity = true
		cfg.EnableGoalDriven = true
		cfg.EnableAST = true
		cfg.EnableContrastive = true
		cfg.NgramEnabled = true
		cfg.EnableEvaluator = true
		cfg.EnableGist = true
		cfg.EnableHierarchical = true
		cfg.EnableCompaction = true
		cfg.EnableAttribution = true
		cfg.EnableH2O = true
		cfg.EnableAttentionSink = true
	}

	return cfg
}

// QuickProcessPreset runs compression with a named preset.
func QuickProcessPreset(input string, mode Mode, preset PipelinePreset) (string, int) {
	cfg := PresetConfig(preset, mode)
	p := NewPipelineCoordinator(cfg)
	output, stats := p.Process(input)
	return output, stats.TotalSaved
}
