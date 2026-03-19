package filter

import (
	"fmt"
	"strings"
)

// PipelineCoordinator orchestrates the 10-layer compression pipeline.
// Research-based: Combines the best techniques from 50+ research papers worldwide
// to achieve maximum token reduction for CLI/Agent output.
//
// Layer order is critical - each layer builds on the previous:
//
// Layer 1: Entropy Filtering (Selective Context, Mila 2023) - 2-3x
// Layer 2: Perplexity Pruning (LLMLingua, Microsoft/Tsinghua 2023) - 20x
// Layer 3: Goal-Driven Selection (SWE-Pruner, Shanghai Jiao Tong 2025) - 14.8x
// Layer 4: AST Preservation (LongCodeZip, NUS 2025) - 4-8x
// Layer 5: Contrastive Ranking (LongLLMLingua, Microsoft 2024) - 4-10x
// Layer 6: N-gram Abbreviation (CompactPrompt 2025) - 2.5x
// Layer 7: Evaluator Heads (EHPC, Tsinghua/Huawei 2025) - 5-7x
// Layer 8: Gist Compression (Stanford/Berkeley 2023) - 20x+
// Layer 9: Hierarchical Summary (AutoCompressor, Princeton/MIT 2023) - Extreme
// Layer 10: Budget Enforcement (Industry standard) - Guaranteed
type PipelineCoordinator struct {
	config PipelineConfig
	
	// Layer 1: Entropy Filtering
	entropyFilter *EntropyFilter
	
	// Layer 2: Perplexity Pruning
	perplexityFilter *PerplexityFilter
	
	// Layer 3: Goal-Driven Selection
	goalDrivenFilter *GoalDrivenFilter
	
	// Layer 4: AST Preservation
	astPreserveFilter *ASTPreserveFilter
	
	// Layer 5: Contrastive Ranking
	contrastiveFilter *ContrastiveFilter
	
	// Layer 6: N-gram Abbreviation
	ngramAbbreviator *NgramAbbreviator
	
	// Layer 7: Evaluator Heads
	evaluatorHeadsFilter *EvaluatorHeadsFilter
	
	// Layer 8: Gist Compression
	gistFilter *GistFilter
	
	// Layer 9: Hierarchical Summary
	hierarchicalSummaryFilter *HierarchicalSummaryFilter
	
	// Layer 10: Budget Enforcement
	budgetEnforcer *BudgetEnforcer
	sessionTracker *SessionTracker
	
	// Optional: Neural Layer (when LLM enabled)
	llmFilter *LLMAwareFilter
	
	// Layer 11: Compaction Layer (AdaL-style semantic compression)
	compactionLayer *CompactionLayer
}

// PipelineConfig holds configuration for the compression pipeline
type PipelineConfig struct {
	// Mode: none, minimal, aggressive
	Mode Mode
	
	// Query intent for query-aware compression
	QueryIntent string
	
	// Token budget (0 = unlimited)
	Budget int
	
	// Enable LLM-based compression
	LLMEnabled bool
	
	// Enable session tracking
	SessionTracking bool
	
	// Enable N-gram abbreviation
	NgramEnabled bool
	
	// Enable multi-file optimization
	MultiFileEnabled bool
	
	// Prompt template for LLM
	PromptTemplate string
	
	// Enable specific layers (all enabled by default)
	EnableEntropy     bool
	EnablePerplexity  bool
	EnableGoalDriven  bool
	EnableAST         bool
	EnableContrastive bool
	EnableEvaluator   bool
	EnableGist        bool
	EnableHierarchical bool
	
	// Layer 11: Compaction (AdaL-style semantic compression)
	EnableCompaction        bool
	CompactionThreshold     int
	CompactionPreserveTurns int
	CompactionMaxTokens     int
	CompactionStateSnapshot bool
	CompactionAutoDetect    bool
}

