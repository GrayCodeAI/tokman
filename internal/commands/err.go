package commands

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var errCmd = &cobra.Command{
	Use:   "err <command>",
	Short: "Run command and show only errors/warnings",
	Long: `Execute a command and filter output to show only errors and warnings.

Useful for running build commands, linters, or tests where you only want
to see failures and warnings, not successful output.

Examples:
  tokman err npm run build
  tokman err cargo build
  tokman err go test ./...`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Error: err requires a command to run")
			os.Exit(1)
		}

		verbose, _ := cmd.Flags().GetBool("verbose")
		if err := runErr(args, verbose); err != nil {
			// Exit with the command's exit code if available
			if exitErr, ok := err.(*exec.ExitError); ok {
				os.Exit(exitErr.ExitCode())
			}
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(errCmd)
}

func runErr(args []string, verbose bool) error {
	// Execute the command
	execCmd := exec.Command(args[0], args[1:]...)
	
	// Capture both stdout and stderr
	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	stderr, err := execCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	if err := execCmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Filter patterns for errors and warnings
	errorPatterns := []string{
		"error", "Error", "ERROR",
		"warning", "Warning", "WARNING",
		"failed", "Failed", "FAILED",
		"fail:", "FAIL:",
		"panic:", "PANIC:",
		"fatal", "Fatal", "FATAL",
		"critical", "Critical", "CRITICAL",
		"exception", "Exception", "EXCEPTION",
		"✗", "✖", "❌",
		"✘", "×",
	}

	// Process stdout
	go filterErrOutput(stdout, errorPatterns, verbose)
	// Process stderr (always show)
	go filterErrOutput(stderr, errorPatterns, true) // Always show stderr

	return execCmd.Wait()
}

func filterErrOutput(reader io.Reader, patterns []string, showAll bool) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		
		// Check if line matches any error pattern
		matches := false
		for _, pattern := range patterns {
			if strings.Contains(line, pattern) {
				matches = true
				break
			}
		}

		// Print if matches or if showAll is true
		if matches || showAll {
			fmt.Println(line)
		}
	}
}
