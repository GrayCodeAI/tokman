package commands

import (
	"testing"
)

func TestIsTableFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"table with separator", " id | name\n----+------\n 1  | foo", true},
		{"plain text", "hello world", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTableFormat(tt.input)
			if got != tt.want {
				t.Errorf("isTableFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsExpandedFormat(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"expanded", "-[ RECORD 1 ]-\nid   | 1\nname | foo", true},
		{"plain text", "hello world", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isExpandedFormat(tt.input)
			if got != tt.want {
				t.Errorf("isExpandedFormat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFilterTable(t *testing.T) {
	input := " id | name\n----+------\n 1  | foo\n 2  | bar\n(2 rows)\n"
	got := filterTable(input)
	if got == "" {
		t.Error("filterTable should return non-empty output")
	}
}

func TestFilterExpanded(t *testing.T) {
	input := "-[ RECORD 1 ]-\nid   | 1\nname | foo\n-[ RECORD 2 ]-\nid   | 2\nname | bar\n"
	got := filterExpanded(input)
	if got == "" {
		t.Error("filterExpanded should return non-empty output")
	}
}
