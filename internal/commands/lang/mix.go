package lang

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

var mixCmd = &cobra.Command{
	Use:   "mix [task] [args...]",
	Short: "Elixir Mix build commands with compact output",
	Long: `Execute Elixir Mix commands with token-optimized output.

Specialized filters for:
  - compile: Compact compilation output
  - test: Compact test results
  - deps: Compact dependency listing

Examples:
  tokman mix compile
  tokman mix test
  tokman mix deps.get`,
	DisableFlagParsing: true,
	RunE:               runMix,
}

func init() {
	registry.Add(func() { registry.Register(mixCmd) })
}

func runMix(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"help"}
	}

	switch args[0] {
	case "compile":
		return runMixCompile(args[1:])
	case "test":
		return runMixTest(args[1:])
	case "deps":
		return runMixDeps(args[1:])
	case "format", "fmt":
		return runMixFormat(args[1:])
	default:
		return runMixPassthrough(args)
	}
}

func runMixCompile(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: mix compile %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("mix", append([]string{"compile"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterMixCompileOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "mix_compile", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("mix compile", "tokman mix compile", originalTokens, filteredTokens)

	return err
}

func filterMixCompileOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Keep warnings
		if strings.Contains(trimmed, "warning:") {
			if !shared.UltraCompact {
				result = append(result, shared.TruncateLine(line, 120))
			}
			continue
		}

		// Keep errors
		if strings.Contains(trimmed, "error:") || strings.Contains(trimmed, "** (") {
			result = append(result, line)
			continue
		}

		// Skip verbose compilation output in ultra-compact mode
		if shared.UltraCompact {
			continue
		}

		// Keep compiled messages
		if strings.Contains(trimmed, "Compiled") {
			result = append(result, shared.TruncateLine(line, 100))
		}
	}

	return strings.Join(result, "\n")
}

func runMixTest(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: mix test %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("mix", append([]string{"test"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterMixTestOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "mix_test", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("mix test", "tokman mix test", originalTokens, filteredTokens)

	return err
}

func filterMixTestOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Keep test summary
		if strings.Contains(trimmed, "test(s),") && strings.Contains(trimmed, "failure(s)") {
			result = append(result, line)
			continue
		}

		// Keep failures
		if strings.Contains(trimmed, "1) test") || strings.Contains(trimmed, "Failure:") {
			result = append(result, line)
			continue
		}

		// Keep errors
		if strings.Contains(trimmed, "** (") {
			result = append(result, line)
			continue
		}

		// Skip dots progress in ultra-compact mode
		if shared.UltraCompact {
			continue
		}

		// Keep test names
		if strings.HasPrefix(trimmed, ".") || strings.Contains(trimmed, "test ") {
			result = append(result, shared.TruncateLine(line, 100))
		}
	}

	return strings.Join(result, "\n")
}

func runMixDeps(args []string) error {
	timer := tracking.Start()

	fullArgs := append([]string{"deps"}, args...)
	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: mix %s\n", strings.Join(fullArgs, " "))
	}

	execCmd := exec.Command("mix", fullArgs...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterMixDepsOutput(raw)

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("mix deps", "tokman mix deps", originalTokens, filteredTokens)

	return err
}

func filterMixDepsOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Skip getting/fetching messages in ultra-compact mode
		if shared.UltraCompact && (strings.Contains(trimmed, "Getting") || strings.Contains(trimmed, "Fetching")) {
			continue
		}

		// Keep dependency status lines
		if strings.Contains(trimmed, "*") || strings.Contains(trimmed, "=>") {
			if shared.UltraCompact {
				// Just show package name
				parts := strings.Fields(trimmed)
				if len(parts) >= 2 {
					result = append(result, parts[1])
				}
			} else {
				result = append(result, shared.TruncateLine(line, 100))
			}
		}
	}

	return strings.Join(result, "\n")
}

func runMixFormat(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: mix format %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("mix", append([]string{"format"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	// Format typically produces no output on success
	if raw == "" {
		fmt.Println("✅ Formatted successfully")
	} else {
		fmt.Println(raw)
	}

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(raw)
	timer.Track("mix format", "tokman mix format", originalTokens, filteredTokens)

	return err
}

func runMixPassthrough(args []string) error {
	timer := tracking.Start()

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: mix %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("mix", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterMixBasicOutput(raw)

	if err != nil {
		if hint := shared.TeeOnFailure(raw, "mix", err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track("mix", "tokman mix", originalTokens, filteredTokens)

	return err
}

func filterMixBasicOutput(raw string) string {
	if shared.UltraCompact {
		lines := strings.Split(raw, "\n")
		var result []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "error:") || strings.Contains(trimmed, "** (") ||
				strings.Contains(trimmed, "warning:") {
				result = append(result, shared.TruncateLine(line, 100))
			}
		}
		return strings.Join(result, "\n")
	}
	return raw
}
