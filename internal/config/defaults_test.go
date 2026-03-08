package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConfigPath(t *testing.T) {
	// Save original env vars
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	origAppData := os.Getenv("APPDATA")
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", origXDG)
		os.Setenv("APPDATA", origAppData)
	}()

	tests := []struct {
		name     string
		xdg      string
		appData  string
		expected string
	}{
		{
			name:     "XDG override",
			xdg:      "/custom/config",
			expected: "/custom/config/tokman/config.toml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("XDG_CONFIG_HOME", tt.xdg)
			os.Setenv("APPDATA", tt.appData)

			result := ConfigPath()
			if result != tt.expected {
				t.Errorf("ConfigPath() = %q, want %q", result, tt.expected)
			}
		})
	}

	// Test default path (no env vars)
	t.Run("default path", func(t *testing.T) {
		os.Unsetenv("XDG_CONFIG_HOME")
		os.Unsetenv("APPDATA")

		result := ConfigPath()
		if runtime.GOOS == "windows" {
			// Should use home directory on Windows if APPDATA not set
			if result == "" {
				t.Error("ConfigPath() returned empty string")
			}
		} else {
			// Should contain .config on Unix
			if !filepath.IsAbs(result) {
				t.Errorf("ConfigPath() = %q, expected absolute path", result)
			}
		}
	})
}

func TestDataPath(t *testing.T) {
	origXDG := os.Getenv("XDG_DATA_HOME")
	origLocalAppData := os.Getenv("LOCALAPPDATA")
	origAppData := os.Getenv("APPDATA")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origXDG)
		os.Setenv("LOCALAPPDATA", origLocalAppData)
		os.Setenv("APPDATA", origAppData)
	}()

	t.Run("XDG override", func(t *testing.T) {
		os.Setenv("XDG_DATA_HOME", "/custom/data")
		os.Unsetenv("LOCALAPPDATA")
		os.Unsetenv("APPDATA")

		result := DataPath()
		expected := "/custom/data/tokman"
		if result != expected {
			t.Errorf("DataPath() = %q, want %q", result, expected)
		}
	})

	t.Run("default path", func(t *testing.T) {
		os.Unsetenv("XDG_DATA_HOME")
		os.Unsetenv("LOCALAPPDATA")
		os.Unsetenv("APPDATA")

		result := DataPath()
		if !filepath.IsAbs(result) {
			t.Errorf("DataPath() = %q, expected absolute path", result)
		}
	})
}

func TestDatabasePath(t *testing.T) {
	orig := os.Getenv("TOKMAN_DATABASE_PATH")
	defer os.Setenv("TOKMAN_DATABASE_PATH", orig)

	t.Run("custom path", func(t *testing.T) {
		os.Setenv("TOKMAN_DATABASE_PATH", "/custom/db.sqlite")
		result := DatabasePath()
		if result != "/custom/db.sqlite" {
			t.Errorf("DatabasePath() = %q, want /custom/db.sqlite", result)
		}
	})

	t.Run("default path", func(t *testing.T) {
		os.Unsetenv("TOKMAN_DATABASE_PATH")
		result := DatabasePath()
		if !filepath.IsAbs(result) {
			t.Errorf("DatabasePath() = %q, expected absolute path", result)
		}
		// Should end with tracking.db
		if filepath.Base(result) != "tracking.db" {
			t.Errorf("DatabasePath() = %q, expected to end with tracking.db", result)
		}
	})
}

func TestLogPath(t *testing.T) {
	result := LogPath()
	if !filepath.IsAbs(result) {
		t.Errorf("LogPath() = %q, expected absolute path", result)
	}
	if filepath.Base(result) != "tokman.log" {
		t.Errorf("LogPath() = %q, expected to end with tokman.log", result)
	}
}

func TestProjectPath(t *testing.T) {
	result := ProjectPath()
	if result == "" {
		t.Error("ProjectPath() returned empty string")
	}
	if result == "." {
		// Current directory fallback is acceptable
		t.Logf("ProjectPath() returned '.' (fallback)")
	}
}

