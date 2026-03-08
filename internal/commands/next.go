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

var nextCmd = &cobra.Command{
	Use:   "next [args...]",
	Short: "Next.js build with compact output",
	Long: `Execute Next.js with token-optimized output.

Strips build noise and shows route summary.

Examples:
  tokman next build
  tokman next dev`,
	DisableFlagParsing: true,
	RunE:               runNext,
}

func init() {
	rootCmd.AddCommand(nextCmd)
}

func runNext(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		args = []string{"build"}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: next %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("next", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterNextOutputCompact(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("next %s", strings.Join(args, " ")), "tokman next", originalTokens, filteredTokens)

	return err
}

func filterNextOutputCompact(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var routes []string
	var staticPages, ssrPages, ssgPages int
	var errors []string
	var warnings []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect route types
		if strings.Contains(line, "○") {
			staticPages++
			routes = append(routes, truncateLine(line, 60))
		} else if strings.Contains(line, "●") {
			ssgPages++
			routes = append(routes, truncateLine(line, 60))
		} else if strings.Contains(line, "λ") || strings.Contains(line, "ƒ") {
			ssrPages++
			routes = append(routes, truncateLine(line, 60))
		}

		// Detect errors/warnings
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") {
			errors = append(errors, truncateLine(line, 100))
		} else if strings.Contains(lower, "warn") {
			warnings = append(warnings, truncateLine(line, 100))
		}
	}

	// Build result
	result = append(result, "🚀 Next.js Build Summary:")

	if staticPages > 0 || ssrPages > 0 || ssgPages > 0 {
		result = append(result, fmt.Sprintf("   📄 %d static | %d SSG | %d SSR pages", staticPages, ssgPages, ssrPages))
	}

	// Show route summary
	if len(routes) > 0 {
		result = append(result, "")
		result = append(result, "Routes:")
		for i, r := range routes {
			if i >= 15 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(routes)-15))
				break
			}
			result = append(result, fmt.Sprintf("   %s", r))
		}
	}

	// Show errors
	if len(errors) > 0 {
		result = append(result, "")
		result = append(result, fmt.Sprintf("❌ Errors (%d):", len(errors)))
		for _, e := range errors {
			result = append(result, fmt.Sprintf("   %s", e))
		}
	}

	// Show warnings
	if len(warnings) > 0 {
		result = append(result, "")
		result = append(result, fmt.Sprintf("⚠️  Warnings (%d):", len(warnings)))
		for i, w := range warnings {
			if i >= 5 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(warnings)-5))
				break
			}
			result = append(result, fmt.Sprintf("   %s", w))
		}
	}

	return strings.Join(result, "\n")
}
