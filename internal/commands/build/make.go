package build

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

var makeCmd = &cobra.Command{
	Use:   "make [target] [args...]",
	Short: "Make build commands with compact output",
	Long: `Execute Make commands with token-optimized output.

Specialized filters for:
  - target execution: Compact build output
  - help: Compact target listing

Examples:
  tokman make build
  tokman make test
  tokman make help`,
	DisableFlagParsing: true,
	RunE:               runMake,
}

func init() {
	registry.Add(func() { registry.Register(makeCmd) })
}

func runMake(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// No args - show targets
		return runMakeHelp()
	}

	// Check for help flag
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			return runMakeHelp()
		}
	}

	return runMakeTarget(args)
}

func runMakeTarget(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: make %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("make", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterMakeOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "make", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("make", "tokman make", originalTokens, filteredTokens)

	return err
}

func filterMakeOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var lastWasEnter bool

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Skip entering/leaving directory messages unless verbose
		if strings.HasPrefix(trimmed, "make[") && strings.Contains(trimmed, "Entering directory") {
			if shared.Verbose > 0 {
				result = append(result, shared.TruncateLine(line, 100))
			}
			lastWasEnter = true
			continue
		}
		if strings.HasPrefix(trimmed, "make[") && strings.Contains(trimmed, "Leaving directory") {
			continue
		}

		// Keep error messages
		if strings.Contains(trimmed, "error:") || strings.Contains(trimmed, "Error") {
			result = append(result, line)
			continue
		}

		// Keep warnings unless ultra-compact
		if !shared.UltraCompact && strings.Contains(trimmed, "warning:") {
			result = append(result, shared.TruncateLine(line, 120))
			continue
		}

		// In ultra-compact mode, be very selective
		if shared.UltraCompact {
			// Keep lines that look like target completion or errors
			if strings.HasPrefix(trimmed, "make[") || strings.Contains(trimmed, "Error") {
				result = append(result, shared.TruncateLine(line, 80))
			}
			continue
		}

		// Keep compilation commands if verbose
		if shared.Verbose > 0 && (strings.Contains(line, "gcc") || strings.Contains(line, "g++") ||
			strings.Contains(line, "cc") || strings.Contains(line, "ld")) {
			result = append(result, shared.TruncateLine(line, 100))
			continue
		}

		// Keep progress messages
		if !lastWasEnter {
			result = append(result, shared.TruncateLine(line, 120))
		}
		lastWasEnter = false
	}

	return strings.Join(result, "\n")
}

func runMakeHelp() error {
	timer := tracking.Start()

	execCmd := exec.Command("make", "-p", "-q", "DEFAULT")
	output, _ := execCmd.CombinedOutput()
	raw := string(output)

	// Extract targets from make database dump
	filtered := extractMakeTargets(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("make help", "tokman make help", originalTokens, filteredTokens)

	return nil
}

func extractMakeTargets(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Look for target definitions
		if strings.Contains(line, ":") && !strings.HasPrefix(trimmed, "#") {
			// Check if it's a valid target (not a variable)
			if !strings.Contains(trimmed, "=") {
				target := strings.Split(trimmed, ":")[0]
				target = strings.TrimSpace(target)

				// Skip pattern rules and special targets
				if !strings.Contains(target, "%") &&
					!strings.HasPrefix(target, ".") &&
					target != "" &&
					!strings.Contains(target, "(") {
					result = append(result, target)
				}
			}
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	unique := make([]string, 0)
	for _, t := range result {
		if !seen[t] {
			seen[t] = true
			unique = append(unique, t)
		}
	}

	if len(unique) == 0 {
		return "No targets found. Run 'make' directly."
	}

	return "Available targets:\n" + strings.Join(unique, "\n")
}
