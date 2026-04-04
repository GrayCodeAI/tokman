package filter

import (
	"path/filepath"
	"testing"
)

func TestNewReversibleStoreUsesDataPath(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	store := NewReversibleStore()
	want := filepath.Join(dataHome, "tokman", "reversible")
	if store.baseDir != want {
		t.Fatalf("ReversibleStore baseDir = %q, want %q", store.baseDir, want)
	}
}