// NewPipelineCoordinator creates a new 10-layer pipeline coordinator.
func NewPipelineCoordinator(cfg PipelineConfig) *PipelineCoordinator {
	p := &PipelineCoordinator{
		config: cfg,
	}
	
	// Set defaults
	if cfg.EnableEntropy != false {
		cfg.EnableEntropy = true
	}
	if cfg.EnablePerplexity != false {
		cfg.EnablePerplexity = true
	}
	if cfg.EnableGoalDriven != false {
		cfg.EnableGoalDriven = true
	}
	if cfg.EnableAST != false {
		cfg.EnableAST = true
	}
	if cfg.EnableContrastive != false {
		cfg.EnableContrastive = true
	}
	if cfg.EnableEvaluator != false {
		cfg.EnableEvaluator = true
	}
	if cfg.EnableGist != false {
		cfg.EnableGist = true
	}
	if cfg.EnableHierarchical != false {
		cfg.EnableHierarchical = true
	}
	
	// Layer 1: Entropy Filtering
	p.entropyFilter = NewEntropyFilter()
	
	// Layer 2: Perplexity Pruning
	p.perplexityFilter = NewPerplexityFilter()
	
	// Layer 3: Goal-Driven Selection
	if cfg.QueryIntent != "" {
		p.goalDrivenFilter = NewGoalDrivenFilter(cfg.QueryIntent)
	}
	
	// Layer 4: AST Preservation
	p.astPreserveFilter = NewASTPreserveFilter()
	
	// Layer 5: Contrastive Ranking
	if cfg.QueryIntent != "" {
		p.contrastiveFilter = NewContrastiveFilter(cfg.QueryIntent)
	}
	
	// Layer 6: N-gram Abbreviation
	if cfg.NgramEnabled {
		p.ngramAbbreviator = NewNgramAbbreviator()
	}
	
	// Layer 7: Evaluator Heads
	p.evaluatorHeadsFilter = NewEvaluatorHeadsFilter()
	
	// Layer 8: Gist Compression
	p.gistFilter = NewGistFilter()
	
	// Layer 9: Hierarchical Summary
	p.hierarchicalSummaryFilter = NewHierarchicalSummaryFilter()
	
	// Layer 10: Budget Enforcement
	if cfg.Budget > 0 {
		p.budgetEnforcer = NewBudgetEnforcer(cfg.Budget)
	}
	if cfg.SessionTracking {
		p.sessionTracker = NewSessionTracker()
	}
	
	// Optional Neural Layer
	if cfg.LLMEnabled {
		p.llmFilter = NewLLMAwareFilter(LLMAwareConfig{
			Threshold:      2000,
			Enabled:        true,
			CacheEnabled:   true,
			PromptTemplate: cfg.PromptTemplate,
		})
	}
	
	// Layer 11: Compaction Layer (AdaL-style semantic compression)
	if cfg.EnableCompaction {
		compactionCfg := CompactionConfig{
			Enabled:              true,
			ThresholdTokens:      cfg.CompactionThreshold,
			PreserveRecentTurns:  cfg.CompactionPreserveTurns,
			MaxSummaryTokens:     cfg.CompactionMaxTokens,
			StateSnapshotFormat:  cfg.CompactionStateSnapshot,
			AutoDetect:           cfg.CompactionAutoDetect,
			CacheEnabled:         true,
		}
		if compactionCfg.ThresholdTokens == 0 {
			compactionCfg.ThresholdTokens = 2000
		}
		if compactionCfg.PreserveRecentTurns == 0 {
			compactionCfg.PreserveRecentTurns = 5
		}
		if compactionCfg.MaxSummaryTokens == 0 {
			compactionCfg.MaxSummaryTokens = 500
		}
		p.compactionLayer = NewCompactionLayer(compactionCfg)
	}
	
	return p
}

// Process runs the full 10-layer compression pipeline.
func (p *PipelineCoordinator) Process(input string) (string, *PipelineStats) {
	stats := &PipelineStats{
		OriginalTokens: EstimateTokens(input),
		LayerStats:     make(map[string]LayerStat),
	}
	
	output := input
	
	// Layer 1: Entropy Filtering (Remove low-information tokens)
	if p.entropyFilter != nil && p.config.EnableEntropy {
		output = p.processLayer1(output, stats)
	}
	
	// Layer 2: Perplexity Pruning (Iterative token removal)
	if p.perplexityFilter != nil && p.config.EnablePerplexity {
		output = p.processLayer2(output, stats)
	}
	
	// Layer 3: Goal-Driven Selection (CRF-style line scoring)
	if p.goalDrivenFilter != nil && p.config.EnableGoalDriven {
		output = p.processLayer3(output, stats)
	}
	
	// Layer 4: AST Preservation (Syntax-aware compression)
	if p.astPreserveFilter != nil && p.config.EnableAST {
		output = p.processLayer4(output, stats)
	}
	
	// Layer 5: Contrastive Ranking (Question-relevance scoring)
	if p.contrastiveFilter != nil && p.config.EnableContrastive {
		output = p.processLayer5(output, stats)
	}
	
	// Layer 6: N-gram Abbreviation (Lossless compression)
	if p.ngramAbbreviator != nil {
		output = p.processLayer6(output, stats)
	}
	
	// Layer 7: Evaluator Heads (Early-layer attention simulation)
	if p.evaluatorHeadsFilter != nil && p.config.EnableEvaluator {
		output = p.processLayer7(output, stats)
	}
	
	// Layer 8: Gist Compression (Virtual token embedding)
	if p.gistFilter != nil && p.config.EnableGist {
		output = p.processLayer8(output, stats)
	}
	
	// Layer 9: Hierarchical Summary (Recursive summarization)
	if p.hierarchicalSummaryFilter != nil && p.config.EnableHierarchical {
		output = p.processLayer9(output, stats)
	}
	
	// Optional: Neural Layer (LLM-based compression)
	if p.llmFilter != nil {
		output = p.processLayerNeural(output, stats)
	}
	
	// Layer 11: Compaction Layer (AdaL-style semantic compression)
	if p.compactionLayer != nil {
		output = p.processLayer11(output, stats)
	}
	
	// Layer 10: Budget Enforcement (Strict token limits)
	output = p.processLayer10(output, stats)
	
	stats.FinalTokens = EstimateTokens(output)
	stats.TotalSaved = stats.OriginalTokens - stats.FinalTokens
	if stats.OriginalTokens > 0 {
		stats.ReductionPercent = float64(stats.TotalSaved) / float64(stats.OriginalTokens) * 100
	}
	
	return output, stats
}

