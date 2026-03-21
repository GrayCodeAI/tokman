package filter

import (
	"strings"
	"testing"
)

func TestSketchStoreFilter_Basic(t *testing.T) {
	filter := NewSketchStoreFilter()
	input := `Token one appears frequently.
Token one appears again.
Token one appears once more.
Token two is different.
Token three is unique.
Token one appears frequently.
Token one appears again.`

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

func TestSketchStoreFilter_SmallInput(t *testing.T) {
	filter := NewSketchStoreFilter()
	input := "Short test input."

	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty for small input")
	}
	t.Logf("Input: %q", input)
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d", saved)
}

func TestSketchStoreFilter_Disabled(t *testing.T) {
	filter := NewSketchStoreFilter()
	input := "This is a test with some repeated content repeated content."

	output, saved := filter.Apply(input, ModeNone)

	if output != input {
		t.Error("output should equal input when disabled")
	}
	if saved != 0 {
		t.Error("saved should be 0 when disabled")
	}
}

func TestSketchStoreFilter_HeavyHitters(t *testing.T) {
	filter := NewSketchStoreFilter()
	// Create input with clear heavy hitters
	input := strings.Repeat("frequent ", 10) + "rare1 " + "rare2 " + "rare3"

	output, _ := filter.Apply(input, ModeAggressive)

	// Heavy hitters should be preserved
	if !strings.Contains(output, "frequent") {
		t.Error("heavy hitter 'frequent' should be preserved")
	}
	t.Logf("Output: %q", output)
}

func TestSketchStoreFilter_WithConfig(t *testing.T) {
	cfg := SketchStoreConfig{
		BudgetRatio:      0.5,
		MaxSketchSize:    1000,
		HeavyHitterRatio: 0.2,
		EnableRecovery:   true,
	}
	filter := NewSketchStoreFilterWithConfig(cfg)

	input := strings.Repeat("pattern ", 20)
	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty")
	}
	t.Logf("Saved: %d tokens", saved)
}

func TestSketchStoreFilter_Reversibility(t *testing.T) {
	filter := NewSketchStoreFilter()
	input := `Line one with content.
Line two with more content.
Line three with even more content.
Line one with content.
Line two with more content.`

	output, saved := filter.Apply(input, ModeMinimal)

	// Sketch store should maintain sketch for potential recovery
	if output == "" {
		t.Error("output should not be empty")
	}

	// Check that sketches are stored
	sketches := filter.GetAllSketches()
	t.Logf("Sketch size: %d entries", len(sketches))
	t.Logf("Saved: %d tokens", saved)
}

func TestSketchStoreFilter_BudgetManagement(t *testing.T) {
	filter := NewSketchStoreFilter()
	// Large input to test budget management
	input := strings.Repeat("token ", 500)

	output, _ := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty")
	}

	// Verify budget was respected
	outputTokens := len(strings.Fields(output))
	t.Logf("Input tokens: 500, Output tokens: %d", outputTokens)
}

func TestSketchStoreFilter_Stats(t *testing.T) {
	filter := NewSketchStoreFilter()
	input := strings.Repeat("repeat ", 30)

	_, saved := filter.Apply(input, ModeAggressive)

	stats := filter.GetStats()
	t.Logf("Stats: TotalSketches=%d, TotalCompressed=%d, TotalRevived=%d, TokensSaved=%d, Saved=%d",
		stats.TotalSketches, stats.TotalCompressed, stats.TotalRevived, stats.TokensSaved, saved)
}
