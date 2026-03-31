package filter

// PipelinePreset defines a compression pipeline mode with specific layers enabled.
// Provide fast/balanced/full presets for different use cases.
// Maps to the adaptive tier system for automatic layer selection.
type PipelinePreset string

const (
	// PresetFast maps to Tier1_Simple (3 layers).
	// ~3x faster than full, ~60% of the compression.
	// Best for: Quick commands, small output, speed-critical operations.
	PresetFast PipelinePreset = "fast"

	// PresetBalanced maps to Tier2_Medium (8 layers).
	// ~1.5x faster than full, ~85% of the compression.
	// Best for: Most CLI output, git status, build logs, test results.
	PresetBalanced PipelinePreset = "balanced"

	// PresetFull maps to Tier3_Full (20 layers).
	// Best for: Large outputs, documentation, conversation history.
	PresetFull PipelinePreset = "full"

	// PresetAuto automatically selects tier based on content analysis.
	// Analyzes content size, type, and complexity to pick optimal tier.
	// Best for: When you don't know the content size/type in advance.
	PresetAuto PipelinePreset = "auto"
)

// PresetConfig returns a PipelineConfig for the given preset.
func PresetConfig(preset PipelinePreset, baseMode Mode) PipelineConfig {
	cfg := PipelineConfig{
		Mode:            baseMode,
		SessionTracking: true,
	}

	switch preset {
	case PresetFast:
		// Fast preset: Minimal layers for speed
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
		cfg.EnableTFIDF = false
		cfg.EnableReasoningTrace = false
		cfg.EnableSymbolicCompress = false
		cfg.EnablePhraseGrouping = false
		cfg.EnableNumericalQuant = false
		cfg.EnableDynamicRatio = false

	case PresetBalanced:
		// Balanced preset: Core layers for most use cases
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
		cfg.EnableTFIDF = true
		cfg.EnableReasoningTrace = false
		cfg.EnableSymbolicCompress = false
		cfg.EnablePhraseGrouping = false
		cfg.EnableNumericalQuant = true
		cfg.EnableDynamicRatio = true

	default: // PresetFull + PresetAuto
		// Full preset: All layers for maximum compression
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
		cfg.EnableTFIDF = true
		cfg.EnableReasoningTrace = true
		cfg.EnableSymbolicCompress = true
		cfg.EnablePhraseGrouping = true
		cfg.EnableNumericalQuant = true
		cfg.EnableDynamicRatio = true
		// Phase 2 layers
		cfg.EnableHypernym = true
		cfg.EnableSemanticCache = true
		cfg.EnableScope = true
		cfg.EnableSmallKV = true
		cfg.EnableKVzip = true
	}

	return cfg
}

// QuickProcessPreset runs compression with a named preset.
// For PresetAuto, uses the adaptive pipeline to select optimal tier.
func QuickProcessPreset(input string, mode Mode, preset PipelinePreset) (string, int) {
	cfg := PresetConfig(preset, mode)
	p := NewPipelineCoordinator(cfg)
	output, stats := p.Process(input)
	return output, stats.TotalSaved
}
