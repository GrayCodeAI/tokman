package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var ccusageDaily bool
var ccusageWeekly bool
var ccusageMonthly bool
var ccusageAll bool
var ccusageFormat string

var ccusageCmd = &cobra.Command{
	Use:   "ccusage",
	Short: "Show Claude Code API usage metrics via ccusage CLI",
	Long: `Display Claude Code API usage statistics from the ccusage npm package.

Requires ccusage to be installed: npm i -g ccusage

Shows token consumption and costs for Claude Code sessions.

Examples:
  tokman ccusage --daily
  tokman ccusage --weekly
  tokman ccusage --monthly
  tokman ccusage --all`,
	RunE: runCcusage,
}

func init() {
	rootCmd.AddCommand(ccusageCmd)
	ccusageCmd.Flags().BoolVarP(&ccusageDaily, "daily", "d", false, "Show daily breakdown")
	ccusageCmd.Flags().BoolVarP(&ccusageWeekly, "weekly", "w", false, "Show weekly breakdown")
	ccusageCmd.Flags().BoolVarP(&ccusageMonthly, "monthly", "m", false, "Show monthly breakdown")
	ccusageCmd.Flags().BoolVarP(&ccusageAll, "all", "a", false, "Show all breakdowns")
	ccusageCmd.Flags().StringVarP(&ccusageFormat, "format", "f", "text", "Output format (text, json)")
}

func runCcusage(cmd *cobra.Command, args []string) error {
	// Determine granularity
	granularity := "daily"
	if ccusageWeekly {
		granularity = "weekly"
	} else if ccusageMonthly {
		granularity = "monthly"
	} else if ccusageAll {
		// Show all three
		return runCcusageAll()
	}

	// Check if ccusage is available
	ccusageCmd, err := buildCcusageCommand()
	if err != nil {
		return err
	}

	// Build command args
	cmdArgs := []string{granularity, "--json", "--since", "20250101"}
	ccusageCmd.Args = append(ccusageCmd.Args, cmdArgs...)

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: %s %s\n", ccusageCmd.Path, strings.Join(cmdArgs, " "))
	}

	output, err := ccusageCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ccusage execution failed: %w\n%s", err, string(output))
	}

	// Parse and display
	periods, err := parseCcusageJSON(string(output), granularity)
	if err != nil {
		return fmt.Errorf("failed to parse ccusage output: %w", err)
	}

	if ccusageFormat == "json" {
		data, _ := json.MarshalIndent(periods, "", "  ")
		fmt.Println(string(data))
		return nil
	}

	// Display text format
	fmt.Printf("📊 Claude Code Usage (%s)\n", strings.Title(granularity))
	fmt.Println(strings.Repeat("-", 50))

	var totalInput, totalOutput, totalTokens uint64
	var totalCost float64

	for _, p := range periods {
		fmt.Printf("%s: %d in / %d out / %d total = $%.2f\n",
			p.Key, p.InputTokens, p.OutputTokens, p.TotalTokens, p.TotalCost)
		totalInput += p.InputTokens
		totalOutput += p.OutputTokens
		totalTokens += p.TotalTokens
		totalCost += p.TotalCost
	}

	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("TOTAL: %d in / %d out / %d tokens = $%.2f\n",
		totalInput, totalOutput, totalTokens, totalCost)

	return nil
}

func runCcusageAll() error {
	granularities := []string{"daily", "weekly", "monthly"}

	for _, g := range granularities {
		ccusageCmd, err := buildCcusageCommand()
		if err != nil {
			return err
		}

		cmdArgs := []string{g, "--json", "--since", "20250101"}
		ccusageCmd.Args = append(ccusageCmd.Args, cmdArgs...)

		output, err := ccusageCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("ccusage %s failed: %w", g, err)
		}

		periods, err := parseCcusageJSON(string(output), g)
		if err != nil {
			return err
		}

		fmt.Printf("\n📊 %s\n", strings.Title(g))
		fmt.Println(strings.Repeat("-", 40))

		var totalTokens uint64
		var totalCost float64
		for _, p := range periods {
			totalTokens += p.TotalTokens
			totalCost += p.TotalCost
		}
		fmt.Printf("%d periods: %d tokens total = $%.2f\n",
			len(periods), totalTokens, totalCost)
	}

	return nil
}

// CcusagePeriod represents usage data for a time period.
type CcusagePeriod struct {
	Key                 string  `json:"key"`
	InputTokens         uint64  `json:"inputTokens"`
	OutputTokens        uint64  `json:"outputTokens"`
	CacheCreationTokens uint64  `json:"cacheCreationTokens"`
	CacheReadTokens     uint64  `json:"cacheReadTokens"`
	TotalTokens         uint64  `json:"totalTokens"`
	TotalCost           float64 `json:"totalCost"`
}

// buildCcusageCommand creates a command to run ccusage (direct or via npx).
func buildCcusageCommand() (*exec.Cmd, error) {
	// Try direct ccusage first
	if _, err := exec.LookPath("ccusage"); err == nil {
		return exec.Command("ccusage"), nil
	}

	// Try npx ccusage
	npxCheck := exec.Command("npx", "ccusage", "--help")
	npxCheck.Stdout = nil
	npxCheck.Stderr = nil
	if err := npxCheck.Run(); err == nil {
		return exec.Command("npx", "ccusage"), nil
	}

	return nil, fmt.Errorf("ccusage not found. Install with: npm i -g ccusage")
}

// parseCcusageJSON parses ccusage JSON output.
func parseCcusageJSON(jsonStr, granularity string) ([]CcusagePeriod, error) {
	switch granularity {
	case "daily":
		var resp struct {
			Daily []struct {
				Date string `json:"date"`
				CcusagePeriod
			} `json:"daily"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
			return nil, err
		}
		periods := make([]CcusagePeriod, len(resp.Daily))
		for i, d := range resp.Daily {
			p := d.CcusagePeriod
			p.Key = d.Date
			periods[i] = p
		}
		return periods, nil

	case "weekly":
		var resp struct {
			Weekly []struct {
				Week string `json:"week"`
				CcusagePeriod
			} `json:"weekly"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
			return nil, err
		}
		periods := make([]CcusagePeriod, len(resp.Weekly))
		for i, w := range resp.Weekly {
			p := w.CcusagePeriod
			p.Key = w.Week
			periods[i] = p
		}
		return periods, nil

	case "monthly":
		var resp struct {
			Monthly []struct {
				Month string `json:"month"`
				CcusagePeriod
			} `json:"monthly"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
			return nil, err
		}
		periods := make([]CcusagePeriod, len(resp.Monthly))
		for i, m := range resp.Monthly {
			p := m.CcusagePeriod
			p.Key = m.Month
			periods[i] = p
		}
		return periods, nil
	}

	return nil, fmt.Errorf("unknown granularity: %s", granularity)
}
