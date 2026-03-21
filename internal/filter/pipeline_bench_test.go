package filter

import (
	"fmt"
	"strings"
	"testing"
)

// BenchmarkPipelineFull benchmarks the complete 14-layer pipeline
func BenchmarkPipelineFull(b *testing.B) {
	input := generateTestContent(1000) // 1000 lines

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p := NewPipelineCoordinator(PipelineConfig{
			Mode:                ModeMinimal,
			SessionTracking:     true,
			NgramEnabled:        true,
			EnableCompaction:    true,
			EnableAttribution:   true,
			EnableH2O:           true,
			EnableAttentionSink: true,
		})
		p.Process(input)
	}
}

// BenchmarkPipelineScaling tests performance across different input sizes
func BenchmarkPipelineScaling(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("lines_%d", size), func(b *testing.B) {
			input := generateTestContent(size)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := NewPipelineCoordinator(PipelineConfig{
					Mode:                ModeMinimal,
					SessionTracking:     true,
					NgramEnabled:        true,
					EnableCompaction:    false, // Disable LLM for benchmarks
					EnableAttribution:   true,
					EnableH2O:           true,
					EnableAttentionSink: true,
				})
				p.Process(input)
			}
		})
	}
}

// BenchmarkPipelineContextSizes tests large context handling
// Important: Validates 1M-2M token capacity
func BenchmarkPipelineContextSizes(b *testing.B) {
	// Token sizes: roughly 100K, 500K, 1M, 2M tokens
	// ~4 chars per token, ~80 chars per line
	contextSizes := []int{
		25000,  // ~100K tokens (25K lines * 4 tokens/line)
		125000, // ~500K tokens
		250000, // ~1M tokens
		500000, // ~2M tokens
	}

	for _, lines := range contextSizes {
		b.Run(fmt.Sprintf("tokens_%dK", (lines*4)/1000), func(b *testing.B) {
			input := generateLargeContext(lines)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				p := NewPipelineCoordinator(PipelineConfig{
					Mode:                ModeMinimal,
					EnableEntropy:       true,
					EnablePerplexity:    true,
					EnableGoalDriven:    false, // Skip for performance
					EnableAST:           true,
					EnableContrastive:   false,
					EnableEvaluator:     true,
					EnableGist:          true,
					EnableHierarchical:  false,
					EnableCompaction:    false,
					EnableAttribution:   true,
					EnableH2O:           true,
					EnableAttentionSink: true,
				})
				_, _ = p.Process(input)
			}
		})
	}
}

// BenchmarkLayerByLayer isolates each layer's performance
func BenchmarkLayerByLayer(b *testing.B) {
	input := generateTestContent(1000)

	layers := []struct {
		name string
		fn   func(string, Mode) (string, int)
	}{
		{"1_entropy", NewEntropyFilter().Apply},
		{"2_perplexity", NewPerplexityFilter().Apply},
		{"4_ast", NewASTPreserveFilter().Apply},
		{"7_evaluator", NewEvaluatorHeadsFilter().Apply},
		{"8_gist", NewGistFilter().Apply},
		{"12_attribution", NewAttributionFilter().Apply},
		{"13_h2o", NewH2OFilter().Apply},
		{"14_attention_sink", NewAttentionSinkFilter().Apply},
	}

	for _, layer := range layers {
		b.Run(layer.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				layer.fn(input, ModeMinimal)
			}
		})
	}
}

// BenchmarkTokenEstimation tests the token counting performance
func BenchmarkTokenEstimation(b *testing.B) {
	inputs := []struct {
		name  string
		input string
	}{
		{"small", strings.Repeat("test line\n", 100)},
		{"medium", strings.Repeat("test line with more content\n", 1000)},
		{"large", strings.Repeat("test line with even more content for benchmarking\n", 5000)},
	}

	for _, tc := range inputs {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				EstimateTokens(tc.input)
			}
		})
	}
}

