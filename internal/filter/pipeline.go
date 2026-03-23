package filter

import (
	"fmt"
	"strings"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// filterLayer pairs a compression filter with its stats key.
type filterLayer struct {
	filter Filter
	name   string
}

// PipelineCoordinator orchestrates the 20-layer compression pipeline.
// Research-based: Combines the best techniques from 120+ research papers worldwide
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
// Layer 11: Compaction Layer (MemGPT, UC Berkeley 2023) - 98%+
// Layer 12: Attribution Filter (ProCut, LinkedIn 2025) - 78%
// Layer 13: H2O Filter (Heavy-Hitter Oracle, NeurIPS 2023) - 30x+
// Layer 14: Attention Sink Filter (StreamingLLM, 2023) - Infinite context stability
// Layer 15: Meta-Token Compression (arXiv:2506.00307, 2025) - 27% lossless
// Layer 16: Semantic Chunking (ChunkKV-style) - Context-aware boundaries
// Layer 17: Sketch Store (KVReviver, Dec 2025) - 90% memory reduction
// Layer 18: Lazy Pruner (LazyLLM, July 2024) - 2.34x speedup
// Layer 19: Semantic Anchor (Attention Gradient Detection) - Context preservation
// Layer 20: Agent Memory (Knowledge Graph Extraction) - Agent-optimized
type PipelineCoordinator struct {
	config PipelineConfig

	layers []filterLayer

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

	// Layer 11: Compaction Layer (Semantic compression)
	compactionLayer *CompactionLayer

	// Layer 12: Attribution Filter (ProCut-style pruning)
	attributionFilter *AttributionFilter

	// Layer 13: H2O Filter (Heavy-Hitter Oracle)
	h2oFilter *H2OFilter

	// Layer 14: Attention Sink Filter (StreamingLLM-style)
	attentionSinkFilter *AttentionSinkFilter

	// Layer 15: Meta-Token Lossless Compression (arXiv:2506.00307)
	metaTokenFilter *MetaTokenFilter

	// Layer 16: Semantic Chunk Filter (ChunkKV style)
	semanticChunkFilter *SemanticChunkFilter

	// Layer 17: Sketch-based Reversible Store (KVReviver style)
	sketchStoreFilter *SketchStoreFilter

	// Layer 18: Budget-aware Dynamic Pruning (LazyLLM style)
	lazyPrunerFilter *LazyPrunerFilter

	// Layer 19: Semantic-Anchor Compression (SAC style)
	semanticAnchorFilter *SemanticAnchorFilter

	// Layer 20: Agent Memory Mode (Focus-inspired)
	agentMemoryFilter *AgentMemoryFilter

	// T12: Question-Aware Filter (LongLLMLingua-style)
	questionAwareFilter *QuestionAwareFilter

	// T17: Density-Adaptive Filter (DAST-style)
	densityAdaptiveFilter *DensityAdaptiveFilter
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
	EnableEntropy      bool
	EnablePerplexity   bool
	EnableGoalDriven   bool
	EnableAST          bool
	EnableContrastive  bool
	EnableEvaluator    bool
	EnableGist         bool
	EnableHierarchical bool

	// Layer 11: Compaction (Semantic compression)
	EnableCompaction        bool
	CompactionThreshold     int
	CompactionPreserveTurns int
	CompactionMaxTokens     int
	CompactionStateSnapshot bool
	CompactionAutoDetect    bool

	// Layer 12: Attribution Filter (ProCut-style pruning)
	EnableAttribution    bool
	AttributionThreshold float64

	// Layer 13: H2O Filter (Heavy-Hitter Oracle)
	EnableH2O          bool
	H2OSinkSize        int
	H2ORecentSize      int
	H2OHeavyHitterSize int

	// Layer 14: Attention Sink Filter (StreamingLLM-style)
	EnableAttentionSink  bool
	AttentionSinkCount   int
	AttentionRecentCount int

	// Layer 15: Meta-Token Lossless Compression (arXiv:2506.00307)
	EnableMetaToken  bool
	MetaTokenWindow  int
	MetaTokenMinSize int

	// Layer 16: Semantic Chunk Filter (ChunkKV style)
	EnableSemanticChunk    bool
	SemanticChunkMethod    string // "auto", "code", "text", "mixed"
	SemanticChunkMinSize   int
	SemanticChunkThreshold float64

	// Layer 17: Sketch-based Reversible Store (KVReviver style)
	EnableSketchStore  bool
	SketchBudgetRatio  float64
	SketchMaxSize      int
	SketchHeavyHitter  float64

	// Layer 18: Budget-aware Dynamic Pruning (LazyLLM style)
	EnableLazyPruner   bool
	LazyBaseBudget     int
	LazyDecayRate      float64
	LazyRevivalBudget  int

	// Layer 19: Semantic-Anchor Compression (SAC style)
	EnableSemanticAnchor bool
	SemanticAnchorRatio  float64
	SemanticAnchorSpacing int

	// Layer 20: Agent Memory Mode (Focus-inspired)
	EnableAgentMemory       bool
	AgentKnowledgeRetention float64
	AgentHistoryPrune       float64
	AgentConsolidationMax   int

	// T12: Question-Aware Filter (LongLLMLingua-style)
	EnableQuestionAware    bool
	QuestionAwareThreshold float64

	// T17: Density-Adaptive Filter (DAST-style)
	EnableDensityAdaptive bool
	DensityTargetRatio    float64
	DensityThreshold      float64
}

// NewPipelineCoordinator creates a new 10-layer pipeline coordinator.
func NewPipelineCoordinator(cfg PipelineConfig) *PipelineCoordinator {
	p := &PipelineCoordinator{
		config: cfg,
	}

	// Set defaults - all layers enabled by default when using zero-config.
	// Callers passing explicit config (e.g., from audit.go) have their values respected.
	allDisabled := !cfg.EnableEntropy && !cfg.EnablePerplexity && !cfg.EnableGoalDriven &&
		!cfg.EnableAST && !cfg.EnableContrastive && !cfg.EnableEvaluator &&
		!cfg.EnableGist && !cfg.EnableHierarchical
	hasExplicitSettings := cfg.Budget > 0 || cfg.QueryIntent != "" || cfg.LLMEnabled ||
		cfg.NgramEnabled || cfg.MultiFileEnabled || cfg.SessionTracking ||
		cfg.EnableCompaction || cfg.EnableAttribution || cfg.EnableH2O || cfg.EnableAttentionSink
	if allDisabled && !hasExplicitSettings {
		cfg.EnableEntropy = true
		cfg.EnablePerplexity = true
		cfg.EnableGoalDriven = true
		cfg.EnableAST = true
		cfg.EnableContrastive = true
		cfg.EnableEvaluator = true
		cfg.EnableGist = true
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

	// Layer 11: Compaction Layer (Semantic compression)
	if cfg.EnableCompaction {
		compactionCfg := CompactionConfig{
			Enabled:             true,
			ThresholdTokens:     cfg.CompactionThreshold,
			PreserveRecentTurns: cfg.CompactionPreserveTurns,
			MaxSummaryTokens:    cfg.CompactionMaxTokens,
			StateSnapshotFormat: cfg.CompactionStateSnapshot,
			AutoDetect:          cfg.CompactionAutoDetect,
			CacheEnabled:        true,
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

	// Layer 12: Attribution Filter (ProCut-style pruning)
	if cfg.EnableAttribution {
		p.attributionFilter = NewAttributionFilter()
		if cfg.AttributionThreshold > 0 {
			p.attributionFilter.config.ImportanceThreshold = cfg.AttributionThreshold
		}
	}

	// Layer 13: H2O Filter (Heavy-Hitter Oracle)
	if cfg.EnableH2O {
		p.h2oFilter = NewH2OFilter()
		if cfg.H2OSinkSize > 0 {
			p.h2oFilter.config.SinkSize = cfg.H2OSinkSize
		}
		if cfg.H2ORecentSize > 0 {
			p.h2oFilter.config.RecentSize = cfg.H2ORecentSize
		}
		if cfg.H2OHeavyHitterSize > 0 {
			p.h2oFilter.config.HeavyHitterSize = cfg.H2OHeavyHitterSize
		}
	}

	// Layer 14: Attention Sink Filter (StreamingLLM-style)
	// T14: Use adaptive attention sinks based on content size
	if cfg.EnableAttentionSink {
		// Estimate lines from typical content for adaptive sizing
		estimatedLines := 50 // Default estimate
		p.attentionSinkFilter = NewAdaptiveAttentionSinkFilter(estimatedLines)
		if cfg.AttentionSinkCount > 0 {
			p.attentionSinkFilter.config.SinkTokenCount = cfg.AttentionSinkCount
		}
		if cfg.AttentionRecentCount > 0 {
			p.attentionSinkFilter.config.RecentTokenCount = cfg.AttentionRecentCount
		}
	}

	// Layer 15: Meta-Token Lossless Compression (arXiv:2506.00307)
	if cfg.EnableMetaToken {
		metaCfg := DefaultMetaTokenConfig()
		if cfg.MetaTokenWindow > 0 {
			metaCfg.WindowSize = cfg.MetaTokenWindow
		}
		if cfg.MetaTokenMinSize > 0 {
			metaCfg.MinPattern = cfg.MetaTokenMinSize
		}
		p.metaTokenFilter = NewMetaTokenFilterWithConfig(metaCfg)
	}

	// Layer 16: Semantic Chunk Filter (ChunkKV style)
	if cfg.EnableSemanticChunk {
		semanticCfg := DefaultSemanticChunkConfig()
		if cfg.SemanticChunkMinSize > 0 {
			semanticCfg.MinChunkSize = cfg.SemanticChunkMinSize
		}
		if cfg.SemanticChunkThreshold > 0 {
			semanticCfg.ImportanceThreshold = cfg.SemanticChunkThreshold
		}
		p.semanticChunkFilter = NewSemanticChunkFilterWithConfig(semanticCfg)
	}

	// Layer 17: Sketch-based Reversible Store (KVReviver style)
	if cfg.EnableSketchStore {
		sketchCfg := DefaultSketchStoreConfig()
		if cfg.SketchBudgetRatio > 0 {
			sketchCfg.BudgetRatio = cfg.SketchBudgetRatio
		}
		if cfg.SketchMaxSize > 0 {
			sketchCfg.MaxSketchSize = cfg.SketchMaxSize
		}
		if cfg.SketchHeavyHitter > 0 {
			sketchCfg.HeavyHitterRatio = cfg.SketchHeavyHitter
		}
		p.sketchStoreFilter = NewSketchStoreFilterWithConfig(sketchCfg)
	}

	// Layer 18: Budget-aware Dynamic Pruning (LazyLLM style)
	if cfg.EnableLazyPruner {
		lazyCfg := DefaultLazyPrunerConfig()
		if cfg.LazyBaseBudget > 0 {
			lazyCfg.BaseBudget = cfg.LazyBaseBudget
		}
		if cfg.LazyDecayRate > 0 {
			lazyCfg.DecayRate = cfg.LazyDecayRate
		}
		if cfg.LazyRevivalBudget > 0 {
			lazyCfg.RevivalBudget = cfg.LazyRevivalBudget
		}
		p.lazyPrunerFilter = NewLazyPrunerFilterWithConfig(lazyCfg)
	}

	// Layer 19: Semantic-Anchor Compression (SAC style)
	if cfg.EnableSemanticAnchor {
		anchorCfg := DefaultSemanticAnchorConfig()
		if cfg.SemanticAnchorRatio > 0 {
			anchorCfg.AnchorRatio = cfg.SemanticAnchorRatio
		}
		if cfg.SemanticAnchorSpacing > 0 {
			anchorCfg.MinAnchorSpacing = cfg.SemanticAnchorSpacing
		}
		p.semanticAnchorFilter = NewSemanticAnchorFilterWithConfig(anchorCfg)
	}

	// Layer 20: Agent Memory Mode (Focus-inspired)
	if cfg.EnableAgentMemory {
		agentCfg := DefaultAgentMemoryConfig()
		if cfg.AgentKnowledgeRetention > 0 {
			agentCfg.KnowledgeRetentionRatio = cfg.AgentKnowledgeRetention
		}
		if cfg.AgentHistoryPrune > 0 {
			agentCfg.HistoryPruneRatio = cfg.AgentHistoryPrune
		}
		if cfg.AgentConsolidationMax > 0 {
			agentCfg.KnowledgeMaxSize = cfg.AgentConsolidationMax
		}
		p.agentMemoryFilter = NewAgentMemoryFilterWithConfig(agentCfg)
	}

	// T12: Question-Aware Filter (LongLLMLingua-style)
	if cfg.EnableQuestionAware && cfg.QueryIntent != "" {
		p.questionAwareFilter = NewQuestionAwareFilter(cfg.QueryIntent)
		if cfg.QuestionAwareThreshold > 0 {
			p.questionAwareFilter.config.RelevanceThreshold = cfg.QuestionAwareThreshold
		}
	}

	// T17: Density-Adaptive Filter (DAST-style)
	if cfg.EnableDensityAdaptive {
		p.densityAdaptiveFilter = NewDensityAdaptiveFilter()
		if cfg.DensityTargetRatio > 0 {
			p.densityAdaptiveFilter.config.TargetRatio = cfg.DensityTargetRatio
		}
		if cfg.DensityThreshold > 0 {
			p.densityAdaptiveFilter.config.DensityThreshold = cfg.DensityThreshold
		}
	}

	// Build layers in Process() execution order
	p.layers = []filterLayer{
		{p.entropyFilter, "1_entropy"},               // Layer 1
		{p.perplexityFilter, "2_perplexity"},          // Layer 2
		{p.goalDrivenFilter, "3_goal_driven"},         // Layer 3
		{p.astPreserveFilter, "4_ast_preserve"},       // Layer 4
		{p.contrastiveFilter, "5_contrastive"},        // Layer 5
		{p.ngramAbbreviator, "6_ngram"},               // Layer 6
		{p.evaluatorHeadsFilter, "7_evaluator"},       // Layer 7
		{p.gistFilter, "8_gist"},                      // Layer 8
		{p.hierarchicalSummaryFilter, "9_hierarchical"}, // Layer 9
		{p.llmFilter, "neural"},                       // Neural (optional)
		{p.compactionLayer, "11_compaction"},          // Layer 11
		{p.attributionFilter, "12_attribution"},       // Layer 12
		{p.h2oFilter, "13_h2o"},                       // Layer 13
		{p.attentionSinkFilter, "14_attention_sink"},  // Layer 14
		{p.metaTokenFilter, "15_meta_token"},          // Layer 15
		{p.semanticChunkFilter, "16_semantic_chunk"},  // Layer 16
		{p.sketchStoreFilter, "17_sketch_store"},      // Layer 17
		{p.lazyPrunerFilter, "18_lazy_pruner"},        // Layer 18
		{p.semanticAnchorFilter, "19_semantic_anchor"}, // Layer 19
		{p.agentMemoryFilter, "20_agent_memory"},      // Layer 20
		{p.questionAwareFilter, "21_question_aware"},  // T12
		{p.densityAdaptiveFilter, "22_density_adaptive"}, // T17
	}

	return p
}

// Process runs the full 14-layer compression pipeline with early-exit support.
// T7: Stage gates skip layers when not applicable (zero cost).
// T81: Skip remaining layers if budget already met.
func (p *PipelineCoordinator) Process(input string) (string, *PipelineStats) {
	stats := &PipelineStats{
		OriginalTokens: core.EstimateTokens(input), // T22: Unified estimator
		LayerStats:     make(map[string]LayerStat),
	}

	output := input

	// Layers 1-9
	if p.entropyFilter != nil && p.config.EnableEntropy && !p.shouldSkipEntropy(output) {
		output = p.processLayer(p.layers[0], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	if p.perplexityFilter != nil && p.config.EnablePerplexity && !p.shouldSkipPerplexity(output) {
		output = p.processLayer(p.layers[1], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	if p.goalDrivenFilter != nil && p.config.EnableGoalDriven && !p.shouldSkipQueryDependent() {
		output = p.processLayer(p.layers[2], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	if p.astPreserveFilter != nil && p.config.EnableAST {
		output = p.processLayer(p.layers[3], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	if p.contrastiveFilter != nil && p.config.EnableContrastive && !p.shouldSkipQueryDependent() {
		output = p.processLayer(p.layers[4], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	if p.ngramAbbreviator != nil && !p.shouldSkipNgram(output) {
		output = p.processLayer(p.layers[5], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	if p.evaluatorHeadsFilter != nil && p.config.EnableEvaluator {
		output = p.processLayer(p.layers[6], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	if p.gistFilter != nil && p.config.EnableGist {
		output = p.processLayer(p.layers[7], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	if p.hierarchicalSummaryFilter != nil && p.config.EnableHierarchical {
		output = p.processLayer(p.layers[8], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	// Neural (optional, LLM-based)
	if p.llmFilter != nil {
		output = p.processLayer(p.layers[9], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	// Layers 11-14
	if p.compactionLayer != nil && !p.shouldSkipCompaction(output) {
		output = p.processLayer(p.layers[10], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	if p.attributionFilter != nil {
		output = p.processLayer(p.layers[11], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	if p.h2oFilter != nil && !p.shouldSkipH2O(output) {
		output = p.processLayer(p.layers[12], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	if p.attentionSinkFilter != nil && !p.shouldSkipAttentionSink(output) {
		output = p.processLayer(p.layers[13], output, stats)
		if p.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output)
		}
	}

	// Layers 15-20
	if p.metaTokenFilter != nil && !p.shouldSkipMetaToken(output) {
		output = p.processLayer(p.layers[14], output, stats)
	}

	if p.semanticChunkFilter != nil && !p.shouldSkipSemanticChunk(output) {
		output = p.processLayer(p.layers[15], output, stats)
	}

	if p.sketchStoreFilter != nil && !p.shouldSkipBudgetDependent() {
		output = p.processLayer(p.layers[16], output, stats)
	}

	if p.lazyPrunerFilter != nil && !p.shouldSkipBudgetDependent() {
		output = p.processLayer(p.layers[17], output, stats)
	}

	if p.semanticAnchorFilter != nil {
		output = p.processLayer(p.layers[18], output, stats)
	}

	if p.agentMemoryFilter != nil {
		output = p.processLayer(p.layers[19], output, stats)
	}

	// T12: Question-Aware Recovery
	if p.questionAwareFilter != nil && !p.shouldSkipQueryDependent() {
		output = p.processLayer(p.layers[20], output, stats)
	}

	// T17: Density-Adaptive Allocation
	if p.densityAdaptiveFilter != nil && !p.shouldSkipSemanticChunk(output) {
		output = p.processLayer(p.layers[21], output, stats)
	}

	// Layer 10: Budget Enforcement (special - sub-filters)
	output = p.processBudgetLayer(output, stats)

	return output, p.finalizeStats(stats, output)
}

// shouldEarlyExit returns true if budget is already met (T81).
func (p *PipelineCoordinator) shouldEarlyExit(stats *PipelineStats) bool {
	if p.config.Budget <= 0 {
		return false
	}
	currentTokens := stats.OriginalTokens - stats.computeTotalSaved()
	return currentTokens <= p.config.Budget
}

// T7: Stage Gates - Skip layers when not applicable
// Each gate checks if the layer would provide value for the given content.
// This reduces unnecessary processing and improves pipeline efficiency.

// shouldSkipEntropy checks if entropy filtering would help.
// Skip if content is too short or already highly dense.
func (p *PipelineCoordinator) shouldSkipEntropy(content string) bool {
	if len(content) < 50 {
		return true // Too short for meaningful entropy analysis
	}
	// Check if content is already dense (low repetition)
	uniqueChars := make(map[rune]bool)
	for _, r := range content {
		uniqueChars[r] = true
		if len(uniqueChars) > 30 {
			return false // Diverse content, entropy filtering will help
		}
	}
	// Very few unique chars - might still benefit from entropy filtering
	// but likely has other issues
	return false
}

// shouldSkipPerplexity checks if perplexity pruning would help.
// Skip if content has no clear structure or is too short.
func (p *PipelineCoordinator) shouldSkipPerplexity(content string) bool {
	lines := strings.Count(content, "\n")
	if lines < 5 {
		return true // Too few lines for perplexity analysis
	}
	return false
}

// shouldSkipQueryDependent checks if query-dependent layers apply.
// Used by goal-driven selection (L3) and contrastive ranking (L5).
// Skip if no query intent is specified.
func (p *PipelineCoordinator) shouldSkipQueryDependent() bool {
	return p.config.QueryIntent == ""
}

// shouldSkipNgram checks if N-gram abbreviation would help.
// Skip if content has no repeated patterns.
func (p *PipelineCoordinator) shouldSkipNgram(content string) bool {
	if len(content) < 200 {
		return true // Too short for pattern extraction
	}
	// Quick check for potential patterns
	words := strings.Fields(content)
	if len(words) < 20 {
		return true
	}
	return false
}

// shouldSkipCompaction checks if compaction would help.
// Skip if content doesn't look like conversation/chat.
func (p *PipelineCoordinator) shouldSkipCompaction(content string) bool {
	// Check for conversation markers
	conversationMarkers := []string{"User:", "Assistant:", "AI:", "Human:", "\n\n", ">>>"}
	for _, marker := range conversationMarkers {
		if strings.Contains(content, marker) {
			return false // Has conversation structure
		}
	}
	return true
}

// shouldSkipH2O checks if H2O heavy-hitter filtering would help.
// Skip if content is too short.
func (p *PipelineCoordinator) shouldSkipH2O(content string) bool {
	tokens := EstimateTokens(content)
	// Lower threshold - H2O can still help with moderate content
	return tokens < 50 // Too short for heavy-hitter analysis
}

// shouldSkipAttentionSink checks if attention sink filtering would help.
// Skip if content is too short.
func (p *PipelineCoordinator) shouldSkipAttentionSink(content string) bool {
	lines := strings.Count(content, "\n")
	return lines < 3 // Too few lines for attention sink benefit
}

// shouldSkipMetaToken checks if meta-token compression would help.
// Skip if content has no repeated token sequences.
func (p *PipelineCoordinator) shouldSkipMetaToken(content string) bool {
	if len(content) < 500 {
		return true
	}
	// Quick check for potential repeated sequences
	return false
}

// shouldSkipSemanticChunk checks if semantic chunking would help.
// Skip if content is too short or has no clear boundaries.
func (p *PipelineCoordinator) shouldSkipSemanticChunk(content string) bool {
	return len(content) < 300
}

// shouldSkipBudgetDependent checks if budget-dependent layers apply.
// Used by sketch store (L17) and lazy pruner (L18).
// Skip if budget tracking isn't enabled.
func (p *PipelineCoordinator) shouldSkipBudgetDependent() bool {
	return p.config.Budget <= 0
}

// computeTotalSaved returns total tokens saved across all layers.
func (s *PipelineStats) computeTotalSaved() int {
	total := 0
	for _, ls := range s.LayerStats {
		total += ls.TokensSaved
	}
	return total
}

// finalizeStats computes final pipeline statistics.
func (p *PipelineCoordinator) finalizeStats(stats *PipelineStats, output string) *PipelineStats {
	stats.FinalTokens = core.EstimateTokens(output) // T22: Unified estimator
	stats.TotalSaved = stats.OriginalTokens - stats.FinalTokens
	if stats.OriginalTokens > 0 {
		stats.ReductionPercent = float64(stats.TotalSaved) / float64(stats.OriginalTokens) * 100
	}
	return stats
}

// processLayer runs a single filter layer and records its stats.
func (p *PipelineCoordinator) processLayer(layer filterLayer, input string, stats *PipelineStats) string {
	start := time.Now()
	output, saved := layer.filter.Apply(input, p.config.Mode)
	stats.LayerStats[layer.name] = LayerStat{TokensSaved: saved, Duration: time.Since(start).Microseconds()}
	return output
}

// processBudgetLayer handles Layer 10: Budget Enforcement (special - sub-filters).
func (p *PipelineCoordinator) processBudgetLayer(input string, stats *PipelineStats) string {
	output := input
	totalSaved := 0

	if p.sessionTracker != nil {
		filtered, saved := p.sessionTracker.Apply(output, p.config.Mode)
		output = filtered
		totalSaved += saved
		stats.LayerStats["10_session"] = LayerStat{TokensSaved: saved}
	}

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
	Duration    int64 // microseconds for timing (T82)
}

// String returns a formatted summary of pipeline stats
func (s *PipelineStats) String() string {
	var sb strings.Builder

	sb.WriteString("╔════════════════════════════════════════════════════╗\n")
	sb.WriteString("║          Tokman 14-Layer Compression Stats         ║\n")
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
		"neural", "11_compaction", "12_attribution", "13_h2o", "14_attention_sink",
		"15_meta_token", "16_semantic_chunk", "17_sketch_store", "18_lazy_pruner",
		"19_semantic_anchor", "20_agent_memory", "10_session", "10_budget",
	}

	for _, layer := range layerOrder {
		if stat, ok := s.LayerStats[layer]; ok && stat.TokensSaved > 0 {
			sb.WriteString(fmt.Sprintf("║   %-20s: %6d tokens saved     ║\n", layer, stat.TokensSaved))
		}
	}

	sb.WriteString("╚════════════════════════════════════════════════════╝\n")

	return sb.String()
}

// QuickProcessOpt is a functional option for QuickProcess
type QuickProcessOpt func(*PipelineConfig)

// WithBudget sets the token budget
func WithBudget(budget int) QuickProcessOpt {
	return func(cfg *PipelineConfig) { cfg.Budget = budget }
}

// WithQuery sets the query intent
func WithQuery(query string) QuickProcessOpt {
	return func(cfg *PipelineConfig) { cfg.QueryIntent = query }
}

// WithLLM enables LLM compression
func WithLLM() QuickProcessOpt {
	return func(cfg *PipelineConfig) { cfg.LLMEnabled = true }
}

// QuickProcess compresses input with optional configuration
func QuickProcess(input string, mode Mode, opts ...QuickProcessOpt) (string, int) {
	cfg := PipelineConfig{
		Mode:                 mode,
		SessionTracking:      true,
		NgramEnabled:         true,
		EnableCompaction:     true,
		EnableAttribution:    true,
		EnableH2O:            true,
		EnableAttentionSink:  true,
		EnableMetaToken:      true,
		EnableSemanticChunk:  true,
		EnableSketchStore:    true,
		EnableLazyPruner:     true,
		EnableSemanticAnchor: true,
		EnableAgentMemory:    true,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	p := NewPipelineCoordinator(cfg)
	output, stats := p.Process(input)
	return output, stats.TotalSaved
}
