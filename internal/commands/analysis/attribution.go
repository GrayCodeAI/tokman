package analysis

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var attributionCmd = &cobra.Command{
	Use:   "attribution",
	Short: "Show token savings by AI agent, model, and provider",
	Long: `Display detailed breakdown of token savings grouped by:
  - AI Agent (Claude Code, OpenCode, Cursor, etc.)
  - Model (claude-3-opus, gpt-4, gemini-pro, etc.)
  - Provider (Anthropic, OpenAI, Google, etc.)

This helps understand which tools and models benefit most from compression.

Examples:
  tokman attribution                    # Full breakdown
  tokman attribution --by-model         # Group by model
  tokman attribution --by-provider      # Group by provider
  tokman attribution --by-agent         # Group by agent`,
	RunE: runAttribution,
}

func init() {
	attributionCmd.Flags().Bool("by-model", false, "Group by model name")
	attributionCmd.Flags().Bool("by-provider", false, "Group by provider")
	attributionCmd.Flags().Bool("by-agent", true, "Group by agent name (default)")
	attributionCmd.Flags().Bool("by-repo", false, "Also show per-repository breakdown")
	attributionCmd.Flags().Int("limit", 20, "Limit number of results")

	registry.Add(func() { registry.Register(attributionCmd) })
}

func runAttribution(cmd *cobra.Command, args []string) error {
	tracker, err := shared.OpenTracker()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Tracking not initialized: %v\n", err)
		return nil
	}
	defer tracker.Close()

	byModel, _ := cmd.Flags().GetBool("by-model")
	byProvider, _ := cmd.Flags().GetBool("by-provider")
	byRepo, _ := cmd.Flags().GetBool("by-repo")
	limit, _ := cmd.Flags().GetInt("limit")

	// Query based on grouping
	var groupBy string
	switch {
	case byModel:
		groupBy = "model_name"
	case byProvider:
		groupBy = "provider"
	default:
		groupBy = "agent_name"
	}

	query := fmt.Sprintf(`
		SELECT 
			%s,
			COUNT(*) as count,
			SUM(saved_tokens) as saved,
			SUM(original_tokens) as original,
			ROUND(100.0 * SUM(saved_tokens) / NULLIF(SUM(original_tokens), 0), 1) as pct
		FROM commands
		WHERE %s IS NOT NULL AND %s != ''
		GROUP BY %s
		ORDER BY saved DESC
		LIMIT ?
	`, groupBy, groupBy, groupBy, groupBy)

	rows, err := tracker.Query(query, limit)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	// Print header
	fmt.Printf("\n%s\n", "Token Savings by "+strings.ToUpper(groupBy[:1])+groupBy[1:])
	fmt.Printf("%s\n", strings.Repeat("═", 75))

	fmt.Printf("%-30s %8s %12s %12s %8s\n",
		strings.ToUpper(groupBy[:1])+groupBy[1:], "Commands", "Saved", "Original", "Reduction")
	fmt.Println(strings.Repeat("─", 75))

	totalSaved := 0
	totalOriginal := 0
	totalCommands := 0

	for rows.Next() {
		var name string
		var count, saved, original int
		var pct float64
		if err := rows.Scan(&name, &count, &saved, &original, &pct); err != nil {
			continue
		}

		// Truncate long names
		displayName := name
		if len(displayName) > 28 {
			displayName = displayName[:25] + "..."
		}

		fmt.Printf("%-30s %8d %12s %12s %7.1f%%\n",
			displayName, count, formatTokens(saved), formatTokens(original), pct)

		totalSaved += saved
		totalOriginal += original
		totalCommands += count
	}

	if totalCommands > 0 {
		fmt.Println(strings.Repeat("─", 75))
		overallPct := 0.0
		if totalOriginal > 0 {
			overallPct = float64(totalSaved) / float64(totalOriginal) * 100
		}
		fmt.Printf("%-30s %8d %12s %12s %7.1f%%\n",
			"TOTAL", totalCommands, formatTokens(totalSaved), formatTokens(totalOriginal), overallPct)
	}

	// If by-repo flag, show repository breakdown
	if byRepo {
		fmt.Println()
		showRepoBreakdown(tracker, limit)
	}

	return nil
}

func showRepoBreakdown(tracker *tracking.Tracker, limit int) {
	query := `
		SELECT 
			project_path,
			agent_name,
			COUNT(*) as count,
			SUM(saved_tokens) as saved,
			SUM(original_tokens) as original,
			ROUND(100.0 * SUM(saved_tokens) / NULLIF(SUM(original_tokens), 0), 1) as pct
		FROM commands
		WHERE agent_name IS NOT NULL AND agent_name != ''
		GROUP BY project_path, agent_name
		ORDER BY saved DESC
		LIMIT ?
	`

	rows, err := tracker.Query(query, limit)
	if err != nil {
		return
	}
	defer rows.Close()

	fmt.Printf("\n%s\n", "By Repository + Agent")
	fmt.Printf("%s\n", strings.Repeat("═", 65))
	fmt.Printf("%-25s %-15s %10s %8s\n", "Repository", "Agent", "Saved", "Reduction")
	fmt.Println(strings.Repeat("─", 65))

	for rows.Next() {
		var path, agent string
		var count, saved, original int
		var pct float64
		if err := rows.Scan(&path, &agent, &count, &saved, &original, &pct); err != nil {
			continue
		}

		// Extract repo name from path
		repoName := path
		if idx := strings.LastIndex(path, "/"); idx >= 0 {
			repoName = path[idx+1:]
		}
		if len(repoName) > 23 {
			repoName = repoName[:20] + "..."
		}
		if len(agent) > 13 {
			agent = agent[:10] + "..."
		}

		fmt.Printf("%-25s %-15s %10s %7.1f%%\n", repoName, agent, formatTokens(saved), pct)
	}
}

func formatTokens(n int) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}
