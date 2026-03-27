package benchmarks

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

// ============================================================================
// LAYER-LEVEL PROFILING TEST
// Identifies which layers cause P99 latency spikes
// ============================================================================

// LayerProfileResult contains timing data for a single layer
type LayerProfileResult struct {
	Name         string
	TotalTime    time.Duration
	CallCount    int
	AvgTime      time.Duration
	MaxTime      time.Duration
	P50Time      time.Duration
	P99Time      time.Duration
	TokensSaved  int
	IsBottleneck bool
}

// TestLayerProfiling measures individual layer performance
func TestLayerProfiling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping layer profiling in short mode")
	}

	// Generate diverse inputs to find edge cases
	testCases := generateProfileInputs()

	// Collect layer timings
	layerTimings := make(map[string][]time.Duration)
	layerTokensSaved := make(map[string]int)

	for i, tc := range testCases {
		fmt.Printf("Profiling input %d/%d (%s, %d tokens)...\n", 
			i+1, len(testCases), tc.Type, tc.TokenCount)

		cfg := filter.PipelineConfig{
			Mode:              tc.Mode,
			EnableEntropy:      true,
			EnablePerplexity:   true,
			EnableGoalDriven:   true,
			EnableAST:          true,
			EnableContrastive:  true,
			EnableEvaluator:    true,
			EnableGist:         true,
			EnableHierarchical: true,
			EnableCompaction:   true,
			EnableAttribution:  true,
			EnableH2O:          true,
			EnableAttentionSink: true,
		}

		// Profile each layer individually
		results := profilePipelineLayers(tc.Input, cfg)

		for layer, result := range results {
			layerTimings[layer] = append(layerTimings[layer], result.Duration)
			layerTokensSaved[layer] += result.TokensSaved
		}
	}

	// Analyze and report
	profileResults := analyzeLayerProfiles(layerTimings, layerTokensSaved)
	printLayerProfileReport(t, profileResults)
}

// ProfileInput represents a test input for profiling
type ProfileInput struct {
	Type       string
	Input      string
	TokenCount int
	Mode       filter.Mode
}

// LayerTiming represents a single layer execution timing
type LayerTiming struct {
	Duration    time.Duration
	TokensSaved int
}

func generateProfileInputs() []ProfileInput {
	var inputs []ProfileInput

	// Small inputs
	for i := 0; i < 50; i++ {
		inputs = append(inputs, ProfileInput{
			Type:       "small_code",
			Input:      generateCodeInput(100),
			TokenCount: 100,
			Mode:       filter.ModeMinimal,
		})
	}

	// Medium inputs
	for i := 0; i < 30; i++ {
		inputs = append(inputs, ProfileInput{
			Type:       "medium_code",
			Input:      generateCodeInput(2000),
			TokenCount: 2000,
			Mode:       filter.ModeMinimal,
		})
	}

	// Large inputs (potential P99 triggers)
	for i := 0; i < 20; i++ {
		inputs = append(inputs, ProfileInput{
			Type:       "large_code",
			Input:      generateCodeInput(20000),
			TokenCount: 20000,
			Mode:       filter.ModeAggressive,
		})
	}

	// Log inputs (different pattern)
	for i := 0; i < 20; i++ {
		inputs = append(inputs, ProfileInput{
			Type:       "logs",
			Input:      generateLogInput(5000),
			TokenCount: 5000,
			Mode:       filter.ModeAggressive,
		})
	}

	// Mixed inputs with edge cases
	for i := 0; i < 30; i++ {
		inputs = append(inputs, ProfileInput{
			Type:       "mixed_edge",
			Input:      generateMixedEdgeCase(),
			TokenCount: 3000,
			Mode:       filter.ModeMinimal,
		})
	}

	return inputs
}

