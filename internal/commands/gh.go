package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var ghCmd = &cobra.Command{
	Use:   "gh [args...]",
	Short: "GitHub CLI with token-optimized output",
	Long: `Execute GitHub CLI commands with compact output.

Provides specialized filtering for pr, issue, run, and repo commands.

Examples:
  tokman gh pr list
  tokman gh issue list --repo owner/repo
  tokman gh run list`,
	DisableFlagParsing: true,
	RunE:               runGh,
}

func init() {
	rootCmd.AddCommand(ghCmd)
}

func runGh(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"--help"}
	}

	// Route to specialized handlers
	switch args[0] {
	case "pr":
		return runGhPr(args[1:])
	case "issue":
		return runGhIssue(args[1:])
	case "run":
		return runGhRun(args[1:])
	case "repo":
		return runGhRepo(args[1:])
	default:
		return runGhPassthrough(args)
	}
}

func runGhPr(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gh pr %s\n", strings.Join(args, " "))
	}

	// Add --json for structured output if listing
	if len(args) > 0 && args[0] == "list" {
		args = append(args, "--json", "number,title,author,headRefName,state")
	}

	execCmd := exec.Command("gh", append([]string{"pr"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGhPrOutput(raw, args)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gh pr %s", strings.Join(args, " ")), "tokman gh pr", originalTokens, filteredTokens)

	return err
}

func runGhIssue(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gh issue %s\n", strings.Join(args, " "))
	}

	if len(args) > 0 && args[0] == "list" {
		args = append(args, "--json", "number,title,author,state")
	}

	execCmd := exec.Command("gh", append([]string{"issue"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGhIssueOutput(raw, args)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gh issue %s", strings.Join(args, " ")), "tokman gh issue", originalTokens, filteredTokens)

	return err
}

func runGhRun(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gh run %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gh", append([]string{"run"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGhRunOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gh run %s", strings.Join(args, " ")), "tokman gh run", originalTokens, filteredTokens)

	return err
}

func runGhRepo(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gh repo %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gh", append([]string{"repo"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGhRepoOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gh repo %s", strings.Join(args, " ")), "tokman gh repo", originalTokens, filteredTokens)

	return err
}

func runGhPassthrough(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gh %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gh", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGhOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gh %s", strings.Join(args, " ")), "tokman gh", originalTokens, filteredTokens)

	return err
}

// Filter functions

type GhPR struct {
	Number      int    `json:"number"`
	Title       string `json:"title"`
	Author      string `json:"author"`
	HeadRefName string `json:"headRefName"`
	State       string `json:"state"`
}

func filterGhPrOutput(raw string, args []string) string {
	// Try JSON parsing for list command
	if len(args) > 0 && args[0] == "list" {
		var prs []GhPR
		if err := json.Unmarshal([]byte(raw), &prs); err == nil {
			var result []string
			result = append(result, fmt.Sprintf("📋 Pull Requests (%d):", len(prs)))
			for i, pr := range prs {
				if i >= 15 {
					result = append(result, fmt.Sprintf("   ... +%d more", len(prs)-15))
					break
				}
				state := "○"
				if pr.State == "OPEN" {
					state = "●"
				} else if pr.State == "MERGED" {
					state = "✓"
				}
				result = append(result, fmt.Sprintf("   %s #%d: %s (%s)", state, pr.Number, truncateLine(pr.Title, 50), pr.Author))
			}
			return strings.Join(result, "\n")
		}
	}
	return raw
}

type GhIssue struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Author string `json:"author"`
	State  string `json:"state"`
}

func filterGhIssueOutput(raw string, args []string) string {
	if len(args) > 0 && args[0] == "list" {
		var issues []GhIssue
		if err := json.Unmarshal([]byte(raw), &issues); err == nil {
			var result []string
			result = append(result, fmt.Sprintf("📋 Issues (%d):", len(issues)))
			for i, issue := range issues {
				if i >= 15 {
					result = append(result, fmt.Sprintf("   ... +%d more", len(issues)-15))
					break
				}
				state := "○"
				if issue.State == "OPEN" {
					state = "●"
				} else if issue.State == "CLOSED" {
					state = "✓"
				}
				result = append(result, fmt.Sprintf("   %s #%d: %s (%s)", state, issue.Number, truncateLine(issue.Title, 50), issue.Author))
			}
			return strings.Join(result, "\n")
		}
	}
	return raw
}

func filterGhRunOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Compact workflow run output
		result = append(result, truncateLine(line, 100))
	}

	if len(result) > 20 {
		return strings.Join(result[:20], "\n") + fmt.Sprintf("\n... (%d more lines)", len(result)-20)
	}
	return strings.Join(result, "\n")
}

func filterGhRepoOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, truncateLine(line, 100))
		}
	}

	if len(result) > 15 {
		return strings.Join(result[:15], "\n") + fmt.Sprintf("\n... (%d more lines)", len(result)-15)
	}
	return strings.Join(result, "\n")
}

func filterGhOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, truncateLine(line, 120))
		}
	}

	if len(result) > 30 {
		return strings.Join(result[:30], "\n") + fmt.Sprintf("\n... (%d more lines)", len(result)-30)
	}
	return strings.Join(result, "\n")
}
