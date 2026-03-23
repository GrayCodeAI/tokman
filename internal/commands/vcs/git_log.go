package vcs

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var (
	gitLogCount int
)

var gitLogCmd = &cobra.Command{
	Use:   "log [args...]",
	Short: "Show commit logs (filtered)",
	Long: `Show git log with output filtering:
- Default: oneline format
- Commit count limited to 20
- Full output only with --verbose flag`,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	Run: func(cmd *cobra.Command, args []string) {
		gitArgs := extractGitArgs(args)

		if gitLogCount > 0 {
			gitArgs = append([]string{fmt.Sprintf("-n%d", gitLogCount)}, gitArgs...)
		}

		shared.ExecuteAndRecord("git log", func() (string, string, error) {
			return runGitLog(gitArgs, shared.Verbose > 0)
		})
	},
}

var gitShowCmd = &cobra.Command{
	Use:   "show [args...]",
	Short: "Show commit or object (filtered)",
	Long: `Show git show with output filtering:
- Compact summary first (hash + subject)
- Diff limited to 30 lines per hunk`,
	Run: func(cmd *cobra.Command, args []string) {
		shared.ExecuteAndRecord("git show", func() (string, string, error) {
			return runGitShow(args, shared.Verbose)
		})
	},
}

func runGitLog(args []string, fullOutput bool) (string, string, error) {
	hasCount := false
	for _, arg := range args {
		if arg == "-n" || strings.HasPrefix(arg, "-n") || strings.HasPrefix(arg, "--count") {
			hasCount = true
			break
		}
	}

	hasFormat := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "--format") || strings.HasPrefix(arg, "--pretty") {
			hasFormat = true
			break
		}
	}

	// Build args for raw (full) git log - use verbose format for tracking
	rawLogArgs := []string{"--format=commit %H%nAuthor: %an <%ae>%nDate:   %ad%n%n    %s%n%b---END---", "--date=short"}
	rawLogArgs = append(rawLogArgs, args...)
	if !hasCount {
		rawLogArgs = append(rawLogArgs, "-n", "20")
	}

	// Capture raw verbose output
	rawCmd := buildGitCmd("log", rawLogArgs...)
	var rawOut bytes.Buffer
	rawCmd.Stdout = &rawOut
	rawCmd.Run()
	raw := rawOut.String()

	// Build args for filtered (compact) output
	var logArgs []string
	if !hasFormat && !fullOutput {
		logArgs = []string{"--format=%h %s (%ar) <%an>"}
	}
	logArgs = append(logArgs, args...)
	if !hasCount {
		logArgs = append(logArgs, "-n", "20")
	}

	filterCmd := buildGitCmd("log", logArgs...)
	var filterOut bytes.Buffer
	filterCmd.Stdout = &filterOut
	if err := filterCmd.Run(); err != nil {
		return raw, "", fmt.Errorf("git log failed: %w", err)
	}

	filtered := filterOut.String()
	if !fullOutput {
		filtered = filterLog(filtered)
	}

	// If raw is empty (no commits), use filtered as raw
	if strings.TrimSpace(raw) == "" {
		raw = filtered
	}

	return raw, filtered, nil
}

func filterLog(output string) string {
	output = filter.StripANSI(output)

	lines := strings.Split(output, "\n")
	if len(lines) > 50 {
		lines = lines[:50]
		lines = append(lines, gray("... (log truncated, use --verbose for full output)"))
	}

	// Truncate each line to 80 chars
	for i, line := range lines {
		runes := []rune(line)
		if len(runes) > 80 {
			lines[i] = string(runes[:77]) + "..."
		}
	}

	return strings.Join(lines, "\n")
}

// filterLogVerbose processes verbose git log output (with commit bodies).
// Splits on ---END--- markers, keeps first body line, strips trailers.
func filterLogVerbose(output string, maxCommits int) string {
	output = filter.StripANSI(output)

	commits := strings.Split(output, "---END---")
	var result []string

	for i, block := range commits {
		if i >= maxCommits {
			break
		}
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}

		blockLines := strings.Split(block, "\n")
		header := truncateLogLine(strings.TrimSpace(blockLines[0]), 80)

		// Find first non-empty body line, skip trailers
		var bodyLine string
		for _, l := range blockLines[1:] {
			l = strings.TrimSpace(l)
			if l == "" || strings.HasPrefix(l, "Signed-off-by:") || strings.HasPrefix(l, "Co-authored-by:") {
				continue
			}
			bodyLine = truncateLogLine(l, 80)
			break
		}

		if bodyLine != "" {
			result = append(result, header+"\n  "+bodyLine)
		} else {
			result = append(result, header)
		}
	}

	return strings.Join(result, "\n")
}

func truncateLogLine(line string, width int) string {
	runes := []rune(line)
	if len(runes) > width {
		return string(runes[:width-3]) + "..."
	}
	return line
}

func runGitShow(args []string, verboseLevel int) (string, string, error) {
	for _, arg := range args {
		if strings.Contains(arg, ":") && !strings.HasPrefix(arg, "-") {
			showCmd := buildGitCmd("show", args...)
			out, err := showCmd.Output()
			if err != nil {
				return "", "", fmt.Errorf("git show failed: %w", err)
			}
			raw := string(out)
			return raw, raw, nil
		}
	}

	// Capture full raw git show output for tracking
	rawCmd := buildGitCmd("show", args...)
	var rawOut bytes.Buffer
	rawCmd.Stdout = &rawOut
	rawCmd.Run()
	raw := rawOut.String()

	// Build filtered output
	summaryArgs := []string{"--no-patch", "--pretty=format:%h %s (%ar) <%an>"}
	summaryArgs = append(summaryArgs, args...)
	summaryCmd := buildGitCmd("show", summaryArgs...)
	var summaryOut bytes.Buffer
	summaryCmd.Stdout = &summaryOut
	summaryCmd.Run()

	statArgs := []string{"--stat", "--pretty=format:"}
	statArgs = append(statArgs, args...)
	statCmd := buildGitCmd("show", statArgs...)
	var statOut bytes.Buffer
	statCmd.Stdout = &statOut
	statCmd.Run()

	diffArgs := []string{"--pretty=format:"}
	diffArgs = append(diffArgs, args...)
	diffCmd := buildGitCmd("show", diffArgs...)
	var diffOut bytes.Buffer
	diffCmd.Stdout = &diffOut
	diffCmd.Run()

	var result strings.Builder
	result.WriteString(summaryOut.String())
	result.WriteString("\n")
	if statOut.Len() > 0 {
		result.WriteString(statOut.String())
		result.WriteString("\n")
	}
	if diffOut.Len() > 0 {
		result.WriteString("\n--- Changes ---\n")
		result.WriteString(filterDiff(diffOut.String()))
	}

	return raw, result.String(), nil
}
