package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/Patel230/tokman/internal/config"
	"github.com/Patel230/tokman/internal/filter"
	"github.com/Patel230/tokman/internal/tracking"
)

// Git status codes from RTK
const (
	GitStaged    = "staged"
	GitModified  = "modified"
	GitUntracked = "untracked"
	GitDeleted   = "deleted"
	GitConflicted = "conflicted"
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git command wrappers with output filtering",
	Long: `Wrap git commands with intelligent output filtering to reduce
token usage while preserving important information.`,
}

var gitStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show working tree status (filtered)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitStatus()
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(output)

		// Record to tracker
		if err := recordCommand("git status", output, output, execTime, true); err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", err)
		}
	},
}

var gitDiffCmd = &cobra.Command{
	Use:   "diff [args...]",
	Short: "Show changes (filtered)",
	Long: `Show git diff with output filtering:
- Stats summary first
- Diff hunks limited to 30 lines each
- ANSI colors stripped`,
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitDiff(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(output)

		// Record to tracker
		if err := recordCommand("git diff", output, output, execTime, true); err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", err)
		}
	},
}

var gitLogCmd = &cobra.Command{
	Use:   "log [args...]",
	Short: "Show commit logs (filtered)",
	Long: `Show git log with output filtering:
- Default: oneline format
- Commit count limited to 20
- Full output only with --verbose flag`,
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitLog(args, verbose)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(output)

		// Record to tracker
		if err := recordCommand("git log", output, output, execTime, true); err != nil && verbose {
			fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", err)
		}
	},
}

// GitStatus represents parsed git status output
type GitStatus struct {
	Branch     string
	Ahead      int
	Behind     int
	Staged     []GitFile
	Modified   []GitFile
	Untracked  []GitFile
	Conflicted []GitFile
}

// GitFile represents a file in git status
type GitFile struct {
	Path string
	Code string
}

func init() {
	rootCmd.AddCommand(gitCmd)
	gitCmd.AddCommand(gitStatusCmd)
	gitCmd.AddCommand(gitDiffCmd)
	gitCmd.AddCommand(gitLogCmd)
}

// runGitStatus executes git status with porcelain parsing
func runGitStatus() (string, error) {
	// Get porcelain output for parsing
	porcelainCmd := exec.Command("git", "status", "--porcelain", "-b")
	var porcelainOut bytes.Buffer
	porcelainCmd.Stdout = &porcelainOut
	if err := porcelainCmd.Run(); err != nil {
		return "", fmt.Errorf("git status failed: %w", err)
	}

	// Parse porcelain output
	status := parsePorcelain(porcelainOut.String())

	// Format output
	return formatStatus(status), nil
}

// parsePorcelain parses git status --porcelain -b output
func parsePorcelain(output string) *GitStatus {
	status := &GitStatus{}
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for i, line := range lines {
		if i == 0 {
			// Branch line: "## main...origin/main [ahead 2, behind 1]"
			status.Branch, status.Ahead, status.Behind = parseBranchLine(line)
			continue
		}

		if len(line) < 2 {
			continue
		}

		code := line[:2]
		filePath := strings.TrimSpace(line[3:])
		
		// Handle renamed files (old -> new)
		if strings.Contains(filePath, " -> ") {
			parts := strings.Split(filePath, " -> ")
			if len(parts) == 2 {
				filePath = parts[1] // Show new name
			}
		}

		gitFile := GitFile{Path: filePath, Code: code}

		switch {
		case strings.Contains(code, "U"):
			// Unmerged/conflicted
			status.Conflicted = append(status.Conflicted, gitFile)
		case code == "??":
			// Untracked
			status.Untracked = append(status.Untracked, gitFile)
		case code == "!!":
			// Ignored - skip
		case code[0] != ' ' && code[0] != '?':
			// Staged changes
			status.Staged = append(status.Staged, gitFile)
			if code[1] != ' ' {
				// Also modified in working tree
				status.Modified = append(status.Modified, gitFile)
			}
		case code[1] != ' ':
			// Modified in working tree (not staged)
			status.Modified = append(status.Modified, gitFile)
		}
	}

	return status
}

