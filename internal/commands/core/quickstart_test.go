package core

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectAgentsRecognizesLegacyQuickstartHook(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(settingsPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	legacyHookPath := filepath.Join(home, ".claude", "hooks", "tokman.sh")
	if err := os.MkdirAll(filepath.Dir(legacyHookPath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(legacyHookPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	agents := detectAgents()
	for _, agent := range agents {
		if agent.Name != "Claude Code" {
			continue
		}
		if !agent.Detected {
			t.Fatal("Claude Code should be detected")
		}
		if !agent.Configured {
			t.Fatal("legacy quickstart hook should mark Claude Code as configured")
		}
		if filepath.Base(agent.HookPath) != "tokman-rewrite.sh" {
			t.Fatalf("HookPath = %q, want tokman-rewrite.sh", agent.HookPath)
		}
		return
	}

	t.Fatal("Claude Code agent entry not found")
}
