package commands

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var suggestCmd = &cobra.Command{
	Use:   "suggest [command...]",
	Short: "Suggest optimal compression settings or workflow optimizations",
	Long: `Analyze command output and suggest the best compression strategy.

When run without arguments, analyzes your recent command history and suggests
workflow optimizations based on token savings opportunities.

Examples:
  tokman suggest                  # Analyze workflow patterns
  tokman suggest git status       # Suggest best strategy for a command
  tokman suggest npm test         # Analyze test output compression`,
	RunE: runSuggest,
}

func init() {
	rootCmd.AddCommand(suggestCmd)
}

func runSuggest(cmd *cobra.Command, args []string) error {
	// No args = workflow analysis mode
	if len(args) == 0 {
		return runWorkflowSuggestions()
	}
	// With args = command analysis mode
	return runCommandSuggestion(args)
}

// runWorkflowSuggestions analyzes recent commands and provides optimization suggestions
func runWorkflowSuggestions() error {
	dbPath := tracking.DatabasePath()
	tracker, err := tracking.NewTracker(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open tracking database: %w", err)
	}
	defer tracker.Close()

	// Get command statistics
	stats, err := tracker.GetCommandStats("")
	if err != nil {
		return fmt.Errorf("failed to get command stats: %w", err)
	}

	// Get recent commands for pattern analysis
	recent, err := tracker.GetRecentCommands("", 100)
	if err != nil {
		return fmt.Errorf("failed to get recent commands: %w", err)
	}

	// Get total savings
	saved24h, _ := tracker.TokensSaved24h()
	savedTotal, _ := tracker.TokensSavedTotal()
	savingsPct, _ := tracker.OverallSavingsPct()

	fmt.Println("💡 Suggestions based on your workflow")
	fmt.Println("═══════════════════════════════════════")
	fmt.Printf("\n📊 Summary: %d tokens saved (24h), %d total (%.1f%% reduction)\n\n", saved24h, savedTotal, savingsPct)

	// Analyze command patterns
	suggestions := analyzeCommandPatterns(stats, recent)

	if len(suggestions) == 0 {
		fmt.Println("✅ Your workflow is well-optimized! No major improvements found.")
		return nil
	}

	// Display suggestions
	for i, s := range suggestions {
		fmt.Printf("%d. %s\n", i+1, s.Title)
		fmt.Printf("   %s\n", s.Description)
		if s.Command != "" {
			fmt.Printf("   Run: %s\n", s.Command)
		}
		if s.EstimatedSavings > 0 {
			fmt.Printf("   💰 Estimated savings: %s tokens/run\n", formatNumber(s.EstimatedSavings))
		}
		fmt.Println()
	}

	return nil
}

type suggestion struct {
	Title            string
	Description      string
	Command          string
	EstimatedSavings int64
	Priority         int // lower = higher priority
}

