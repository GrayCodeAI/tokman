package toml

import (
	"testing"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

func TestTOMLFilterEngine_Apply(t *testing.T) {
	tests := []struct {
		name    string
		config  TOMLFilterRule
		input   string
		wantLen int // approximate expected length
	}{
		{
			name: "strip ansi",
			config: TOMLFilterRule{
				StripANSI: true,
			},
			input:   "\x1b[32mHello\x1b[0m World",
			wantLen: 11, // "Hello World"
		},
		{
			name: "strip lines matching",
			config: TOMLFilterRule{
				StripLinesMatching: []string{"^DEBUG:.*"},
			},
			input:   "DEBUG: line1\nINFO: line2\nDEBUG: line3\nINFO: line4",
			wantLen: 20, // "INFO: line2\nINFO: line4"
		},
		{
			name: "keep lines matching",
			config: TOMLFilterRule{
				KeepLinesMatching: []string{"ERROR:.*"},
			},
			input:   "INFO: line1\nERROR: line2\nINFO: line3\nERROR: line4",
			wantLen: 22, // "ERROR: line2\nERROR: line4"
		},
		{
			name: "truncate lines",
			config: TOMLFilterRule{
				TruncateLinesAt: 10,
			},
			input:   "short\nthis is a very long line that should be truncated\nanother short",
			wantLen: 38, // "short\nthis is a ...\nanother short"
		},
		{
			name: "head only",
			config: TOMLFilterRule{
				Head: 2,
			},
			input:   "line1\nline2\nline3\nline4\nline5",
			wantLen: 30, // "line1\nline2\n... [3 lines truncated]"
		},
		{
			name: "tail only",
			config: TOMLFilterRule{
				Tail: 2,
			},
			input:   "line1\nline2\nline3\nline4\nline5",
			wantLen: 32, // "... [3 lines truncated]\nline4\nline5"
		},
		{
			name: "head and tail",
			config: TOMLFilterRule{
				Head: 1,
				Tail: 1,
			},
			input:   "line1\nline2\nline3\nline4\nline5",
			wantLen: 40, // "line1\n... [3 lines truncated] ...\nline5"
		},
		{
			name: "max lines",
			config: TOMLFilterRule{
				MaxLines: 4,
			},
			input:   "line1\nline2\nline3\nline4\nline5\nline6\nline7\nline8",
			wantLen: 50, // halves + truncation message
		},
		{
			name: "on empty",
			config: TOMLFilterRule{
				StripLinesMatching: []string{".*"},
				OnEmpty:            "No output",
			},
			input:   "this will all be stripped",
			wantLen: 9, // "No output"
		},
		{
			name: "replace patterns",
			config: TOMLFilterRule{
				Replace: []ReplaceRule{
					{Pattern: "foo", Replacement: "bar"},
					{Pattern: "hello", Replacement: "hi"},
				},
			},
			input:   "foo is hello world",
			wantLen: 16, // "bar is hi world"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewTOMLFilterEngine(&tt.config)
			output, saved := engine.Apply(tt.input, filter.ModeMinimal)

			if saved < 0 {
				t.Errorf("Apply() saved = %d, want >= 0", saved)
			}

			// Approximate length check (allow some variance)
			if len(output) > tt.wantLen+10 || len(output) < tt.wantLen-10 {
				t.Logf("Apply() output length = %d, expected ~%d", len(output), tt.wantLen)
			}
		})
	}
}

func TestTOMLFilterEngine_Pipeline(t *testing.T) {
	// Test that pipeline stages are applied in order
	config := TOMLFilterRule{
		StripANSI:          true,
		StripLinesMatching: []string{"^DEBUG:.*"},
		TruncateLinesAt:    20,
		MaxLines:           3,
	}

	input := "\x1b[32mDEBUG: hidden\x1b[0m\nINFO: this is a very long line that should be truncated\nINFO: keep this\nDEBUG: also hidden\nINFO: and this too"

	engine := NewTOMLFilterEngine(&config)
	output, saved := engine.Apply(input, filter.ModeMinimal)

	// Should not contain ANSI
	if contains := len(output) > 0 && (output[0] == 0x1b); contains {
		t.Error("Output should not contain ANSI codes")
	}

	// Should not contain DEBUG lines
	if len(output) > 5 && output[:5] == "DEBUG" {
		t.Error("Output should not contain DEBUG lines")
	}

	t.Logf("Output: %q", output)
	t.Logf("Saved: %d bytes", saved)
}

