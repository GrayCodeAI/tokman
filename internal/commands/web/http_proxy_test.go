package web

import (
	"path/filepath"
	"testing"
)

func TestGetProxyPIDFileUsesDataPath(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	want := filepath.Join(dataHome, "tokman", "proxy.pid")
	if got := getProxyPIDFile(); got != want {
		t.Fatalf("getProxyPIDFile() = %q, want %q", got, want)
	}
}
