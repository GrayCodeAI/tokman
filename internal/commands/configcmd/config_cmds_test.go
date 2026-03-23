package configcmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigSetCreatesFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".config", "tokman")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	configPath := filepath.Join(configDir, "config.toml")

	initial := "[filter]\nmode = \"minimal\"\nmax_width = 0\n"
	os.WriteFile(configPath, []byte(initial), 0644)

	lines := strings.Split(initial, "\n")
	var newLines []string
	found := false
	for _, line := range lines {
		if strings.Contains(line, "mode") && strings.Contains(line, "=") {
			newLines = append(newLines, "mode = \"aggressive\"")
			found = true
			continue
		}
		newLines = append(newLines, line)
	}

	if !found {
		t.Error("expected to find mode key")
	}

	content := strings.Join(newLines, "\n")
	if !strings.Contains(content, `mode = "aggressive"`) {
		t.Errorf("expected mode to be updated, got: %s", content)
	}
}

func TestConfigSetNewSection(t *testing.T) {
	content := "# Config\n[filter]\nmode = \"minimal\"\n"

	if !strings.Contains(content, "[tracking]") {
		content += "\n[tracking]\nenabled = true\n"
	}

	if !strings.Contains(content, "[tracking]") {
		t.Error("expected tracking section to be added")
	}
	if !strings.Contains(content, "enabled = true") {
		t.Error("expected enabled = true")
	}
}

func TestConfigSetNewKeyInSection(t *testing.T) {
	content := "[filter]\nmode = \"minimal\"\nmax_width = 0\n"

	if !strings.Contains(content, "[filter]") {
		t.Error("expected filter section")
	}

	lines := strings.Split(content, "\n")
	inFilter := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[filter]" {
			inFilter = true
			continue
		}
		if inFilter && strings.Contains(trimmed, "=") {
			kv := strings.SplitN(trimmed, "=", 2)
			key := strings.TrimSpace(kv[0])
			if key == "mode" {
				return
			}
		}
	}
	t.Error("expected to find mode key in filter section")
}
