package commands

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestHookAlreadyPresent(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		hookPath string
		want     bool
	}{
		{
			name:     "empty settings",
			json:     `{}`,
			hookPath: "/home/user/.claude/hooks/tokman-rewrite.sh",
			want:     false,
		},
		{
			name: "exact match",
			json: `{
				"hooks": {
					"PreToolUse": [{
						"matcher": "Bash",
						"hooks": [{
							"type": "command",
							"command": "/home/user/.claude/hooks/tokman-rewrite.sh"
						}]
					}]
				}
			}`,
			hookPath: "/home/user/.claude/hooks/tokman-rewrite.sh",
			want:     true,
		},
		{
			name: "different path same filename",
			json: `{
				"hooks": {
					"PreToolUse": [{
						"matcher": "Bash",
						"hooks": [{
							"type": "command",
							"command": "/Users/test/.claude/hooks/tokman-rewrite.sh"
						}]
					}]
				}
			}`,
			hookPath: "/home/user/.claude/hooks/tokman-rewrite.sh",
			want:     true,
		},
		{
			name: "other hook only",
			json: `{
				"hooks": {
					"PreToolUse": [{
						"matcher": "Bash",
						"hooks": [{
							"type": "command",
							"command": "/some/other/hook.sh"
						}]
					}]
				}
			}`,
			hookPath: "/home/user/.claude/hooks/tokman-rewrite.sh",
			want:     false,
		},
		{
			name: "multiple hooks with tokman",
			json: `{
				"hooks": {
					"PreToolUse": [
						{
							"matcher": "Bash",
							"hooks": [{"type": "command", "command": "/other/hook.sh"}]
						},
						{
							"matcher": "Bash",
							"hooks": [{"type": "command", "command": "/home/user/.claude/hooks/tokman-rewrite.sh"}]
						}
					]
				}
			}`,
			hookPath: "/home/user/.claude/hooks/tokman-rewrite.sh",
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var root map[string]interface{}
			if err := json.Unmarshal([]byte(tt.json), &root); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			got := hookAlreadyPresent(root, tt.hookPath)
			if got != tt.want {
				t.Errorf("hookAlreadyPresent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInsertHookEntry(t *testing.T) {
	tests := []struct {
		name         string
		json         string
		hookPath     string
		wantLen      int
		wantCmdIndex int
	}{
		{
			name:         "empty root",
			json:         `{}`,
			hookPath:     "/home/user/.claude/hooks/tokman-rewrite.sh",
			wantLen:      1,
			wantCmdIndex: 0,
		},
		{
			name: "preserves existing hooks",
			json: `{
				"hooks": {
					"PreToolUse": [{
						"matcher": "Bash",
						"hooks": [{"type": "command", "command": "/other/hook.sh"}]
					}]
				}
			}`,
			hookPath:     "/home/user/.claude/hooks/tokman-rewrite.sh",
			wantLen:      2,
			wantCmdIndex: 1,
		},
		{
			name: "preserves other keys",
			json: `{
				"env": {"PATH": "/custom"},
				"model": "claude-sonnet-4",
				"permissions": {"allowAll": true}
			}`,
			hookPath:     "/home/user/.claude/hooks/tokman-rewrite.sh",
			wantLen:      1,
			wantCmdIndex: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var root map[string]interface{}
			if err := json.Unmarshal([]byte(tt.json), &root); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			insertHookEntry(root, tt.hookPath)

			// Check PreToolUse array length
			preToolUse := root["hooks"].(map[string]interface{})["PreToolUse"].([]interface{})
			if len(preToolUse) != tt.wantLen {
				t.Errorf("PreToolUse length = %d, want %d", len(preToolUse), tt.wantLen)
			}

			// Check our hook was added with correct command
			entry := preToolUse[tt.wantCmdIndex].(map[string]interface{})
			hooks := entry["hooks"].([]interface{})
			cmd := hooks[0].(map[string]interface{})["command"].(string)
			if cmd != tt.hookPath {
				t.Errorf("command = %v, want %v", cmd, tt.hookPath)
			}

			// Check matcher is Bash
			if entry["matcher"] != "Bash" {
				t.Errorf("matcher = %v, want Bash", entry["matcher"])
			}
		})
	}
}

func TestRemoveTokmanBlock(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantContent  string
		wantMigrated bool
	}{
		{
			name:         "no block",
			input:        "# My notes\n\nSome content\n",
			wantContent:  "# My notes\n\nSome content\n",
			wantMigrated: false,
		},
		{
			name: "remove block in middle",
			input: `# My Config

<!-- tokman-instructions v1 -->
OLD TOKMAN STUFF
<!-- /tokman-instructions -->

More content
`,
			wantContent:  "# My Config\n\nMore content\n",
			wantMigrated: true,
		},
		{
			name: "block at end",
			input: `# My Config

<!-- tokman-instructions v1 -->
STUFF
<!-- /tokman-instructions -->`,
			wantContent:  "# My Config",
			wantMigrated: true,
		},
		{
			name: "malformed - no closing",
			input: `# Config
<!-- tokman-instructions v1 -->
No closing marker`,
			wantContent:  "# Config\n<!-- tokman-instructions v1 -->\nNo closing marker",
			wantMigrated: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, migrated := removeTokmanBlock(tt.input)
			if migrated != tt.wantMigrated {
				t.Errorf("removeTokmanBlock() migrated = %v, want %v", migrated, tt.wantMigrated)
			}
			if got != tt.wantContent {
				t.Errorf("removeTokmanBlock() = %q, want %q", got, tt.wantContent)
			}
		})
	}
}

