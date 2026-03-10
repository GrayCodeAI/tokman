package commands

import (
	"testing"
)

func TestDetectWcMode(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected WcMode
	}{
		{
			name:     "no flags - full mode",
			args:     []string{"file.txt"},
			expected: WcModeFull,
		},
		{
			name:     "lines only -l",
			args:     []string{"-l", "file.txt"},
			expected: WcModeLines,
		},
		{
			name:     "words only -w",
			args:     []string{"-w", "file.txt"},
			expected: WcModeWords,
		},
		{
			name:     "bytes only -c",
			args:     []string{"-c", "file.txt"},
			expected: WcModeBytes,
		},
		{
			name:     "chars only -m",
			args:     []string{"-m", "file.txt"},
			expected: WcModeChars,
		},
		{
			name:     "combined flags -lw",
			args:     []string{"-lw", "file.txt"},
			expected: WcModeMixed,
		},
		{
			name:     "separate flags -l -w",
			args:     []string{"-l", "-w", "file.txt"},
			expected: WcModeMixed,
		},
		{
			name:     "all flags -lwc",
			args:     []string{"-lwc", "file.txt"},
			expected: WcModeMixed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectWcMode(tt.args)
			if result != tt.expected {
				t.Errorf("detectWcMode(%v) = %v, want %v", tt.args, result, tt.expected)
			}
		})
	}
}

func TestFormatSingleLine(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		mode     WcMode
		expected string
	}{
		{
			name:     "full mode with path",
			line:     "      30      96     978 src/main.go",
			mode:     WcModeFull,
			expected: "30L 96W 978B src/main.go",
		},
		{
			name:     "full mode stdin (no path)",
			line:     "      30      96     978",
			mode:     WcModeFull,
			expected: "30L 96W 978B",
		},
		{
			name:     "lines only",
			line:     "      30 src/main.go",
			mode:     WcModeLines,
			expected: "30",
		},
		{
			name:     "words only",
			line:     "      96 src/main.go",
			mode:     WcModeWords,
			expected: "96",
		},
		{
			name:     "bytes only",
			line:     "     978 src/main.go",
			mode:     WcModeBytes,
			expected: "978",
		},
		{
			name:     "mixed mode",
			line:     "      30      96 src/main.go",
			mode:     WcModeMixed,
			expected: "30 96 src/main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSingleLine(tt.line, tt.mode)
			if result != tt.expected {
				t.Errorf("formatSingleLine(%q, %v) = %q, want %q", tt.line, tt.mode, result, tt.expected)
			}
		})
	}
}

func TestFormatMultiLine(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		mode     WcMode
		contains []string
	}{
		{
			name:     "full mode multiple files",
			lines:    []string{"      30      96     978 src/main.go", "      50     120    1500 src/lib.go", "      80     216    2478 total"},
			mode:     WcModeFull,
			contains: []string{"30L 96W 978B", "50L 120W 1500B", "Σ 80L 216W 2478B"},
		},
		{
			name:     "lines mode multiple files",
			lines:    []string{"      30 src/main.go", "      50 src/lib.go", "      80 total"},
			mode:     WcModeLines,
			contains: []string{"30 main.go", "50 lib.go", "Σ 80"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMultiLine(tt.lines, tt.mode)
			for _, c := range tt.contains {
				if !contains(result, c) {
					t.Errorf("formatMultiLine() = %q, should contain %q", result, c)
				}
			}
		})
	}
}

func TestFindCommonPrefix(t *testing.T) {
	tests := []struct {
		name     string
		paths    []string
		expected string
	}{
		{
			name:     "common src prefix",
			paths:    []string{"src/main.go", "src/lib.go", "src/utils.go"},
			expected: "src/",
		},
		{
			name:     "no common prefix",
			paths:    []string{"main.go", "lib.go"},
			expected: "",
		},
		{
			name:     "deep common prefix",
			paths:    []string{"src/cmd/wc.go", "src/cmd/ls.go"},
			expected: "src/cmd/",
		},
		{
			name:     "single path",
			paths:    []string{"src/main.go"},
			expected: "",
		},
		{
			name:     "empty paths",
			paths:    []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findCommonPrefix(tt.paths)
			if result != tt.expected {
				t.Errorf("findCommonPrefix(%v) = %q, want %q", tt.paths, result, tt.expected)
			}
		})
	}
}

func TestStripPrefix(t *testing.T) {
	tests := []struct {
		path     string
		prefix   string
		expected string
	}{
		{"src/main.go", "src/", "main.go"},
		{"src/cmd/wc.go", "src/cmd/", "wc.go"},
		{"main.go", "", "main.go"},
		{"src/main.go", "lib/", "src/main.go"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := stripPrefix(tt.path, tt.prefix)
			if result != tt.expected {
				t.Errorf("stripPrefix(%q, %q) = %q, want %q", tt.path, tt.prefix, result, tt.expected)
			}
		})
	}
}

func TestIsNumeric(t *testing.T) {
	tests := []struct {
		s        string
		expected bool
	}{
		{"123", true},
		{"0", true},
		{"123abc", false},
		{"", false},
		{"-123", false},
		{"12.3", false},
	}

	for _, tt := range tests {
		t.Run(tt.s, func(t *testing.T) {
			result := isNumeric(tt.s)
			if result != tt.expected {
				t.Errorf("isNumeric(%q) = %v, want %v", tt.s, result, tt.expected)
			}
		})
	}
}
