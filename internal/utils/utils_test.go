package utils

import (
	"testing"
)

func TestShortenPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short path", "/home/user/project", 50, "/home/user/project"},
		{"long path", "/home/user/projects/very/deeply/nested/directory/structure", 30, ".../nested/directory/structure"},
		{"exact length", "/home/user/project", 18, "/home/user/project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShortenPath(tt.input, tt.maxLen)
			if len(result) > tt.maxLen && tt.maxLen > 0 {
				t.Errorf("ShortenPath(%q, %d) = %q (len=%d), expected len <= %d",
					tt.input, tt.maxLen, result, len(result), tt.maxLen)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"bytes", 500, "500B"},
		{"kilobytes", 2048, "2.0K"},
		{"megabytes", 1048576, "1.0M"},
		{"gigabytes", 1073741824, "1.0G"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name     string
		ms       int64
		contains string
	}{
		{"milliseconds", 100, "ms"},
		{"seconds", 2500, "s"},
		{"minutes", 125000, "m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.ms)
			if !contains(result, tt.contains) {
				t.Errorf("FormatDuration(%d) = %q, expected to contain %q", tt.ms, result, tt.contains)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
