package pkgmgr

import (
	"bytes"
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
	registry.Add(func() { registry.Register(npmCmd) })
}

func runNpm(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		args = []string{"--help"}
	}

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

	filtered := filterNpmOutput(output)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("npm %s", strings.Join(args, " ")), "tokman npm", originalTokens, filteredTokens)

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", originalTokens-filteredTokens)
	}

	return err
}

func runNpmTest(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
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

	if err != nil {
		if hint := shared.TeeOnFailure(output, "npm_test", err); hint != "" {
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

		if strings.Contains(trimmed, "\\") || strings.Contains(trimmed, "|") || strings.Contains(trimmed, "/") {
			continue
		}
		if strings.HasPrefix(trimmed, "npm WARN") && !strings.Contains(trimmed, "deprecated") {
			continue
		}
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

		if strings.Contains(line, "PASS") {
			testSuitesPassed++
		}
		if strings.Contains(line, "FAIL") {
			testSuitesFailed++
			inFailure = true
			currentFailure = []string{origLine}
		}

		if strings.Contains(line, "Tests:") || strings.Contains(line, "passed") {
			parts := strings.Fields(line)
			for i, p := range parts {
			if p == "passed" || p == "passing" {
				if i > 0 {
					if _, err := fmt.Sscanf(parts[i-1], "%d", &passed); err != nil {
						passed = 0
					}
				}
			}
			if p == "failed" || p == "failing" {
				if i > 0 {
					if _, err := fmt.Sscanf(parts[i-1], "%d", &failed); err != nil {
						failed = 0
					}
				}
			}
			if p == "skipped" || p == "pending" {
				if i > 0 {
					if _, err := fmt.Sscanf(parts[i-1], "%d", &skipped); err != nil {
						skipped = 0
					}
				}
			}
			}
		}

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

		if strings.Contains(line, "1) ") || strings.Contains(line, "2) ") || strings.Contains(line, "3) ") {
			failures = append(failures, line)
		}
	}

	if shared.UltraCompact {
		return filterNpmTestOutputUltraCompact(passed, failed, skipped, testSuitesPassed, testSuitesFailed, failures)
	}

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
					result = append(result, fmt.Sprintf("   %s", shared.TruncateLine(strings.TrimSpace(l), 80)))
				}
			}
		}
	}

	if passed == 0 && failed == 0 && len(result) <= 2 {
		result = result[:1]
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				result = append(result, shared.TruncateLine(strings.TrimSpace(line), 100))
				if len(result) > 20 {
					result = append(result, fmt.Sprintf("   ... (%d more lines)", len(lines)-20))
					break
				}
			}
		}
	}

	return strings.Join(result, "\n")
}

func filterNpmTestOutputUltraCompact(passed, failed, skipped, suitesPassed, suitesFailed int, failures []string) string {
	var parts []string

	if suitesPassed > 0 || suitesFailed > 0 {
		parts = append(parts, fmt.Sprintf("S:%d/%d", suitesPassed, suitesPassed+suitesFailed))
	}

	parts = append(parts, fmt.Sprintf("P:%d", passed))
	if failed > 0 {
		parts = append(parts, fmt.Sprintf("F:%d", failed))
	}
	if skipped > 0 {
		parts = append(parts, fmt.Sprintf("S:%d", skipped))
	}

	var result []string
	result = append(result, strings.Join(parts, " "))

	if len(failures) > 0 {
		for i, f := range failures {
			if i >= 3 {
				result = append(result, fmt.Sprintf("... +%d more", len(failures)-3))
				break
			}
			lines := strings.Split(f, "\n")
			for _, l := range lines {
				l = strings.TrimSpace(l)
				if l != "" && len(l) > 3 {
					result = append(result, shared.TruncateLine(l, 60))
					break
				}
			}
		}
	}

	return strings.Join(result, "\n")
}