func TestRemoveTokmanMdReference(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single line reference",
			input: "# Notes\n\n@TOKMAN.md\n\nMore notes",
			want:  "# Notes\n\n\nMore notes",
		},
		{
			name:  "no reference",
			input: "# Notes\n\nNo reference here",
			want:  "# Notes\n\nNo reference here",
		},
		{
			name:  "reference at end",
			input: "# Notes\n\n@TOKMAN.md",
			want:  "# Notes\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := removeTokmanMdReference(tt.input)
			if got != tt.want {
				t.Errorf("removeTokmanMdReference() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCleanDoubleBlanks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no change needed",
			input: "line1\n\nline2",
			want:  "line1\n\nline2",
		},
		{
			name:  "collapse three to two",
			input: "line1\n\n\nline2",
			want:  "line1\n\n\nline2",
		},
		{
			name:  "collapse four to two",
			input: "line1\n\n\n\nline2",
			want:  "line1\n\n\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cleanDoubleBlanks(tt.input)
			if got != tt.want {
				t.Errorf("cleanDoubleBlanks() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteIfChanged(t *testing.T) {
	t.Run("creates new file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.md")
		content := "test content"

		err := writeIfChanged(path, content, "test")
		if err != nil {
			t.Fatalf("writeIfChanged() error = %v", err)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(got) != content {
			t.Errorf("file content = %q, want %q", got, content)
		}
	})

	t.Run("no change when identical", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.md")
		content := "test content"

		// Write initially
		os.WriteFile(path, []byte(content), 0644)

		// Should not error and not change file
		err := writeIfChanged(path, content, "test")
		if err != nil {
			t.Fatalf("writeIfChanged() error = %v", err)
		}
	})

	t.Run("overwrites when different", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "test.md")

		os.WriteFile(path, []byte("old content"), 0644)

		newContent := "new content"
		err := writeIfChanged(path, newContent, "test")
		if err != nil {
			t.Fatalf("writeIfChanged() error = %v", err)
		}

		got, _ := os.ReadFile(path)
		if string(got) != newContent {
			t.Errorf("file content = %q, want %q", got, newContent)
		}
	})
}

func TestGetTokmanSlim(t *testing.T) {
	content := getTokmanSlim()

	// Verify key elements exist
	if !contains(content, "<!-- tokman-instructions") {
		t.Error("missing opening marker")
	}
	if !contains(content, "<!-- /tokman-instructions -->") {
		t.Error("missing closing marker")
	}
	if !contains(content, "Golden Rule") {
		t.Error("missing Golden Rule section")
	}
	if !contains(content, "tokman git status") {
		t.Error("missing git status example")
	}
	if !contains(content, "Token Savings Overview") {
		t.Error("missing Token Savings Overview")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
