package pkgmgr

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var pipCmd = &cobra.Command{
	Use:   "pip [args...]",
	Short: "Pip package manager with compact output",
	Long: `Pip package manager with token-optimized output.

Auto-detects uv if available for faster operations.

Examples:
  tokman pip list
  tokman pip install package
  tokman pip outdated`,
	RunE: runPip,
}

func init() {
	registry.Add(func() { registry.Register(pipCmd) })
}

func runPip(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	var c *exec.Cmd
	if _, err := exec.LookPath("uv"); err == nil {
		uvArgs := append([]string{"pip"}, args...)
		c = exec.Command("uv", uvArgs...)
	} else {
		c = exec.Command("pip", args...)
	}

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	filtered := filterPipOutput(output)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("pip %s", strings.Join(args, " ")), "tokman pip", originalTokens, filteredTokens)

	return err
}

func filterPipOutput(output string) string {
	var result strings.Builder
	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, "Collecting") && strings.Contains(trimmed, "Downloading") {
			continue
		}
		if strings.HasPrefix(trimmed, "Requirement already satisfied") {
			continue
		}
		if trimmed == "" {
			continue
		}

		result.WriteString(line + "\n")
	}
	return result.String()
}
