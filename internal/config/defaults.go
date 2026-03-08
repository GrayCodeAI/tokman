package config

import (
	"os"
	"path/filepath"
)

// ConfigPath returns the path to the configuration file.
// Follows XDG Base Directory Specification.
func ConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "tokman", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "tokman", "config.toml")
}

// DataPath returns the path to the data directory.
// Follows XDG Base Directory Specification.
func DataPath() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "tokman")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "tokman")
}

// DatabasePath returns the path to the SQLite database.
func DatabasePath() string {
	if custom := os.Getenv("TOKMAN_DATABASE_PATH"); custom != "" {
		return custom
	}
	return filepath.Join(DataPath(), "history.db")
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
