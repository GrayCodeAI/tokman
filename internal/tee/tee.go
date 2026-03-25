package tee

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Configuration defaults
const (
	MinTeeSize      = 500     // Minimum output size to tee (bytes)
	DefaultMaxFiles = 20      // Default max files to keep
	DefaultMaxSize  = 1 << 20 // Default max file size (1MB)
)

// Mode controls when tee writes files.
type Mode int

const (
	ModeNever    Mode = iota // Never write tee files
	ModeFailures             // Write only on command failures (default)
	ModeAlways               // Always write tee files
)

// Config configures the tee feature.
type Config struct {
	Enabled     bool   // Whether tee is enabled
	Mode        Mode   // When to write tee files
	MaxFiles    int    // Maximum number of files to keep
	MaxFileSize int    // Maximum file size in bytes
	Directory   string // Directory for tee files (empty = default)
}

// DefaultConfig returns the default tee configuration.
func DefaultConfig() Config {
	return Config{
		Enabled:     true,
		Mode:        ModeFailures,
		MaxFiles:    DefaultMaxFiles,
		MaxFileSize: DefaultMaxSize,
		Directory:   "",
	}
}

// Tee handles writing raw output to recovery files.
type Tee struct {
	config Config
}

// New creates a new Tee instance with the given configuration.
func New(config Config) *Tee {
	return &Tee{config: config}
}

// GetDirectory returns the tee directory path.
func (t *Tee) GetDirectory() (string, error) {
	// Check environment variable override
	if dir := os.Getenv("TOKMAN_TEE_DIR"); dir != "" {
		return dir, nil
	}

	// Use configured directory
	if t.config.Directory != "" {
		return t.config.Directory, nil
	}

	// Default: ~/.local/share/tokman/tee/
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", "tokman", "tee"), nil
}

// ShouldTee determines if output should be tee'd based on config and conditions.
func (t *Tee) ShouldTee(rawLen int, exitCode int) bool {
	// Check environment override
	if os.Getenv("TOKMAN_TEE") == "0" {
		return false
	}

	if !t.config.Enabled {
		return false
	}

	switch t.config.Mode {
	case ModeNever:
		return false
	case ModeFailures:
		if exitCode == 0 {
			return false
		}
	case ModeAlways:
		// Continue
	}

	// Skip small outputs
	if rawLen < MinTeeSize {
		return false
	}

	return true
}

// Write writes the raw output to a tee file.
// Returns the file path if written, or empty string if skipped.
func (t *Tee) Write(raw string, commandSlug string, exitCode int) string {
	if !t.ShouldTee(len(raw), exitCode) {
		return ""
	}

	dir, err := t.GetDirectory()
	if err != nil {
		return ""
	}

	// Create directory if needed
	if err := os.MkdirAll(dir, 0700); err != nil {
		return ""
	}

	// Sanitize slug for filename
	slug := sanitizeSlug(commandSlug)

	// Generate filename with epoch timestamp
	epoch := time.Now().Unix()
	filename := fmt.Sprintf("%d_%s.log", epoch, slug)
	filepath := filepath.Join(dir, filename)

	// Truncate if needed
	content := raw
	if len(raw) > t.config.MaxFileSize {
		content = fmt.Sprintf("%s\n\n--- truncated at %d bytes ---",
			raw[:t.config.MaxFileSize], t.config.MaxFileSize)
	}

	// Write file
	if err := os.WriteFile(filepath, []byte(content), 0600); err != nil {
		return ""
	}

	// Rotate old files
	t.cleanupOldFiles(dir)

	return filepath
}

// WriteAndHint writes the raw output and returns a formatted hint.
func (t *Tee) WriteAndHint(raw string, commandSlug string, exitCode int) string {
	path := t.Write(raw, commandSlug, exitCode)
	if path == "" {
		return ""
	}
	return FormatHint(path)
}

// cleanupOldFiles removes old tee files, keeping only the newest MaxFiles.
func (t *Tee) cleanupOldFiles(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	// Filter .log files
	var logFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".log") {
			logFiles = append(logFiles, entry)
		}
	}

	if len(logFiles) <= t.config.MaxFiles {
		return
	}

	// Sort by name (starts with epoch timestamp, so chronological)
	// We want to keep newest, so sort ascending and remove oldest
	sorted := make([]string, len(logFiles))
	for i, e := range logFiles {
		sorted[i] = e.Name()
	}
	sort.Strings(sorted)

	// Remove oldest files
	toRemove := len(sorted) - t.config.MaxFiles
	for i := 0; i < toRemove; i++ {
		if err := os.Remove(filepath.Join(dir, sorted[i])); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", sorted[i], err)
		}
	}
}

// sanitizeSlugRe is compiled once for use in sanitizeSlug.
var sanitizeSlugRe = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

// sanitizeSlug sanitizes a command slug for use in filenames.
// Replaces non-alphanumeric chars (except underscore/hyphen) with underscore,
// truncates at 40 chars.
func sanitizeSlug(slug string) string {
	// Replace non-alphanumeric (except _ and -) with _
	sanitized := sanitizeSlugRe.ReplaceAllString(slug, "_")

	// Truncate at 40 chars
	if len(sanitized) > 40 {
		sanitized = sanitized[:40]
	}

	return sanitized
}

// FormatHint formats a file path as a hint string with ~ shorthand.
func FormatHint(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}

	display := path

	if home != "" {
		if strings.HasPrefix(path, home) {
			display = "~" + strings.TrimPrefix(path, home)
		}
	}

	return fmt.Sprintf("[full output: %s]", display)
}

// Global default tee instance
var defaultTee = New(DefaultConfig())

// Write writes using the default tee instance.
func Write(raw string, commandSlug string, exitCode int) string {
	return defaultTee.Write(raw, commandSlug, exitCode)
}

// WriteAndHint writes and returns hint using the default tee instance.
func WriteAndHint(raw string, commandSlug string, exitCode int) string {
	return defaultTee.WriteAndHint(raw, commandSlug, exitCode)
}