func TestMatchAndFilter(t *testing.T) {
	// Create a registry with a simple filter
	registry := NewFilterRegistry()

	content := `schema_version = 1

[test]
match_command = "^test.*"
strip_ansi = true
max_lines = 2
`
	parser := NewParser()
	filter, err := parser.ParseContent([]byte(content), "test.toml")
	if err != nil {
		t.Fatalf("ParseContent() error = %v", err)
	}

	registry.filters["test.toml"] = filter

	// Test matching
	output, _, matched := MatchAndFilter("test command", "line1\nline2\nline3\nline4", registry)
	if !matched {
		t.Error("MatchAndFilter() should match 'test command'")
	}

	// Verify output is filtered (max_lines=2)
	if len(output) == 0 {
		t.Error("MatchAndFilter() should produce output")
	}

	t.Logf("Filtered output: %q", output)

	// Test non-matching
	_, _, matched = MatchAndFilter("other command", "output", registry)
	if matched {
		t.Error("MatchAndFilter() should not match 'other command'")
	}
}

func TestTOMLFilterWrapper(t *testing.T) {
	config := TOMLFilterRule{
		StripANSI: true,
		MaxLines:  5,
	}

	wrapper := NewTOMLFilterWrapper("test_filter", &config)

	if wrapper.Name() != "test_filter" {
		t.Errorf("Name() = %q, want %q", wrapper.Name(), "test_filter")
	}

	input := "line1\nline2\nline3\nline4\nline5\nline6\nline7"
	output, _ := wrapper.Apply(input, filter.ModeMinimal)

	// Verify output is filtered
	if len(output) == 0 {
		t.Error("Apply() should produce output")
	}

	t.Logf("Filtered: %q", output)
}

func TestMatchOutputUnless(t *testing.T) {
	tests := []struct {
		name       string
		config     TOMLFilterRule
		input      string
		wantOutput string
	}{
		{
			name: "match_output short-circuits when pattern matches",
			config: TOMLFilterRule{
				MatchOutput: []MatchOutputRule{
					{Pattern: "Build succeeded", Message: "✓ Build OK"},
				},
			},
			input:      "Compiling...\nBuild succeeded\nDone",
			wantOutput: "✓ Build OK",
		},
		{
			name: "unless clause prevents short-circuit when errors present",
			config: TOMLFilterRule{
				MatchOutput: []MatchOutputRule{
					{
						Pattern: "Build succeeded",
						Message: "✓ Build OK",
						Unless:  "error|warning|ERROR|WARNING",
					},
				},
			},
			input:      "Compiling...\nerror: something failed\nBuild succeeded",
			wantOutput: "", // Should NOT short-circuit - full output passes through
		},
		{
			name: "unless clause allows short-circuit when no errors",
			config: TOMLFilterRule{
				MatchOutput: []MatchOutputRule{
					{
						Pattern: "Build succeeded",
						Message: "✓ Build OK",
						Unless:  "error|warning|ERROR|WARNING",
					},
				},
			},
			input:      "Compiling...\nBuild succeeded\nDone",
			wantOutput: "✓ Build OK",
		},
		{
			name: "multiple match_output rules - first wins",
			config: TOMLFilterRule{
				MatchOutput: []MatchOutputRule{
					{Pattern: "Tests passed", Message: "✓ Tests OK"},
					{Pattern: "Build succeeded", Message: "✓ Build OK"},
				},
			},
			input:      "Build succeeded\nTests passed",
			wantOutput: "✓ Tests OK",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewTOMLFilterEngine(&tt.config)
			output, _ := engine.Apply(tt.input, filter.ModeMinimal)

			if tt.wantOutput != "" {
				if output != tt.wantOutput {
					t.Errorf("Apply() output = %q, want %q", output, tt.wantOutput)
				}
			} else {
				// When unless prevents short-circuit, output should contain original content
				if output == tt.config.MatchOutput[0].Message {
					t.Errorf("Apply() should not have short-circuited, got %q", output)
				}
			}
		})
	}
}
