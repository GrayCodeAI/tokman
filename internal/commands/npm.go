package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var npmCmd = &cobra.Command{
	Use:   "npm [args...]",
	Short: "npm run with filtered output",
	Long: `npm run with token-optimized output.

Strips boilerplate and progress bars from npm output.
Special handling for npm test with 90% token reduction.

Examples:
  tokman npm run build
  tokman npm install
  tokman npm test
  tokman npm test -- --coverage`,
	DisableFlagParsing: true,
	RunE:               runNpm,
}

func init() {
	rootCmd.AddCommand(npmCmd)
}

func runNpm(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		args = []string{"--help"}
	}

	// Route npm test to specialized handler
	if len(args) > 0 && args[0] == "test" {
		return runNpmTest(args[1:])
	}

	npmArgs := append([]string{}, args...)

	c := exec.Command("npm", npmArgs...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	// Filter npm output
	filtered := filterNpmOutput(output)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("npm %s", strings.Join(args, " ")), "tokman npm", originalTokens, filteredTokens)

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", originalTokens-filteredTokens)
	}

	return err
}

func runNpmTest(args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: npm test %s\n", strings.Join(args, " "))
	}

	npmArgs := append([]string{"test"}, args...)
	c := exec.Command("npm", npmArgs...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	filtered := filterNpmTestOutput(output)

	// Add tee hint on failure
	if err != nil {
		if hint := TeeOnFailure(output, "npm_test", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("npm test %s", strings.Join(args, " ")), "tokman npm test", originalTokens, filteredTokens)

	return err
}

func filterNpmOutput(output string) string {
	var result strings.Builder
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)

		// Skip progress bars and spinners
		if strings.Contains(trimmed, "\\") || strings.Contains(trimmed, "|") || strings.Contains(trimmed, "/") {
			continue
		}
		// Skip npm WARN without actionable info
		if strings.HasPrefix(trimmed, "npm WARN") && !strings.Contains(trimmed, "deprecated") {
			continue
		}
		// Skip empty lines
		if trimmed == "" {
			continue
		}

		result.WriteString(line + "\n")
	}
	return result.String()
}

func filterNpmTestOutput(output string) string {
	lines := strings.Split(output, "\n")
	var result []string
	var passed, failed, skipped int
	var failures []string
	var inFailure bool
	var currentFailure []string
	var testSuitesPassed, testSuitesFailed int

	for _, line := range lines {
		origLine := line
		line = strings.TrimSpace(line)

		// Detect test suite status
		if strings.Contains(line, "PASS") {
			testSuitesPassed++
		}
		if strings.Contains(line, "FAIL") {
			testSuitesFailed++
			inFailure = true
			currentFailure = []string{origLine}
		}

		// Extract numbers from summary (handles both Jest and other test runners)
		if strings.Contains(line, "Tests:") || strings.Contains(line, "passed") {
			// Parse "Tests:       10 passed, 2 failed, 1 skipped" or "10 passing"
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "passed" || p == "passing" {
					if i > 0 {
						fmt.Sscanf(parts[i-1], "%d", &passed)
					}
				}
				if p == "failed" || p == "failing" {
					if i > 0 {
						fmt.Sscanf(parts[i-1], "%d", &failed)
					}
				}
				if p == "skipped" || p == "pending" {
					if i > 0 {
						fmt.Sscanf(parts[i-1], "%d", &skipped)
					}
				}
			}
		}

		// Collect failure context
		if inFailure {
			if strings.HasPrefix(line, "●") || strings.Contains(line, "expect(") ||
				strings.Contains(line, "AssertionError") || strings.Contains(line, "Error:") {
				currentFailure = append(currentFailure, origLine)
			} else if line == "" && len(currentFailure) > 1 {
				failures = append(failures, strings.Join(currentFailure, "\n"))
				inFailure = false
				currentFailure = nil
			}
		}

		// Also catch mocha-style failures
		if strings.Contains(line, "1) ") || strings.Contains(line, "2) ") || strings.Contains(line, "3) ") {
			failures = append(failures, line)
		}
	}

	// Build result
	result = append(result, "📋 npm test Results:")
	if testSuitesPassed > 0 || testSuitesFailed > 0 {
		result = append(result, fmt.Sprintf("   %d suites passed, %d suites failed", testSuitesPassed, testSuitesFailed))
	}
	if passed > 0 {
		result = append(result, fmt.Sprintf("   ✅ %d tests passed", passed))
	}
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
				if len(strings.TrimSpace(l)) > 3 {
					result = append(result, fmt.Sprintf("   %s", truncateLine(strings.TrimSpace(l), 80)))
				}
			}
		}
	}

	if passed == 0 && failed == 0 && len(result) <= 2 {
		// Fallback: show compact output
		result = result[:1]
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				result = append(result, truncateLine(strings.TrimSpace(line), 100))
				if len(result) > 20 {
					result = append(result, fmt.Sprintf("   ... (%d more lines)", len(lines)-20))
					break
				}
			}
		}
	}

	return strings.Join(result, "\n")
}
