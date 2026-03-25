package output

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/tee"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var errCmd = &cobra.Command{
	Use:   "err <command>",
	Short: "Run command and show only errors/warnings",
	Long: `Execute a command and filter output to show only errors and warnings.

Useful for running build commands, linters, or tests where you only want
to see failures and warnings, not successful output.

Supports language-specific error patterns for:
- Rust (error[E####], --> file:line:col)
- Python (Traceback, File "path", line N)
- JavaScript/TypeScript (at path:line:col)
- Go (file.go:line: message)

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
		exitCode := runErr(args, verbose)
		os.Exit(exitCode)
	},
}

func init() {
	registry.Add(func() { registry.Register(errCmd) })
}

func runErr(args []string, verbose bool) int {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command(args[0], args[1:]...)

	stdout, err := execCmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	stderr, err := execCmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	if err := execCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	var stdoutBuf, stderrBuf strings.Builder
	doneOut := make(chan struct{})
	doneErr := make(chan struct{})

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			stdoutBuf.WriteString(scanner.Text() + "\n")
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "stdout read error: %v\n", err)
		}
		close(doneOut)
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			stderrBuf.WriteString(scanner.Text() + "\n")
		}
		if err := scanner.Err(); err != nil {
			fmt.Fprintf(os.Stderr, "stderr read error: %v\n", err)
		}
		close(doneErr)
	}()

	<-doneOut
	<-doneErr

	execCmd.Wait()

	exitCode := 0
	if execCmd.ProcessState != nil {
		exitCode = execCmd.ProcessState.ExitCode()
	}

	raw := stdoutBuf.String() + stderrBuf.String()
	filtered := filterErrorsAdvanced(raw, verbose)

	var result strings.Builder

	if filtered == "" {
		if exitCode == 0 {
			result.WriteString("✅ Command completed successfully (no errors)\n")
		} else {
			result.WriteString(fmt.Sprintf("❌ Command failed (exit code: %d)\n", exitCode))
			lines := strings.Split(raw, "\n")
			start := len(lines) - 10
			if start < 0 {
				start = 0
			}
			for i := start; i < len(lines); i++ {
				if lines[i] != "" {
					result.WriteString(fmt.Sprintf("  %s\n", lines[i]))
				}
			}
		}
	} else {
		result.WriteString(filtered)
	}

	if hint := tee.WriteAndHint(raw, "err", exitCode); hint != "" {
		result.WriteString(hint + "\n")
	}

	fmt.Print(result.String())

	timer.Track(strings.Join(args, " "), "tokman err", tracking.EstimateTokens(raw), tracking.EstimateTokens(filtered))

	return exitCode
}

var errorPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^.*error[\s:\[].*$`),
	regexp.MustCompile(`(?i)^.*\berr\b.*$`),
	regexp.MustCompile(`(?i)^.*warning[\s:\[].*$`),
	regexp.MustCompile(`(?i)^.*\bwarn\b.*$`),
	regexp.MustCompile(`(?i)^.*failed.*$`),
	regexp.MustCompile(`(?i)^.*failure.*$`),
	regexp.MustCompile(`(?i)^.*exception.*$`),
	regexp.MustCompile(`(?i)^.*panic.*$`),

	regexp.MustCompile(`^error\[E\d+\]:.*$`),
	regexp.MustCompile(`^\s*--> .*:\d+:\d+$`),

	regexp.MustCompile(`^Traceback.*$`),
	regexp.MustCompile(`^\s*File ".*", line \d+.*$`),

	regexp.MustCompile(`^\s*at .*:\d+:\d+.*$`),

	regexp.MustCompile(`^.*\.go:\d+:.*$`),

	regexp.MustCompile(`^\s*at .+\(.+\.java:\d+\)$`),

	regexp.MustCompile(`^.*:\d+:\d+: error:.*$`),
	regexp.MustCompile(`^.*:\d+: error:.*$`),

	regexp.MustCompile(`^.*/[^:]+:\s+line\s+\d+:\s+.*$`),

	regexp.MustCompile(`(?i)^.*\b(cannot|unable|denied|forbidden)\b.*$`),
}

func filterErrorsAdvanced(output string, verbose bool) string {
	var result []string
	inErrorBlock := false
	blankCount := 0

	for _, line := range strings.Split(output, "\n") {
		isErrorLine := false

		for _, pattern := range errorPatterns {
			if pattern.MatchString(line) {
				isErrorLine = true
				break
			}
		}

		if isErrorLine {
			inErrorBlock = true
			blankCount = 0
			result = append(result, line)
		} else if inErrorBlock {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				blankCount++
				if blankCount >= 2 {
					inErrorBlock = false
				} else {
					result = append(result, line)
				}
			} else if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
				result = append(result, line)
				blankCount = 0
			} else {
				inErrorBlock = false
			}
		}
	}

	return strings.Join(result, "\n")
}
