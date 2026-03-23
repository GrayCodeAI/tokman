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

var cargoCmd = &cobra.Command{
	Use:   "cargo [subcommand] [args...]",
	Short: "Cargo commands with compact output",
	Long: `Cargo commands with token-optimized output.

Subcommands:
  build   - Build with compact output (strip Compiling lines, keep errors)
  test    - Test with failures-only output
  nextest - Nextest test runner with failures-only output
  check   - Check with compact output
  clippy  - Clippy with warnings grouped by lint rule

Examples:
  tokman cargo build
  tokman cargo test --lib
  tokman cargo clippy -- -W clippy::all`,
	RunE: runCargo,
}

func init() {
	registry.Add(func() { registry.Register(cargoCmd) })
}

func runCargo(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		args = []string{"--help"}
	}

	subcommand := args[0]
	cargoArgs := append([]string{}, args...)

	c := exec.Command("cargo", cargoArgs...)
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	var filtered string
	switch subcommand {
	case "build", "check":
		if shared.UltraCompact {
			filtered = filterCargoBuildUltraCompact(output)
		} else {
			filtered = filterCargoBuild(output)
		}
	case "test":
		if shared.UltraCompact {
			filtered = filterCargoTestUltraCompact(output)
		} else {
			filtered = filterCargoTest(output)
		}
	case "nextest":
		filtered = filterCargoNextest(output)
	case "clippy":
		filtered = filterCargoClippy(output)
	default:
		filtered = output
	}

	if err != nil {
		if hint := shared.TeeOnFailure(output, "cargo_"+subcommand, err); hint != "" {
			filtered += "\n" + hint
		}
	}

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("cargo %s", strings.Join(args, " ")), "tokman cargo", originalTokens, filteredTokens)

	if shared.Verbose > 0 {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", originalTokens-filteredTokens)
	}

	return err
}

func filterCargoBuild(output string) string {
	var result strings.Builder
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "Compiling") {
			continue
		}
		if strings.HasPrefix(line, "Finished") {
			result.WriteString("✓ build complete\n")
			continue
		}
		if strings.Contains(line, "error") || strings.Contains(line, "warning") {
			result.WriteString(line + "\n")
		}
	}
	return result.String()
}

func filterCargoTest(output string) string {
	var result strings.Builder
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "test result:") {
			result.WriteString(line + "\n")
			continue
		}
		if strings.Contains(line, "FAILED") || strings.Contains(line, "----") {
			result.WriteString(line + "\n")
			continue
		}
		if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			result.WriteString(line + "\n")
		}
	}
	return result.String()
}

func filterCargoNextest(output string) string {
	var result strings.Builder
	var failures []string
	var summary string

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "test result:") || strings.Contains(line, "passed") && strings.Contains(line, "failed") {
			summary = line
			continue
		}
		if strings.HasPrefix(line, "FAIL") || strings.Contains(line, "[FAIL]") {
			failures = append(failures, line)
		}
		if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			failures = append(failures, line)
		}
	}

	if len(failures) > 0 {
		result.WriteString(fmt.Sprintf("Failures (%d):\n", len(failures)))
		for _, f := range failures {
			result.WriteString("  " + shared.TruncateLine(f, 80) + "\n")
		}
	}

	if summary != "" {
		result.WriteString(summary + "\n")
	} else if result.Len() == 0 {
		result.WriteString("✓ all tests passed\n")
	}

	return result.String()
}

func filterCargoClippy(output string) string {
	warnings := make(map[string][]string)
	var errors []string

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "warning:") {
			parts := strings.SplitN(line, ":", 4)
			if len(parts) >= 4 {
				warnType := strings.TrimSpace(parts[3])
				warnings[warnType] = append(warnings[warnType], line)
			}
		} else if strings.Contains(line, "error:") {
			errors = append(errors, line)
		}
	}

	var result strings.Builder
	if len(warnings) > 0 {
		result.WriteString(fmt.Sprintf("Warnings (%d types):\n", len(warnings)))
		for wtype, lines := range warnings {
			result.WriteString(fmt.Sprintf("  %s: %d occurrences\n", wtype, len(lines)))
		}
	}
	for _, e := range errors {
		result.WriteString(e + "\n")
	}

	return result.String()
}

func filterCargoBuildUltraCompact(output string) string {
	errors := 0
	warnings := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "error") {
			errors++
		}
		if strings.Contains(line, "warning") {
			warnings++
		}
	}
	if errors > 0 {
		return fmt.Sprintf("build failed: %d errors, %d warnings", errors, warnings)
	}
	if warnings > 0 {
		return fmt.Sprintf("build ok: %d warnings", warnings)
	}
	return "build ok"
}

func filterCargoTestUltraCompact(output string) string {
	passed := 0
	failed := 0
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "test result:") {
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
			}
		}
	}
	if failed > 0 {
		return fmt.Sprintf("tests: %d passed, %d failed", passed, failed)
	}
	return fmt.Sprintf("tests: %d passed", passed)
}
