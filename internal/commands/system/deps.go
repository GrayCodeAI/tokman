package system

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
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
	registry.Add(func() { registry.Register(depsCmd) })
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
	data, err := os.ReadFile(filepath.Join(path, "package.json"))
	if err != nil {
		return ""
	}

	var result strings.Builder
	result.WriteString("=== JS/TS Dependencies ===\n")

	// Simple JSON parsing - extract "dependencies" and "devDependencies" keys
	// without importing encoding/json for a lightweight parser
	inDeps := false
	inDevDeps := false
	depth := 0
	count := 0
	devCount := 0

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Detect section starts
		if strings.Contains(trimmed, `"dependencies"`) && strings.Contains(trimmed, ":") {
			inDeps = true
			inDevDeps = false
			depth = 0
			continue
		}
		if strings.Contains(trimmed, `"devDependencies"`) && strings.Contains(trimmed, ":") {
			inDevDeps = true
			inDeps = false
			depth = 0
			continue
		}

		if inDeps || inDevDeps {
			depth += strings.Count(trimmed, "{") - strings.Count(trimmed, "}")
			if depth <= 0 && strings.Contains(trimmed, "}") {
				inDeps = false
				inDevDeps = false
				continue
			}

			// Extract package name from "package-name": "version"
			if strings.Contains(trimmed, `":`) {
				parts := strings.SplitN(trimmed, `":`, 2)
				name := strings.Trim(strings.TrimSpace(parts[0]), `"`)
				if name != "" && !strings.Contains(name, " ") {
					if inDevDeps {
						result.WriteString(fmt.Sprintf("  %s (dev)\n", name))
						devCount++
					} else {
						result.WriteString(fmt.Sprintf("  %s\n", name))
						count++
					}
				}
			}
		}
	}

	total := count + devCount
	result.WriteString(fmt.Sprintf("Total: %d dependencies\n\n", total))
	return result.String()
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
	var result strings.Builder
	result.WriteString("=== Python Dependencies ===\n")
	count := 0

	// Parse requirements.txt
	if data, err := os.ReadFile(filepath.Join(path, "requirements.txt")); err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			// Extract package name (before ==, >=, <=, ~=, !=)
			name := line
			for _, sep := range []string{"==", ">=", "<=", "~=", "!=", ">", "<"} {
				if idx := strings.Index(name, sep); idx > 0 {
					name = name[:idx]
					break
				}
			}
			name = strings.TrimSpace(name)
			if name != "" {
				result.WriteString(fmt.Sprintf("  %s\n", name))
				count++
			}
		}
	}

	// Parse pyproject.toml dependencies section
	if data, err := os.ReadFile(filepath.Join(path, "pyproject.toml")); err == nil {
		lines := strings.Split(string(data), "\n")
		inDeps := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "dependencies") && strings.Contains(trimmed, "[") {
				inDeps = true
				continue
			}
			if inDeps {
				if strings.HasPrefix(trimmed, "[") {
					break
				}
				trimmed = strings.Trim(trimmed, `", `)
				if trimmed == "" {
					continue
				}
				// Extract package name
				name := trimmed
				for _, sep := range []string{">=", "<=", "==", "~=", "!=", ">", "<"} {
					if idx := strings.Index(name, sep); idx > 0 {
						name = name[:idx]
						break
					}
				}
				name = strings.TrimSpace(name)
				if name != "" {
					result.WriteString(fmt.Sprintf("  %s\n", name))
					count++
				}
			}
		}
	}

	result.WriteString(fmt.Sprintf("Total: %d dependencies\n\n", count))
	return result.String()
}
