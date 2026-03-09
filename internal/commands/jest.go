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

var jestCmd = &cobra.Command{
	Use:   "jest [args...]",
	Short: "Jest test runner with filtered output (90% token reduction)",
	Long: `Execute Jest with token-optimized output.

Shows only test failures and summary.

Examples:
  tokman jest
  tokman jest --coverage
  tokman jest src/__tests__/mytest.test.ts`,
	DisableFlagParsing: true,
	RunE:               runJest,
}

func init() {
	rootCmd.AddCommand(jestCmd)
}

func runJest(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: jest %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("jest", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterJestOutput(raw)
	
	// Add tee hint on failure
	if err != nil {
		if hint := TeeOnFailure(raw, "jest", err); hint != "" {
			filtered += "\n" + hint
		}
	}
	
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("jest %s", strings.Join(args, " ")), "tokman jest", originalTokens, filteredTokens)

	return err
}

func filterJestOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var passed, failed, skipped int
	var failures []string
	var inFailure bool
	var currentFailure []string
	var testSuitesPassed, testSuitesFailed int

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect test summary
		if strings.Contains(line, "PASS") {
			testSuitesPassed++
		}
		if strings.Contains(line, "FAIL") {
			testSuitesFailed++
			inFailure = true
			currentFailure = []string{line}
		}

		// Extract numbers from summary
		if strings.Contains(line, "Tests:") {
			// Parse "Tests:       10 passed, 2 failed, 1 skipped"
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "passed" && i > 0 {
					fmt.Sscanf(parts[i-1], "%d", &passed)
				}
				if p == "failed" && i > 0 {
					fmt.Sscanf(parts[i-1], "%d", &failed)
				}
				if p == "skipped" && i > 0 {
					fmt.Sscanf(parts[i-1], "%d", &skipped)
				}
			}
		}

		// Collect failure context
		if inFailure {
			if strings.HasPrefix(line, "●") || strings.Contains(line, "expect(") || strings.Contains(line, "Received:") {
				currentFailure = append(currentFailure, line)
			} else if line == "" && len(currentFailure) > 1 {
				failures = append(failures, strings.Join(currentFailure, "\n"))
				inFailure = false
				currentFailure = nil
			}
		}
	}

	// Build result
	result = append(result, "📋 Jest Results:")
	result = append(result, fmt.Sprintf("   %d suites passed, %d suites failed", testSuitesPassed, testSuitesFailed))
	result = append(result, fmt.Sprintf("   ✅ %d tests passed", passed))
	if failed > 0 {
		result = append(result, fmt.Sprintf("   ❌ %d tests failed", failed))
	}
	if skipped > 0 {
		result = append(result, fmt.Sprintf("   ⏭️  %d tests skipped", skipped))
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
				if len(l) > 3 {
					result = append(result, fmt.Sprintf("   %s", truncateLine(l, 80)))
				}
			}
		}
	}

	return strings.Join(result, "\n")
}
