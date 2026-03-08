package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy <command> [args...]",
	Short: "Execute command without filtering but track usage",
	Long: `Execute a command without applying any output filtering.

Unlike other TokMan commands that filter output to reduce tokens,
proxy runs the command as-is while still tracking execution metrics.
Useful for commands where you need full unfiltered output.`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if verbose > 0 {
			fmt.Fprintf(os.Stderr, "Proxy mode: %s\n", strings.Join(args, " "))
		}

		runProxy(args)
	},
}

func init() {
	rootCmd.AddCommand(proxyCmd)
}

func runProxy(args []string) {
	// Start timing
	timer := tracking.Start()

	// Create command
	execCmd := exec.Command(args[0], args[1:]...)
	execCmd.Stdin = os.Stdin

	// Capture stdout and stderr
	stdoutPipe, err := execCmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating stdout pipe: %v\n", err)
		os.Exit(1)
	}

	stderrPipe, err := execCmd.StderrPipe()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating stderr pipe: %v\n", err)
		os.Exit(1)
	}

	// Start the command
	if err := execCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting command: %v\n", err)
		os.Exit(1)
	}

	// Stream stdout and stderr in parallel, capturing for tracking
	outputChan := make(chan []byte)
	go func() {
		var output []byte
		buf := make([]byte, 8192)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				output = append(output, buf[:n]...)
				os.Stdout.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		outputChan <- output
	}()

	errChan := make(chan []byte)
	go func() {
		var errOutput []byte
		buf := make([]byte, 8192)
		for {
			n, err := stderrPipe.Read(buf)
			if n > 0 {
				errOutput = append(errOutput, buf[:n]...)
				os.Stderr.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		errChan <- errOutput
	}()

	// Wait for streams to finish
	stdout := <-outputChan
	stderr := <-errChan

	// Wait for command to complete
	err = execCmd.Wait()

	// Track usage (input = output since no filtering)
	fullOutput := string(stdout) + string(stderr)
	cmdStr := strings.Join(args, " ")
	originalTokens := tracking.EstimateTokens(fullOutput)
	timer.Track(cmdStr, fmt.Sprintf("tokman proxy %s", cmdStr), originalTokens, originalTokens)

	// Exit with same code as child process
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		os.Exit(1)
	}
}
