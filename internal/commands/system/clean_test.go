package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanDirHelpersUseDataPath(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	if got := cleanTeeDir(); got != filepath.Join(dataHome, "tokman", "tee") {
		t.Fatalf("cleanTeeDir() = %q", got)
	}
	if got := cleanReversibleDir(); got != filepath.Join(dataHome, "tokman", "reversible") {
		t.Fatalf("cleanReversibleDir() = %q", got)
	}
}

func TestRunCleanAllRemovesRecentReversibleEntries(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	revDir := cleanReversibleDir()
	if err := os.MkdirAll(revDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	recentFile := filepath.Join(revDir, "recent.txt")
	if err := os.WriteFile(recentFile, []byte("data"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	oldDays, oldAll, oldTee, oldReversible := cleanDays, cleanAll, cleanTee, cleanReversible
	cleanDays = 30
	cleanAll = true
	cleanTee = false
	cleanReversible = true
	t.Cleanup(func() {
		cleanDays, cleanAll, cleanTee, cleanReversible = oldDays, oldAll, oldTee, oldReversible
	})

	if err := runClean(nil, nil); err != nil {
		t.Fatalf("runClean() error = %v", err)
	}
	if _, err := os.Stat(recentFile); !os.IsNotExist(err) {
		t.Fatalf("recent reversible file still exists after clean --all")
	}
}