// parseBranchLine extracts branch name and ahead/behind counts
func parseBranchLine(line string) (branch string, ahead, behind int) {
	// Remove "## " prefix
	line = strings.TrimPrefix(line, "## ")

	// Check for detached HEAD
	if strings.HasPrefix(line, "HEAD detached") {
		return "HEAD detached", 0, 0
	}

	// Check for initial branch (no tracking)
	if !strings.Contains(line, "...") {
		// Check for ahead/behind without tracking
		if strings.Contains(line, "[") {
			parts := strings.SplitN(line, "[", 2)
			branch = strings.TrimSpace(parts[0])
			ahead, behind = parseAheadBehind(parts[1])
		} else {
			branch = strings.TrimSpace(line)
		}
		return branch, ahead, behind
	}

	// Normal branch with tracking: "main...origin/main [ahead 2, behind 1]"
	parts := strings.SplitN(line, "...", 2)
	branch = strings.Fields(parts[0])[0] // Handle "main (no branch)"

	if len(parts) > 1 && strings.Contains(parts[1], "[") {
		// Extract ahead/behind
		abParts := strings.SplitN(parts[1], "[", 2)
		ahead, behind = parseAheadBehind(abParts[1])
	}

	return branch, ahead, behind
}

// parseAheadBehind parses "ahead 2, behind 1]" format
func parseAheadBehind(s string) (ahead, behind int) {
	// Remove trailing ]
	s = strings.TrimSuffix(s, "]")

	// Match ahead N
	aheadRe := regexp.MustCompile(`ahead (\d+)`)
	if matches := aheadRe.FindStringSubmatch(s); len(matches) > 1 {
		ahead, _ = strconv.Atoi(matches[1])
	}

	// Match behind N
	behindRe := regexp.MustCompile(`behind (\d+)`)
	if matches := behindRe.FindStringSubmatch(s); len(matches) > 1 {
		behind, _ = strconv.Atoi(matches[1])
	}

	return ahead, behind
}

