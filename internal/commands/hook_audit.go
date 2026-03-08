package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var hookAuditCmd = &cobra.Command{
	Use:   "hook-audit",
	Short: "Show hook rewrite audit metrics",
	Long: `Display metrics from the hook audit log.

Shows statistics about hook invocations including:
- Total invocations
- Rewrites vs skips
- Skip reasons breakdown
- Top rewritten commands

Requires TOKMAN_HOOK_AUDIT=1 environment variable to enable logging.`,
	Run: func(cmd *cobra.Command, args []string) {
		since, _ := cmd.Flags().GetInt("since")
		runHookAudit(since, verbose)
	},
}

func init() {
	rootCmd.AddCommand(hookAuditCmd)

	hookAuditCmd.Flags().IntP("since", "s", 7, "Show entries from last N days (0 = all time)")
}

// AuditEntry represents a single audit log entry
type AuditEntry struct {
	Timestamp    string
	Action       string
	OriginalCmd  string
	RewrittenCmd string
}

func runHookAudit(sinceDays int, verbose int) {
	logPath := getAuditLogPath()

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		fmt.Printf("No audit log found at %s\n", logPath)
		fmt.Println("Enable audit mode: export TOKMAN_HOOK_AUDIT=1 in your shell, then use TokMan.")
		return
	}

	file, err := os.Open(logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading audit log: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// Parse log entries
	var entries []AuditEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if entry := parseAuditLine(line); entry != nil {
			entries = append(entries, *entry)
		}
	}

	if len(entries) == 0 {
		fmt.Println("Audit log is empty.")
		return
	}

	// Filter by time
	filtered := filterEntriesByDays(entries, sinceDays)

	if len(filtered) == 0 {
		fmt.Printf("No entries in the last %d days.\n", sinceDays)
		return
	}

	// Count by action
	actionCounts := make(map[string]int)
	cmdCounts := make(map[string]int)

	for _, entry := range filtered {
		actionCounts[entry.Action]++
		if entry.Action == "rewrite" {
			base := baseCommand(entry.OriginalCmd)
			cmdCounts[base]++
		}
	}

	// Calculate statistics
	total := len(filtered)
	rewrites := actionCounts["rewrite"]
	skips := total - rewrites
	rewritePct := float64(rewrites) / float64(total) * 100
	skipPct := float64(skips) / float64(total) * 100

	// Period label
	period := fmt.Sprintf("last %d days", sinceDays)
	if sinceDays == 0 {
		period = "all time"
	}

	// Output
	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Printf("Hook Audit (%s)\n", period)
	fmt.Println(strings.Repeat("─", 30))
	fmt.Printf("Total invocations: %d\n", total)
	fmt.Printf("Rewrites:          %s (%.1f%%)\n", green(rewrites), rewritePct)
	fmt.Printf("Skips:             %d (%.1f%%)\n", skips, skipPct)

	// Skip breakdown
	var skipActions []struct {
		action string
		count  int
	}
	for action, count := range actionCounts {
		if strings.HasPrefix(action, "skip:") {
			skipActions = append(skipActions, struct {
				action string
				count  int
			}{action, count})
		}
	}

	if len(skipActions) > 0 {
		sort.Slice(skipActions, func(i, j int) bool {
			return skipActions[i].count > skipActions[j].count
		})
		for _, sa := range skipActions {
			reason := strings.TrimPrefix(sa.action, "skip:")
			padding := strings.Repeat(" ", 14-len(reason))
			if len(reason) > 13 {
				padding = " "
			}
			fmt.Printf("  %s:%s%d\n", reason, padding, sa.count)
		}
	}

	// Top commands
	if len(cmdCounts) > 0 {
		var sorted []struct {
			cmd   string
			count int
		}
		for cmd, count := range cmdCounts {
			sorted = append(sorted, struct {
				cmd   string
				count int
			}{cmd, count})
		}
		sort.Slice(sorted, func(i, j int) bool {
			return sorted[i].count > sorted[j].count
		})

		var top []string
		for i := 0; i < 5 && i < len(sorted); i++ {
			top = append(top, fmt.Sprintf("%s (%d)", sorted[i].cmd, sorted[i].count))
		}
		fmt.Printf("Top commands: %s\n", cyan(strings.Join(top, ", ")))
	}

	if verbose > 0 {
		fmt.Printf("\nLog: %s\n", logPath)
	}
}

func getAuditLogPath() string {
	if dir := os.Getenv("TOKMAN_AUDIT_DIR"); dir != "" {
		return filepath.Join(dir, "hook-audit.log")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "tokman", "hook-audit.log")
}

func parseAuditLine(line string) *AuditEntry {
	parts := strings.SplitN(line, " | ", 4)
	if len(parts) < 3 {
		return nil
	}
	return &AuditEntry{
		Timestamp:    parts[0],
		Action:       parts[1],
		OriginalCmd:  parts[2],
		RewrittenCmd: func() string { if len(parts) > 3 { return parts[3] }; return "-" }(),
	}
}

func baseCommand(cmd string) string {
	// Strip env var prefixes
	words := strings.Fields(cmd)
	var stripped []string
	for _, w := range words {
		if strings.Contains(w, "=") {
			continue
		}
		stripped = append(stripped, w)
		if len(stripped) == 2 {
			break
		}
	}
	if len(stripped) == 0 {
		return cmd
	}
	return strings.Join(stripped, " ")
}

func filterEntriesByDays(entries []AuditEntry, days int) []AuditEntry {
	if days == 0 {
		return entries
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	cutoffStr := cutoff.Format("2006-01-02T15:04:05Z")

	var filtered []AuditEntry
	for _, e := range entries {
		if e.Timestamp >= cutoffStr {
			filtered = append(filtered, e)
		}
	}
	return filtered
}
