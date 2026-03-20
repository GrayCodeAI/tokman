package commands

import (
	"strings"
	"testing"
)

func TestCompactURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com/api", "example.com/api"},
		{"http://example.com", "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := compactURL(tt.input)
			if got != tt.want {
				t.Errorf("compactURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		input uint64
		want  string
	}{
		{0, "?"},
		{500, "500B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1073741824, "1.0GB"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatSize(tt.input)
			if got != tt.want {
				t.Errorf("formatSize(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestExtractFilename(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		url    string
		args   []string
		want   string
	}{
		{"from -O flag", "", "https://example.com/file.tar.gz", []string{"-O", "output.tar.gz"}, "output.tar.gz"},
		{"from URL", "", "https://example.com/file.txt", []string{}, "file.txt"},
		{"from stderr", "Saving to: 'data.json'\n", "https://example.com", []string{}, "data.json"},
		{"fallback", "", "https://example.com/", []string{}, "index.html"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFilename(tt.stderr, tt.url, tt.args)
			if got != tt.want {
				t.Errorf("extractFilename() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseWgetError(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"404 Not Found", "404"},
		{"500 Internal Server Error", "500"},
		{"Connection refused", "Connection refused"},
		{"no error here", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseWgetError(tt.input)
			if tt.want != "" && !strings.Contains(got, tt.want) {
				t.Errorf("parseWgetError(%q) = %q, want to contain %q", tt.input, got, tt.want)
			}
		})
	}
}
