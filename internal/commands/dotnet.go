package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var dotnetCmd = &cobra.Command{
	Use:   "dotnet [command]",
	Short: ".NET commands with compact output",
	Long: `.NET commands with compact output (build/test/restore/format).
Filters verbose MSBuild output while preserving errors and warnings.`,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
}

var dotnetBuildCmd = &cobra.Command{
	Use:                "build [args...]",
	Short:              "Build with compact output",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	Run: func(cmd *cobra.Command, args []string) {
		runDotnetCommand("build", args)
	},
}

var dotnetTestCmd = &cobra.Command{
	Use:                "test [args...]",
	Short:              "Test with compact output",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	Run: func(cmd *cobra.Command, args []string) {
		runDotnetCommand("test", args)
	},
}

var dotnetRestoreCmd = &cobra.Command{
	Use:                "restore [args...]",
	Short:              "Restore with compact output",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	Run: func(cmd *cobra.Command, args []string) {
		runDotnetCommand("restore", args)
	},
}

var dotnetFormatCmd = &cobra.Command{
	Use:                "format [args...]",
	Short:              "Format with compact output",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	Run: func(cmd *cobra.Command, args []string) {
		runDotnetCommand("format", args)
	},
}

func init() {
	rootCmd.AddCommand(dotnetCmd)
	dotnetCmd.AddCommand(dotnetBuildCmd)
	dotnetCmd.AddCommand(dotnetTestCmd)
	dotnetCmd.AddCommand(dotnetRestoreCmd)
	dotnetCmd.AddCommand(dotnetFormatCmd)
}

func runDotnetCommand(subCmd string, args []string) {
	startTime := time.Now()

	dotnetArgs := append([]string{subCmd}, args...)
	c := exec.Command("dotnet", dotnetArgs...)

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	// Filter output
	filtered := filterDotnetOutput(output)

	fmt.Print(filtered)

	execTime := time.Since(startTime).Milliseconds()
	if err := recordCommand("dotnet "+subCmd, output, filtered, execTime, err == nil); err != nil && verbose > 0 {
		fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", err)
	}

	if err != nil {
		os.Exit(1)
	}
}

func filterDotnetOutput(output string) string {
	lines := strings.Split(output, "\n")
	var result []string
	var errors, warnings int

	for _, line := range lines {
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Skip verbose build lines
		if strings.Contains(line, "Microsoft ") ||
			strings.Contains(line, "  Determining projects to restore") ||
			strings.Contains(line, "  Restored ") ||
			strings.Contains(line, "  dotnet ") ||
			strings.HasPrefix(line, "  ") && !strings.Contains(line, "error") && !strings.Contains(line, "warning") {
			continue
		}

		// Count errors and warnings
		if strings.Contains(strings.ToLower(line), "error") {
			errors++
		}
		if strings.Contains(strings.ToLower(line), "warning") {
			warnings++
		}

		// Truncate long paths
		if len(line) > 100 {
			line = line[:97] + "..."
		}

		result = append(result, line)
	}

	// Add summary if we have results
	if len(result) > 0 {
		summary := fmt.Sprintf("\n---\n%d errors, %d warnings", errors, warnings)
		result = append(result, summary)
	}

	return strings.Join(result, "\n")
}
