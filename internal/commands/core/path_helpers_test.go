package core

import (
	"path/filepath"
	"testing"
)

func TestGetEnabledMarkerPathUsesDataPath(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	want := filepath.Join(dataHome, "tokman", ".enabled")
	if got := getEnabledMarkerPath(); got != want {
		t.Fatalf("getEnabledMarkerPath() = %q, want %q", got, want)
	}
}

func TestGetTeeDirFromHomeUsesDataPath(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	want := filepath.Join(dataHome, "tokman", "tee")
	if got := getTeeDirFromHome(); got != want {
		t.Fatalf("getTeeDirFromHome() = %q, want %q", got, want)
	}
}
