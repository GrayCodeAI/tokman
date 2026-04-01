package config

import (
	"testing"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

func TestToFilterPipelineConfigMapsKeyFields(t *testing.T) {
	cfg := PipelineConfig{
		EnableEntropy:           true,
		EnableNgram:             true,
		EnableCompaction:        true,
		CompactionThreshold:     123,
		CompactionPreserveTurns: 7,
		CompactionMaxTokens:     456,
		EnableMetaToken:         true,
		MetaTokenWindow:         64,
		MetaTokenMinMatch:       5,
		EnableSemanticChunk:     true,
		SemanticThreshold:       0.7,
		ChunkMinSize:            80,
		EnableSketchStore:       true,
		SketchMemoryRatio:       90,
		EnableLazyPruner:        true,
		LazyBudgetRatio:         0.25,
		LazyLayerDecay:          0.8,
		DefaultBudget:           2000,
		EnableSemanticAnchor:    true,
		AnchorMinPreserve:       6,
		EnableAgentMemory:       true,
		AgentMemoryMaxNodes:     42,
	}

	runtime := cfg.ToFilterPipelineConfig(PipelineRuntimeOptions{
		Mode:        filter.ModeAggressive,
		QueryIntent: "debug",
		Budget:      1500,
		LLMEnabled:  true,
	})

	if runtime.Mode != filter.ModeAggressive {
		t.Fatalf("Mode = %q, want %q", runtime.Mode, filter.ModeAggressive)
	}
	if runtime.QueryIntent != "debug" {
		t.Fatalf("QueryIntent = %q, want debug", runtime.QueryIntent)
	}
	if runtime.Budget != 1500 {
		t.Fatalf("Budget = %d, want 1500", runtime.Budget)
	}
	if !runtime.EnableCompaction || runtime.CompactionThreshold != 123 {
		t.Fatalf("compaction mapping failed: %+v", runtime)
	}
	if runtime.MetaTokenMinSize != 5 {
		t.Fatalf("MetaTokenMinSize = %d, want 5", runtime.MetaTokenMinSize)
	}
	if runtime.SemanticChunkMinSize != 80 {
		t.Fatalf("SemanticChunkMinSize = %d, want 80", runtime.SemanticChunkMinSize)
	}
	if runtime.SemanticChunkThreshold != 0.7 {
		t.Fatalf("SemanticChunkThreshold = %v, want 0.7", runtime.SemanticChunkThreshold)
	}
	if runtime.SketchBudgetRatio != 0.9 {
		t.Fatalf("SketchBudgetRatio = %v, want 0.9", runtime.SketchBudgetRatio)
	}
	if runtime.LazyBaseBudget != 500 {
		t.Fatalf("LazyBaseBudget = %d, want 500", runtime.LazyBaseBudget)
	}
	if runtime.LazyDecayRate != 0.8 {
		t.Fatalf("LazyDecayRate = %v, want 0.8", runtime.LazyDecayRate)
	}
	if runtime.SemanticAnchorSpacing != 6 {
		t.Fatalf("SemanticAnchorSpacing = %d, want 6", runtime.SemanticAnchorSpacing)
	}
	if runtime.AgentConsolidationMax != 42 {
		t.Fatalf("AgentConsolidationMax = %d, want 42", runtime.AgentConsolidationMax)
	}
}
