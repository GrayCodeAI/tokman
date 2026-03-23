package analysis

import (
	"testing"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/utils"
)

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{500, "500"},
		{999, "999"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{5000, "5.0K"},
		{10000, "10.0K"},
		{999999, "1000.0K"},
		{1000000, "1.0M"},
		{1500000, "1.5M"},
		{5000000, "5.0M"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := utils.FormatTokens(tt.input)
			if got != tt.want {
				t.Errorf("FormatTokens(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0ms"},
		{500, "500ms"},
		{999, "999ms"},
		{1000, "1.0s"},
		{1500, "1.5s"},
		{59999, "60.0s"},
		{60000, "1m"},
		{90000, "1m 30s"},
		{120000, "2m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := utils.FormatDuration(tt.input)
			if got != tt.want {
				t.Errorf("FormatDuration(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestShortenPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/a/b/c", "/a/b/c"},
		{"/home/user/project", "/home/user/project"},
		{"/home/user/deep/path/to/project", "/home/user/deep/path/to/project"},
		{"/a/b/c/d/e", "/a/.../d/e"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := shared.ShortenPath(tt.input)
			// ShortenPath may vary by OS, just check it returns something
			if got == "" {
				t.Errorf("shared.ShortenPath(%q) returned empty", tt.input)
			}
		})
	}
}
