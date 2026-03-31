package filter

import "github.com/GrayCodeAI/tokman/internal/cache"

// Pipeline defines the interface for compression pipelines.
// This allows mock testing and future pipeline implementations.
type Pipeline interface {
	Process(input string) (string, *PipelineStats)
}

// filterLayer pairs a compression filter with its stats key.
type filterLayer struct {
	filter Filter
	name   string
}

// PipelineCoordinator orchestrates the 26-layer compression pipeline.
// Research-based: Combines the best techniques from 120+ research papers worldwide
// to achieve maximum token reduction for CLI/Agent output.
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

	// Question-Aware Filter (LongLLMLingua-style)
	questionAwareFilter *QuestionAwareFilter

	// Density-Adaptive Filter (DAST-style)
	densityAdaptiveFilter *DensityAdaptiveFilter

	// NEW: TF-IDF Coarse Filter (DSPC, Sep 2025)
	tfidfFilter *TFIDFFilter

	// NEW: Symbolic Instruction Compression (MetaGlyph, Jan 2026)
	symbolicCompressFilter *SymbolicCompressFilter

	// NEW: Phrase Grouping Filter (CompactPrompt, 2025)
	phraseGroupingFilter *PhraseGroupingFilter

	// NEW: Numerical Quantization (CompactPrompt, 2025)
	numericalQuantizer *NumericalQuantizer

	// NEW: Dynamic Compression Ratio (PruneSID, Mar 2026)
	dynamicRatioFilter *DynamicRatioFilter

	// NEW: Inter-Layer Feedback Mechanism
	feedback *InterLayerFeedback

	// NEW: Quality Estimator for feedback
	qualityEstimator *QualityEstimator

	// TOML Filter Integration (declarative filters)
	tomlFilterWrapper Filter
	tomlFilterName    string

	// Phase 2: Hypernym Concept Compression (Mercury-style)
	hypernymCompressor *HypernymCompressor

	// Phase 2: SemantiCache Clustered Merging (Mar 2026)
	semanticCacheFilter *SemanticCacheFilter

	// Phase 2: SCOPE Prefill/Decode Separation (ACL 2025)
	scopeFilter *ScopeFilter

	// Phase 2: SmallKV Model Compensation (2025)
	smallKVCompensator *SmallKVCompensator

	// Phase 2: KVzip Query-Agnostic Reconstruction (2025)
	kvzipFilter *KVzipFilter

	// Phase 2: Pipeline result cache for repeated inputs
	resultCache    *cache.FingerprintCache
	cacheEnabled   bool
	cacheHitCount  int64
	cacheMissCount int64
}

// PipelineConfig holds configuration for the compression pipeline
type PipelineConfig struct {
	Mode                    Mode
	QueryIntent             string
	Budget                  int
	LLMEnabled              bool
	SessionTracking         bool
	NgramEnabled            bool
	MultiFileEnabled        bool
	PromptTemplate          string
	EnableEntropy           bool
	EnablePerplexity        bool
	EnableGoalDriven        bool
	EnableAST               bool
	EnableContrastive       bool
	EnableEvaluator         bool
	EnableGist              bool
	EnableHierarchical      bool
	EnableCompaction        bool
	CompactionThreshold     int
	CompactionPreserveTurns int
	CompactionMaxTokens     int
	CompactionStateSnapshot bool
	CompactionAutoDetect    bool
	EnableAttribution       bool
	AttributionThreshold    float64
	EnableH2O               bool
	H2OSinkSize             int
	H2ORecentSize           int
	H2OHeavyHitterSize      int
	EnableAttentionSink     bool
	AttentionSinkCount      int
	AttentionRecentCount    int
	EnableMetaToken         bool
	MetaTokenWindow         int
	MetaTokenMinSize        int
	EnableSemanticChunk     bool
	SemanticChunkMethod     string
	SemanticChunkMinSize    int
	SemanticChunkThreshold  float64
	EnableSketchStore       bool
	SketchBudgetRatio       float64
	SketchMaxSize           int
	SketchHeavyHitter       float64
	EnableLazyPruner        bool
	LazyBaseBudget          int
	LazyDecayRate           float64
	LazyRevivalBudget       int
	EnableSemanticAnchor    bool
	SemanticAnchorRatio     float64
	SemanticAnchorSpacing   int
	EnableAgentMemory       bool
	AgentKnowledgeRetention float64
	AgentHistoryPrune       float64
	AgentConsolidationMax   int
	EnableQuestionAware     bool
	QuestionAwareThreshold  float64
	EnableDensityAdaptive   bool
	DensityTargetRatio      float64
	DensityThreshold        float64
	EnableTFIDF             bool
	TFIDFThreshold          float64
	EnableReasoningTrace    bool
	MaxReflectionLoops      int
	EnableSymbolicCompress  bool
	EnablePhraseGrouping    bool
	EnableNumericalQuant    bool
	DecimalPlaces           int
	EnableDynamicRatio      bool
	DynamicRatioBase        float64
	EnableHypernym          bool
	EnableSemanticCache     bool
	EnableScope             bool
	EnableSmallKV           bool
	EnableKVzip             bool
	EnableTOMLFilter        bool
	TOMLFilterCommand       string
}

// PipelineStats holds statistics from the compression pipeline
type PipelineStats struct {
	OriginalTokens   int
	FinalTokens      int
	TotalSaved       int
	ReductionPercent float64
	LayerStats       map[string]LayerStat
	runningSaved     int
	CacheHit         bool
}

// LayerStat holds statistics for a single layer
type LayerStat struct {
	TokensSaved int
	Duration    int64
}
