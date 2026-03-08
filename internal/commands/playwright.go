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

var playwrightCmd = &cobra.Command{
	Use:   "playwright [args...]",
	Short: "Playwright E2E tests with compact output",
	Long: `Execute Playwright with token-optimized output.

Shows only test failures and summary.

Examples:
  tokman playwright test
  tokman playwright test --project=chromium`,
	DisableFlagParsing: true,
	RunE:               runPlaywright,
}

func init() {
	rootCmd.AddCommand(playwrightCmd)
}

func runPlaywright(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		args = []string{"test"}
	}

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: playwright %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("playwright", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterPlaywrightOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("playwright %s", strings.Join(args, " ")), "tokman playwright", originalTokens, filteredTokens)

	return err
}

func filterPlaywrightOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var passed, failed, skipped int
	var failures []string
	var inFailure bool
	var currentFailure []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Parse test results
		if strings.Contains(line, "passed") {
			fmt.Sscanf(line, "%d passed", &passed)
		}
		if strings.Contains(line, "failed") {
			fmt.Sscanf(line, "%d failed", &failed)
		}
		if strings.Contains(line, "skipped") {
			fmt.Sscanf(line, "%d skipped", &skipped)
		}

		// Detect individual test failures
		if strings.Contains(line, "✘") || strings.Contains(line, "FAIL") {
			inFailure = true
			currentFailure = []string{line}
		}

		// Collect failure context
		if inFailure {
			currentFailure = append(currentFailure, line)
			if strings.HasPrefix(line, "   at ") || line == "" {
				if len(currentFailure) > 1 {
					failures = append(failures, strings.Join(currentFailure, "\n"))
				}
				inFailure = false
				currentFailure = nil
			}
		}
	}

	// Build result
	result = append(result, "🎭 Playwright Results:")
	result = append(result, fmt.Sprintf("   ✅ %d passed", passed))
	if failed > 0 {
		result = append(result, fmt.Sprintf("   ❌ %d failed", failed))
	}
	if skipped > 0 {
		result = append(result, fmt.Sprintf("   ⏭️  %d skipped", skipped))
	}

	// Show failures
	if len(failures) > 0 {
		result = append(result, "")
		result = append(result, "Failures:")
		for i, f := range failures {
			if i >= 5 {
				result = append(result, fmt.Sprintf("   ... +%d more failures", len(failures)-5))
				break
			}
			for _, l := range strings.Split(f, "\n") {
				if len(l) > 5 {
					result = append(result, fmt.Sprintf("   %s", truncateLine(l, 80)))
				}
			}
		}
	}

	return strings.Join(result, "\n")
}