func TestConfigPath_Windows(t *testing.T) {
	origXDG := os.Getenv("XDG_CONFIG_HOME")
	origAppData := os.Getenv("APPDATA")
	defer func() {
		os.Setenv("XDG_CONFIG_HOME", origXDG)
		os.Setenv("APPDATA", origAppData)
	}()

	t.Run("Windows APPDATA path", func(t *testing.T) {
		os.Unsetenv("XDG_CONFIG_HOME")
		os.Setenv("APPDATA", `C:\Users\TestUser\AppData\Roaming`)

		result := ConfigPath()
		// On non-Windows, APPDATA is ignored, so we just verify it doesn't crash
		t.Logf("ConfigPath with APPDATA set: %s", result)
		if result == "" {
			t.Error("ConfigPath() returned empty string")
		}
	})
}

func TestDataPath_Windows(t *testing.T) {
	origXDG := os.Getenv("XDG_DATA_HOME")
	origLocalAppData := os.Getenv("LOCALAPPDATA")
	origAppData := os.Getenv("APPDATA")
	defer func() {
		os.Setenv("XDG_DATA_HOME", origXDG)
		os.Setenv("LOCALAPPDATA", origLocalAppData)
		os.Setenv("APPDATA", origAppData)
	}()

	t.Run("Windows LOCALAPPDATA path", func(t *testing.T) {
		os.Unsetenv("XDG_DATA_HOME")
		os.Setenv("LOCALAPPDATA", `C:\Users\TestUser\AppData\Local`)
		os.Setenv("APPDATA", `C:\Users\TestUser\AppData\Roaming`)

		result := DataPath()
		t.Logf("DataPath with LOCALAPPDATA set: %s", result)
		if result == "" {
			t.Error("DataPath() returned empty string")
		}
	})

	t.Run("Windows fallback to APPDATA", func(t *testing.T) {
		os.Unsetenv("XDG_DATA_HOME")
		os.Unsetenv("LOCALAPPDATA")
		os.Setenv("APPDATA", `C:\Users\TestUser\AppData\Roaming`)

		result := DataPath()
		t.Logf("DataPath with APPDATA fallback: %s", result)
		if result == "" {
			t.Error("DataPath() returned empty string")
		}
	})
}

func TestHooksPath(t *testing.T) {
	result := HooksPath()
	if !filepath.IsAbs(result) {
		t.Errorf("HooksPath() = %q, expected absolute path", result)
	}
	if filepath.Base(result) != "hooks" {
		t.Errorf("HooksPath() = %q, expected to end with hooks", result)
	}
}

func TestPathConsistency(t *testing.T) {
	// Verify that all paths use consistent separators
	paths := []struct {
		name  string
		value string
	}{
		{"ConfigPath", ConfigPath()},
		{"DataPath", DataPath()},
		{"DatabasePath", DatabasePath()},
		{"LogPath", LogPath()},
		{"HooksPath", HooksPath()},
	}

	for _, p := range paths {
		t.Run(p.name, func(t *testing.T) {
			// Verify path is cleaned (no duplicate separators)
			cleaned := filepath.Clean(p.value)
			if p.value != cleaned {
				t.Errorf("%s = %q, should be cleaned to %q", p.name, p.value, cleaned)
			}
		})
	}
}

func TestDatabasePathInDataPath(t *testing.T) {
	// Verify database is inside data directory
	dataPath := DataPath()
	dbPath := DatabasePath()

	if !filepath.IsAbs(dbPath) {
		t.Errorf("DatabasePath() = %q, expected absolute path", dbPath)
	}

	// Database should be inside data path
	expectedDir := filepath.Dir(dbPath)
	if expectedDir != dataPath {
		t.Errorf("DatabasePath() = %q, expected to be in %q", dbPath, dataPath)
	}
}
