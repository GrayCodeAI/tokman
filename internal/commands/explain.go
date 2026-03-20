package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var explainCmd = &cobra.Command{
	Use:   "explain [command...]",
	Short: "Explain which compression layers removed what",
	Long: `Execute a command and show per-layer compression statistics
to understand what each pipeline stage does.

Example:
  tokman explain git log --oneline -50`,
	RunE: runExplain,
}

func init() {
	rootCmd.AddCommand(explainCmd)
}

func runExplain(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: tokman explain <command> [args...]")
	}

	// Execute the command
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

	// Run pipeline with all layers to get per-layer stats
	pipeline := filter.NewPipelineCoordinator(filter.PipelineConfig{
		Mode:                filter.ModeMinimal,
		QueryIntent:         GetQueryIntent(),
		Budget:              GetTokenBudget(),
		LLMEnabled:          IsLLMEnabled(),
		SessionTracking:     true,
		NgramEnabled:        true,
		EnableCompaction:    true,
		EnableAttribution:   true,
		EnableH2O:           true,
		EnableAttentionSink: true,
	})

	_, stats := pipeline.Process(rawOutput)

	fmt.Printf("%-25s %s\n", "Layer", "Tokens Saved")
	fmt.Printf("%-25s %s\n", "─────────────────────────", "────────────")

	layerOrder := []string{
		"1_entropy", "2_perplexity", "3_goal_driven", "4_ast_preserve",
		"5_contrastive", "6_ngram", "7_evaluator", "8_gist", "9_hierarchical",
		"neural", "11_compaction", "12_attribution", "13_h2o",
		"10_session", "10_budget",
	}

	layerDescriptions := map[string]string{
		"1_entropy":      "Entropy Filtering (low-info removal)",
		"2_perplexity":   "Perplexity Pruning (surprise scoring)",
		"3_goal_driven":  "Goal-Driven Selection (relevance)",
		"4_ast_preserve": "AST Preservation (syntax-aware)",
		"5_contrastive":  "Contrastive Ranking (question-relevance)",
		"6_ngram":        "N-gram Abbreviation (pattern compression)",
		"7_evaluator":    "Evaluator Heads (attention simulation)",
		"8_gist":         "Gist Compression (virtual tokens)",
		"9_hierarchical": "Hierarchical Summary (recursive)",
		"neural":         "LLM-based Compression",
		"11_compaction":  "Compaction (MemGPT-style)",
		"12_attribution": "Attribution Filter (ProCut-style)",
		"13_h2o":         "H2O Heavy-Hitter Oracle",
		"10_session":     "Session Deduplication",
		"10_budget":      "Budget Enforcement (final cap)",
	}

	totalLayerSaved := 0
	for _, layer := range layerOrder {
		if stat, ok := stats.LayerStats[layer]; ok && stat.TokensSaved > 0 {
			desc := layerDescriptions[layer]
			if desc == "" {
				desc = layer
			}
			fmt.Printf("%-25s %d tokens\n", desc, stat.TokensSaved)
			totalLayerSaved += stat.TokensSaved
		}
	}

	fmt.Printf("\nTotal saved: %d tokens (%.1f%% reduction)\n",
		stats.TotalSaved, stats.ReductionPercent)

	return nil
}
