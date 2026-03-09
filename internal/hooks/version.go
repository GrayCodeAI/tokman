package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CurrentHookVersion is the latest hook script version.
// Increment this when the hook script changes in a breaking way.
const CurrentHookVersion = 2

// WarnIntervalSecs is how often to warn about outdated hooks (24 hours).
const WarnIntervalSecs = 24 * 3600

// VersionChecker handles hook version checking.
type VersionChecker struct {
	dataDir string
}

// NewVersionChecker creates a new version checker.
func NewVersionChecker() *VersionChecker {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, ".local", "share", "tokman")
	return &VersionChecker{dataDir: dataDir}
}

// MaybeWarn checks if the installed hook is outdated and warns once per day.
// This is non-blocking and fails silently on errors.
func (v *VersionChecker) MaybeWarn() {
	// Don't block startup — fail silently
	_ = v.checkAndWarn()
}

func (v *VersionChecker) checkAndWarn() error {
	hookPath := v.hookInstalledPath()
	if hookPath == "" {
		return nil // No hook installed
	}

	content, err := os.ReadFile(hookPath)
	if err != nil {
		return nil
	}

	installedVersion := ParseHookVersion(string(content))
	if installedVersion >= CurrentHookVersion {
		return nil // Up to date
	}

	// Rate limit: warn once per day
	marker := v.warnMarkerPath()
	if info, err := os.Stat(marker); err == nil {
		if time.Since(info.ModTime()).Seconds() < WarnIntervalSecs {
			return nil // Already warned recently
		}
	}

	// Touch marker
	os.MkdirAll(filepath.Dir(marker), 0755)
	os.WriteFile(marker, []byte{}, 0644)

	// Print warning
	fmt.Fprintf(os.Stderr, "[tokman] Hook outdated — run `tokman init` to update\n")
	return nil
}

// ParseHookVersion extracts the version number from hook script content.
// Looks for "# tokman-hook-version: N" in the first 5 lines.
func ParseHookVersion(content string) uint8 {
	lines := splitLines(content)
	for i := 0; i < 5 && i < len(lines); i++ {
		line := lines[i]
		// Check for version tag: "# tokman-hook-version: N"
		prefix := "# tokman-hook-version:"
		if strings.HasPrefix(line, prefix) {
			var v uint8
			if _, err := fmt.Sscanf(strings.TrimPrefix(line, prefix), "%d", &v); err == nil {
				return v
			}
		}
	}
	return 0 // No version tag = version 0 (outdated)
}

func (v *VersionChecker) hookInstalledPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Check standard location
	path := filepath.Join(home, ".claude", "hooks", "tokman-rewrite.sh")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	// Check tokman hooks directory
	path = filepath.Join(v.dataDir, "hooks", "tokman-rewrite.sh")
	if _, err := os.Stat(path); err == nil {
		return path
	}

	return ""
}

func (v *VersionChecker) warnMarkerPath() string {
	return filepath.Join(v.dataDir, ".hook_warn_last")
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// Global version checker
var defaultChecker = NewVersionChecker()

// MaybeWarn checks for outdated hooks using the default checker.
func MaybeWarn() {
	defaultChecker.MaybeWarn()
}
