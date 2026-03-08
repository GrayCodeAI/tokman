package discover

import (
	"testing"
)

func TestRewrite(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "git status",
			input:    "git status",
			expected: "tokman git status",
		},
		{
			name:     "git status with args",
			input:    "git status --short",
			expected: "tokman git status --short",
		},
		{
			name:     "git diff",
			input:    "git diff",
			expected: "tokman git diff",
		},
		{
			name:     "git log",
			input:    "git log",
			expected: "tokman git log",
		},
		{
			name:     "ls",
			input:    "ls",
			expected: "tokman ls",
		},
		{
			name:     "ls -la",
			input:    "ls -la",
			expected: "tokman ls",
		},
		{
			name:     "ls with path",
			input:    "ls /home/user",
			expected: "tokman ls /home/user",
		},
		{
			name:     "go test",
			input:    "go test",
			expected: "tokman test",
		},
		{
			name:     "go test with args",
			input:    "go test ./...",
			expected: "tokman test ./...",
		},
		{
			name:     "go build",
			input:    "go build",
			expected: "tokman build",
		},
		{
			name:     "unknown command",
			input:    "cat file.txt",
			expected: "cat file.txt",
		},
		{
			name:     "partial match not rewritten",
			input:    "git clone",
			expected: "git clone",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Rewrite(tt.input)
			if result != tt.expected {
				t.Errorf("Rewrite() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestShouldRewrite(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "git status",
			input:    "git status",
			expected: true,
		},
		{
			name:     "git status with args",
			input:    "git status --short",
			expected: true,
		},
		{
			name:     "unknown command",
			input:    "cat file.txt",
			expected: false,
		},
		{
			name:     "partial git command",
			input:    "git clone",
			expected: false,
		},
		{
			name:     "ls",
			input:    "ls",
			expected: true,
		},
		{
			name:     "go test",
			input:    "go test",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldRewrite(tt.input)
			if result != tt.expected {
				t.Errorf("ShouldRewrite() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetMapping(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantFound  bool
		wantCmd    string
	}{
		{
			name:      "git status",
			input:     "git status",
			wantFound: true,
			wantCmd:   "tokman git status",
		},
		{
			name:      "unknown command",
			input:     "cat file.txt",
			wantFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapping, found := GetMapping(tt.input)
			if found != tt.wantFound {
				t.Errorf("GetMapping() found = %v, want %v", found, tt.wantFound)
			}
			if found && mapping.TokManCmd != tt.wantCmd {
				t.Errorf("GetMapping() TokManCmd = %q, want %q", mapping.TokManCmd, tt.wantCmd)
			}
		})
	}
}

func TestListRewrites(t *testing.T) {
	rewrites := ListRewrites()

	if len(rewrites) == 0 {
		t.Error("ListRewrites() returned empty list")
	}

	// Check all returned rewrites are enabled
	for _, r := range rewrites {
		if !r.Enabled {
			t.Errorf("ListRewrites() returned disabled mapping: %s", r.Original)
		}
	}

	// Check expected commands are present
	expected := []string{"git status", "git diff", "git log", "ls", "go test"}
	for _, exp := range expected {
		found := false
		for _, r := range rewrites {
			if r.Original == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ListRewrites() missing expected command: %s", exp)
		}
	}
}
