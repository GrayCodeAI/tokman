package commands

import (
	"strings"
	"testing"
)

func TestFilterLogs(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"timestamp stripping", "2026-03-10T12:00:00 INFO started\n2026-03-10T12:00:01 INFO ready", "INFO started\nINFO ready"},
		{"deduplication", "same line\nsame line\nother line", "same line\nother line"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterLogs(tt.input)
			if tt.name == "timestamp stripping" && strings.Contains(got, "2026-03") {
				t.Errorf("filterLogs should strip timestamps, got: %s", got)
			}
			if tt.name == "deduplication" && strings.Count(got, "same line") != 1 {
				t.Errorf("filterLogs should deduplicate, got: %s", got)
			}
		})
	}
}