// generateMixedEdgeCase creates inputs that might trigger edge cases
func generateMixedEdgeCase() string {
	var sb strings.Builder

	// Repeated patterns (caching opportunities)
	for i := 0; i < 100; i++ {
		sb.WriteString("func process() error { return nil }\n")
	}

	// Long single line (perplexity edge case)
	sb.WriteString("// ")
	for i := 0; i < 500; i++ {
		sb.WriteString("very long comment line that might cause issues ")
	}
	sb.WriteString("\n")

	// Deeply nested code (AST edge case)
	sb.WriteString("func deep() {\n")
	for i := 0; i < 50; i++ {
		sb.WriteString(strings.Repeat("\t", i) + "if condition {\n")
	}
	for i := 0; i < 50; i++ {
		sb.WriteString(strings.Repeat("\t", 49-i) + "}\n")
	}
	sb.WriteString("}\n")

	// Many short tokens (entropy edge case)
	for i := 0; i < 500; i++ {
		sb.WriteString("a b c d e f g h i j ")
	}

	return sb.String()
}

// profilePipelineLayers executes each layer individually and measures timing
func profilePipelineLayers(input string, cfg filter.PipelineConfig) map[string]LayerTiming {
	results := make(map[string]LayerTiming)

	// Create a fresh coordinator for each layer
	coord := filter.NewPipelineCoordinator(cfg)

	// Group 0: Sequential initial filters
	if tfidf := coord.GetTFIDFFilter(); tfidf != nil {
		results["0_tfidf"] = timeLayer(tfidf, input, cfg.Mode)
	}

	if entropy := coord.GetEntropyFilter(); entropy != nil && cfg.EnableEntropy {
		results["1_entropy"] = timeLayer(entropy, input, cfg.Mode)
	}

	if perplexity := coord.GetPerplexityFilter(); perplexity != nil && cfg.EnablePerplexity {
		results["2_perplexity"] = timeLayer(perplexity, input, cfg.Mode)
	}

	// Group 1: Parallel intent-aware filters
	if goalDriven := coord.GetGoalDrivenFilter(); goalDriven != nil && cfg.EnableGoalDriven {
		results["3_goal_driven"] = timeLayer(goalDriven, input, cfg.Mode)
	}

	if ast := coord.GetASTPreserveFilter(); ast != nil && cfg.EnableAST {
		results["4_ast_preserve"] = timeLayer(ast, input, cfg.Mode)
	}

	if contrastive := coord.GetContrastiveFilter(); contrastive != nil && cfg.EnableContrastive {
		results["5_contrastive"] = timeLayer(contrastive, input, cfg.Mode)
	}

	// Group 2: Sequential mid-pipeline
	if ngram := coord.GetNgramAbbreviator(); ngram != nil {
		results["6_ngram"] = timeLayer(ngram, input, cfg.Mode)
	}

	if evaluator := coord.GetEvaluatorHeadsFilter(); evaluator != nil && cfg.EnableEvaluator {
		results["7_evaluator"] = timeLayer(evaluator, input, cfg.Mode)
	}

	if gist := coord.GetGistFilter(); gist != nil && cfg.EnableGist {
		results["8_gist"] = timeLayer(gist, input, cfg.Mode)
	}

	if hierarchical := coord.GetHierarchicalSummaryFilter(); hierarchical != nil && cfg.EnableHierarchical {
		results["9_hierarchical"] = timeLayer(hierarchical, input, cfg.Mode)
	}

	// Group 3: Parallel compaction + attribution
	if compaction := coord.GetCompactionLayer(); compaction != nil {
		results["11_compaction"] = timeLayer(compaction, input, cfg.Mode)
	}

	if attribution := coord.GetAttributionFilter(); attribution != nil {
		results["12_attribution"] = timeLayer(attribution, input, cfg.Mode)
	}

	// Group 4: Sequential H2O + AttentionSink
	if h2o := coord.GetH2OFilter(); h2o != nil {
		results["13_h2o"] = timeLayer(h2o, input, cfg.Mode)
	}

	if attentionSink := coord.GetAttentionSinkFilter(); attentionSink != nil {
		results["14_attention_sink"] = timeLayer(attentionSink, input, cfg.Mode)
	}

	return results
}

