package system

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var watchCmd = &cobra.Command{
	Use:   "watch [command...]",
	Short: "Live monitoring of token savings",
	Long:  `Run a command and display real-time compression stats as layers process.`,
	RunE:  runWatch,
}

func init() {
	registry.Add(func() { registry.Register(watchCmd) })
}

func runWatch(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: tokman watch <command> [args...]")
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
	fmt.Printf("Original: ~%d tokens\n\n", originalTokens)
	fmt.Println("Processing layers...")
	fmt.Println()

	// Show layer-by-layer processing
	pipeline := filter.NewPipelineCoordinator(filter.PipelineConfig{
		Mode:                filter.ModeMinimal,
		QueryIntent:         shared.GetQueryIntent(),
		Budget:              shared.GetTokenBudget(),
		SessionTracking:     true,
		NgramEnabled:        true,
		EnableCompaction:    true,
		EnableAttribution:   true,
		EnableH2O:           true,
		EnableAttentionSink: true,
	})

	_, stats := pipeline.Process(rawOutput)

	// Display results with animation
	layers := []string{
		"1_entropy", "2_perplexity", "3_goal_driven", "4_ast_preserve",
		"5_contrastive", "6_ngram", "7_evaluator", "8_gist", "9_hierarchical",
		"11_compaction", "12_attribution", "13_h2o", "10_budget",
	}

	runningTotal := originalTokens
	for _, layer := range layers {
		stat, ok := stats.LayerStats[layer]
		if !ok {
			continue
		}

		time.Sleep(50 * time.Millisecond) // Visual effect
		runningTotal -= stat.TokensSaved
		bar := progressBarVis(float64(runningTotal), float64(originalTokens))
		fmt.Printf("  %-20s %s %d tokens\n", layer, bar, stat.TokensSaved)
	}

	fmt.Printf("\nFinal: ~%d tokens (%.1f%% reduction)\n", stats.FinalTokens, stats.ReductionPercent)
	return nil
}

func progressBarVis(current, total float64) string {
	width := 20
	if total == 0 {
		return "[" + strings.Repeat(" ", width) + "]"
	}
	filled := int(current / total * float64(width))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}
