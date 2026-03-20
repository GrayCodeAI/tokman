package commands

import (
	"testing"
)

func TestTruncateLine(t *testing.T) {
	tests := []struct {
		name  string
		input string
		max   int
		want  string
	}{
		{"short", "hello", 10, "hello"},
		{"exact", "hello", 5, "hello"},
		{"long", "hello world foo bar", 10, "hello w..."},
		{"empty", "", 10, ""},
		{"one char", "a", 1, "a"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateLine(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("truncateLine(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

func TestExtractNumber(t *testing.T) {
	tests := []struct {
		name   string
		text   string
		after  string
		expect int
	}{
		{"passed", "5 passed", "passed", 5},
		{"failed", "3 failed", "failed", 3},
		{"skipped", "2 skipped", "skipped", 2},
		{"no match", "no numbers here", "passed", 0},
		{"zero", "0 errors", "errors", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractNumber(tt.text, tt.after)
			if got != tt.expect {
				t.Errorf("extractNumber(%q, %q) = %d, want %d", tt.text, tt.after, got, tt.expect)
			}
		})
	}
}

func TestDetectOutputType(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		command string
		typ     OutputType
	}{
		{"test output", "PASS TestFoo", "go test", OutputTypeTest},
		{"build error", "error[E0308]", "cargo build", OutputTypeBuild},
		{"json output", `{"key": "value"}`, "curl", OutputTypeJSON},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectOutputType(tt.output, tt.command)
			if got != tt.typ {
				t.Errorf("detectOutputType(%q, %q) = %v, want %v", tt.output, tt.command, got, tt.typ)
			}
		})
	}
}