// formatStatus formats git status for output
func formatStatus(status *GitStatus) string {
	var buf strings.Builder

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	// Branch line
	buf.WriteString(bold("📌 ") + cyan(status.Branch))
	if status.Ahead > 0 || status.Behind > 0 {
		buf.WriteString(" [")
		if status.Ahead > 0 {
			buf.WriteString(fmt.Sprintf("ahead %d", status.Ahead))
		}
		if status.Behind > 0 {
			if status.Ahead > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("behind %d", status.Behind))
		}
		buf.WriteString("]")
	}
	buf.WriteString("\n")

	// Staged files
	if len(status.Staged) > 0 {
		buf.WriteString(green("\n✅ Staged:\n"))
		for _, f := range status.Staged {
			statusChar := getStatusCode(f.Code[0])
			buf.WriteString(fmt.Sprintf("   %s %s\n", statusChar, f.Path))
		}
	}

	// Modified files
	if len(status.Modified) > 0 {
		buf.WriteString(yellow("\n📝 Modified:\n"))
		for _, f := range status.Modified {
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	// Untracked files
	if len(status.Untracked) > 0 {
		buf.WriteString(red("\n❓ Untracked:\n"))
		// Limit display
		maxShow := 10
		for i, f := range status.Untracked {
			if i >= maxShow {
				buf.WriteString(fmt.Sprintf("   ... and %d more\n", len(status.Untracked)-maxShow))
				break
			}
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	// Conflicted files
	if len(status.Conflicted) > 0 {
		buf.WriteString(red("\n⚠️  Conflicted:\n"))
		for _, f := range status.Conflicted {
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	// Clean state
	if len(status.Staged) == 0 && len(status.Modified) == 0 &&
		len(status.Untracked) == 0 && len(status.Conflicted) == 0 {
		buf.WriteString(green("\n✓ Clean working tree\n"))
	}

	return buf.String()
}

// getStatusCode returns a status character for staged changes
func getStatusCode(code byte) string {
	switch code {
	case 'M':
		return "M" // Modified
	case 'A':
		return "A" // Added
	case 'D':
		return "D" // Deleted
	case 'R':
		return "R" // Renamed
	case 'C':
		return "C" // Copied
	default:
		return string(code)
	}
}

// runGitDiff executes git diff with filtering
func runGitDiff(args []string) (string, error) {
	// Run git diff --stat first for summary
	statArgs := append([]string{"diff", "--stat"}, args...)
	statCmd := exec.Command("git", statArgs...)
	var statOut bytes.Buffer
	statCmd.Stdout = &statOut
	statCmd.Run() // Ignore error, diff may have no changes

	// Run git diff for content
	diffArgs := append([]string{"diff"}, args...)
	diffCmd := exec.Command("git", diffArgs...)
	var diffOut bytes.Buffer
	diffCmd.Stdout = &diffOut
	if err := diffCmd.Run(); err != nil {
		return "", fmt.Errorf("git diff failed: %w", err)
	}

	// Filter the diff output
	filtered := filterDiff(diffOut.String())

	// Combine stat and filtered diff
	if statOut.Len() > 0 {
		return statOut.String() + "\n" + filtered, nil
	}
	return filtered, nil
}

// filterDiff filters diff hunks to max 30 lines (from RTK)
func filterDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var result []string
	hunkLineCount := 0
	inHunk := false
	maxHunkLines := 30

	for _, line := range lines {
		// Check for hunk header
		if strings.HasPrefix(line, "@@") {
			inHunk = true
			hunkLineCount = 0
			result = append(result, line)
			continue
		}

		// Check for file header
		if strings.HasPrefix(line, "diff --git") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "--- ") ||
			strings.HasPrefix(line, "+++ ") {
			inHunk = false
			result = append(result, line)
			continue
		}

		// Filter hunk content
		if inHunk {
			hunkLineCount++
			if hunkLineCount <= maxHunkLines {
				result = append(result, line)
			} else if hunkLineCount == maxHunkLines+1 {
				result = append(result, gray("... (hunk truncated)"))
			}
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// gray returns a gray-colored string
func gray(s string) string {
	dim := color.New(color.FgHiBlack).SprintFunc()
	return dim(s)
}

// runGitLog executes git log with filtering
func runGitLog(args []string, fullOutput bool) (string, error) {
	// Default args if none provided
	logArgs := args
	if len(logArgs) == 0 && !fullOutput {
		// Use oneline format with count limit
		logArgs = []string{"--oneline", "-n", "20"}
	} else if len(logArgs) == 0 {
		logArgs = []string{"-n", "20"}
	}

	// Prepend "log" to args
	cmdArgs := append([]string{"log"}, logArgs...)
	cmd := exec.Command("git", cmdArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git log failed: %w", err)
	}

	// Filter output
	output := out.String()
	if !fullOutput {
		output = filterLog(output)
	}

	return output, nil
}

// filterLog filters git log output
func filterLog(output string) string {
	// Strip ANSI codes
	output = filter.StripANSI(output)

	// Limit to reasonable size
	lines := strings.Split(output, "\n")
	if len(lines) > 50 {
		lines = lines[:50]
		lines = append(lines, gray("... (log truncated, use --verbose for full output)"))
	}

	return strings.Join(lines, "\n")
}

// recordCommand records a command execution to the tracker
func recordCommand(command, originalOutput, filteredOutput string, execTimeMs int64, success bool) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return err
	}

	if !cfg.Tracking.Enabled {
		return nil
	}

	tracker, err := tracking.NewTracker(cfg.GetDatabasePath())
	if err != nil {
		return err
	}
	defer tracker.Close()

	originalTokens := tracking.EstimateTokens(originalOutput)
	filteredTokens := tracking.EstimateTokens(filteredOutput)
	savedTokens := 0
	if originalTokens > filteredTokens {
		savedTokens = originalTokens - filteredTokens
	}

	record := &tracking.CommandRecord{
		Command:        command,
		OriginalOutput: originalOutput,
		FilteredOutput: filteredOutput,
		OriginalTokens: originalTokens,
		FilteredTokens: filteredTokens,
		SavedTokens:    savedTokens,
		ProjectPath:    config.ProjectPath(),
		ExecTimeMs:     execTimeMs,
		Timestamp:      time.Now(),
		ParseSuccess:   success,
	}

	return tracker.Record(record)
}
