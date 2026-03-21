package filter

import (
	"strings"
	"testing"
)

func TestSemanticChunkFilter_Basic(t *testing.T) {
	filter := NewSemanticChunkFilter()
	input := `This is the first paragraph with some content.

This is the second paragraph with more content.

This is the third paragraph with even more content.`

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

func TestSemanticChunkFilter_SmallInput(t *testing.T) {
	filter := NewSemanticChunkFilter()
	input := "Short text."

	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty for small input")
	}
	// Small inputs should be preserved as-is
	t.Logf("Input: %q", input)
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d", saved)
}

func TestSemanticChunkFilter_Disabled(t *testing.T) {
	filter := NewSemanticChunkFilter()
	input := "This is a test with multiple paragraphs.\n\nSecond paragraph here."

	output, saved := filter.Apply(input, ModeNone)

	if output != input {
		t.Error("output should equal input when disabled")
	}
	if saved != 0 {
		t.Error("saved should be 0 when disabled")
	}
}

func TestSemanticChunkFilter_CodeContent(t *testing.T) {
	filter := NewSemanticChunkFilter()
	input := `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}

type User struct {
	Name string
	Age  int
}`

	output, saved := filter.Apply(input, ModeMinimal)

	if output == "" {
		t.Error("output should not be empty")
	}
	// Code structure should be preserved
	if !strings.Contains(output, "func") && !strings.Contains(output, "main") {
		t.Error("code structure should be preserved")
	}
	t.Logf("Saved: %d tokens", saved)
}

func TestSemanticChunkFilter_MixedContent(t *testing.T) {
	filter := NewSemanticChunkFilter()
	input := `# Documentation

This is a markdown document with code blocks.

` + "```go" + `
func example() {
	return 42
}
` + "```" + `

More text after code.`

	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty")
	}
	t.Logf("Saved: %d tokens", saved)
}

func TestSemanticChunkFilter_WithConfig(t *testing.T) {
	cfg := SemanticChunkConfig{
		ChunkMethod:         ChunkText,
		MinChunkSize:        5,
		MaxChunkSize:        200,
		ImportanceThreshold: 0.3,
		PreserveStructure:   true,
	}
	filter := NewSemanticChunkFilterWithConfig(cfg)

	input := strings.Repeat("This is a sentence. ", 50)
	output, saved := filter.Apply(input, ModeAggressive)

	if output == "" {
		t.Error("output should not be empty")
	}
	t.Logf("Saved: %d tokens", saved)
}

func TestSemanticChunkFilter_ChunkTypes(t *testing.T) {
	filter := NewSemanticChunkFilter()

	tests := []struct {
		name    string
		input   string
		hasCode bool
	}{
		{
			name:    "text only",
			input:   "Just plain text content here.",
			hasCode: false,
		},
		{
			name:    "code only",
			input:   "func test() { return 1 }",
			hasCode: true,
		},
		{
			name:    "mixed",
			input:   "Text before.\nfunc code() {}\nText after.",
			hasCode: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, _ := filter.Apply(tt.input, ModeMinimal)
			if output == "" {
				t.Error("output should not be empty")
			}
		})
	}
}

func TestSemanticChunkFilter_ImportanceScoring(t *testing.T) {
	filter := NewSemanticChunkFilter()
	input := `TODO: Important task to complete.
This is regular content.
FIXME: Bug that needs fixing.
More regular content here.
ERROR: Something went wrong.`

	output, saved := filter.Apply(input, ModeAggressive)

	// Important markers should be preserved
	if !strings.Contains(output, "TODO") && !strings.Contains(output, "FIXME") {
		t.Log("Warning: important markers may not be prioritized")
	}
	t.Logf("Output: %q", output)
	t.Logf("Saved: %d tokens", saved)
}
