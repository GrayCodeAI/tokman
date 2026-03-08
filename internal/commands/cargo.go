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

var cargoCmd = &cobra.Command{
	Use:   "cargo [subcommand] [args...]",
	Short: "Cargo commands with compact output",
	Long: `Cargo commands with token-optimized output.

Subcommands:
  build  - Build with compact output (strip Compiling lines, keep errors)
  test   - Test with failures-only output
  check  - Check with compact output
  clippy - Clippy with warnings grouped by lint rule

Examples:
  tokman cargo build
  tokman cargo test --lib
  tokman cargo clippy -- -W clippy::all`,
	RunE: runCargo,
}

func init() {
	rootCmd.AddCommand(cargoCmd)
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

	// Apply filtering based on subcommand
	var filtered string
	switch subcommand {
	case "build", "check":
		filtered = filterCargoBuild(output)
	case "test":
		filtered = filterCargoTest(output)
	case "clippy":
		filtered = filterCargoClippy(output)
	default:
		filtered = output
	}

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("cargo %s", strings.Join(args, " ")), "tokman cargo", originalTokens, filteredTokens)

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", originalTokens-filteredTokens)
	}

	return err
}

func filterCargoBuild(output string) string {
	var result strings.Builder
	for _, line := range strings.Split(output, "\n") {
		// Skip "Compiling" lines
		if strings.HasPrefix(line, "Compiling") {
			continue
		}
		// Skip "Finished" line
		if strings.HasPrefix(line, "Finished") {
			result.WriteString("✓ build complete\n")
			continue
		}
		// Keep errors and warnings
		if strings.Contains(line, "error") || strings.Contains(line, "warning") {
			result.WriteString(line + "\n")
		}
	}
	return result.String()
}

func filterCargoTest(output string) string {
	var result strings.Builder
	for _, line := range strings.Split(output, "\n") {
		// Keep test result summary
		if strings.HasPrefix(line, "test result:") {
			result.WriteString(line + "\n")
			continue
		}
		// Keep failures
		if strings.Contains(line, "FAILED") || strings.Contains(line, "----") {
			result.WriteString(line + "\n")
			continue
		}
		// Keep error lines
		if strings.Contains(line, "error") || strings.Contains(line, "Error") {
			result.WriteString(line + "\n")
		}
	}
	return result.String()
}

func filterCargoClippy(output string) string {
	// Group warnings by type
	warnings := make(map[string][]string)
	var errors []string

	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, "warning:") {
			// Extract warning type
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
