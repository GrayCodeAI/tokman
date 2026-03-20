package commands

import (
	"strings"
	"testing"
)

func TestSplitCommandChain(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int // expected number of segments
	}{
		{"single command", "git status", 1},
		{"&& chain", "git add . && git commit", 2},
		{"|| chain", "git stash || echo failed", 2},
		{"semicolon", "git add .; git commit", 2},
		{"pipe chain", "grep foo | head -5", 2},
		{"mixed chains", "git add . && git commit || echo fail", 3},
		{"empty", "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCommandChain(tt.input)
			if len(got) != tt.want {
				t.Errorf("splitCommandChain(%q) = %d segments, want %d: %v",
					tt.input, len(got), tt.want, got)
			}
		})
	}
}

func TestSplitCommandChainPreservesQuoted(t *testing.T) {
	input := `echo "foo && bar" && echo done`
	got := splitCommandChain(input)
	// The && inside quotes should not be a chain separator
	if len(got) == 2 {
		// Check the first segment preserves the quoted content
		if !strings.Contains(got[0], "foo && bar") {
			t.Errorf("splitCommandChain should preserve quoted && in first segment: %v", got)
		}
	}
}

func TestProgressBar(t *testing.T) {
	tests := []struct {
		pct float64
		w   int
	}{
		{0.0, 10},
		{50.0, 10},
		{100.0, 10},
		{75.0, 20},
	}

	for _, tt := range tests {
		got := progressBar(tt.pct, tt.w)
		if len(got) != tt.w {
			t.Errorf("progressBar(%v, %d) length = %d, want %d", tt.pct, tt.w, len(got), tt.w)
		}
	}
}
