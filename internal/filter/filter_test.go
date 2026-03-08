package filter

import (
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "ansi color codes",
			input:    "\x1b[32mgreen\x1b[0m text",
			expected: "green text",
		},
		{
			name:     "multiple ansi codes",
			input:    "\x1b[1;31mred bold\x1b[0m \x1b[34mblue\x1b[0m",
			expected: "red bold blue",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("StripANSI() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterEngine(t *testing.T) {
	engine := NewEngine(ModeMinimal)

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "ansi codes",
			input: "\x1b[31mred\x1b[0m",
		},
		{
			name:  "plain text",
			input: "hello world",
		},
		{
			name:  "mixed content",
			input: "\x1b[1mbold\x1b[0m text\nwith newlines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, saved := engine.Process(tt.input)
			if saved < 0 {
				t.Errorf("Process() saved tokens = %d, should be >= 0", saved)
			}
			if result == "" && tt.input != "" {
				t.Errorf("Process() returned empty for non-empty input")
			}
		})
	}
}

func TestFilterMode(t *testing.T) {
	if ModeMinimal != "minimal" {
		t.Errorf("ModeMinimal = %q, want 'minimal'", ModeMinimal)
	}
	if ModeAggressive != "aggressive" {
		t.Errorf("ModeAggressive = %q, want 'aggressive'", ModeAggressive)
	}
}