// TestPipelineCompressionRatio validates compression effectiveness
func TestPipelineCompressionRatio(t *testing.T) {
	testCases := []struct {
		name         string
		lines        int
		minReduction float64 // Minimum expected reduction percentage
	}{
		{"small_output", 100, 20.0},
		{"medium_output", 1000, 30.0},
		{"large_output", 5000, 40.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			input := generateTestContent(tc.lines)

			p := NewPipelineCoordinator(PipelineConfig{
				Mode:                ModeAggressive,
				SessionTracking:     true,
				NgramEnabled:        true,
				EnableCompaction:    false,
				EnableAttribution:   true,
				EnableH2O:           true,
				EnableAttentionSink: true,
			})

			output, stats := p.Process(input)

			t.Logf("Original: %d tokens, Final: %d tokens, Saved: %d (%.1f%%)",
				stats.OriginalTokens, stats.FinalTokens, stats.TotalSaved, stats.ReductionPercent)

			if stats.ReductionPercent < tc.minReduction {
				t.Errorf("Expected reduction >= %.1f%%, got %.1f%%", tc.minReduction, stats.ReductionPercent)
			}

			if len(output) == 0 {
				t.Error("Output should not be empty")
			}
		})
	}
}

// TestPipelineLayerActivation verifies the pipeline produces valid output
func TestPipelineLayerActivation(t *testing.T) {
	input := generateTestContent(500)

	p := NewPipelineCoordinator(PipelineConfig{
		Mode:                ModeMinimal,
		SessionTracking:     true,
		NgramEnabled:        true,
		QueryIntent:         "test query",
		EnableCompaction:    true,
		EnableAttribution:   true,
		EnableH2O:           true,
		EnableAttentionSink: true,
		CompactionThreshold: 100, // Low threshold to trigger
	})

	output, stats := p.Process(input)

	// Verify compression happened
	if stats.OriginalTokens == 0 {
		t.Error("Expected original tokens > 0")
	}

	// Verify output is not empty
	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}

	// Verify some compression occurred
	if stats.TotalSaved <= 0 {
		t.Errorf("Expected total saved tokens > 0, got %d", stats.TotalSaved)
	}

	// Verify at least some layers contributed
	if len(stats.LayerStats) == 0 {
		t.Error("Expected at least one layer to contribute")
	}

	t.Logf("Pipeline stats:\n%s", stats.String())
}

// generateTestContent creates realistic test content
func generateTestContent(lines int) string {
	var sb strings.Builder

	// Add headers
	sb.WriteString("=== Build Output ===\n")
	sb.WriteString("Time: 2025-01-15 10:30:00\n")
	sb.WriteString("Command: npm run build\n\n")

	// Add content lines
	for i := 0; i < lines; i++ {
		switch i % 10 {
		case 0:
			sb.WriteString(fmt.Sprintf("INFO: Processing file %d of %d\n", i, lines))
		case 1:
			sb.WriteString(fmt.Sprintf("DEBUG: Module %d loaded successfully\n", i))
		case 2:
			sb.WriteString(fmt.Sprintf("File: /home/user/project/src/module%d.go\n", i))
		case 3:
			sb.WriteString(fmt.Sprintf("Line: %d - Function: processItem\n", i*10))
		case 4:
			sb.WriteString("WARNING: Deprecated API usage detected\n")
		case 5:
			sb.WriteString(fmt.Sprintf("http://localhost:3000/api/item/%d\n", i))
		case 6:
			sb.WriteString("Error: Connection timeout after 30 seconds\n")
		case 7:
			sb.WriteString("SUCCESS: Build completed successfully\n")
		case 8:
			sb.WriteString("CRITICAL: Memory usage at 85%\n")
		case 9:
			sb.WriteString(fmt.Sprintf("item_%d: { id: %d, name: 'test item', value: %d }\n", i, i, i*100))
		}
	}

	// Add footer
	sb.WriteString("\n=== Build Complete ===\n")
	sb.WriteString("Total files: 1000\n")
	sb.WriteString("Build time: 45 seconds\n")

	return sb.String()
}

// generateLargeContext creates large context for 1M-2M token tests
func generateLargeContext(lines int) string {
	var sb strings.Builder

	// Simulate a large codebase scan or log output
	sb.WriteString("=== Large Context Processing ===\n")
	sb.WriteString("Context Size: Large\n\n")

	chunk := strings.Repeat("Line with content for testing large context handling. ", 20) + "\n"

	for i := 0; i < lines; i++ {
		if i%1000 == 0 {
			sb.WriteString(fmt.Sprintf("\n--- Section %d ---\n", i/1000))
		}
		sb.WriteString(chunk)
	}

	sb.WriteString("\n=== End of Context ===\n")

	return sb.String()
}
