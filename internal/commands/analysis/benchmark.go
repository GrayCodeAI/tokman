package analysis

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark [command...]",
	Short: "Benchmark token compression for a command",
	Long: `Run a command through the compression pipeline and compare
savings across fast/balanced/full presets.

Example:
  tokman benchmark git status
  tokman benchmark docker ps`,
	RunE: runBenchmark,
}

func init() {
	registry.Add(func() { registry.Register(benchmarkCmd) })
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: tokman benchmark <command> [args...]")
	}

	// Execute the command
	commandStr := strings.Join(args, " ")
	fmt.Printf("Running: %s\n\n", commandStr)

	exePath, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", args[0])
	}

	execCmd := exec.Command(exePath, args[1:]...)
	execCmd.Env = os.Environ()
	output, _ := execCmd.CombinedOutput()
	rawOutput := string(output)

	originalTokens := core.EstimateTokens(rawOutput)
	fmt.Printf("Original output: %d chars, ~%d tokens\n\n", len(rawOutput), originalTokens)

	// Benchmark each preset
	presets := []filter.PipelinePreset{
		filter.PresetFast,
		filter.PresetBalanced,
		filter.PresetFull,
	}

	fmt.Printf("%-12s %8s %8s %8s %10s\n", "Preset", "Tokens", "Saved", "Pct", "Duration")
	fmt.Printf("%-12s %8s %8s %8s %10s\n", "------", "------", "-----", "---", "--------")

	for _, preset := range presets {
		cfg := filter.PresetConfig(preset, filter.ModeMinimal)
		pipeline := filter.NewPipelineCoordinator(cfg)

		start := time.Now()
		result, stats := pipeline.Process(rawOutput)
		duration := time.Since(start)

		finalTokens := core.EstimateTokens(result)
		saved := originalTokens - finalTokens
		pct := float64(0)
		if originalTokens > 0 {
			pct = float64(saved) / float64(originalTokens) * 100
		}

		_ = stats // use stats for layer breakdown in verbose mode
		fmt.Printf("%-12s %8d %8d %7.1f%% %10s\n",
			preset, finalTokens, saved, pct, duration.Round(time.Microsecond))
	}

	return nil
}
