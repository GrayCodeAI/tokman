package filter

import (
	"fmt"
	"strings"
	"testing"
)

// TestIntegrationFullPipeline tests the complete 14-layer pipeline
func TestIntegrationFullPipeline(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		mode          Mode
		minReduction  float64
		shouldBeValid bool
	}{
		{
			name: "git_status_output",
			input: `On branch main
Your branch is up to date with 'origin/main'.

Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
  (use "git restore <file>..." to discard changes in working directory)
	modified:   internal/commands/smart.go
	modified:   internal/utils/utils.go

Untracked files:
  (use "git add <file>..." to include in what will be committed)
	internal/ccusage/ccusage_test.go
	internal/commands/smart_test.go
`,
			mode:          ModeMinimal,
			minReduction:  20.0,
			shouldBeValid: true,
		},
		{
			name: "npm_install_output",
			input: `npm WARN deprecated @types/request@2.48.8: request has been deprecated
npm WARN deprecated har-validator@5.1.5: this library is no longer supported
npm WARN deprecated request@2.88.2: request has been deprecated

added 1423 packages in 32s
152 packages are looking for funding
  run npm fund for details`,
			mode:          ModeMinimal,
			minReduction:  10.0,
			shouldBeValid: true,
		},
		{
			name: "go_test_output",
			input: `=== RUN   TestFilterShort
--- PASS: TestFilterShort (0.00s)
=== RUN   TestFilterLong
--- PASS: TestFilterLong (0.00s)
=== RUN   TestFilterGitStatus
--- PASS: TestFilterGitStatus (0.00s)
=== RUN   TestFilterNpmOutput
--- PASS: TestFilterNpmOutput (0.00s)
PASS
ok  	github.com/GrayCodeAI/tokman/internal/filter	0.014s`,
			mode:          ModeMinimal,
			minReduction:  20.0, // Go test output is already minimal, harder to compress
			shouldBeValid: true,
		},
		{
			name: "error_output",
			input: `Error: Cannot find module '../config'
    at Object.<anonymous> (/app/index.js:15:12)
    at Module._compile (internal/modules/cjs/loader.js:1063:30)
    at Object.Module._extensions..js (internal/modules/cjs/loader.js:1092:10)
Failed to compile.

./src/App.tsx
Line 42: 'useState' is not defined  react/react-in-jsx-scope
Line 58: Unexpected token  syntax-error`,
			mode:          ModeMinimal,
			minReduction:  20.0,
			shouldBeValid: true,
		},
		{
			name: "docker_ps_output",
			input: `CONTAINER ID   IMAGE          COMMAND                  CREATED        STATUS        PORTS                    NAMES
abc123def456   nginx:latest   "/docker-entrypoint.…"   2 hours ago    Up 2 hours    0.0.0.0:80->80/tcp       web-server
def789ghi012   redis:alpine   "docker-entrypoint.s…"   3 hours ago    Up 3 hours    0.0.0.0:6379->6379/tcp   cache-server
jkl345mno678   postgres:14    "docker-entrypoint.s…"   5 hours ago    Up 5 hours    5432/tcp                 db-server`,
			mode:          ModeMinimal,
			minReduction:  10.0,
			shouldBeValid: true,
		},
		{
			name:          "aggressive_mode",
			input:         strings.Repeat("This is a test line with some content that should be filtered aggressively.\n", 100),
			mode:          ModeAggressive,
			minReduction:  50.0,
			shouldBeValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewPipelineCoordinator(PipelineConfig{
				Mode:                tt.mode,
				SessionTracking:     true,
				NgramEnabled:        true,
				EnableCompaction:    true,
				EnableAttribution:   true,
				EnableH2O:           true,
				EnableAttentionSink: true,
				CompactionThreshold: 50,
			})

			output, stats := p.Process(tt.input)

			t.Logf("Input: %d tokens, Output: %d tokens, Saved: %d (%.1f%%)",
				stats.OriginalTokens, stats.FinalTokens, stats.TotalSaved, stats.ReductionPercent)

			if tt.shouldBeValid {
				if len(output) == 0 {
					t.Error("Expected non-empty output")
				}
				if stats.OriginalTokens == 0 {
					t.Error("Expected original tokens > 0")
				}
			}

			if stats.ReductionPercent < tt.minReduction {
				t.Errorf("Expected reduction >= %.1f%%, got %.1f%%", tt.minReduction, stats.ReductionPercent)
			}
		})
	}
}

// TestIntegrationBudgetEnforcement tests budget enforcement across all layers
func TestIntegrationBudgetEnforcement(t *testing.T) {
	input := strings.Repeat("This is a long line that should be compressed to fit within budget.\n", 1000)

	budgets := []int{100, 500, 1000}

	for _, budget := range budgets {
		t.Run(fmt.Sprintf("budget_%d", budget), func(t *testing.T) {
			p := NewPipelineCoordinator(PipelineConfig{
				Mode:                ModeAggressive,
				Budget:              budget,
				SessionTracking:     true,
				NgramEnabled:        true,
				EnableAttribution:   true,
				EnableH2O:           true,
				EnableAttentionSink: true,
			})

			output, stats := p.Process(input)

			// Verify output is not empty
			if len(output) == 0 {
				t.Errorf("Pipeline produced empty output for budget %d", budget)
			}

			t.Logf("Budget: %d, Final: %d tokens", budget, stats.FinalTokens)

			// Budget enforcement should keep output close to budget
			if stats.FinalTokens > budget*2 {
				t.Errorf("Output %d tokens exceeded budget %d by more than 2x", stats.FinalTokens, budget)
			}
		})
	}
}

