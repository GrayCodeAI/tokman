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

func runRailsCmd(args []string) error {
	if len(args) == 0 {
		args = []string{"--help"}
	}

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rails %s\n", strings.Join(args, " "))
	}

	// Route to specialized handlers
	switch args[0] {
	case "test", "t":
		return runRailsTestCmd(args[1:])
	case "db:migrate":
		return runRailsDbMigrateCmd(args[1:])
	default:
		return runRailsPassthrough(args)
	}
}

func runRailsTestCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rails test %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("rails", append([]string{"test"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRailsTestOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "rails_test", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("rails test", "tokman ruby rails test", originalTokens, filteredTokens)

	return err
}

func filterRailsTestOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var passed, failed int
	var failures []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for test result lines
		if strings.HasPrefix(line, ".") || strings.HasPrefix(line, "F") || strings.HasPrefix(line, "E") {
			for _, c := range line {
				if c == '.' {
					passed++
				} else if c == 'F' || c == 'E' {
					failed++
				}
			}
		} else if strings.Contains(line, " runs, ") {
			result = append(result, line)
		} else if strings.Contains(line, "FAIL") || strings.Contains(line, "ERROR") {
			failures = append(failures, shared.TruncateLine(line, 100))
		}
	}

	// Ultra-compact mode
	if shared.UltraCompact {
		var parts []string
		parts = append(parts, fmt.Sprintf("P:%d", passed))
		if failed > 0 {
			parts = append(parts, fmt.Sprintf("F:%d", failed))
		}
		return strings.Join(parts, " ")
	}

	// Normal output
	if len(result) == 0 {
		if passed > 0 || failed > 0 {
			result = append(result, "Rails Test Results:")
			result = append(result, fmt.Sprintf("   Passed: %d", passed))
			if failed > 0 {
				result = append(result, fmt.Sprintf("   Failed: %d", failed))
			}
		}
	}

	if len(failures) > 0 {
		result = append(result, "")
		result = append(result, "Failures:")
		for i, f := range failures {
			if i >= 10 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(failures)-10))
				break
			}
			result = append(result, fmt.Sprintf("   - %s", f))
		}
	}

	if len(result) == 0 {
		return "Tests passed"
	}

	return strings.Join(result, "\n")
}

func runRailsDbMigrateCmd(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rails db:migrate %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("rails", append([]string{"db:migrate"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRailsDbMigrateOutput(raw)

	// Add tee hint on failure
	if err != nil {
		if hint := shared.TeeOnFailure(raw, "rails_db_migrate", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("rails db:migrate", "tokman ruby rails db:migrate", originalTokens, filteredTokens)

	return err
}

func filterRailsDbMigrateOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var migrations []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Look for migration output
		if strings.Contains(line, "==") && strings.Contains(line, "migrating") {
			// Migration start
			parts := strings.Fields(line)
			if len(parts) >= 1 {
				migrations = append(migrations, parts[0])
			}
		} else if strings.Contains(line, "migrated") {
			result = append(result, "+ "+shared.TruncateLine(line, 80))
		} else if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			result = append(result, "- "+shared.TruncateLine(line, 100))
		}
	}

	if len(result) == 0 {
		if len(migrations) > 0 {
			return fmt.Sprintf("%d migrations completed", len(migrations))
		}
		return "Database migrated"
	}

	return strings.Join(result, "\n")
}

func runRailsPassthrough(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: rails %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("rails", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterRailsOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("rails %s", args[0]), "tokman ruby rails", originalTokens, filteredTokens)

	return err
}

func filterRailsOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, shared.TruncateLine(line, 120))
		}
	}

	if len(result) > 30 {
		return strings.Join(result[:30], "\n") + fmt.Sprintf("\n... (%d more lines)", len(result)-30)
	}
	return strings.Join(result, "\n")
}
