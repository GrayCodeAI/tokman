package reversible

import (
	"path/filepath"
	"testing"
)

func TestDefaultConfigUsesDataPath(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	cfg := DefaultConfig()
	want := filepath.Join(dataHome, "tokman", "rewind.db")
	if cfg.StorePath != want {
		t.Fatalf("DefaultConfig().StorePath = %q, want %q", cfg.StorePath, want)
	}
}
