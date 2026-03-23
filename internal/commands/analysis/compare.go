package analysis

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var compareCmd = &cobra.Command{
	Use:   "compare [command...]",
	Short: "Compare compression across all presets",
	Long: `Run a command and compare compression results across
fast, balanced, and full presets side by side.`,
	RunE: runCompare,
}

func init() {
	registry.Add(func() { registry.Register(compareCmd) })
}

func runCompare(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: tokman compare <command> [args...]")
	}

	// Execute command
	exePath, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", args[0])
	}

	execCmd := exec.Command(exePath, args[1:]...)
	execCmd.Env = os.Environ()
	output, err := execCmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		return fmt.Errorf("command failed: %s (%w)", args[0], err)
	}
	rawOutput := string(output)

	originalTokens := core.EstimateTokens(rawOutput)

	fmt.Printf("Command: %s\n", strings.Join(args, " "))
	fmt.Printf("Original: ~%d tokens (%d chars)\n\n", originalTokens, len(rawOutput))

	fmt.Printf("%-12s %8s %8s %7s %8s\n", "Preset", "Tokens", "Saved", "Pct", "Quality")
	fmt.Printf("%-12s %8s %8s %7s %8s\n", "────────────", "────────", "────────", "───────", "────────")

	equiv := filter.NewSemanticEquivalence()

	for _, preset := range []filter.PipelinePreset{
		filter.PresetFast,
		filter.PresetBalanced,
		filter.PresetFull,
	} {
		cfg := filter.PresetConfig(preset, filter.ModeMinimal)
		pipeline := filter.NewPipelineCoordinator(cfg)

		compressed, stats := pipeline.Process(rawOutput)
		report := equiv.Check(rawOutput, compressed)

		fmt.Printf("%-12s %8d %8d %6.1f%% %7.0f%%\n",
			preset,
			stats.FinalTokens,
			stats.TotalSaved,
			stats.ReductionPercent,
			report.Score*100,
		)
	}

	return nil
}
