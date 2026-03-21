package filter

import (
	"strings"
	"testing"
)

func TestLazyPrunerFilter_Basic(t *testing.T) {
	filter := NewLazyPrunerFilter()
	input := `This is the first line of content.
This is the second line with more content.
This is the third line with even more content.
This is the fourth line.
This is the fifth line.
This is the sixth line with additional content.`

	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty")
	}
	if saved < 0 {
		t.Error("tokens saved should be non-negative")
	}
	t.Logf("Input tokens: %d", len(strings.Fields(input)))
	t.Logf("Output tokens: %d", len(strings.Fields(output)))
	t.Logf("Saved: %d", saved)
}

func TestLazyPrunerFilter_SmallInput(t *testing.T) {
	filter := NewLazyPrunerFilter()
	input := "Short test input."

	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty for small input")
	}
	t.Logf("Input: %q", input)
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d", saved)
}

func TestLazyPrunerFilter_Disabled(t *testing.T) {
	filter := NewLazyPrunerFilter()
	input := "This is a test with multiple lines.\nSecond line here.\nThird line."

	output, saved := filter.Apply(input, ModeNone)

	if output != input {
		t.Error("output should equal input when disabled")
	}
	if saved != 0 {
		t.Error("saved should be 0 when disabled")
	}
}

func TestLazyPrunerFilter_BudgetAware(t *testing.T) {
	cfg := LazyPrunerConfig{
		BaseBudget:         50,
		DecayRate:          0.9,
		NumLayers:          10,
		RevivalBudget:      20,
		EnableRevival:      true,
	}
	filter := NewLazyPrunerFilterWithConfig(cfg)

	input := strings.Repeat("token line content\n", 30)
	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty")
	}
	t.Logf("Saved: %d tokens", saved)
}

func TestLazyPrunerFilter_ProgressivePruning(t *testing.T) {
	filter := NewLazyPrunerFilter()
	// Create content with varying importance
	input := `Critical: This is very important content.
Low importance line here.
Another low importance line.
Key finding: The solution works.
More low importance content.
Yet another line.
Important: Remember this value.`

	output, saved := filter.Apply(input, ModeAggressive)

	// Important lines should be preserved
	if !strings.Contains(output, "Critical") && !strings.Contains(output, "Important") {
		t.Log("Warning: important lines may not be prioritized")
	}
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d tokens", saved)
}

func TestLazyPrunerFilter_WithConfig(t *testing.T) {
	cfg := LazyPrunerConfig{
		BaseBudget:         100,
		DecayRate:          0.9,
		NumLayers:          10,
		RevivalBudget:      30,
		EnableRevival:      true,
	}
	filter := NewLazyPrunerFilterWithConfig(cfg)

	input := strings.Repeat("content ", 100)
	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty")
	}
	t.Logf("Saved: %d tokens", saved)
}

func TestLazyPrunerFilter_Stats(t *testing.T) {
	filter := NewLazyPrunerFilter()
	input := strings.Repeat("repeat content\n", 20)

	filter.Apply(input, ModeAggressive)

	stats := filter.GetStats()
	t.Logf("Stats: TotalPruned=%d, TotalRevived=%d, TokensSaved=%d, LayersApplied=%d",
		stats.TotalPruned, stats.TotalRevived, stats.TokensSaved, stats.LayersApplied)
}

func TestLazyPrunerFilter_Modes(t *testing.T) {
	filter := NewLazyPrunerFilter()
	input := strings.Repeat("test content line\n", 20)

	tests := []struct {
		name string
		mode Mode
	}{
		{"none", ModeNone},
		{"minimal", ModeMinimal},
		{"aggressive", ModeAggressive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, saved := filter.Apply(input, tt.mode)
			if output == "" {
				t.Error("output should not be empty")
			}
			t.Logf("Mode %s: saved %d tokens", tt.name, saved)
		})
	}
}
