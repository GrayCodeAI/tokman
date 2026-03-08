package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigPath returns the path to the configuration file.
// Follows XDG Base Directory Specification on Unix.
// Uses %APPDATA% on Windows.
func ConfigPath() string {
	// Check for explicit override first
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "tokman", "config.toml")
	}

	// Windows: use %APPDATA%
	if runtime.GOOS == "windows" {
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "tokman", "config.toml")
		}
	}

	// Unix: default to ~/.config
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tokman", "config.toml")
}

// DataPath returns the path to the data directory.
// Follows XDG Base Directory Specification on Unix.
// Uses %LOCALAPPDATA% on Windows.
func DataPath() string {
	// Check for explicit override first
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "tokman")
	}

	// Windows: use %LOCALAPPDATA%
	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "tokman")
		}
		// Fallback to APPDATA if LOCALAPPDATA not set
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "tokman", "data")
		}
	}

	// Unix: default to ~/.local/share
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "tokman")
}

// DatabasePath returns the path to the SQLite database.
func DatabasePath() string {
	if custom := os.Getenv("TOKMAN_DATABASE_PATH"); custom != "" {
		return custom
	}
	return filepath.Join(DataPath(), "tracking.db")
}

// LogPath returns the path to the log file.
func LogPath() string {
	return filepath.Join(DataPath(), "tokman.log")
}

// HooksPath returns the path to the hooks directory.
func HooksPath() string {
	return filepath.Join(DataPath(), "hooks")
}

// ProjectPath returns the canonical path for the current working directory.
// Resolves symlinks for accurate project matching.
func ProjectPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "."
	}
	canonical, err := filepath.EvalSymlinks(cwd)
	if err != nil {
		return cwd
	}
	return canonical
}
