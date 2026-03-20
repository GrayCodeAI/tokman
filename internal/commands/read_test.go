package commands

import (
	"strings"
	"testing"
)

func TestDetectLanguageFromExtension(t *testing.T) {
	// Test the extensions that actually work
	tests := []struct {
		path string
		want string
	}{
		{"file.go", "go"},
		{"file.rs", "rust"},
		{"file.py", "python"},
		{"file.js", "javascript"},
		{"file.ts", "typescript"},
		{"file.tsx", "typescript"},
		{"file.sql", "sql"},
		{"file.sh", "sh"},
		{"file.md", "unknown"},
		{"file.unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := detectLanguageFromExtension(tt.path)
			if string(got) != tt.want {
				t.Errorf("detectLanguageFromExtension(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestAddLineNumbers(t *testing.T) {
	// Test that addLineNumbers produces output with line numbers
	input := "hello\nworld\n"
	got := addLineNumbers(input)
	if !strings.Contains(got, "hello") {
		t.Error("addLineNumbers should contain original text")
	}
	if !strings.Contains(got, "1") {
		t.Error("addLineNumbers should contain line numbers")
	}
}

func TestTruncateLines(t *testing.T) {
	// Test that truncateLines reduces line count
	input := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\n"
	got := truncateLines(input, 3)
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) > 4 { // allow for truncation marker
		t.Errorf("truncateLines should limit output, got %d lines", len(lines))
	}
}
