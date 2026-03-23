package web

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var proxyCmd = &cobra.Command{
	Use:   "proxy -- <command> [args...]",
	Short: "Execute command without filtering but track usage",
	Long: `Execute a command without applying any output filtering.

Unlike other TokMan commands that filter output to reduce tokens,
proxy runs the command as-is while still tracking execution metrics.
Useful for commands where you need full unfiltered output.`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		if shared.Verbose > 0 {
			fmt.Fprintf(os.Stderr, "Proxy mode: %s\n", strings.Join(args, " "))
		}

		runProxy(args)
	},
}

func init() {
	registry.Add(func() { registry.Register(proxyCmd) })
}

func runProxy(args []string) {
	timer := tracking.Start()

	execCmd := exec.Command(args[0], args[1:]...)
	execCmd.Stdin = os.Stdin

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

	if err := execCmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting command: %v\n", err)
		os.Exit(1)
	}

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

	stdout := <-outputChan
	stderr := <-errChan

	err = execCmd.Wait()

	fullOutput := string(stdout) + string(stderr)
	cmdStr := strings.Join(args, " ")
	originalTokens := tracking.EstimateTokens(fullOutput)
	timer.Track(cmdStr, fmt.Sprintf("tokman proxy %s", cmdStr), originalTokens, originalTokens)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
		os.Exit(1)
	}
}
