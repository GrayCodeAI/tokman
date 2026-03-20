package commands

import (
	"strings"
	"testing"
)

func TestFilterCurlOutput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"json output", `{"key": "value", "num": 42}`},
		{"plain text", "hello world"},
		{"empty", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterCurlOutput(tt.input, 3, 100, 200)
			_ = got // just check no panic
		})
	}
}

func TestTryJSONSchema(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid json", `{"name": "test"}`, false},
		{"invalid json", "not json", true},
		{"empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tryJSONSchema(tt.input, 3)
			if tt.wantErr && got != "" {
				t.Errorf("tryJSONSchema(%q) should return empty for invalid input", tt.input)
			}
			if !tt.wantErr && got == "" {
				t.Errorf("tryJSONSchema(%q) should return non-empty for valid input", tt.input)
			}
			if !tt.wantErr && !strings.Contains(got, "string") {
				t.Errorf("tryJSONSchema(%q) should contain type info, got: %s", tt.input, got)
			}
		})
	}
}

func TestTruncateLineCurl(t *testing.T) {
	long := strings.Repeat("x", 200)
	got := truncateLine(long, 50)
	if len(got) > 55 { // allow for "..."
		t.Errorf("truncateLine should limit length, got %d chars", len(got))
	}
}
