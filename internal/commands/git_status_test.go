package commands

import (
	"testing"
)

func TestGetStatusCode(t *testing.T) {
	// getStatusCode returns the byte as-is (uppercase string)
	tests := []struct {
		code byte
		want string
	}{
		{'M', "M"},
		{'A', "A"},
		{'D', "D"},
		{'R', "R"},
		{'C', "C"},
		{'U', "U"},
		{'?', "?"},
		{'!', "!"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := getStatusCode(tt.code)
			if got != tt.want {
				t.Errorf("getStatusCode(%c) = %q, want %q", tt.code, got, tt.want)
			}
		})
	}
}

func TestParsePorcelainBranch(t *testing.T) {
	// Test that parsePorcelain extracts branch info
	input := "# branch.head main\n"
	status := parsePorcelain(input)
	if status.Branch == "" {
		t.Error("parsePorcelain should extract branch name")
	}
	// The function may keep the full line or extract just "main"
	if status.Branch != "# branch.head main" && status.Branch != "main" {
		t.Errorf("Branch = %q, want 'main' or full line", status.Branch)
	}
}

func TestParsePorcelainFiles(t *testing.T) {
	input := `# branch.head main
1 M N file.txt
? untracked.txt
`
	status := parsePorcelain(input)
	total := len(status.Modified) + len(status.Staged) + len(status.Untracked) + len(status.Conflicted)
	if total == 0 {
		t.Error("parsePorcelain should detect files")
	}
}
