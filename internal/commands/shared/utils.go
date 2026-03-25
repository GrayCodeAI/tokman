package shared

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// OutputType represents the type of command output.
type OutputType int

const (
	OutputTypeTest OutputType = iota
	OutputTypeBuild
	OutputTypeLog
	OutputTypeList
	OutputTypeJSON
	OutputTypeGeneric
)

// ShortenPath shortens a file path for display.
func ShortenPath(path string) string {
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) <= 4 {
		return path
	}
	return filepath.Join(parts[0], "...", parts[len(parts)-2], parts[len(parts)-1])
}

// TruncateLine truncates a line to maxLen characters.
func TruncateLine(line string, maxLen int) string {
	if len(line) <= maxLen {
		return line
	}
	return line[:maxLen-3] + "..."
}

// Truncate truncates a string to maxLen characters.
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// BuildGitCmd creates a git command with the given subcommand and args.
func BuildGitCmd(subCmd string, args ...string) *exec.Cmd {
	gitArgs := []string{}
	return exec.Command("git", append(append(gitArgs, subCmd), args...)...)
}

// TryJSONSchema generates a JSON schema from a JSON string.
func TryJSONSchema(jsonStr string, maxDepth int) string {
	var v any
	if err := json.Unmarshal([]byte(jsonStr), &v); err != nil {
		return ""
	}
	return generateSchemaFromJSON(v, 0, maxDepth)
}

func generateSchemaFromJSON(v any, depth, maxDepth int) string {
	if depth > maxDepth {
		return "..."
	}

	switch val := v.(type) {
	case nil:
		return "null"
	case bool:
		return "bool"
	case float64:
		return "number"
	case string:
		return "string"
	case []any:
		if len(val) == 0 {
			return "[]"
		}
		elemType := generateSchemaFromJSON(val[0], depth+1, maxDepth)
		return fmt.Sprintf("[%s, ...]", elemType)
	case map[string]any:
		if len(val) == 0 {
			return "{}"
		}
		var parts []string
		for k, v := range val {
			schema := generateSchemaFromJSON(v, depth+1, maxDepth)
			parts = append(parts, fmt.Sprintf("%s: %s", k, schema))
		}
		indent := strings.Repeat("  ", depth)
		return fmt.Sprintf("{\n%s  %s\n%s}", indent, strings.Join(parts, ",\n"+indent+"  "), indent)
	default:
		return fmt.Sprintf("%T", v)
	}
}

const maxArgLength = 4096
const maxArgsCount = 256

// SanitizeArgs validates and sanitizes command arguments.
// Returns error if arguments contain dangerous patterns.
func SanitizeArgs(args []string) error {
	if len(args) > maxArgsCount {
		return fmt.Errorf("too many arguments: %d (max %d)", len(args), maxArgsCount)
	}

	for i, arg := range args {
		if len(arg) > maxArgLength {
			return fmt.Errorf("argument %d exceeds max length %d", i, maxArgLength)
		}
		if strings.ContainsRune(arg, '\x00') {
			return fmt.Errorf("argument %d contains null byte", i)
		}
	}
	return nil
}

// SanitizePath validates a file path to prevent traversal attacks.
// Returns the cleaned path or an error.
func SanitizePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	if len(path) > 4096 {
		return "", fmt.Errorf("path too long: %d chars", len(path))
	}
	if strings.ContainsRune(path, '\x00') {
		return "", fmt.Errorf("path contains null byte")
	}
	cleaned := filepath.Clean(path)
	// Reject absolute paths and traversal attempts
	if filepath.IsAbs(cleaned) {
		return "", fmt.Errorf("absolute paths not allowed")
	}
	if strings.HasPrefix(cleaned, "..") {
		return "", fmt.Errorf("path traversal not allowed")
	}
	return cleaned, nil
}

// RubyExec returns an exec.Cmd for a Ruby tool, using "bundle exec" if a
// Gemfile exists in the current directory. This consolidates the duplicate
// rubyExec/rspecRubyExec/rakeRubyExec functions across packages.
func RubyExec(tool string, args ...string) *exec.Cmd {
	if _, err := os.Stat("Gemfile"); err == nil {
		if bundlePath, err := exec.LookPath("bundle"); err == nil {
			bundleArgs := []string{"exec", tool}
			bundleArgs = append(bundleArgs, args...)
			return exec.Command(bundlePath, bundleArgs...)
		}
	}
	return exec.Command(tool, args...)
}