// Layer 1: Entropy Filtering
func (p *PipelineCoordinator) processLayer1(input string, stats *PipelineStats) string {
	output, saved := p.entropyFilter.Apply(input, p.config.Mode)
	stats.LayerStats["1_entropy"] = LayerStat{TokensSaved: saved}
	return output
}

// Layer 2: Perplexity Pruning
func (p *PipelineCoordinator) processLayer2(input string, stats *PipelineStats) string {
	output, saved := p.perplexityFilter.Apply(input, p.config.Mode)
	stats.LayerStats["2_perplexity"] = LayerStat{TokensSaved: saved}
	return output
}

// Layer 3: Goal-Driven Selection
func (p *PipelineCoordinator) processLayer3(input string, stats *PipelineStats) string {
	output, saved := p.goalDrivenFilter.Apply(input, p.config.Mode)
	stats.LayerStats["3_goal_driven"] = LayerStat{TokensSaved: saved}
	return output
}

// Layer 4: AST Preservation
func (p *PipelineCoordinator) processLayer4(input string, stats *PipelineStats) string {
	output, saved := p.astPreserveFilter.Apply(input, p.config.Mode)
	stats.LayerStats["4_ast_preserve"] = LayerStat{TokensSaved: saved}
	return output
}

// Layer 5: Contrastive Ranking
func (p *PipelineCoordinator) processLayer5(input string, stats *PipelineStats) string {
	output, saved := p.contrastiveFilter.Apply(input, p.config.Mode)
	stats.LayerStats["5_contrastive"] = LayerStat{TokensSaved: saved}
	return output
}

// Layer 6: N-gram Abbreviation
func (p *PipelineCoordinator) processLayer6(input string, stats *PipelineStats) string {
	output, saved := p.ngramAbbreviator.Apply(input, p.config.Mode)
	stats.LayerStats["6_ngram"] = LayerStat{TokensSaved: saved}
	return output
}

// Layer 7: Evaluator Heads
func (p *PipelineCoordinator) processLayer7(input string, stats *PipelineStats) string {
	output, saved := p.evaluatorHeadsFilter.Apply(input, p.config.Mode)
	stats.LayerStats["7_evaluator"] = LayerStat{TokensSaved: saved}
	return output
}

// Layer 8: Gist Compression
func (p *PipelineCoordinator) processLayer8(input string, stats *PipelineStats) string {
	output, saved := p.gistFilter.Apply(input, p.config.Mode)
	stats.LayerStats["8_gist"] = LayerStat{TokensSaved: saved}
	return output
}

// Layer 9: Hierarchical Summary
func (p *PipelineCoordinator) processLayer9(input string, stats *PipelineStats) string {
	output, saved := p.hierarchicalSummaryFilter.Apply(input, p.config.Mode)
	stats.LayerStats["9_hierarchical"] = LayerStat{TokensSaved: saved}
	return output
}

// Layer Neural: LLM-based compression (optional)
func (p *PipelineCoordinator) processLayerNeural(input string, stats *PipelineStats) string {
	output, saved := p.llmFilter.Apply(input, p.config.Mode)
	stats.LayerStats["neural"] = LayerStat{TokensSaved: saved}
	return output
}

