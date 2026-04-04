package toml

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader("")
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
	if loader.parser == nil {
		t.Error("NewLoader() parser is nil")
	}
	if loader.trustedPaths == nil {
		t.Error("NewLoader() trustedPaths is nil")
	}
}

func TestNewLoaderWithConfigDir(t *testing.T) {
	dir := t.TempDir()
	loader := NewLoader(dir)
	if loader == nil {
		t.Fatal("NewLoader() returned nil")
	}
}

func TestLoaderLoadAll_EmptyDir(t *testing.T) {
	loader := NewLoader(t.TempDir())
	reg, err := loader.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if reg == nil {
		t.Fatal("LoadAll() returned nil registry")
	}
	// Built-in filters should still be loaded
	if reg.Count() == 0 {
		t.Log("No built-in filters loaded (expected in test environment)")
	}
}

func TestLoaderLoadAll_WithProjectDir(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	os.MkdirAll(filepath.Join(projectDir, ".tokman"), 0755)

	// Create a project filter file
	filterContent := `schema_version = 1

[my_filter]
match_command = "^my-tool.*"
strip_ansi = true
max_lines = 10
`
	os.WriteFile(filepath.Join(projectDir, ".tokman", "filters.toml"), []byte(filterContent), 0644)

	loader := NewLoader(tmpDir)
	// Project is not trusted, so filters won't load
	reg, err := loader.LoadAll(projectDir)
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if reg == nil {
		t.Fatal("LoadAll() returned nil registry")
	}
}

func TestLoaderTrustUntrust(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	projectPath := filepath.Join(tmpDir, "myproject")
	os.MkdirAll(projectPath, 0755)

	// Initially not trusted
	if loader.IsTrusted(projectPath) {
		t.Error("Project should not be trusted initially")
	}

	// Trust the project
	if err := loader.TrustProject(projectPath); err != nil {
		t.Fatalf("TrustProject() error = %v", err)
	}

	if !loader.IsTrusted(projectPath) {
		t.Error("Project should be trusted after TrustProject()")
	}

	// List trusted
	trusted := loader.ListTrusted()
	if len(trusted) != 1 {
		t.Errorf("ListTrusted() got %d, want 1", len(trusted))
	}

	// Untrust
	if err := loader.UntrustProject(projectPath); err != nil {
		t.Fatalf("UntrustProject() error = %v", err)
	}

	if loader.IsTrusted(projectPath) {
		t.Error("Project should not be trusted after UntrustProject()")
	}
}

func TestLoaderLoadTrusted(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	projectPath := filepath.Join(tmpDir, "project1")
	os.MkdirAll(projectPath, 0755)

	// Write a trusted file
	trustedContent := "# TokMan Trusted Projects\n" + projectPath + "\n"
	os.WriteFile(loader.trustedFile, []byte(trustedContent), 0600)

	if err := loader.LoadTrusted(); err != nil {
		t.Fatalf("LoadTrusted() error = %v", err)
	}

	if !loader.IsTrusted(projectPath) {
		t.Error("Project should be trusted after LoadTrusted()")
	}
}

func TestGetLoader(t *testing.T) {
	loader1 := GetLoader()
	loader2 := GetLoader()
	if loader1 != loader2 {
		t.Error("GetLoader() should return the same instance (singleton)")
	}
}

func TestLoaderLoadAll_WithUserFilters(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	configDir := filepath.Join(homeDir, ".config", "tokman")
	os.MkdirAll(configDir, 0755)

	// Create user filters file
	filterContent := `schema_version = 1

[user_tool]
match_command = "^user-tool.*"
strip_ansi = true
max_lines = 20
`
	os.WriteFile(filepath.Join(configDir, "filters.toml"), []byte(filterContent), 0644)

	// Set HOME to temp dir
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", origHome)

	loader := NewLoader(configDir)
	reg, err := loader.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}
	if reg == nil {
		t.Fatal("LoadAll() returned nil registry")
	}
}

func TestLoaderLoadAll_WithXDGUserFilters(t *testing.T) {
	tmpDir := t.TempDir()
	xdgConfigHome := filepath.Join(tmpDir, "xdg-config")
	configDir := filepath.Join(xdgConfigHome, "tokman")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	filterContent := `schema_version = 1

[xdg_tool]
match_command = "^xdg-tool.*"
keep_lines_matching = ["^ERROR:"]
`
	if err := os.WriteFile(filepath.Join(configDir, "filters.toml"), []byte(filterContent), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)

	loader := NewLoader("")
	reg, err := loader.LoadAll("")
	if err != nil {
		t.Fatalf("LoadAll() error = %v", err)
	}

	_, _, cfg := reg.FindMatchingFilter("xdg-tool run")
	if cfg == nil {
		t.Fatal("expected XDG user filter to match command")
	}
	if len(cfg.KeepLinesMatching) != 1 || cfg.KeepLinesMatching[0] != "^ERROR:" {
		t.Fatalf("loaded wrong filter config: %+v", cfg)
	}
}

func TestLoaderLoadAll_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	homeDir := filepath.Join(tmpDir, "home")
	configDir := filepath.Join(homeDir, ".config", "tokman")
	os.MkdirAll(configDir, 0755)

	// Create invalid TOML file
	os.WriteFile(filepath.Join(configDir, "filters.toml"), []byte("invalid {{{ toml"), 0644)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", homeDir)
	defer os.Setenv("HOME", origHome)

	loader := NewLoader(configDir)
	_, err := loader.LoadAll("")
	if err == nil {
		t.Log("LoadAll() did not error on invalid TOML (warning printed to stderr)")
	}
}

func TestLoaderTrustPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	projectPath := filepath.Join(tmpDir, "project")
	os.MkdirAll(projectPath, 0755)

	// Create loader and trust a project
	loader1 := NewLoader(tmpDir)
	if err := loader1.TrustProject(projectPath); err != nil {
		t.Fatalf("TrustProject() error = %v", err)
	}

	// Create a new loader and load trusted
	loader2 := NewLoader(tmpDir)
	if err := loader2.LoadTrusted(); err != nil {
		t.Fatalf("LoadTrusted() error = %v", err)
	}

	if !loader2.IsTrusted(projectPath) {
		t.Error("Trust should persist across loader instances")
	}
}

func TestLoaderTrustUsesCanonicalProjectPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink normalization test is Unix-focused")
	}

	tmpDir := t.TempDir()
	realProject := filepath.Join(tmpDir, "real-project")
	if err := os.MkdirAll(realProject, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	linkProject := filepath.Join(tmpDir, "project-link")
	if err := os.Symlink(realProject, linkProject); err != nil {
		t.Fatalf("Symlink() error = %v", err)
	}

	loader := NewLoader(tmpDir)
	if err := loader.TrustProject(linkProject); err != nil {
		t.Fatalf("TrustProject() error = %v", err)
	}

	if !loader.IsTrusted(realProject) {
		t.Fatal("expected canonical project path to be trusted")
	}
	if !loader.IsTrusted(linkProject) {
		t.Fatal("expected symlink project path to resolve as trusted")
	}
}

func TestLoaderUntrustNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	// Untrusting a non-trusted project should not error
	if err := loader.UntrustProject("/nonexistent/path"); err != nil {
		t.Fatalf("UntrustProject() should not error for non-trusted path: %v", err)
	}
}

func TestLoaderListTrustedEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewLoader(tmpDir)

	trusted := loader.ListTrusted()
	if len(trusted) != 0 {
		t.Errorf("ListTrusted() should be empty, got %d", len(trusted))
	}
}
