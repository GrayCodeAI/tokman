package system

import (
	"path/filepath"
	"testing"
)

func TestMemoryStorePathUsesDataPath(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	want := filepath.Join(dataHome, "tokman", "memory.json")
	if got := memoryStorePath(); got != want {
		t.Fatalf("memoryStorePath() = %q, want %q", got, want)
	}
}
