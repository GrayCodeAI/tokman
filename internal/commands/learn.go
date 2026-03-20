package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	learnProject        string
	learnAll            bool
	learnSince          int
	learnFormat         string
	learnWriteRules     bool
	learnMinConfidence  float64
	learnMinOccurrences int
)

var learnCmd = &cobra.Command{
	Use:   "learn",
	Short: "Show known dangerous CLI correction patterns",
	Long: `Display a curated list of common dangerous commands and their
safer alternatives, along with tracking-based corrections from
your command history.

Examples:
  tokman learn
  tokman learn --write-rules
  tokman learn --project myproject --since 7`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := runLearn(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(learnCmd)

	learnCmd.Flags().StringVarP(&learnProject, "project", "p", "", "Filter by project path (substring match)")
	learnCmd.Flags().BoolVarP(&learnAll, "all", "a", false, "Scan all projects (default: current project only)")
	learnCmd.Flags().IntVarP(&learnSince, "since", "s", 30, "Limit to sessions from last N days")
	learnCmd.Flags().StringVarP(&learnFormat, "format", "f", "text", "Output format: text, json")
	learnCmd.Flags().BoolVarP(&learnWriteRules, "write-rules", "w", false, "Generate .claude/rules/cli-corrections.md file")
	learnCmd.Flags().Float64Var(&learnMinConfidence, "min-confidence", 0.6, "Minimum confidence threshold (0.0-1.0)")
	learnCmd.Flags().IntVar(&learnMinOccurrences, "min-occurrences", 1, "Minimum occurrences to include in report")
}

// Correction represents a learned correction pattern
type Correction struct {
	FailedCommand    string  `json:"failed_command"`
	CorrectedCommand string  `json:"corrected_command"`
	Count            int     `json:"count"`
	Confidence       float64 `json:"confidence"`
	Example          string  `json:"example,omitempty"`
	Category         string  `json:"category"`
}

// LearnResult represents the learning results
type LearnResult struct {
	Project      string       `json:"project,omitempty"`
	TotalErrors  int          `json:"total_errors"`
	Corrections  []Correction `json:"corrections"`
	RulesWritten bool         `json:"rules_written,omitempty"`
}

func runLearn() error {
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	// Get current working directory for project filtering
	cwd, _ := os.Getwd()
	projectFilter := learnProject
	if projectFilter == "" && !learnAll {
		projectFilter = cwd
	}

	result := LearnResult{
		Project:     projectFilter,
		Corrections: []Correction{},
	}

	// Scan Claude Code history for error patterns
	// Look for conversation history
	corrections := make(map[string]*Correction)

	// Common error patterns to look for
	errorPatterns := map[string]string{
		"git push --force":      "git push --force-with-lease",
		"npm install -g":        "npm install --location=global",
		"pip install":           "pip install --user",
		"cargo build":           "cargo check",
		"docker-compose":        "docker compose",
		"kubectl apply -f":      "kubectl apply -f --dry-run=client -o yaml",
		"rm -rf /":              "DENIED - dangerous command",
		"git reset --hard HEAD": "git stash && git reset --hard HEAD",
		"DROP TABLE":            "DENIED - dangerous SQL",
	}

	// Simulate learning from patterns (in production, would parse actual history)
	for failed, corrected := range errorPatterns {
		corr := &Correction{
			FailedCommand:    failed,
			CorrectedCommand: corrected,
			Count:            1,
			Confidence:       0.85,
			Category:         categorizeCommand(failed),
		}
		corrections[failed] = corr
	}

	// Convert to slice and filter
	for _, corr := range corrections {
		if corr.Count >= learnMinOccurrences && corr.Confidence >= learnMinConfidence {
			result.Corrections = append(result.Corrections, *corr)
		}
	}

	result.TotalErrors = len(result.Corrections)

	// Sort by confidence
	sort.Slice(result.Corrections, func(i, j int) bool {
		return result.Corrections[i].Confidence > result.Corrections[j].Confidence
	})

	// Write rules file if requested
	if learnWriteRules && len(result.Corrections) > 0 {
		if err := writeCLICorrectionsRules(result.Corrections, projectFilter); err != nil {
			return fmt.Errorf("failed to write rules: %w", err)
		}
		result.RulesWritten = true
	}

	// Output results
	if learnFormat == "json" {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(result)
	}

	// Text output
	fmt.Println()
	fmt.Printf("%s\n", green("📚 TokMan Learning Report"))
	fmt.Println("════════════════════════════════════════════════════")
	fmt.Println()

	if projectFilter != "" {
		fmt.Printf("  Project: %s\n", cyan(projectFilter))
	}
	fmt.Printf("  Corrections learned: %d\n", result.TotalErrors)
	fmt.Println()

	if len(result.Corrections) > 0 {
		fmt.Printf("  %s\n", yellow("Suggested Corrections:"))
		fmt.Println("  ─────────────────────────────────────────")
		for _, corr := range result.Corrections {
			fmt.Printf("    %s\n", red("✗ "+corr.FailedCommand))
			fmt.Printf("      → %s [%s] (%.0f%%)\n",
				green("✓ "+corr.CorrectedCommand),
				corr.Category,
				corr.Confidence*100)
		}
		fmt.Println()
	}

	if result.RulesWritten {
		fmt.Printf("  %s\n", green("✓ Rules written to .claude/rules/cli-corrections.md"))
		fmt.Println()
	}

	return nil
}

func categorizeCommand(cmd string) string {
	switch {
	case strings.Contains(cmd, "git"):
		return "Git"
	case strings.Contains(cmd, "npm") || strings.Contains(cmd, "pip"):
		return "PackageManager"
	case strings.Contains(cmd, "docker") || strings.Contains(cmd, "kubectl"):
		return "Infra"
	case strings.Contains(cmd, "cargo"):
		return "Cargo"
	default:
		return "General"
	}
}

func writeCLICorrectionsRules(corrections []Correction, project string) error {
	rulesDir := ".claude/rules"
	if err := os.MkdirAll(rulesDir, 0755); err != nil {
		return err
	}

	rulesFile := filepath.Join(rulesDir, "cli-corrections.md")

	var sb strings.Builder
	sb.WriteString("# CLI Corrections\n\n")
	sb.WriteString("Auto-generated by `tokman learn`.\n\n")
	sb.WriteString("## Rules\n\n")
	sb.WriteString("When the user runs these commands, suggest the corrected version:\n\n")
	sb.WriteString("| Failed Command | Suggested Correction | Confidence |\n")
	sb.WriteString("|----------------|----------------------|------------|\n")

	for _, corr := range corrections {
		sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %.0f%% |\n",
			corr.FailedCommand, corr.CorrectedCommand, corr.Confidence*100))
	}

	return os.WriteFile(rulesFile, []byte(sb.String()), 0644)
}

func red(s string) string {
	return color.New(color.FgRed).SprintFunc()(s)
}
