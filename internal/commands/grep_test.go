package commands

import (
	"strings"
	"testing"
)

func TestCompactGrepOutputSimple(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		maxRes int
	}{
		{"short output", "file.go:10:foo\nfile.go:20:bar\n", 80, 100},
		{"long lines", "file.go:10:" + strings.Repeat("x", 200) + "\n", 50, 100},
		{"many results", strings.Repeat("file.go:1:line\n", 100), 80, 10},
		{"empty", "", 80, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compactGrepOutputSimple(tt.input, tt.maxLen, tt.maxRes)
			// Just verify no panic and returns something reasonable
			if tt.input == "" && got != "" {
				t.Errorf("compactGrepOutputSimple empty input should return empty, got: %q", got)
			}
			if tt.input != "" && len(got) == 0 {
				t.Errorf("compactGrepOutputSimple should return non-empty for non-empty input")
			}
		})
	}
}

func TestCompactFindOutput(t *testing.T) {
	input := "./src/main.go\n./src/lib.go\n./docs/README.md\n./src/\n"
	got := compactFindOutput(input)
	if got == "" {
		t.Error("compactFindOutput should return non-empty output")
	}
}