// timeLayer measures a single layer's execution time
func timeLayer(f filter.Filter, input string, mode filter.Mode) LayerTiming {
	start := time.Now()
	_, saved := f.Apply(input, mode)
	return LayerTiming{
		Duration:    time.Since(start),
		TokensSaved: saved,
	}
}

// analyzeLayerProfiles computes statistics from raw timings
func analyzeLayerProfiles(timings map[string][]time.Duration, tokensSaved map[string]int) []LayerProfileResult {
	var results []LayerProfileResult

	for layer, times := range timings {
		if len(times) == 0 {
			continue
		}

		sort.Slice(times, func(i, j int) bool {
			return times[i] < times[j]
		})

		var total time.Duration
		for _, t := range times {
			total += t
		}

		p50 := times[len(times)*50/100]
		p99 := times[len(times)*99/100]
		max := times[len(times)-1]

		// Flag as bottleneck if P99 > 10ms or max > 50ms
		isBottleneck := p99 > 10*time.Millisecond || max > 50*time.Millisecond

		results = append(results, LayerProfileResult{
			Name:         layer,
			TotalTime:    total,
			CallCount:    len(times),
			AvgTime:      total / time.Duration(len(times)),
			MaxTime:      max,
			P50Time:      p50,
			P99Time:      p99,
			TokensSaved:  tokensSaved[layer],
			IsBottleneck: isBottleneck,
		})
	}

	// Sort by P99 time (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].P99Time > results[j].P99Time
	})

	return results
}

// printLayerProfileReport displays profiling results
func printLayerProfileReport(t *testing.T, results []LayerProfileResult) {
	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                         LAYER PROFILING RESULTS (P99 Bottleneck Analysis)                     ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ %-20s | %8s | %8s | %8s | %8s | %10s | %-8s ║\n",
		"Layer", "Avg", "P50", "P99", "Max", "TokensSaved", "Status")
	fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════════════════════╣\n")

	bottleneckCount := 0
	for _, r := range results {
		status := "OK"
		if r.IsBottleneck {
			status = "⚠️ SLOW"
			bottleneckCount++
		}

		fmt.Printf("║ %-20s | %8s | %8s | %8s | %8s | %10d | %-8s ║\n",
			r.Name,
			r.AvgTime.Round(time.Microsecond),
			r.P50Time.Round(time.Microsecond),
			r.P99Time.Round(time.Microsecond),
			r.MaxTime.Round(time.Microsecond),
			r.TokensSaved,
			status)
	}

	fmt.Printf("╠══════════════════════════════════════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ TOTAL BOTTLENECKS: %d layers with P99 > 10ms or Max > 50ms\n", bottleneckCount)
	fmt.Printf("╚══════════════════════════════════════════════════════════════════════════════════════════════╝\n\n")

	// Print optimization recommendations
	if bottleneckCount > 0 {
		fmt.Printf("OPTIMIZATION RECOMMENDATIONS:\n")
		for i, r := range results {
			if r.IsBottleneck {
				fmt.Printf("  %d. %s (P99: %v) - ", i+1, r.Name, r.P99Time)
				switch {
				case strings.Contains(r.Name, "entropy"):
					fmt.Printf("Use SIMD entropy calculation\n")
				case strings.Contains(r.Name, "perplexity"):
					fmt.Printf("Reduce iterations or add early-exit\n")
				case strings.Contains(r.Name, "ast"):
					fmt.Printf("Use lazy parsing or caching\n")
				case strings.Contains(r.Name, "hierarchical"):
					fmt.Printf("Implement hierarchical shortcuts\n")
				default:
					fmt.Printf("Profile and optimize inner loop\n")
				}
			}
		}
		fmt.Printf("\n")
	}
}
