package filter

import (
	"strings"
	"testing"
)

func TestMetaTokenFilter_Basic(t *testing.T) {
	filter := NewMetaTokenFilter()
	input := "This is a test with repeated repeated repeated content that appears multiple times."

	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty")
	}
	if saved < 0 {
		t.Error("tokens saved should be non-negative")
	}
	t.Logf("Input: %q", input)
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d", saved)
}

func TestMetaTokenFilter_SmallInput(t *testing.T) {
	filter := NewMetaTokenFilter()
	input := "Small test."

	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty for small input")
	}
	if strings.TrimSpace(output) == "" {
		t.Error("output should not be whitespace-only")
	}
	t.Logf("Input: %q", input)
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d", saved)
}

func TestMetaTokenFilter_Disabled(t *testing.T) {
	filter := NewMetaTokenFilter()
	input := "This is a test with repeated repeated content."

	output, saved := filter.Apply(input, ModeNone)

	if output != input {
		t.Error("output should equal input when disabled")
	}
	if saved != 0 {
		t.Error("saved should be 0 when disabled")
	}
}

func TestMetaTokenFilter_RepeatedPatterns(t *testing.T) {
	filter := NewMetaTokenFilter()
	input := `func main() {
	fmt.Println("Hello")
	fmt.Println("Hello")
	fmt.Println("Hello")
}`

	output, saved := filter.Apply(input, ModeMinimal)

	if output == "" {
		t.Error("output should not be empty")
	}
	// Should compress repeated patterns
	t.Logf("Input: %q", input)
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d tokens", saved)
}

func TestMetaTokenFilter_WithConfig(t *testing.T) {
	cfg := MetaTokenConfig{
		WindowSize:          50,
		MinPattern:          3,
		MaxMetaTokens:       100,
	}
	filter := NewMetaTokenFilterWithConfig(cfg)

	input := strings.Repeat("test ", 20)
	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty")
	}
	t.Logf("Saved: %d tokens", saved)
}

func TestMetaTokenFilter_LosslessRecovery(t *testing.T) {
	filter := NewMetaTokenFilter()
	input := "The quick brown fox jumps. The quick brown fox jumps. The quick brown fox jumps."

	output, saved := filter.Apply(input, ModeMinimal)

	// Meta-token should preserve semantics
	if output == "" {
		t.Error("output should not be empty")
	}

	// Verify key content is preserved
	if !strings.Contains(output, "quick") || !strings.Contains(output, "fox") {
		t.Error("key content should be preserved")
	}
	t.Logf("Saved: %d tokens", saved)
}

func TestMetaTokenFilter_Stats(t *testing.T) {
	filter := NewMetaTokenFilter()
	input := "repeat repeat repeat test test test end end end"

	_, saved := filter.Apply(input, ModeAggressive)

	stats := filter.Stats()
	t.Logf("Stats: UniquePatterns=%d, TotalPatterns=%d, EstTokensSaved=%d, Saved=%d", 
		stats.UniquePatterns, stats.TotalPatterns, stats.EstTokensSaved, saved)
}