// analyzeCommandPatterns identifies optimization opportunities
func analyzeCommandPatterns(stats []tracking.CommandStats, recent []tracking.CommandRecord) []suggestion {
	var suggestions []suggestion

	// Sort stats by total saved (already sorted by DB, but ensure)
	sort.Slice(stats, func(i, j int) bool {
		return stats[i].TotalSaved > stats[j].TotalSaved
	})

	// High-frequency, low-savings commands - potential improvement
	for _, s := range stats {
		if s.ExecutionCount >= 5 && float64(s.TotalSaved) < float64(s.TotalOriginal)*0.3 {
			// Command runs often but saves less than 30%
			avgSaved := s.TotalSaved / s.ExecutionCount
			if avgSaved < 500 && s.TotalOriginal > 1000 {
				suggestions = append(suggestions, suggestion{
					Title:            fmt.Sprintf("'%s' could save more tokens", s.Command),
					Description:      fmt.Sprintf("Run %d times, only %.0f%% reduction. Try aggressive mode.", s.ExecutionCount, s.ReductionPct),
					Command:          fmt.Sprintf("tokman --mode aggressive %s", s.Command),
					EstimatedSavings: int64(float64(s.TotalOriginal)*0.5) - int64(s.TotalSaved),
					Priority:         2,
				})
			}
		}
	}

	// Commands with large output not using tokman
	commandCounts := make(map[string]int)
	commandTokens := make(map[string]int64)
	for _, r := range recent {
		cmd := extractBaseCommand(r.Command)
		commandCounts[cmd]++
		if r.OriginalTokens > 0 {
			commandTokens[cmd] += int64(r.OriginalTokens)
		}
	}

	for cmd, count := range commandCounts {
		avgTokens := commandTokens[cmd] / int64(count)
		// High-output commands that could benefit
		if avgTokens > 3000 && count >= 3 {
			suggestions = append(suggestions, suggestion{
				Title:            fmt.Sprintf("'%s' outputs are large (~%s tokens)", cmd, formatNumber(avgTokens)),
				Description:      "Consider wrapping with tokman for automatic compression.",
				Command:          fmt.Sprintf("tokman %s", cmd),
				EstimatedSavings: int64(float64(avgTokens) * 0.7),
				Priority:         1,
			})
		}
	}

	// Check for git commands without tokman
	gitCommands := []string{"git status", "git log", "git diff"}
	for _, gc := range gitCommands {
		found := false
		for _, s := range stats {
			if strings.HasPrefix(s.Command, gc) {
				found = true
				break
			}
		}
		if !found && commandCounts[gc] >= 3 {
			suggestions = append(suggestions, suggestion{
				Title:            fmt.Sprintf("You run '%s' frequently", gc),
				Description:      "TokMan can significantly reduce git output size.",
				Command:          "tokman init -g",
				EstimatedSavings: 1500,
				Priority:         1,
			})
		}
	}

	// Check for test commands
	testCommands := []string{"npm test", "pytest", "cargo test", "go test"}
	for _, tc := range testCommands {
		for _, s := range stats {
			if strings.Contains(s.Command, tc) && s.TotalOriginal > 5000 {
				suggestions = append(suggestions, suggestion{
					Title:            fmt.Sprintf("Test output from '%s' is large", tc),
					Description:      "Use focused test mode to show only failures.",
					Command:          fmt.Sprintf("tokman --mode aggressive %s", s.Command),
					EstimatedSavings: int64(float64(s.TotalOriginal) * 0.8 / float64(s.ExecutionCount)),
					Priority:         2,
				})
				break
			}
		}
	}

	// Suggest enabling aggressive mode if not already used
	avgSavings := savingsPct(recent)
	if avgSavings < 50 && len(recent) > 10 {
		suggestions = append(suggestions, suggestion{
			Title:            "Consider enabling aggressive mode globally",
			Description:      fmt.Sprintf("Current average savings: %.0f%%. Aggressive mode can increase to 80%%+.", avgSavings),
			Command:          "tokman config set mode aggressive",
			EstimatedSavings: 0,
			Priority:         3,
		})
	}

	// Sort by priority
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Priority < suggestions[j].Priority
	})

	// Limit to top 5 suggestions
	if len(suggestions) > 5 {
		suggestions = suggestions[:5]
	}

	return suggestions
}

func extractBaseCommand(cmd string) string {
	parts := strings.Fields(cmd)
	if len(parts) >= 2 {
		return parts[0] + " " + parts[1]
	}
	return cmd
}

func savingsPct(recent []tracking.CommandRecord) float64 {
	var saved, total int64
	for _, r := range recent {
		saved += int64(r.SavedTokens)
		total += int64(r.OriginalTokens)
	}
	if total == 0 {
		return 0
	}
	return float64(saved) / float64(total) * 100
}

func formatNumber(n int64) string {
	if n >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	if n >= 1000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	return fmt.Sprintf("%d", n)
}

// runCommandSuggestion analyzes a specific command
func runCommandSuggestion(args []string) error {
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
