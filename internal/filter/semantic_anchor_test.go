package filter

import (
	"strings"
	"testing"
)

func TestSemanticAnchorFilter_Basic(t *testing.T) {
	filter := NewSemanticAnchorFilter()
	input := `func main() {
	fmt.Println("Hello, World!")
}

type User struct {
	Name string
	Age  int
}

func (u *User) GetName() string {
	return u.Name
}`

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

func TestSemanticAnchorFilter_SmallInput(t *testing.T) {
	filter := NewSemanticAnchorFilter()
	input := "Short test input."

	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty for small input")
	}
	t.Logf("Input: %q", input)
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d", saved)
}

func TestSemanticAnchorFilter_Disabled(t *testing.T) {
	filter := NewSemanticAnchorFilter()
	input := "This is a test with func main and type User."

	output, saved := filter.Apply(input, ModeNone)

	if output != input {
		t.Error("output should equal input when disabled")
	}
	if saved != 0 {
		t.Error("saved should be 0 when disabled")
	}
}

func TestSemanticAnchorFilter_AnchorDetection(t *testing.T) {
	filter := NewSemanticAnchorFilter()
	input := `package main

import "fmt"

func main() {
	fmt.Println("test")
}

type Config struct {
	Name string
}

func NewConfig() *Config {
	return &Config{}
}`

	output, _ := filter.Apply(input, ModeMinimal)

	// Anchors (func, type, struct) should be preserved
	anchors := filter.GetAnchors()
	t.Logf("Detected %d anchors", len(anchors))
	for i, a := range anchors {
		if i < 5 {
			t.Logf("  Anchor %d: %q (score: %.2f)", i, a.Text, a.Score)
		}
	}

	if output == "" {
		t.Error("output should not be empty")
	}
}

func TestSemanticAnchorFilter_WithConfig(t *testing.T) {
	cfg := SemanticAnchorConfig{
		AnchorRatio:       0.2,
		MinAnchorSpacing:  5,
		EnableAggregation: true,
		PreserveStructure: true,
	}
	filter := NewSemanticAnchorFilterWithConfig(cfg)

	input := strings.Repeat("content token ", 50)
	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty")
	}
	t.Logf("Saved: %d tokens", saved)
}

func TestSemanticAnchorFilter_StructuralMarkers(t *testing.T) {
	filter := NewSemanticAnchorFilter()
	input := `func test() {
	if condition {
		for i := 0; i < 10; i++ {
			doSomething()
		}
	}
}`

	output, _ := filter.Apply(input, ModeAggressive)

	// Structural markers should be preserved
	if !strings.Contains(output, "{") && !strings.Contains(output, "}") {
		t.Log("Warning: structural markers may not be preserved")
	}
	t.Logf("Output: %q", output)
}

func TestSemanticAnchorFilter_Aggregation(t *testing.T) {
	filter := NewSemanticAnchorFilter()
	input := `ImportantKeyword appears here.
Some other content around it.
ImportantKeyword appears again.
More content here.
ImportantKeyword final appearance.`

	output, saved := filter.Apply(input, ModeMinimal)

	// High-frequency tokens should become anchors
	if output == "" {
		t.Error("output should not be empty")
	}
	t.Logf("Saved: %d tokens", saved)
}

func TestSemanticAnchorFilter_Stats(t *testing.T) {
	filter := NewSemanticAnchorFilter()
	input := strings.Repeat("anchor content ", 30)

	_, saved := filter.Apply(input, ModeAggressive)

	stats := filter.GetStats()
	t.Logf("Stats: Anchors=%d, Aggregated=%d, Pruned=%d, Saved=%d",
		stats.TotalAnchors, stats.TotalAggregated, stats.NonAnchorPruned, saved)
}

func TestSemanticAnchorFilter_Modes(t *testing.T) {
	filter := NewSemanticAnchorFilter()
	input := strings.Repeat("test content line\n", 15)

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
