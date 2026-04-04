package lang

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

func runRakeCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rake %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("rake", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRakeOutput(raw, args)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "rake", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	taskName := "rake"
	if len(args) > 0 {
		taskName = fmt.Sprintf("rake %s", args[0])
	}
	timer.Track(taskName, "tokman ruby rake", originalTokens, filteredTokens)

	return err
}

func filterRakeOutput(raw string, args []string) string {
	if raw == "" {
		return "✅ Rake completed"
	}

	// Check if this is a test task
	if len(args) > 0 && (args[0] == "test" || args[0] == "spec") {
		return filterRakeTestOutput(raw)
	}

	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Skip verbose rake output
		if strings.HasPrefix(line, "rake aborted!") {
			result = append(result, "- "+line)
		} else if strings.Contains(line, "error:") {
			result = append(result, "- "+shared.TruncateLine(line, 100))
		} else if len(result) < 20 {
			result = append(result, shared.TruncateLine(line, 100))
		}
	}

	if len(result) == 0 {
		return "Rake completed"
	}

	if len(result) > 20 {
		return strings.Join(result[:20], "\n") + fmt.Sprintf("\n... (%d more lines)", len(result)-20)
	}

	return strings.Join(result, "\n")
}

func filterRakeTestOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var testsRun, testsPassed, testsFailed int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for test summary patterns
		if strings.Contains(line, " tests, ") || strings.Contains(line, " assertions, ") {
			result = append(result, line)
		} else if strings.Contains(line, "FAIL") || strings.Contains(line, "error:") {
			testsFailed++
			if testsFailed <= 10 {
				result = append(result, "- "+shared.TruncateLine(line, 100))
			}
		} else if strings.Contains(line, "PASS") || strings.Contains(line, ".") {
			testsPassed++
		}

		// Count tests
		if strings.Contains(line, "test_") {
			testsRun++
		}
	}

	if len(result) == 0 {
		return "All tests passed"
	}

	if testsFailed > 10 {
		result = append(result, fmt.Sprintf("... +%d more failures", testsFailed-10))
	}

	return strings.Join(result, "\n")
}
