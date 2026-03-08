package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var gtCmd = &cobra.Command{
	Use:   "gt [args...]",
	Short: "Graphite (gt) stacked PR commands with compact output",
	Long: `Execute Graphite CLI commands with compact output.

Provides specialized filtering for log, submit, sync, restack, create, and branch.

Examples:
  tokman gt log
  tokman gt submit
  tokman gt sync`,
	DisableFlagParsing: true,
	RunE:               runGt,
}

func init() {
	rootCmd.AddCommand(gtCmd)
}

func runGt(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"--help"}
	}

	// Route to specialized handlers
	switch args[0] {
	case "log":
		return runGtLog(args[1:])
	case "submit":
		return runGtSubmit(args[1:])
	case "sync":
		return runGtSync(args[1:])
	case "restack":
		return runGtRestack(args[1:])
	case "create":
		return runGtCreate(args[1:])
	case "branch":
		return runGtBranch(args[1:])
	default:
		return runGtPassthrough(args)
	}
}

func runGtLog(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gt log %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gt", append([]string{"log"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGtLogOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gt log %s", strings.Join(args, " ")), "tokman gt log", originalTokens, filteredTokens)

	return err
}

func runGtSubmit(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gt submit %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gt", append([]string{"submit"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGtSubmitOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gt submit %s", strings.Join(args, " ")), "tokman gt submit", originalTokens, filteredTokens)

	return err
}

func runGtSync(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gt sync %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gt", append([]string{"sync"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGtSyncOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gt sync %s", strings.Join(args, " ")), "tokman gt sync", originalTokens, filteredTokens)

	return err
}

func runGtRestack(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gt restack %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gt", append([]string{"restack"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGtRestackOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gt restack %s", strings.Join(args, " ")), "tokman gt restack", originalTokens, filteredTokens)

	return err
}

func runGtCreate(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gt create %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gt", append([]string{"create"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGtCreateOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gt create %s", strings.Join(args, " ")), "tokman gt create", originalTokens, filteredTokens)

	return err
}

func runGtBranch(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gt branch %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gt", append([]string{"branch"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGtBranchOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gt branch %s", strings.Join(args, " ")), "tokman gt branch", originalTokens, filteredTokens)

	return err
}

func runGtPassthrough(args []string) error {
	timer := tracking.Start()

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: gt %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("gt", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGtOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("gt %s", strings.Join(args, " ")), "tokman gt", originalTokens, filteredTokens)

	return err
}

// Filter functions

func filterGtLogOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var branches []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Extract branch info
		if strings.HasPrefix(line, "│") || strings.HasPrefix(line, "├") || strings.HasPrefix(line, "└") {
			// Tree structure - extract branch name
			branch := strings.TrimLeft(line, "│├└─ ")
			if branch != "" {
				branches = append(branches, truncateLine(branch, 50))
			}
		} else if line != "" && len(line) > 2 {
			branches = append(branches, truncateLine(line, 50))
		}
	}

	if len(branches) > 0 {
		result = append(result, fmt.Sprintf("🌳 Stack (%d branches):", len(branches)))
		for i, b := range branches {
			if i >= 15 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(branches)-15))
				break
			}
			result = append(result, fmt.Sprintf("   %s", b))
		}
		return strings.Join(result, "\n")
	}
	return raw
}

func filterGtSubmitOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Keep important lines
		if strings.Contains(line, "PR") || strings.Contains(line, "submitted") ||
			strings.Contains(line, "created") || strings.Contains(line, "updated") ||
			strings.Contains(line, "error") || strings.Contains(line, "success") {
			result = append(result, truncateLine(line, 80))
		}
	}

	if len(result) == 0 {
		return "✅ Submit completed"
	}
	return strings.Join(result, "\n")
}

func filterGtSyncOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, truncateLine(line, 80))
		}
	}

	if len(result) == 0 {
		return "✅ Sync completed"
	}
	if len(result) > 10 {
		return strings.Join(result[:10], "\n") + fmt.Sprintf("\n... (%d more)", len(result)-10)
	}
	return strings.Join(result, "\n")
}

func filterGtRestackOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, truncateLine(line, 80))
		}
	}

	if len(result) == 0 {
		return "✅ Restack completed"
	}
	if len(result) > 10 {
		return strings.Join(result[:10], "\n") + fmt.Sprintf("\n... (%d more)", len(result)-10)
	}
	return strings.Join(result, "\n")
}

func filterGtCreateOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, truncateLine(line, 80))
		}
	}

	if len(result) == 0 {
		return "✅ Branch created"
	}
	return strings.Join(result, "\n")
}

func filterGtBranchOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, truncateLine(line, 80))
		}
	}

	if len(result) > 15 {
		return strings.Join(result[:15], "\n") + fmt.Sprintf("\n... (%d more)", len(result)-15)
	}
	return strings.Join(result, "\n")
}

func filterGtOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, truncateLine(line, 100))
		}
	}

	if len(result) > 30 {
		return strings.Join(result[:30], "\n") + fmt.Sprintf("\n... (%d more lines)", len(result)-30)
	}
	return strings.Join(result, "\n")
}
