package filter

import (
	"strings"
	"testing"
)

// T17: Density-Adaptive Filter Tests

func TestDensityAdaptiveFilter_Basic(t *testing.T) {
	filter := NewDensityAdaptiveFilter()

	// Create content with varying density
	input := strings.Repeat("This is a normal line with some content.\n", 20)
	input += "func main() {\n\tfmt.Println(\"dense code block\")\n}\n"
	input += strings.Repeat("Comment line with low information density.\n", 20)

	output, saved := filter.Apply(input, ModeMinimal)

	if len(output) == 0 {
		t.Error("Output should not be empty")
	}

	// Dense code section should be preserved
	if !strings.Contains(output, "func main") {
		t.Error("Dense code section should be preserved")
	}

	if saved <= 0 {
		t.Error("Expected some tokens to be saved")
	}
}

func TestDensityAdaptiveFilter_ShortContent(t *testing.T) {
	filter := NewDensityAdaptiveFilter()

	// Too short for density analysis
	input := "Short content"
	output, saved := filter.Apply(input, ModeMinimal)

	if output != input {
		t.Error("Short content should be unchanged")
	}

	if saved != 0 {
		t.Error("No savings expected for short content")
	}
}

func TestDensityAdaptiveFilter_LineDensity(t *testing.T) {
	filter := NewDensityAdaptiveFilter()

	// Code line should have higher density
	codeDensity := filter.calculateLineDensity("func main() { fmt.Println(\"hello\"); }")
	// Empty line should have lowest density
	emptyDensity := filter.calculateLineDensity("")
	// Comment line should have medium density
	commentDensity := filter.calculateLineDensity("// This is a comment")

	if codeDensity <= emptyDensity {
		t.Errorf("Code line should have higher density than empty: code=%f, empty=%f", codeDensity, emptyDensity)
	}

	if codeDensity <= commentDensity {
		t.Errorf("Code line should have higher density than comment: code=%f, comment=%f", codeDensity, commentDensity)
	}
}

func TestDensityAdaptiveFilter_WindowDensities(t *testing.T) {
	filter := NewDensityAdaptiveFilter()

	densities := []float64{0.1, 0.2, 0.8, 0.9, 0.3, 0.2, 0.1}
	windowDensities := filter.calculateWindowDensities(densities)

	// Verify the function produces valid output
	if len(windowDensities) != len(densities) {
		t.Errorf("Window densities length mismatch: got %d, want %d", len(windowDensities), len(densities))
	}

	// All values should be non-negative
	for i, d := range windowDensities {
		if d < 0 {
			t.Errorf("Window density should be non-negative at index %d, got %f", i, d)
		}
	}

	// Values should be smoothed (not identical to input)
	hasSmoothing := false
	for i := range densities {
		if windowDensities[i] != densities[i] {
			hasSmoothing = true
			break
		}
	}

	if !hasSmoothing && len(densities) > filter.config.WindowSize {
		t.Error("Expected some smoothing effect from window calculation")
	}
}

func TestDensityAdaptiveFilter_BudgetAllocation(t *testing.T) {
	filter := NewDensityAdaptiveFilter()
	filter.config.TargetRatio = 0.5

	lines := []string{
		"Line 1", "Line 2", "Line 3",
		"Dense code line with important content",
		"func importantFunction() { return 42; }",
		"Line 6", "Line 7", "Line 8", "Line 9", "Line 10",
		"Line 11", "Line 12",
	}

	densities := make([]float64, len(lines))
	for i, line := range lines {
		densities[i] = filter.calculateLineDensity(line)
	}

	windowDensities := filter.calculateWindowDensities(densities)

	keep := filter.allocateBudget(lines, windowDensities, 100)

	// First and last should always be kept
	if !keep[0] || !keep[len(lines)-1] {
		t.Error("First and last lines should always be kept")
	}
}

func TestDensityAdaptiveFilter_Disabled(t *testing.T) {
	filter := NewDensityAdaptiveFilter()
	filter.SetEnabled(false)

	input := strings.Repeat("Test content\n", 50)
	output, saved := filter.Apply(input, ModeMinimal)

	if output != input {
		t.Error("Disabled filter should return original")
	}

	if saved != 0 {
		t.Error("No savings expected when disabled")
	}
}

func TestDensityAdaptiveFilter_Configuration(t *testing.T) {
	filter := NewDensityAdaptiveFilter()
	filter.SetTargetRatio(0.3)

	stats := filter.GetStats()

	if stats["target_ratio"] != 0.3 {
		t.Errorf("Expected target_ratio=0.3, got %v", stats["target_ratio"])
	}
}
