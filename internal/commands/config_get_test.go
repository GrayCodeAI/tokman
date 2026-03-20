package commands

import (
	"testing"
)

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"single", "line", 1},
		{"two lines", "a\nb", 2},
		{"trailing newline", "a\nb\n", 2},
		{"multiple", "a\nb\nc\nd", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitLines(tt.input)
			if len(got) != tt.want {
				t.Errorf("splitLines(%q) = %d lines, want %d", tt.input, len(got), tt.want)
			}
		})
	}
}

func TestSplitFirst(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sep  string
		want int
	}{
		{"found", "key=value", "=", 2},
		{"not found", "keyvalue", "=", 1},
		{"multiple seps", "a=b=c", "=", 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitFirst(tt.s, tt.sep)
			if len(got) != tt.want {
				t.Errorf("splitFirst(%q, %q) = %d parts, want %d", tt.s, tt.sep, len(got), tt.want)
			}
		})
	}
}

func TestTrimStr(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  hello  ", "hello"},
		{"hello", "hello"},
		{"\t\nhello\n\t", "hello"},
		{"", ""},
		{"   ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := trimStr(tt.input)
			if got != tt.want {
				t.Errorf("trimStr(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFindStr(t *testing.T) {
	tests := []struct {
		s    string
		sub  string
		want int
	}{
		{"hello world", "world", 6},
		{"hello world", "hello", 0},
		{"hello world", "xyz", -1},
		{"", "a", -1},
	}

	for _, tt := range tests {
		t.Run(tt.s+"_"+tt.sub, func(t *testing.T) {
			got := findStr(tt.s, tt.sub)
			if got != tt.want {
				t.Errorf("findStr(%q, %q) = %d, want %d", tt.s, tt.sub, got, tt.want)
			}
		})
	}
}