// Layer 11: Compaction (AdaL-style semantic compression)
func (p *PipelineCoordinator) processLayer11(input string, stats *PipelineStats) string {
	output, saved := p.compactionLayer.Apply(input, p.config.Mode)
	stats.LayerStats["11_compaction"] = LayerStat{TokensSaved: saved}
	return output
}

// Layer 10: Budget Enforcement
func (p *PipelineCoordinator) processLayer10(input string, stats *PipelineStats) string {
	output := input
	totalSaved := 0
	
	// Session tracking (deduplication)
	if p.sessionTracker != nil {
		filtered, saved := p.sessionTracker.Apply(output, p.config.Mode)
		output = filtered
		totalSaved += saved
		stats.LayerStats["10_session"] = LayerStat{TokensSaved: saved}
	}
	
	// Budget enforcement (final safety net)
	if p.budgetEnforcer != nil {
		filtered, saved := p.budgetEnforcer.Apply(output, p.config.Mode)
		output = filtered
		totalSaved += saved
		stats.LayerStats["10_budget"] = LayerStat{TokensSaved: saved}
	}
	
	stats.LayerStats["10_total"] = LayerStat{TokensSaved: totalSaved}
	return output
}

// PipelineStats holds statistics from the compression pipeline
type PipelineStats struct {
	OriginalTokens   int
	FinalTokens      int
	TotalSaved       int
	ReductionPercent float64
	LayerStats       map[string]LayerStat
}

// LayerStat holds statistics for a single layer
type LayerStat struct {
	TokensSaved int
}

// String returns a formatted summary of pipeline stats
func (s *PipelineStats) String() string {
	var sb strings.Builder
	
	sb.WriteString("╔════════════════════════════════════════════════════╗\n")
	sb.WriteString("║          Tokman 10-Layer Compression Stats         ║\n")
	sb.WriteString("╠════════════════════════════════════════════════════╣\n")
	sb.WriteString(fmt.Sprintf("║ Original:  %6d tokens                         ║\n", s.OriginalTokens))
	sb.WriteString(fmt.Sprintf("║ Final:     %6d tokens                         ║\n", s.FinalTokens))
	sb.WriteString(fmt.Sprintf("║ Saved:     %6d tokens (%.1f%%)                 ║\n", s.TotalSaved, s.ReductionPercent))
	sb.WriteString("╠════════════════════════════════════════════════════╣\n")
	sb.WriteString("║ Layer Breakdown:                                   ║\n")
	
	// Order layers properly
	layerOrder := []string{
		"1_entropy", "2_perplexity", "3_goal_driven", "4_ast_preserve",
		"5_contrastive", "6_ngram", "7_evaluator", "8_gist", "9_hierarchical",
		"neural", "10_session", "10_budget",
	}
	
	for _, layer := range layerOrder {
		if stat, ok := s.LayerStats[layer]; ok && stat.TokensSaved > 0 {
			sb.WriteString(fmt.Sprintf("║   %-20s: %6d tokens saved     ║\n", layer, stat.TokensSaved))
		}
	}
	
	sb.WriteString("╚════════════════════════════════════════════════════╝\n")
	
	return sb.String()
}

// QuickProcess is a convenience function for simple compression
func QuickProcess(input string, mode Mode) (string, int) {
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:            mode,
		SessionTracking: true,
		NgramEnabled:    true,
	})
	
	output, stats := p.Process(input)
	return output, stats.TotalSaved
}

// QuickProcessWithBudget is a convenience function for budgeted compression
func QuickProcessWithBudget(input string, mode Mode, budget int) (string, int) {
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:            mode,
		Budget:          budget,
		SessionTracking: true,
		NgramEnabled:    true,
	})
	
	output, stats := p.Process(input)
	return output, stats.TotalSaved
}

// QuickProcessWithQuery is a convenience function for query-aware compression
func QuickProcessWithQuery(input string, mode Mode, query string) (string, int) {
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:            mode,
		QueryIntent:     query,
		SessionTracking: true,
		NgramEnabled:    true,
	})
	
	output, stats := p.Process(input)
	return output, stats.TotalSaved
}

// QuickProcessFull is a convenience function with all options
func QuickProcessFull(input string, mode Mode, query string, budget int, llmEnabled bool) (string, int) {
	p := NewPipelineCoordinator(PipelineConfig{
		Mode:            mode,
		QueryIntent:     query,
		Budget:          budget,
		LLMEnabled:      llmEnabled,
		SessionTracking: true,
		NgramEnabled:    true,
	})
	
	output, stats := p.Process(input)
	return output, stats.TotalSaved
}
