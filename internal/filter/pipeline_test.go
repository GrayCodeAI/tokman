package filter

import (
	"strings"
	"testing"
)

func TestTenLayerPipeline(t *testing.T) {
	// Test input with various content types
	input := `# Project Title

This is a long introduction with some verbose content that should be compressed.

## Section 1
Error: something went wrong in this section.
Warning: this is a warning message.

func main() {
    fmt.Println("Hello, World!")
}

## Section 2
More content here that repeats repeats repeats.
The quick brown fox jumps over the lazy dog.
The quick brown fox jumps over the lazy dog.
The quick brown fox jumps over the lazy dog.
`

	// Test with different modes
	tests := []struct {
		name string
		mode Mode
	}{
		{"Minimal", ModeMinimal},
		{"Aggressive", ModeAggressive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordinator := NewPipelineCoordinator(PipelineConfig{
				Mode:            tt.mode,
				SessionTracking: true,
				NgramEnabled:    true,
			})

			output, stats := coordinator.Process(input)

			// Verify output is not empty
			if output == "" {
				t.Error("output should not be empty")
			}

			// Verify compression occurred
			if len(output) > len(input) {
				t.Errorf("output (%d bytes) should be <= input (%d bytes)", len(output), len(input))
			}

			// Verify we have layer stats
			if len(stats.LayerStats) == 0 {
				t.Error("expected layer stats to be populated")
			}

			// Verify token savings reported
			if stats.OriginalTokens <= 0 {
				t.Error("original tokens should be positive")
			}

			// Verify some compression happened
			if stats.TotalSaved < 0 {
				t.Error("total saved should be non-negative")
			}
		})
	}
}

func TestPipelineBudget(t *testing.T) {
	coordinator := NewPipelineCoordinator(PipelineConfig{
		Mode:            ModeAggressive,
		Budget:          100,
		SessionTracking: true,
		NgramEnabled:    true,
	})

	// Long input that needs budget enforcement
	input := strings.Repeat("This is a test line that will be compressed.\n", 100)

	output, stats := coordinator.Process(input)

	// Output should be constrained by budget
	tokens := len(output) / 4 // rough token estimate
	if tokens > 150 {         // allow some slack
		t.Errorf("output tokens (%d) should be close to budget (100)", tokens)
	}

	// Verify budget layer ran
	foundBudget := false
	for layer := range stats.LayerStats {
		if strings.Contains(layer, "budget") {
			foundBudget = true
			break
		}
	}
	if !foundBudget {
		t.Error("budget layer should be in stats")
	}
}

func TestQuickProcess(t *testing.T) {
	input := "This is a test with repeated repeated repeated content."

	output, saved := QuickProcess(input, ModeAggressive)

	t.Logf("Input: %q", input)
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d", saved)

	if output == "" {
		t.Error("output should not be empty")
	}

	if saved < 0 {
		t.Error("tokens saved should be non-negative")
	}
}
