package filter

import (
	"strings"
	"testing"
)

func TestAgentMemoryFilter_Basic(t *testing.T) {
	filter := NewAgentMemoryFilter()
	input := "This is a test with some content for the agent memory filter."

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

func TestAgentMemoryFilter_SmallInput(t *testing.T) {
	filter := NewAgentMemoryFilter()
	input := "This is a test."

	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty for small input")
	}
	if strings.TrimSpace(output) == "" {
		t.Error("output should not be whitespace-only for small input")
	}
	t.Logf("Input: %q", input)
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d", saved)
}

func TestAgentMemoryFilter_Disabled(t *testing.T) {
	filter := NewAgentMemoryFilter()
	input := "This is a test."

	output, saved := filter.Apply(input, ModeNone)

	if output != input {
		t.Error("output should equal input when disabled")
	}
	if saved != 0 {
		t.Error("saved should be 0 when disabled")
	}
}

func TestAgentMemoryFilter_KnowledgeExtraction(t *testing.T) {
	filter := NewAgentMemoryFilter()
	input := `I found the issue in the code.
The solution is to fix the regex pattern.
I discovered that the bug was in the parser.
This is some regular content.
Some more content here.`

	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty")
	}
	// Knowledge patterns should be extracted
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d tokens", saved)
}

func TestAgentMemoryFilter_NoiseRemoval(t *testing.T) {
	filter := NewAgentMemoryFilter()
	input := `okay.
sure.
yes.
This is important content.
The key finding was that the algorithm works.
waiting...
loading...`

	output, _ := filter.Apply(input, ModeAggressive)

	// Noise should be filtered out
	if strings.Contains(output, "okay.") || strings.Contains(output, "sure.") {
		t.Error("noise patterns should be filtered")
	}
	t.Logf("Output: %q", output)
}
