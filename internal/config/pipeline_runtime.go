package config

import "github.com/GrayCodeAI/tokman/internal/filter"

// PipelineRuntimeOptions carries per-request overrides for the runtime pipeline.
type PipelineRuntimeOptions struct {
	Mode        filter.Mode
	QueryIntent string
	Budget      int
	LLMEnabled  bool
}

// ToFilterPipelineConfig converts user-facing config into the runtime pipeline config.
// Some fields are best-effort mappings because the public config and runtime pipeline
// have diverged over time; centralizing that mapping keeps behavior consistent.
func (c PipelineConfig) ToFilterPipelineConfig(opts PipelineRuntimeOptions) filter.PipelineConfig {
	cfg := filter.PipelineConfig{
		Mode:                    opts.Mode,
		QueryIntent:             opts.QueryIntent,
		Budget:                  opts.Budget,
		LLMEnabled:              opts.LLMEnabled,
		SessionTracking:         true,
		NgramEnabled:            c.EnableNgram,
		EnableEntropy:           c.EnableEntropy,
		EnablePerplexity:        c.EnablePerplexity,
		EnableGoalDriven:        c.EnableGoalDriven,
		EnableAST:               c.EnableAST,
		EnableContrastive:       c.EnableContrastive,
		EnableEvaluator:         c.EnableEvaluator,
		EnableGist:              c.EnableGist,
		EnableHierarchical:      c.EnableHierarchical,
		EnableCompaction:        c.EnableCompaction,
		CompactionThreshold:     c.CompactionThreshold,
		CompactionPreserveTurns: c.CompactionPreserveTurns,
		CompactionMaxTokens:     c.CompactionMaxTokens,
		CompactionStateSnapshot: c.CompactionStateSnapshot,
		CompactionAutoDetect:    c.CompactionAutoDetect,
		EnableAttribution:       c.EnableAttribution,
		AttributionThreshold:    c.AttributionThreshold,
		EnableH2O:               c.EnableH2O,
		H2OSinkSize:             c.H2OSinkSize,
		H2ORecentSize:           c.H2ORecentSize,
		H2OHeavyHitterSize:      c.H2OHeavyHitterSize,
		EnableAttentionSink:     c.EnableAttentionSink,
		AttentionSinkCount:      c.AttentionSinkCount,
		AttentionRecentCount:    c.AttentionRecentCount,
		EnableMetaToken:         c.EnableMetaToken,
		MetaTokenWindow:         c.MetaTokenWindow,
		MetaTokenMinSize:        c.MetaTokenMinMatch,
		EnableSemanticChunk:     c.EnableSemanticChunk,
		SemanticChunkMinSize:    c.ChunkMinSize,
		SemanticChunkThreshold:  c.SemanticThreshold,
		EnableSketchStore:       c.EnableSketchStore,
		SketchBudgetRatio:       float64(c.SketchMemoryRatio) / 100.0,
		EnableLazyPruner:        c.EnableLazyPruner,
		LazyDecayRate:           c.LazyLayerDecay,
		EnableSemanticAnchor:    c.EnableSemanticAnchor,
		EnableAgentMemory:       c.EnableAgentMemory,
		AgentConsolidationMax:   c.AgentMemoryMaxNodes,
	}

	if c.DefaultBudget > 0 && c.LazyBudgetRatio > 0 {
		cfg.LazyBaseBudget = int(float64(c.DefaultBudget) * c.LazyBudgetRatio)
	}
	if c.AnchorMinPreserve > 0 {
		cfg.SemanticAnchorSpacing = c.AnchorMinPreserve
	}

	return cfg
}
