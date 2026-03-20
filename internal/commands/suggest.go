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

var suggestCmd = &cobra.Command{
	Use:   "suggest [command...]",
	Short: "Suggest optimal compression settings",
	Long:  `Analyze command output and suggest the best compression strategy.`,
	RunE:  runSuggest,
}

func init() {
	rootCmd.AddCommand(suggestCmd)
}

func runSuggest(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: tokman suggest <command> [args...]")
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
	lines := strings.Split(rawOutput, "\n")

	fmt.Printf("Command: %s\n", strings.Join(args, " "))
	fmt.Printf("Output: %d lines, ~%d tokens\n\n", len(lines), originalTokens)

	// Detect content type
	router := filter.NewContentRouter()
	ct, _ := router.Route(rawOutput)

	fmt.Printf("Detected content type: %s\n\n", ct)

	// Try different strategies
	fmt.Println("Compression Strategies:")
	fmt.Printf("%-15s %8s %8s %7s %8s\n", "Strategy", "Tokens", "Saved", "Pct", "Quality")
	fmt.Printf("%-15s %8s %8s %7s %8s\n", "───────────────", "────────", "────────", "───────", "────────")

	equiv := filter.NewSemanticEquivalence()
	bestStrategy := ""
	bestScore := 0.0

	strategies := []struct {
		name   string
		preset filter.PipelinePreset
		mode   filter.Mode
	}{
		{"fast/minimal", filter.PresetFast, filter.ModeMinimal},
		{"balanced/minimal", filter.PresetBalanced, filter.ModeMinimal},
		{"full/minimal", filter.PresetFull, filter.ModeMinimal},
		{"fast/aggressive", filter.PresetFast, filter.ModeAggressive},
		{"balanced/aggressive", filter.PresetBalanced, filter.ModeAggressive},
	}

	for _, s := range strategies {
		cfg := filter.PresetConfig(s.preset, s.mode)
		pipeline := filter.NewPipelineCoordinator(cfg)
		compressed, stats := pipeline.Process(rawOutput)
		report := equiv.Check(rawOutput, compressed)

		score := stats.ReductionPercent * report.Score
		if report.IsGood() && score > bestScore {
			bestScore = score
			bestStrategy = s.name
		}

		fmt.Printf("%-15s %8d %8d %6.1f%% %7.0f%%\n",
			s.name, stats.FinalTokens, stats.TotalSaved,
			stats.ReductionPercent, report.Score*100)
	}

	fmt.Printf("\nRecommended: %s\n", bestStrategy)
	fmt.Printf("Use: tokman --preset %s %s\n",
		strings.Split(bestStrategy, "/")[0], strings.Join(args, " "))

	return nil
}
