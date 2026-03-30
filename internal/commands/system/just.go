package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var justCmd = &cobra.Command{
	Use:   "just [recipe] [args...]",
	Short: "Just command runner with compact output",
	Long: `Execute Just recipes with token-optimized output.

Specialized filters for:
  - Recipe execution: Compact output
  - List: Compact recipe listing

Examples:
  tokman just build
  tokman just test
  tokman just --list`,
	DisableFlagParsing: true,
	RunE:               runJust,
}

func init() {
	registry.Add(func() { registry.Register(justCmd) })
}

func runJust(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return runJustList()
	}

	// Check for list flag
	for _, arg := range args {
		if arg == "-l" || arg == "--list" {
			return runJustList()
		}
	}

	return runJustRecipe(args)
}

func runJustRecipe(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: just %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("just", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterJustOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "just", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("just", "tokman just", originalTokens, filteredTokens)

	return err
}

func filterJustOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Keep errors
		if strings.Contains(trimmed, "error:") || strings.HasPrefix(trimmed, "error:") {
			result = append(result, line)
			continue
		}

		// In ultra-compact mode, be very selective
		if shared.UltraCompact {
			// Keep only error lines
			if strings.Contains(trimmed, "Error") || strings.Contains(trimmed, "FAILED") {
				result = append(result, shared.TruncateLine(line, 100))
			}
			continue
		}

		result = append(result, shared.TruncateLine(line, 120))
	}

	return strings.Join(result, "\n")
}

func runJustList() error {
	timer := tracking.Start()

	execCmd := exec.Command("just", "--list")
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterJustListOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("just list", "tokman just list", originalTokens, filteredTokens)

	return err
}

func filterJustListOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Skip "Available recipes:" header
		if strings.HasPrefix(trimmed, "Available") {
			if !shared.UltraCompact {
				result = append(result, line)
			}
			continue
		}

		if shared.UltraCompact {
			// Just show recipe names
			if strings.HasPrefix(trimmed, "just") {
				// Skip the "just" prefix in listing
				continue
			}
			// Extract recipe name (first word)
			recipe := strings.Fields(trimmed)[0]
			result = append(result, recipe)
		} else {
			result = append(result, shared.TruncateLine(line, 80))
		}
	}

	return strings.Join(result, "\n")
}
