package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/core"
)

var (
	costModel  string
	costTokens int
)

var costCmd = &cobra.Command{
	Use:   "cost",
	Short: "Calculate cost savings from token reduction",
	Long:  `Estimate dollar savings from token compression for different models.`,
	RunE:  runCost,
}

func init() {
	costCmd.Flags().StringVar(&costModel, "model", "gpt-4o-mini", "model to calculate cost for")
	costCmd.Flags().IntVar(&costTokens, "tokens", 0, "number of tokens saved (or use with tracking)")
	rootCmd.AddCommand(costCmd)
}

func runCost(cmd *cobra.Command, args []string) error {
	models := []string{"gpt-4o-mini", "gpt-4o", "claude-3.5-sonnet", "claude-3-haiku"}

	if costTokens > 0 {
		// Calculate for specific token count
		fmt.Printf("Cost savings for %d tokens:\n\n", costTokens)
		fmt.Printf("%-25s %12s\n", "Model", "Savings")
		fmt.Printf("%-25s %12s\n", "─────────────────────────", "────────────")

		for _, model := range models {
			savings := core.CalculateSavings(costTokens, model)
			fmt.Printf("%-25s $%.4f\n", model, savings)
		}
	} else {
		// Show pricing table
		fmt.Println("Model Pricing (per 1M tokens):")
		fmt.Printf("%-25s %12s %12s\n", "Model", "Input", "Output")
		fmt.Printf("%-25s %12s %12s\n", "─────────────────────────", "────────────", "────────────")

		for _, model := range models {
			if p, ok := core.CommonModelPricing[model]; ok {
				fmt.Printf("%-25s $%.2f $%.2f\n", model, p.InputPerMillion, p.OutputPerMillion)
			}
		}

		fmt.Println("\nUse --tokens N to calculate savings for a specific token count.")
	}

	return nil
}
