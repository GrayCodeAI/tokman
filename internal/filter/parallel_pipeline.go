package filter

import (
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ParallelPipeline processes independent layers concurrently.
// Phase 3.1: Groups layers by dependency and runs groups in parallel.
//
// Layer dependency groups:
// Group 0 (sequential): L0-TFIDF → L1-Entropy → L2-Perplexity (each depends on previous output)
// Group 1 (parallel):   L3-GoalDriven, L4-AST, L5-Contrastive (independent filters)
// Group 2 (sequential): L6-Ngram → L7-Evaluator → L8-Gist → L9-Hierarchical
// Group 3 (parallel):   L11-Compaction, L12-Attribution (independent)
// Group 4 (sequential): L13-H2O → L14-AttentionSink
// Group 5 (parallel):   L15-MetaToken, L16-SemanticChunk (independent)
// Group 6 (sequential): L17-SketchStore → L18-LazyPruner → L19-SemanticAnchor → L20-AgentMemory
// Group 7 (parallel):   L21-ReasoningTrace, L22-Symbolic, L23-PhraseGroup, L24-Numerical, L25-DynamicRatio
// Group 8 (parallel):   L26-Hypernym, L27-SemanticCache, L28-SCOPE, L29-KVzip
// Group 9 (sequential): Budget Enforcement → SmallKV Compensation
type ParallelPipeline struct {
	coordinator *PipelineCoordinator
	config      PipelineConfig
}

// NewParallelPipeline creates a parallel-capable pipeline
func NewParallelPipeline(cfg PipelineConfig) *ParallelPipeline {
	return &ParallelPipeline{
		coordinator: NewPipelineCoordinator(cfg),
		config:      cfg,
	}
}

// parallelLayerGroup is a group of layers that can run concurrently
type parallelLayerGroup struct {
	layers []filterLayer
}

// Process runs the pipeline with parallel execution where possible
func (p *ParallelPipeline) Process(input string) (string, *PipelineStats) {
	stats := &PipelineStats{
		OriginalTokens: core.EstimateTokens(input),
		LayerStats:     make(map[string]LayerStat),
	}

	output := input
	startTime := time.Now()

	// Group 0: Sequential initial filters
	output = p.processSequentialGroup([]filterLayer{
		{p.coordinator.tfidfFilter, "0_tfidf"},
		{p.coordinator.entropyFilter, "1_entropy"},
		{p.coordinator.perplexityFilter, "2_perplexity"},
	}, output, stats, func(f Filter) bool {
		switch f.(type) {
		case *TFIDFFilter:
			return p.coordinator.tfidfFilter == nil
		case *EntropyFilter:
			return p.coordinator.entropyFilter == nil || !p.config.EnableEntropy
		case *PerplexityFilter:
			return p.coordinator.perplexityFilter == nil || !p.config.EnablePerplexity
		default:
			return false
		}
	})

	if p.coordinator.shouldEarlyExit(stats) {
		return output, p.finalizeStats(stats, output, startTime)
	}

	// Group 1: Parallel intent-aware filters
	output = p.processParallelGroup([]filterLayer{
		{p.coordinator.goalDrivenFilter, "3_goal_driven"},
		{p.coordinator.astPreserveFilter, "4_ast_preserve"},
		{p.coordinator.contrastiveFilter, "5_contrastive"},
	}, output, stats, func(f Filter) bool {
		switch f.(type) {
		case *GoalDrivenFilter:
			return p.coordinator.goalDrivenFilter == nil || !p.config.EnableGoalDriven
		case *ASTPreserveFilter:
			return p.coordinator.astPreserveFilter == nil || !p.config.EnableAST
		case *ContrastiveFilter:
			return p.coordinator.contrastiveFilter == nil || !p.config.EnableContrastive
		default:
			return false
		}
	})

	if p.coordinator.shouldEarlyExit(stats) {
		return output, p.finalizeStats(stats, output, startTime)
	}

	// Group 2: Sequential mid-pipeline
	output = p.processSequentialGroup([]filterLayer{
		{p.coordinator.ngramAbbreviator, "6_ngram"},
		{p.coordinator.evaluatorHeadsFilter, "7_evaluator"},
		{p.coordinator.gistFilter, "8_gist"},
		{p.coordinator.hierarchicalSummaryFilter, "9_hierarchical"},
	}, output, stats, func(f Filter) bool {
		switch f.(type) {
		case *NgramAbbreviator:
			return p.coordinator.ngramAbbreviator == nil
		case *EvaluatorHeadsFilter:
			return p.coordinator.evaluatorHeadsFilter == nil || !p.config.EnableEvaluator
		case *GistFilter:
			return p.coordinator.gistFilter == nil || !p.config.EnableGist
		case *HierarchicalSummaryFilter:
			return p.coordinator.hierarchicalSummaryFilter == nil || !p.config.EnableHierarchical
		default:
			return false
		}
	})

	if p.coordinator.shouldEarlyExit(stats) {
		return output, p.finalizeStats(stats, output, startTime)
	}

	// Neural (optional)
	if p.coordinator.llmFilter != nil {
		output = p.coordinator.processLayer(filterLayer{p.coordinator.llmFilter, "neural"}, output, stats)
	}

	// Group 3: Parallel compaction + attribution
	output = p.processParallelGroup([]filterLayer{
		{p.coordinator.compactionLayer, "11_compaction"},
		{p.coordinator.attributionFilter, "12_attribution"},
	}, output, stats, func(f Filter) bool {
		switch f.(type) {
		case *CompactionLayer:
			return p.coordinator.compactionLayer == nil
		case *AttributionFilter:
			return p.coordinator.attributionFilter == nil
		default:
			return false
		}
	})

	if p.coordinator.shouldEarlyExit(stats) {
		return output, p.finalizeStats(stats, output, startTime)
	}

	// Group 4: Sequential H2O + AttentionSink
	output = p.processSequentialGroup([]filterLayer{
		{p.coordinator.h2oFilter, "13_h2o"},
		{p.coordinator.attentionSinkFilter, "14_attention_sink"},
	}, output, stats, func(f Filter) bool {
		switch f.(type) {
		case *H2OFilter:
			return p.coordinator.h2oFilter == nil
		case *AttentionSinkFilter:
			return p.coordinator.attentionSinkFilter == nil
		default:
			return false
		}
	})

	if p.coordinator.shouldEarlyExit(stats) {
		return output, p.finalizeStats(stats, output, startTime)
	}

	// Group 5: Parallel MetaToken + SemanticChunk
	output = p.processParallelGroup([]filterLayer{
		{p.coordinator.metaTokenFilter, "15_meta_token"},
		{p.coordinator.semanticChunkFilter, "16_semantic_chunk"},
	}, output, stats, func(f Filter) bool {
		switch f.(type) {
		case *MetaTokenFilter:
			return p.coordinator.metaTokenFilter == nil
		case *SemanticChunkFilter:
			return p.coordinator.semanticChunkFilter == nil
		default:
			return false
		}
	})

	if p.coordinator.shouldEarlyExit(stats) {
		return output, p.finalizeStats(stats, output, startTime)
	}

	// Group 6: Sequential late pipeline
	output = p.processSequentialGroup([]filterLayer{
		{p.coordinator.sketchStoreFilter, "17_sketch_store"},
		{p.coordinator.lazyPrunerFilter, "18_lazy_pruner"},
		{p.coordinator.semanticAnchorFilter, "19_semantic_anchor"},
		{p.coordinator.agentMemoryFilter, "20_agent_memory"},
	}, output, stats, func(f Filter) bool {
		switch f.(type) {
		case *SketchStoreFilter:
			return p.coordinator.sketchStoreFilter == nil
		case *LazyPrunerFilter:
			return p.coordinator.lazyPrunerFilter == nil
		case *SemanticAnchorFilter:
			return p.coordinator.semanticAnchorFilter == nil
		case *AgentMemoryFilter:
			return p.coordinator.agentMemoryFilter == nil
		default:
			return false
		}
	})

	if p.coordinator.shouldEarlyExit(stats) {
		return output, p.finalizeStats(stats, output, startTime)
	}

	// Group 7: Parallel NEW Phase 1 layers (skip if not initialized)
	var group7Layers []filterLayer
	if p.coordinator.reasoningTraceFilter != nil {
		group7Layers = append(group7Layers, filterLayer{p.coordinator.reasoningTraceFilter, "21_reasoning_trace"})
	}
	if p.coordinator.symbolicCompressFilter != nil {
		group7Layers = append(group7Layers, filterLayer{p.coordinator.symbolicCompressFilter, "22_symbolic_compress"})
	}
	if p.coordinator.phraseGroupingFilter != nil {
		group7Layers = append(group7Layers, filterLayer{p.coordinator.phraseGroupingFilter, "23_phrase_grouping"})
	}
	if p.coordinator.numericalQuantizer != nil {
		group7Layers = append(group7Layers, filterLayer{p.coordinator.numericalQuantizer, "24_numerical_quant"})
	}
	if p.coordinator.dynamicRatioFilter != nil {
		group7Layers = append(group7Layers, filterLayer{p.coordinator.dynamicRatioFilter, "25_dynamic_ratio"})
	}
	if len(group7Layers) > 0 {
		output = p.processParallelGroup(group7Layers, output, stats, func(f Filter) bool { return false })
		if p.coordinator.shouldEarlyExit(stats) {
			return output, p.finalizeStats(stats, output, startTime)
		}
	}

	// Group 8: Parallel Phase 2 layers (skip if not initialized)
	var group8Layers []filterLayer
	if p.coordinator.hypernymCompressor != nil {
		group8Layers = append(group8Layers, filterLayer{p.coordinator.hypernymCompressor, "26_hypernym"})
	}
	if p.coordinator.semanticCacheFilter != nil {
		group8Layers = append(group8Layers, filterLayer{p.coordinator.semanticCacheFilter, "27_semantic_cache"})
	}
	if p.coordinator.scopeFilter != nil {
		group8Layers = append(group8Layers, filterLayer{p.coordinator.scopeFilter, "28_scope"})
	}
	if p.coordinator.kvzipFilter != nil {
		group8Layers = append(group8Layers, filterLayer{p.coordinator.kvzipFilter, "29_kvzip"})
	}
	if len(group8Layers) > 0 {
		output = p.processParallelGroup(group8Layers, output, stats, func(f Filter) bool { return false })
	}

	if p.coordinator.shouldEarlyExit(stats) {
		return output, p.finalizeStats(stats, output, startTime)
	}

	// Question-Aware Recovery
	if p.coordinator.questionAwareFilter != nil {
		output = p.coordinator.processLayer(filterLayer{p.coordinator.questionAwareFilter, "21_question_aware"}, output, stats)
	}

	// Density-Adaptive
	if p.coordinator.densityAdaptiveFilter != nil {
		output = p.coordinator.processLayer(filterLayer{p.coordinator.densityAdaptiveFilter, "22_density_adaptive"}, output, stats)
	}

	// Group 9: Sequential budget + compensation
	output = p.coordinator.processBudgetLayer(output, stats)

	if p.coordinator.smallKVCompensator != nil {
		output = p.coordinator.smallKVCompensator.Compensate(input, output, p.config.Mode)
	}

	// Feedback
	if p.coordinator.feedback != nil && p.coordinator.qualityEstimator != nil {
		quality := p.coordinator.qualityEstimator.EstimateQuality(input, output)
		p.coordinator.feedback.RecordSignal(FeedbackSignal{
			LayerName:           "pipeline",
			QualityScore:        quality,
			CompressionRatio:    stats.ReductionPercent / 100.0,
			SuggestedAdjustment: (quality - 0.8) * 0.5,
		})
	}

	return output, p.finalizeStats(stats, output, startTime)
}

// processSequentialGroup runs layers sequentially (each depends on previous)
func (p *ParallelPipeline) processSequentialGroup(layers []filterLayer, input string, stats *PipelineStats, shouldSkip func(Filter) bool) string {
	output := input
	for _, fl := range layers {
		if fl.filter == nil || shouldSkip(fl.filter) {
			continue
		}
		output = p.coordinator.processLayer(fl, output, stats)
	}
	return output
}

// processParallelGroup runs layers in parallel, then merges results.
// Each parallel layer processes the same input independently.
// Results are merged by taking the "best" output (most tokens saved).
func (p *ParallelPipeline) processParallelGroup(layers []filterLayer, input string, stats *PipelineStats, shouldSkip func(Filter) bool) string {
	// Filter to only active layers
	var active []filterLayer
	for _, fl := range layers {
		if fl.filter != nil && !shouldSkip(fl.filter) {
			active = append(active, fl)
		}
	}

	if len(active) == 0 {
		return input
	}

	if len(active) == 1 {
		return p.coordinator.processLayer(active[0], input, stats)
	}
	type layerResult struct {
		name   string
		output string
		saved  int
	}

	results := make(chan layerResult, len(active))
	var wg sync.WaitGroup

	for _, fl := range active {
		wg.Add(1)
		go func(fl filterLayer) {
			defer wg.Done()
			out, saved := fl.filter.Apply(input, p.config.Mode)
			results <- layerResult{name: fl.name, output: out, saved: saved}
		}(fl)
	}

	wg.Wait()
	close(results)

	// Collect results
	var bestOutput string
	bestSaved := -1

	for r := range results {
		if r.saved > bestSaved {
			bestSaved = r.saved
			bestOutput = r.output
		}
		stats.LayerStats[r.name] = LayerStat{
			TokensSaved: r.saved,
		}
	}

	if bestOutput != "" && bestSaved > 0 {
		return bestOutput
	}
	return input
}

// finalizeStats computes final statistics
func (p *ParallelPipeline) finalizeStats(stats *PipelineStats, output string, startTime time.Time) *PipelineStats {
	stats.FinalTokens = core.EstimateTokens(output)
	stats.TotalSaved = stats.OriginalTokens - stats.FinalTokens
	if stats.OriginalTokens > 0 {
		stats.ReductionPercent = float64(stats.TotalSaved) / float64(stats.OriginalTokens) * 100
	}
	return stats
}
