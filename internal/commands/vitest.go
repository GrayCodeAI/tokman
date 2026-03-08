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

var vitestCmd = &cobra.Command{
	Use:   "vitest [args...]",
	Short: "Vitest with filtered output (90% token reduction)",
	Long: `Execute Vitest with token-optimized output.

Shows only test failures and summary.

Examples:
  tokman vitest run
  tokman vitest run --coverage`,
	DisableFlagParsing: true,
	RunE:               runVitest,
}

func init() {
	rootCmd.AddCommand(vitestCmd)
}

func runVitest(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		args = []string{"run"}
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: vitest %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("vitest", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterVitestOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("vitest %s", strings.Join(args, " ")), "tokman vitest", originalTokens, filteredTokens)

	return err
}

func filterVitestOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var passed, failed, skipped int
	var failures []string
	var inFailure bool
	var currentFailure []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect test summary
		if strings.Contains(line, "passed") || strings.Contains(line, "✓") {
			passed++
		}
		if strings.Contains(line, "failed") || strings.Contains(line, "✗") || strings.Contains(line, "FAIL") {
			if !strings.Contains(line, "0 failed") {
				failed++
				inFailure = true
				currentFailure = []string{line}
			}
		}
		if strings.Contains(line, "skipped") {
			skipped++
		}

		// Collect failure context
		if inFailure {
			if line == "" || strings.HasPrefix(line, "✓") || strings.HasPrefix(line, "RUN") {
				if len(currentFailure) > 0 {
					failures = append(failures, strings.Join(currentFailure, "\n"))
				}
				inFailure = false
				currentFailure = nil
			} else {
				currentFailure = append(currentFailure, line)
			}
		}
	}

	// Build result
	result = append(result, "📋 Vitest Results:")
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
