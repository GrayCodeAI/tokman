package shared

import (
	"path/filepath"
	"testing"
)

func TestGetConfigPathUsesCfgFileOverride(t *testing.T) {
	oldCfgFile := CfgFile
	CfgFile = filepath.Join(t.TempDir(), "custom.toml")
	t.Cleanup(func() {
		CfgFile = oldCfgFile
	})

	if got := GetConfigPath(); got != CfgFile {
		t.Fatalf("GetConfigPath() = %q, want %q", got, CfgFile)
	}
	if got := GetConfigDir(); got != filepath.Dir(CfgFile) {
		t.Fatalf("GetConfigDir() = %q, want %q", got, filepath.Dir(CfgFile))
	}
}

func TestSharedPathHelpersUseXDGDirs(t *testing.T) {
	configHome := t.TempDir()
	dataHome := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configHome)
	t.Setenv("XDG_DATA_HOME", dataHome)

	oldCfgFile := CfgFile
	CfgFile = ""
	t.Cleanup(func() {
		CfgFile = oldCfgFile
	})

	if got := GetConfigPath(); got != filepath.Join(configHome, "tokman", "config.toml") {
		t.Fatalf("GetConfigPath() = %q", got)
	}
	if got := GetDataPath(); got != filepath.Join(dataHome, "tokman") {
		t.Fatalf("GetDataPath() = %q", got)
	}
	if got := GetHooksPath(); got != filepath.Join(dataHome, "tokman", "hooks") {
		t.Fatalf("GetHooksPath() = %q", got)
	}
	if got := GetFiltersDir(); got != filepath.Join(configHome, "tokman", "filters") {
		t.Fatalf("GetFiltersDir() = %q", got)
	}
}
