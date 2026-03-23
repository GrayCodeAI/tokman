package web

import (
	"strings"
	"testing"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
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
			_ = got
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
			got := shared.TryJSONSchema(tt.input, 3)
			if tt.wantErr && got != "" {
				t.Errorf("TryJSONSchema(%q) should return empty for invalid input", tt.input)
			}
			if !tt.wantErr && got == "" {
				t.Errorf("TryJSONSchema(%q) should return non-empty for valid input", tt.input)
			}
			if !tt.wantErr && !strings.Contains(got, "string") {
				t.Errorf("TryJSONSchema(%q) should contain type info, got: %s", tt.input, got)
			}
		})
	}
}

func TestTruncateLineCurl(t *testing.T) {
	long := strings.Repeat("x", 200)
	got := shared.TruncateLine(long, 50)
	if len(got) > 55 {
		t.Errorf("shared.TruncateLine should limit length, got %d chars", len(got))
	}
}
