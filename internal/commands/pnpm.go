package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var pnpmDepth int

var pnpmCmd = &cobra.Command{
	Use:   "pnpm [args...]",
	Short: "pnpm with ultra-compact output",
	Long: `Execute pnpm commands with token-optimized output.

Provides compact output for list, outdated, install, and other pnpm commands.

Examples:
  tokman pnpm list
  tokman pnpm list --depth 1
  tokman pnpm outdated
  tokman pnpm install`,
	DisableFlagParsing: true,
	RunE:               runPnpm,
}

func init() {
	rootCmd.AddCommand(pnpmCmd)
}

func runPnpm(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		args = []string{"--help"}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: pnpm %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("pnpm", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterPnpmOutput(raw, args)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("pnpm %s", strings.Join(args, " ")), "tokman pnpm", originalTokens, filteredTokens)

	if err != nil {
		return err
	}
	return nil
}

func filterPnpmOutput(output string, args []string) string {
	if len(args) == 0 {
		return output
	}

	switch args[0] {
	case "list", "ls":
		return filterPnpmList(output)
	case "outdated":
		return filterPnpmOutdated(output)
	case "install", "add", "update":
		return filterPnpmInstall(output)
	default:
		return output
	}
}

func filterPnpmList(output string) string {
	// Compact: strip dependencies tree, show summary
	lines := strings.Split(output, "\n")
	var packages []string
	var devDeps []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "├──") || strings.HasPrefix(line, "└──") {
			// Parse package line
			pkg := strings.TrimPrefix(line, "├── ")
			pkg = strings.TrimPrefix(pkg, "└── ")
			pkg = strings.TrimSpace(pkg)
			if pkg != "" && len(pkg) < 60 {
				if strings.Contains(line, "dev:") || strings.Contains(line, "(dev)") {
					devDeps = append(devDeps, pkg)
				} else {
					packages = append(packages, pkg)
				}
			}
		}
	}

	var result []string
	if len(packages) > 0 {
		result = append(result, fmt.Sprintf("📦 Dependencies (%d):", len(packages)))
		for i, pkg := range packages {
			if i >= 15 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(packages)-15))
				break
			}
			result = append(result, fmt.Sprintf("   %s", pkg))
		}
	}

	if len(devDeps) > 0 {
		result = append(result, fmt.Sprintf("📦 DevDependencies (%d):", len(devDeps)))
		for i, pkg := range devDeps {
			if i >= 10 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(devDeps)-10))
				break
			}
			result = append(result, fmt.Sprintf("   %s", pkg))
		}
	}

	if len(result) == 0 {
		return output
	}
	return strings.Join(result, "\n")
}

func filterPnpmOutdated(output string) string {
	// Compact: "pkg: old → new"
	lines := strings.Split(output, "\n")
	var result []string
	count := 0

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Package") || strings.HasPrefix(line, "─") {
			continue
		}

		// Parse outdated line: package current wanted latest
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pkg := fields[0]
			current := fields[1]
			latest := fields[2]
			if len(fields) >= 4 {
				latest = fields[3] // use 'latest' column if available
			}
			result = append(result, fmt.Sprintf("📦 %s: %s → %s", pkg, current, latest))
			count++
		}
	}

	if count == 0 {
		return "✅ All packages up to date"
	}
	return strings.Join(result, "\n")
}

func filterPnpmInstall(output string) string {
	// Strip progress bars, show summary
	lines := strings.Split(output, "\n")
	var added, removed, changed int
	var warnings []string

	for _, line := range lines {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "added") {
			fmt.Sscanf(line, "added %d", &added)
		}
		if strings.Contains(lower, "removed") {
			fmt.Sscanf(line, "removed %d", &removed)
		}
		if strings.Contains(lower, "changed") {
			fmt.Sscanf(line, "changed %d", &changed)
		}
		if strings.Contains(lower, "warn") {
			warnings = append(warnings, truncateLine(line, 80))
		}
	}

	var result []string
	result = append(result, "📦 Install Summary:")
	if added > 0 {
		result = append(result, fmt.Sprintf("   ✅ %d added", added))
	}
	if removed > 0 {
		result = append(result, fmt.Sprintf("   🗑️  %d removed", removed))
	}
	if changed > 0 {
		result = append(result, fmt.Sprintf("   🔄 %d changed", changed))
	}

	if len(warnings) > 0 {
		result = append(result, "   ⚠️  Warnings:")
		for _, w := range warnings {
			if len(w) > 10 {
				result = append(result, fmt.Sprintf("   • %s", w))
			}
		}
	}

	if len(result) == 1 {
		return "✅ Install complete"
	}
	return strings.Join(result, "\n")
}
