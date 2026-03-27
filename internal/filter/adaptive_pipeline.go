package filter

import (
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// Tier defines complexity-based layer selection.
// 20 highly impactful layers across 4 tiers.
//
// Research: PoC (2026), ATACompressor (2025), METok (2025), D-Rank (2026)
//
//	Tier 0 (Trivial): 0 layers  - empty/whitespace
//	Tier 1 (Simple):  3 layers  - git status, ls, docker ps    <0.5ms
//	Tier 2 (Medium):  8 layers  - git diff, tests, builds      <2ms
//	Tier 3 (Full):   20 layers  - large code, reasoning, logs   <15ms
type Tier int

const (
	Tier0_Trivial Tier = iota // 0 layers
	Tier1_Simple              // 3 layers
	Tier2_Medium              // 8 layers
	Tier3_Full                // 20 layers
)

func (t Tier) String() string {
	switch t {
	case Tier0_Trivial:
		return "trivial(0L)"
	case Tier1_Simple:
		return "simple(3L)"
	case Tier2_Medium:
		return "medium(8L)"
	case Tier3_Full:
		return "full(20L)"
	default:
		return "unknown"
	}
}

func (t Tier) LayerCount() int {
	switch t {
	case Tier0_Trivial:
		return 0
	case Tier1_Simple:
		return 3
	case Tier2_Medium:
		return 8
	case Tier3_Full:
		return 20
	default:
		return 0
	}
}

// AdaptivePipeline selects 0-20 layers based on content complexity.
type AdaptivePipeline struct {
	c   *PipelineCoordinator
	cfg PipelineConfig
}

func NewAdaptive(cfg PipelineConfig) *AdaptivePipeline {
	return &AdaptivePipeline{
		c:   NewPipelineCoordinator(cfg),
		cfg: cfg,
	}
}

// DetectTier analyzes content and selects the right tier
func (p *AdaptivePipeline) DetectTier(input string) Tier {
	trimmed := strings.TrimSpace(input)
	if len(trimmed) == 0 {
		return Tier0_Trivial
	}

	tokens := core.EstimateTokens(input)
	lines := strings.Split(input, "\n")

	codeLines := 0
	errorLines := 0
	for _, line := range lines {
		lower := strings.ToLower(strings.TrimSpace(line))
		if strings.Contains(lower, "func ") || strings.Contains(lower, "class ") ||
			strings.Contains(lower, "import ") || strings.Contains(lower, "{") {
			codeLines++
		}
		if strings.Contains(lower, "error") || strings.Contains(lower, "fail") ||
			strings.Contains(lower, "panic") {
			errorLines++
		}
	}

	switch {
	case tokens < 50:
		return Tier1_Simple
	case tokens < 300 && codeLines == 0 && errorLines == 0:
		return Tier1_Simple
	case tokens < 1000 && codeLines < 5:
		return Tier2_Medium
	default:
		return Tier3_Full
	}
}

// Process runs the appropriate tier
func (p *AdaptivePipeline) Process(input string) (string, *PipelineStats) {
	tier := p.DetectTier(input)

	switch tier {
	case Tier0_Trivial:
		return input, &PipelineStats{OriginalTokens: 0}

	case Tier1_Simple:
		return p.runSimple(input)

	case Tier2_Medium:
		return p.runMedium(input)

	case Tier3_Full:
		return p.runFull(input)

	default:
		return p.c.Process(input)
	}
}

// runSimple: 3 layers - Entropy + Ngram + Budget
func (p *AdaptivePipeline) runSimple(input string) (string, *PipelineStats) {
	stats := &PipelineStats{
		OriginalTokens: core.EstimateTokens(input),
		LayerStats:     make(map[string]LayerStat),
	}
	out := input

	if p.c.entropyFilter != nil {
		o, s := p.c.entropyFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["entropy"] = LayerStat{TokensSaved: s}
	}
	if p.c.ngramAbbreviator != nil {
		o, s := p.c.ngramAbbreviator.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["ngram"] = LayerStat{TokensSaved: s}
	}
	out = p.c.processBudgetLayer(out, stats)

	stats.FinalTokens = core.EstimateTokens(out)
	stats.TotalSaved = stats.OriginalTokens - stats.FinalTokens
	if stats.OriginalTokens > 0 {
		stats.ReductionPercent = float64(stats.TotalSaved) / float64(stats.OriginalTokens) * 100
	}
	return out, stats
}

// runMedium: 8 layers - Entropy + Perplexity + AST + Ngram + Attribution + Numerical + Dynamic + Budget
func (p *AdaptivePipeline) runMedium(input string) (string, *PipelineStats) {
	stats := &PipelineStats{
		OriginalTokens: core.EstimateTokens(input),
		LayerStats:     make(map[string]LayerStat),
	}
	out := input

	layers := []struct {
		f    Filter
		name string
	}{
		{p.c.entropyFilter, "entropy"},
		{p.c.perplexityFilter, "perplexity"},
		{p.c.astPreserveFilter, "ast"},
		{p.c.ngramAbbreviator, "ngram"},
		{p.c.attributionFilter, "attribution"},
		{p.c.numericalQuantizer, "numerical"},
		{p.c.dynamicRatioFilter, "dynamic"},
	}

	for _, l := range layers {
		if l.f != nil {
			o, s := l.f.Apply(out, p.cfg.Mode)
			if s > 0 {
				out = o
			}
			stats.LayerStats[l.name] = LayerStat{TokensSaved: s}
		}
	}
	out = p.c.processBudgetLayer(out, stats)

	stats.FinalTokens = core.EstimateTokens(out)
	stats.TotalSaved = stats.OriginalTokens - stats.FinalTokens
	if stats.OriginalTokens > 0 {
		stats.ReductionPercent = float64(stats.TotalSaved) / float64(stats.OriginalTokens) * 100
	}
	return out, stats
}

// runFull: 20 layers - all impactful layers
//
// The 20 layers:
//  1. Entropy         - fast pre-filter (Selective Context 2023)
//  2. Perplexity      - core 20x compression (LLMLingua 2023)
//  3. Goal-Driven     - intent-aware selection (SWE-Pruner 2025)
//  4. AST Preserve    - code structure (LongCodeZip 2025)
//  5. N-gram          - lossless abbreviation (CompactPrompt 2025)
//  6. Hierarchical    - recursive summarization (AutoCompressor 2023)
//  7. Budget          - token limit enforcement
//  8. Compaction      - semantic compression (MemGPT 2023)
//  9. Attribution     - token importance scoring (ProCut 2025)
//
// 10. H2O             - heavy-hitter detection (NeurIPS 2023)
// 11. Attention Sink  - stability preservation (StreamingLLM 2023)
// 12. Semantic Chunk  - chunk-level compression (ChunkKV 2025)
// 13. Lazy Pruner     - progressive pruning (LazyLLM 2024)
// 14. Semantic Anchor - context preservation
// 15. Reasoning Trace - CoT compression (R-KV 2025)
// 16. Symbolic        - instruction compression (MetaGlyph 2026)
// 17. Numerical       - structured data (CompactPrompt 2025)
// 18. Dynamic Ratio   - adaptive compression (PruneSID 2026)
// 19. SCOPE           - prefill/decode (ACL 2025)
// 20. DynaKV          - token-wise adaptive (arXiv 2026)
func (p *AdaptivePipeline) runFull(input string) (string, *PipelineStats) {
	stats := &PipelineStats{
		OriginalTokens: core.EstimateTokens(input),
		LayerStats:     make(map[string]LayerStat),
	}
	out := input

	// Layer 1: Entropy
	if p.c.entropyFilter != nil {
		o, s := p.c.entropyFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["1_entropy"] = LayerStat{TokensSaved: s}
	}
	// Layer 2: Perplexity
	if p.c.perplexityFilter != nil {
		o, s := p.c.perplexityFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["2_perplexity"] = LayerStat{TokensSaved: s}
	}
	// Layer 3: Goal-Driven
	if p.c.goalDrivenFilter != nil {
		o, s := p.c.goalDrivenFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["3_goal_driven"] = LayerStat{TokensSaved: s}
	}
	// Layer 4: AST Preserve
	if p.c.astPreserveFilter != nil {
		o, s := p.c.astPreserveFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["4_ast"] = LayerStat{TokensSaved: s}
	}
	// Layer 5: N-gram
	if p.c.ngramAbbreviator != nil {
		o, s := p.c.ngramAbbreviator.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["5_ngram"] = LayerStat{TokensSaved: s}
	}
	// Layer 6: Hierarchical
	if p.c.hierarchicalSummaryFilter != nil {
		o, s := p.c.hierarchicalSummaryFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["6_hierarchical"] = LayerStat{TokensSaved: s}
	}
	// Layer 8: Compaction
	if p.c.compactionLayer != nil {
		o, s := p.c.compactionLayer.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["8_compaction"] = LayerStat{TokensSaved: s}
	}
	// Layer 9: Attribution
	if p.c.attributionFilter != nil {
		o, s := p.c.attributionFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["9_attribution"] = LayerStat{TokensSaved: s}
	}
	// Layer 10: H2O
	if p.c.h2oFilter != nil {
		o, s := p.c.h2oFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["10_h2o"] = LayerStat{TokensSaved: s}
	}
	// Layer 11: Attention Sink
	if p.c.attentionSinkFilter != nil {
		o, s := p.c.attentionSinkFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["11_attention_sink"] = LayerStat{TokensSaved: s}
	}
	// Layer 12: Semantic Chunk
	if p.c.semanticChunkFilter != nil {
		o, s := p.c.semanticChunkFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["12_semantic_chunk"] = LayerStat{TokensSaved: s}
	}
	// Layer 13: Lazy Pruner
	if p.c.lazyPrunerFilter != nil {
		o, s := p.c.lazyPrunerFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["13_lazy_pruner"] = LayerStat{TokensSaved: s}
	}
	// Layer 14: Semantic Anchor
	if p.c.semanticAnchorFilter != nil {
		o, s := p.c.semanticAnchorFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["14_semantic_anchor"] = LayerStat{TokensSaved: s}
	}
	// Layer 15: Reasoning Trace
	if p.c.reasoningTraceFilter != nil {
		o, s := p.c.reasoningTraceFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["15_reasoning"] = LayerStat{TokensSaved: s}
	}
	// Layer 16: Symbolic
	if p.c.symbolicCompressFilter != nil {
		o, s := p.c.symbolicCompressFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["16_symbolic"] = LayerStat{TokensSaved: s}
	}
	// Layer 17: Numerical
	if p.c.numericalQuantizer != nil {
		o, s := p.c.numericalQuantizer.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["17_numerical"] = LayerStat{TokensSaved: s}
	}
	// Layer 18: Dynamic Ratio
	if p.c.dynamicRatioFilter != nil {
		o, s := p.c.dynamicRatioFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["18_dynamic"] = LayerStat{TokensSaved: s}
	}
	// Layer 19: SCOPE
	if p.c.scopeFilter != nil {
		o, s := p.c.scopeFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["19_scope"] = LayerStat{TokensSaved: s}
	}
	// Layer 20: Agent Memory (Focus-inspired)
	if p.c.agentMemoryFilter != nil {
		o, s := p.c.agentMemoryFilter.Apply(out, p.cfg.Mode)
		if s > 0 {
			out = o
		}
		stats.LayerStats["20_agent_memory"] = LayerStat{TokensSaved: s}
	}

	// Layer 7: Budget (always last)
	out = p.c.processBudgetLayer(out, stats)

	stats.FinalTokens = core.EstimateTokens(out)
	stats.TotalSaved = stats.OriginalTokens - stats.FinalTokens
	if stats.OriginalTokens > 0 {
		stats.ReductionPercent = float64(stats.TotalSaved) / float64(stats.OriginalTokens) * 100
	}
	return out, stats
}
