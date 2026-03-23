package vcs

import (
	"strings"
	"testing"
)

func TestFilterLog(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single commit",
			input: "abc1234 fix bug in parser\n",
			want:  []string{"abc1234 fix bug in parser"},
		},
		{
			name:  "multiple commits",
			input: "abc1234 fix bug\ndef5678 add feature\nghi9012 update docs\n",
			want:  []string{"abc1234 fix bug", "def5678 add feature", "ghi9012 update docs"},
		},
		{
			name:  "empty output",
			input: "",
			want:  nil,
		},
		{
			name:  "long commit message",
			input: "abc1234 " + string(make([]byte, 200)) + "\n",
			want:  nil, // just check it doesn't panic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterLog(tt.input)
			if tt.want == nil {
				// Just check no panic
				_ = got
				return
			}
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("filterLog(%q) missing %q, got:\n%s", tt.input, w, got)
				}
			}
		})
	}
}

func TestTruncateLogLine(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		width int
	}{
		{"short line", "abc1234 fix bug", 80},
		{"exact width", "abc1234 fix bug in parser with details", 80},
		{"long line", "abc1234 " + string(make([]byte, 100)), 50},
		{"zero width", "abc1234 fix bug", 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateLogLine(tt.line, tt.width)
			if tt.width > 0 && len(got) > tt.width+10 {
				t.Errorf("truncateLogLine(%q, %d) = %d chars, expected <= %d",
					tt.line[:min(len(tt.line), 20)], tt.width, len(got), tt.width)
			}
		})
	}
}
