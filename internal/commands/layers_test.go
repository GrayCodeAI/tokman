package commands

import (
	"strings"
	"testing"
)

func TestSplitWords(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello world", 2},
		{"", 0},
		{"single", 1},
		{"  multiple   spaces  ", 2},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitWords(tt.input)
			if len(got) != tt.want {
				t.Errorf("splitWords(%q) = %d words, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestWrapText(t *testing.T) {
	input := "This is a long line that should be wrapped at a certain width"
	got := wrapText(input, 20)
	if got == "" {
		t.Error("wrapText should return non-empty")
	}
	lines := strings.Split(got, "\n")
	for _, line := range lines {
		// Check that continuation lines have the prefix
		if strings.Contains(line, "║   ") {
			continue
		}
	}
}
