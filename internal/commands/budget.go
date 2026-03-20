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

var budgetCmd = &cobra.Command{
	Use:   "budget [command...]",
	Short: "Suggest optimal token budget for a command",
	Long: `Analyze command output and suggest the minimum budget
that preserves all critical information.

Example:
  tokman budget git log --oneline -50
  tokman budget docker ps`,
	RunE: runBudget,
}

func init() {
	rootCmd.AddCommand(budgetCmd)
}

func runBudget(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: tokman budget <command> [args...]")
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
	fmt.Printf("Original output: ~%d tokens\n\n", originalTokens)

	// Test different budgets
	budgets := []int{50, 100, 200, 500, 1000, 2000, 5000}

	fmt.Printf("%-10s %8s %8s %8s %10s\n", "Budget", "Output", "Saved", "Quality", "Verdict")
	fmt.Printf("%-10s %8s %8s %8s %10s\n", "──────", "──────", "─────", "───────", "───────")

	equiv := filter.NewSemanticEquivalence()
	recommended := 0

	for _, budget := range budgets {
		if budget >= originalTokens {
			continue
		}

		pipeline := filter.NewPipelineCoordinator(filter.PipelineConfig{
			Mode:                filter.ModeMinimal,
			Budget:              budget,
			SessionTracking:     false,
			NgramEnabled:        true,
			EnableCompaction:    true,
			EnableAttribution:   true,
			EnableH2O:           true,
			EnableAttentionSink: true,
		})

		compressed, _ := pipeline.Process(rawOutput)
		finalTokens := core.EstimateTokens(compressed)
		saved := originalTokens - finalTokens

		report := equiv.Check(rawOutput, compressed)
		quality := fmt.Sprintf("%.0f%%", report.Score*100)

		verdict := ""
		if report.IsGood() && recommended == 0 {
			verdict = "← recommended"
			recommended = budget
		} else if !report.ErrorPreserved {
			verdict = "⚠ errors lost"
		} else if report.Score < 0.7 {
			verdict = "⚠ low quality"
		}

		fmt.Printf("%-10d %8d %8d %8s %10s\n", budget, finalTokens, saved, quality, verdict)
	}

	if recommended > 0 {
		fmt.Printf("\nRecommended budget: %d tokens\n", recommended)
		fmt.Printf("Use: tokman --budget %d %s\n", recommended, strings.Join(args, " "))
	} else {
		fmt.Printf("\nNo budget found that preserves all critical information.\n")
		fmt.Printf("Consider using --preset fast for lighter compression.\n")
	}

	return nil
}
