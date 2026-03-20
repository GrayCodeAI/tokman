package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var pipelineCmd = &cobra.Command{
	Use:   "pipeline [command...]",
	Short: "Show detailed pipeline execution",
	Long:  `Run a command through the compression pipeline and show each layer's effect.`,
	RunE:  runPipeline,
}

func init() {
	rootCmd.AddCommand(pipelineCmd)
}

func runPipeline(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: tokman pipeline <command> [args...]")
	}

	// Execute command
	exePath, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", args[0])
	}

	execCmd := exec.Command(exePath, args[1:]...)
	execCmd.Env = os.Environ()
	output, _ := execCmd.CombinedOutput()
	rawOutput := string(output)

	originalTokens := core.EstimateTokens(rawOutput)

	fmt.Printf("Pipeline Execution: %s\n", args[0])
	fmt.Printf("Original: ~%d tokens\n\n", originalTokens)

	pipeline := filter.NewPipelineCoordinator(filter.PipelineConfig{
		Mode:                filter.ModeMinimal,
		QueryIntent:         GetQueryIntent(),
		Budget:              GetTokenBudget(),
		SessionTracking:     false,
		NgramEnabled:        true,
		EnableCompaction:    true,
		EnableAttribution:   true,
		EnableH2O:           true,
		EnableAttentionSink: true,
	})

	_, stats := pipeline.Process(rawOutput)

	fmt.Printf("%-30s %10s %8s\n", "Layer", "Saved", "Status")
	fmt.Printf("%-30s %10s %8s\n", "──────────────────────────────", "──────────", "────────")

	runningTotal := originalTokens
	layers := []string{
		"1_entropy", "2_perplexity", "3_goal_driven", "4_ast_preserve",
		"5_contrastive", "6_ngram", "7_evaluator", "8_gist", "9_hierarchical",
		"11_compaction", "12_attribution", "13_h2o", "10_budget",
	}

	for _, layer := range layers {
		stat, ok := stats.LayerStats[layer]
		if !ok {
			continue
		}
		status := "applied"
		runningTotal -= stat.TokensSaved
		fmt.Printf("%-30s %10d %8s\n", layer, stat.TokensSaved, status)
	}

	fmt.Printf("\nFinal: ~%d tokens (%.1f%% reduction)\n", stats.FinalTokens, stats.ReductionPercent)

	return nil
}
