package test

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
	registry.Add(func() { registry.Register(jestCmd) })
}

func runJest(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: jest %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("jest", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterJestOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "jest", err); hint != "" {
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

		if strings.Contains(line, "PASS") {
			testSuitesPassed++
		}
		if strings.Contains(line, "FAIL") {
			testSuitesFailed++
			inFailure = true
			currentFailure = []string{line}
		}

		if strings.Contains(line, "Tests:") {
			parts := strings.Fields(line)
			for i, p := range parts {
			if p == "passed" && i > 0 {
				if _, err := fmt.Sscanf(parts[i-1], "%d", &passed); err != nil {
					passed = 0
				}
			}
			if p == "failed" && i > 0 {
				if _, err := fmt.Sscanf(parts[i-1], "%d", &failed); err != nil {
					failed = 0
				}
			}
			if p == "skipped" && i > 0 {
				if _, err := fmt.Sscanf(parts[i-1], "%d", &skipped); err != nil {
					skipped = 0
				}
			}
			}
		}

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

	result = append(result, "📋 Jest Results:")
	result = append(result, fmt.Sprintf("   %d suites passed, %d suites failed", testSuitesPassed, testSuitesFailed))
	result = append(result, fmt.Sprintf("   ✅ %d tests passed", passed))
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
				if len(l) > 3 {
					result = append(result, fmt.Sprintf("   %s", shared.TruncateLine(l, 80)))
				}
			}
		}
	}

	return strings.Join(result, "\n")
}