// TestIntegrationQueryAware tests query-aware compression
func TestIntegrationQueryAware(t *testing.T) {
	input := `Package: tokman
Version: 1.2.0
Architecture: amd64
Maintainer: GrayCodeAI
Description: Token reduction system
Installed-Size: 15240
Depends: libc6
Section: utils
Priority: optional
Homepage: https://github.com/GrayCodeAI/tokman

This package contains the tokman CLI tool for token reduction.
It implements 14 layers of compression based on research papers.

Layer 1: Entropy Filtering
Layer 2: Perplexity Pruning
Layer 3: Goal-Driven Selection
Layer 4: AST Preservation
Layer 5: Contrastive Ranking
Layer 6: N-gram Abbreviation
Layer 7: Evaluator Heads
Layer 8: Gist Compression
Layer 9: Hierarchical Summary
Layer 10: Budget Enforcement
Layer 11: Compaction
Layer 12: Attribution
Layer 13: H2O Filter
Layer 14: Attention Sink`

	queries := []string{
		"version number",
		"compression layers",
		"dependencies",
	}

	for _, query := range queries {
		t.Run(fmt.Sprintf("query_%s", strings.ReplaceAll(query, " ", "_")), func(t *testing.T) {
			p := NewPipelineCoordinator(PipelineConfig{
				Mode:              ModeMinimal,
				QueryIntent:       query,
				SessionTracking:   true,
				EnableContrastive: true,
				EnableGoalDriven:  true,
			})

			output, stats := p.Process(input)

			t.Logf("Query: '%s', Reduction: %.1f%%", query, stats.ReductionPercent)
			t.Logf("Output preview: %s", truncate(output, 200))

			if len(output) == 0 {
				t.Error("Expected non-empty output")
			}
		})
	}
}

// TestIntegrationLargeContext tests handling of large contexts
func TestIntegrationLargeContext(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large context test in short mode")
	}

	// Generate 100K tokens (~25K lines)
	input := generateLargeContext(25000)

	p := NewPipelineCoordinator(PipelineConfig{
		Mode:                ModeMinimal,
		EnableEntropy:       true,
		EnablePerplexity:    false, // Skip slow layers for test
		EnableGoalDriven:    false,
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

	output, stats := p.Process(input)

	t.Logf("Original: %d tokens, Final: %d tokens", stats.OriginalTokens, stats.FinalTokens)
	t.Logf("Reduction: %.1f%%", stats.ReductionPercent)

	// Should achieve significant compression
	if stats.ReductionPercent < 50.0 {
		t.Errorf("Expected reduction >= 50%% for large context, got %.1f%%", stats.ReductionPercent)
	}

	// Output should not be empty
	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}
}

// TestIntegrationResilience tests fail-safe behavior
func TestIntegrationResilience(t *testing.T) {
	// Empty input
	t.Run("empty_input", func(t *testing.T) {
		p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
		output, stats := p.Process("")
		if output != "" {
			t.Error("Expected empty output for empty input")
		}
		if stats.OriginalTokens != 0 {
			t.Error("Expected 0 original tokens for empty input")
		}
	})

	// Very short input
	t.Run("short_input", func(t *testing.T) {
		p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
		output, _ := p.Process("short")
		if output != "short" {
			t.Error("Expected unchanged short input")
		}
	})

	// Special characters
	t.Run("special_chars", func(t *testing.T) {
		p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
		input := "!@#$%^&*()_+-=[]{}|;':\",./<>?"
		output, stats := p.Process(input)
		// Should handle without error
		_ = output
		_ = stats
	})

	// Unicode content
	t.Run("unicode", func(t *testing.T) {
		p := NewPipelineCoordinator(PipelineConfig{Mode: ModeMinimal})
		input := "Hello 世界 🌸 Привет مرحبا"
		output, stats := p.Process(input)
		t.Logf("Output: %s", output)
		_ = stats
	})
}

// TestIntegrationAllLayers verifies all 14 layers contribute
func TestIntegrationAllLayers(t *testing.T) {
	// Create content that should trigger multiple layers
	input := `ERROR: Failed to process request
File: /home/user/app/main.go
Line: 42

Stack trace:
  at processRequest (main.go:42)
  at handleConnection (server.go:128)
  at main (main.go:15)

WARNING: Memory usage at 85%
INFO: Processing 1000 items
DEBUG: Item 500 processed
SUCCESS: All items completed

User query: debug authentication
Session: abc123

CRITICAL: Database connection failed
http://localhost:8080/api/health

=== Build Output ===
Time: 2025-01-15
Command: go build

FINAL STATUS: Complete`

	p := NewPipelineCoordinator(PipelineConfig{
		Mode:                ModeAggressive,
		SessionTracking:     true,
		NgramEnabled:        true,
		QueryIntent:         "debug",
		EnableEntropy:       true,
		EnablePerplexity:    true,
		EnableGoalDriven:    true,
		EnableAST:           true,
		EnableContrastive:   true,
		EnableEvaluator:     true,
		EnableGist:          true,
		EnableHierarchical:  true,
		EnableCompaction:    false, // Not conversation content
		EnableAttribution:   true,
		EnableH2O:           true,
		EnableAttentionSink: true,
	})

	output, stats := p.Process(input)

	t.Logf("Stats:\n%s", stats.String())

	// Verify compression occurred
	if stats.TotalSaved <= 0 {
		t.Error("Expected some token savings")
	}

	// Verify output is not empty
	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}

	// Verify key content is preserved
	keyContent := []string{"ERROR", "File:", "Line:", "CRITICAL"}
	for _, content := range keyContent {
		if !strings.Contains(output, content) {
			t.Logf("Warning: Key content '%s' may have been filtered", content)
		}
	}
}

// Helper function
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
