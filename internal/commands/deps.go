package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var depsCmd = &cobra.Command{
	Use:   "deps [path]",
	Short: "Summarize project dependencies",
	Long: `Summarize project dependencies for various ecosystems.

Auto-detects package manager from project files (go.mod, package.json, Cargo.toml, etc).

Examples:
  tokman deps
  tokman deps /path/to/project`,
	RunE: runDeps,
}

func init() {
	rootCmd.AddCommand(depsCmd)
}

func runDeps(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	var output strings.Builder

	// Detect and show dependencies based on project type
	if hasFile(path, "go.mod") {
		output.WriteString(summarizeGoDeps(path))
	}
	if hasFile(path, "package.json") {
		output.WriteString(summarizeJSDeps(path))
	}
	if hasFile(path, "Cargo.toml") {
		output.WriteString(summarizeRustDeps(path))
	}
	if hasFile(path, "requirements.txt") || hasFile(path, "pyproject.toml") {
		output.WriteString(summarizePythonDeps(path))
	}

	result := output.String()
	if result == "" {
		result = "No recognized package manager found.\n"
	}

	fmt.Print(result)

	originalTokens := filter.EstimateTokens(result)
	filteredTokens := filter.EstimateTokens(result)
	timer.Track("deps", "tokman deps", originalTokens, filteredTokens)

	return nil
}

func hasFile(path, name string) bool {
	_, err := os.Stat(filepath.Join(path, name))
	return err == nil
}

func summarizeGoDeps(path string) string {
	data, err := os.ReadFile(filepath.Join(path, "go.mod"))
	if err != nil {
		return ""
	}

	var result strings.Builder
	result.WriteString("=== Go Dependencies ===\n")

	lines := strings.Split(string(data), "\n")
	inRequire := false
	count := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "require (" {
			inRequire = true
			continue
		}
		if inRequire && trimmed == ")" {
			inRequire = false
			continue
		}
		if inRequire && trimmed != "" {
			parts := strings.Fields(trimmed)
			if len(parts) >= 1 {
				result.WriteString(fmt.Sprintf("  %s\n", parts[0]))
				count++
			}
		} else if strings.HasPrefix(trimmed, "require ") {
			// Single-line require
			parts := strings.Fields(trimmed)
			if len(parts) >= 2 {
				result.WriteString(fmt.Sprintf("  %s\n", parts[1]))
				count++
			}
		}
	}

	result.WriteString(fmt.Sprintf("Total: %d dependencies\n\n", count))
	return result.String()
}

func summarizeJSDeps(path string) string {
	// Simplified - just note package.json exists
	result := "=== JS/TS Dependencies ===\n"
	result += "  See package.json for details\n\n"
	return result
}

func summarizeRustDeps(path string) string {
	data, err := os.ReadFile(filepath.Join(path, "Cargo.toml"))
	if err != nil {
		return ""
	}

	var result strings.Builder
	result.WriteString("=== Rust Dependencies ===\n")

	lines := strings.Split(string(data), "\n")
	inDeps := false
	count := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "[dependencies]" {
			inDeps = true
			continue
		}
		if strings.HasPrefix(trimmed, "[") && trimmed != "[dependencies]" {
			inDeps = false
		}
		if inDeps && trimmed != "" && !strings.HasPrefix(trimmed, "#") {
			parts := strings.SplitN(trimmed, "=", 2)
			if len(parts) >= 1 {
				name := strings.TrimSpace(parts[0])
				result.WriteString(fmt.Sprintf("  %s\n", name))
				count++
			}
		}
	}

	result.WriteString(fmt.Sprintf("Total: %d dependencies\n\n", count))
	return result.String()
}

func summarizePythonDeps(path string) string {
	result := "=== Python Dependencies ===\n"
	result += "  See requirements.txt or pyproject.toml for details\n\n"
	return result
}
