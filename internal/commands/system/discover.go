package system

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var (
	discoverProject string
	discoverLimit   int
	discoverAll     bool
	discoverSince   int
	discoverFormat  string
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Discover missed token savings from Claude Code history",
	Long: `Analyze Claude Code session history to find commands that could have
used TokMan wrappers for token savings.

Scans ~/.claude/projects/*/CLAUDE.md and conversation history to identify
patterns of commands that weren't rewritten.

Examples:
  tokman discover
  tokman discover --project myproject
  tokman discover --all --since 7`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDiscover(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	registry.Add(func() { registry.Register(discoverCmd) })

	discoverCmd.Flags().StringVarP(&discoverProject, "project", "p", "", "Filter by project path (substring match)")
	discoverCmd.Flags().IntVarP(&discoverLimit, "limit", "l", 15, "Max commands per section")
	discoverCmd.Flags().BoolVarP(&discoverAll, "all", "a", false, "Scan all projects (default: current project only)")
	discoverCmd.Flags().IntVarP(&discoverSince, "since", "s", 30, "Limit to sessions from last N days")
	discoverCmd.Flags().StringVarP(&discoverFormat, "format", "f", "text", "Output format: text, json")
}

// DiscoveredCommand represents a discovered command pattern
type DiscoveredCommand struct {
	Command     string  `json:"command"`
	Count       int     `json:"count"`
	Category    string  `json:"category"`
	CouldSave   bool    `json:"could_save"`
	SavingsPct  float64 `json:"savings_percent,omitempty"`
	TokensSaved int     `json:"tokens_saved,omitempty"`
	Example     string  `json:"example,omitempty"`
}

// DiscoverResult represents the discovery results
type DiscoverResult struct {
	Project         string              `json:"project,omitempty"`
	TotalCommands   int                 `json:"total_commands"`
	MissedSavings   int                 `json:"missed_savings"`
	Opportunities   []DiscoveredCommand `json:"opportunities"`
	UnsupportedCmds []DiscoveredCommand `json:"unsupported_commands,omitempty"`
}

func runDiscover() error {
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	// Get current working directory for project filtering
	cwd, _ := os.Getwd()
	projectFilter := discoverProject
	if projectFilter == "" && !discoverAll {
		projectFilter = cwd
	}

	// Initialize tracker to get historical data
	tracker, err := tracking.NewTracker(tracking.DatabasePath())
	if err != nil {
		return fmt.Errorf("failed to initialize tracker: %w", err)
	}
	defer tracker.Close()

	// Get command stats from tracker
	stats, err := tracker.GetCommandStats(projectFilter)
	if err != nil {
		return fmt.Errorf("failed to get command stats: %w", err)
	}

	// Analyze commands for missed opportunities
	result := DiscoverResult{
		Project:       projectFilter,
		Opportunities: []DiscoveredCommand{},
	}

	// Known TokMan wrappers for analysis
	tokmanWrappers := map[string]string{
		"git":     "Git",
		"gh":      "GitHub",
		"cargo":   "Cargo",
		"docker":  "Infra",
		"kubectl": "Infra",
		"npm":     "PackageManager",
		"pnpm":    "PackageManager",
		"npx":     "PackageManager",
		"go":      "Go",
		"pytest":  "Python",
		"ruff":    "Python",
		"mypy":    "Build",
		"tsc":     "Build",
		"vitest":  "Tests",
		"curl":    "Network",
		"psql":    "Infra",
		"aws":     "Infra",
		"ls":      "Files",
		"find":    "Files",
		"tree":    "Files",
		"grep":    "Files",
	}

	unsupportedCommands := []DiscoveredCommand{}
	totalCommands := 0

	for _, stat := range stats {
		totalCommands += stat.ExecutionCount

		// Check if this command was already using tokman
		if strings.HasPrefix(stat.Command, "tokman ") {
			continue // Already optimized
		}

		// Extract base command
		parts := strings.Fields(stat.Command)
		if len(parts) == 0 {
			continue
		}
		baseCmd := parts[0]

		// Check if it's a known wrapper opportunity
		if category, ok := tokmanWrappers[baseCmd]; ok {
			result.Opportunities = append(result.Opportunities, DiscoveredCommand{
				Command:     stat.Command,
				Count:       stat.ExecutionCount,
				Category:    category,
				CouldSave:   true,
				SavingsPct:  stat.ReductionPct,
				TokensSaved: stat.TotalSaved,
			})
		} else if stat.TotalSaved == 0 && stat.ExecutionCount >= 3 {
			// Unsupported command that's frequently used
			unsupportedCommands = append(unsupportedCommands, DiscoveredCommand{
				Command:   stat.Command,
				Count:     stat.ExecutionCount,
				Category:  "Unknown",
				CouldSave: false,
			})
		}
	}

	result.TotalCommands = totalCommands
	result.MissedSavings = len(result.Opportunities)
	result.UnsupportedCmds = unsupportedCommands

	// Sort by count (descending)
	sort.Slice(result.Opportunities, func(i, j int) bool {
		return result.Opportunities[i].Count > result.Opportunities[j].Count
	})
	sort.Slice(result.UnsupportedCmds, func(i, j int) bool {
		return result.UnsupportedCmds[i].Count > result.UnsupportedCmds[j].Count
	})

	// Limit results
	if len(result.Opportunities) > discoverLimit {
		result.Opportunities = result.Opportunities[:discoverLimit]
	}
	if len(result.UnsupportedCmds) > discoverLimit {
		result.UnsupportedCmds = result.UnsupportedCmds[:discoverLimit]
	}

	// Output results
	if discoverFormat == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	// Text output
	fmt.Println()
	fmt.Printf("%s\n", green("🔍 TokMan Discovery Report"))
	fmt.Println("════════════════════════════════════════════════════")
	fmt.Println()

	if projectFilter != "" {
		fmt.Printf("  Project: %s\n", cyan(projectFilter))
	}
	fmt.Printf("  Commands analyzed: %d\n", totalCommands)
	fmt.Println()

	if len(result.Opportunities) > 0 {
		fmt.Printf("  %s\n", yellow("Missed Opportunities (could use TokMan):"))
		fmt.Println("  ─────────────────────────────────────────")
		for _, opp := range result.Opportunities {
			pct := ""
			if opp.SavingsPct > 0 {
				pct = fmt.Sprintf("  %4.1f%% saved", opp.SavingsPct)
			}
			fmt.Printf("    %-30s  %3dx  [%s]%s\n", truncate(opp.Command, 30), opp.Count, opp.Category, pct)
		}
		fmt.Println()
	}

	if len(result.UnsupportedCmds) > 0 && shared.Verbose > 0 {
		fmt.Printf("  %s\n", cyan("Unsupported Commands (frequent but no TokMan wrapper):"))
		fmt.Println("  ─────────────────────────────────────────")
		for _, cmd := range result.UnsupportedCmds {
			fmt.Printf("    %-30s  %3dx\n", truncate(cmd.Command, 30), cmd.Count)
		}
		fmt.Println()
	}

	if len(result.Opportunities) == 0 {
		fmt.Printf("  %s\n", green("✓ All commands are already optimized!"))
		fmt.Println()
	}

	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// Get git history for discover (using git log)
func getGitCommandHistory(sinceDays int) ([]string, error) {
	since := fmt.Sprintf("--since=%d days ago", sinceDays)
	cmd := exec.Command("git", "log", since, "--all", "--pretty=format:%s", "--grep=")

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	return lines, nil
}
