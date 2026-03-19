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

	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

// Git status codes
const (
	GitStaged     = "staged"
	GitModified   = "modified"
	GitUntracked  = "untracked"
	GitDeleted    = "deleted"
	GitConflicted = "conflicted"
)

// Global git flags (persisted to all subcommands)
var (
	gitDir          string
	gitWorkTree     string
	gitDirectory    string // -C flag
	gitNoPager      bool
	gitNoOptLocks   bool
	gitBare         bool
	gitLiteralPaths bool
	gitConfigOpts   []string // -c key=value options
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git command wrappers with output filtering",
	Long: `Wrap git commands with intelligent output filtering to reduce
token usage while preserving important information.

Global flags (applied before subcommand):
  -C, --directory <path>      Run git in specified directory
  --git-dir <path>            Set the .git directory path
  --work-tree <path>          Set the working tree path
  --no-pager                  Disable pager
  --no-optional-locks         Skip optional locks
  --bare                      Treat repository as bare
  --literal-pathspecs         Treat pathspecs literally
  -c <key=value>              Set git config option`,
	TraverseChildren:      true, // Allow flags between 'git' and subcommand
	FParseErrWhitelist:    cobra.FParseErrWhitelist{UnknownFlags: true},
}

// buildGitCmd creates a git command with global options prepended
func buildGitCmd(subCmd string, args ...string) *exec.Cmd {
	var cmdArgs []string

	// Prepend global flags (must come BEFORE subcommand)
	if gitNoPager {
		cmdArgs = append(cmdArgs, "--no-pager")
	}
	if gitDirectory != "" {
		cmdArgs = append(cmdArgs, "-C", gitDirectory)
	}
	if gitDir != "" {
		cmdArgs = append(cmdArgs, "--git-dir", gitDir)
	}
	if gitWorkTree != "" {
		cmdArgs = append(cmdArgs, "--work-tree", gitWorkTree)
	}
	if gitNoOptLocks {
		cmdArgs = append(cmdArgs, "--no-optional-locks")
	}
	if gitBare {
		cmdArgs = append(cmdArgs, "--bare")
	}
	if gitLiteralPaths {
		cmdArgs = append(cmdArgs, "--literal-pathspecs")
	}
	for _, cfg := range gitConfigOpts {
		cmdArgs = append(cmdArgs, "-c", cfg)
	}

	// Add subcommand and args
	cmdArgs = append(cmdArgs, subCmd)
	cmdArgs = append(cmdArgs, args...)

	return exec.Command("git", cmdArgs...)
}

// extractGitArgs extracts git-specific args from the args list
// (filters out tokman-specific flags)
func extractGitArgs(args []string) []string {
	var gitArgs []string
	skipNext := false
	for _, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		// Skip tokman-specific flags that take values
		if arg == "--query" {
			skipNext = true
			continue
		}
		// Skip tokman-specific boolean flags
		if strings.HasPrefix(arg, "--ultra-compact") ||
			arg == "-u" ||
			strings.HasPrefix(arg, "--verbose") ||
			arg == "-v" ||
			strings.HasPrefix(arg, "-vv") ||
			strings.HasPrefix(arg, "-vvv") ||
			arg == "--dry-run" ||
			arg == "--llm" ||
			arg == "--skip-env" {
			continue
		}
		// Skip tokman flags with values
		if strings.HasPrefix(arg, "--query=") ||
			strings.HasPrefix(arg, "--config=") ||
			strings.HasPrefix(arg, "-c=") {
			continue
		}
		gitArgs = append(gitArgs, arg)
	}
	return gitArgs
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
		if err := recordCommand("git status", output, output, execTime, true); err != nil && verbose > 0 {
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
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
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
		if err := recordCommand("git diff", output, output, execTime, true); err != nil && verbose > 0 {
			fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", err)
		}
	},
}

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
		// Extract non-flag args (everything that's not a tokman flag)
		gitArgs := extractGitArgs(args)
		
		// If -n flag was set via Cobra, add it to gitArgs
		if gitLogCount > 0 {
			gitArgs = append([]string{fmt.Sprintf("-n%d", gitLogCount)}, gitArgs...)
		}
		
		startTime := time.Now()
		output, err := runGitLog(gitArgs, verbose > 0)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(output)

		// Record to tracker
		if err := recordCommand("git log", output, output, execTime, true); err != nil && verbose > 0 {
			fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", err)
		}
	},
}

// GitStatus represents parsed git status output
type GitStatus struct {
	Branch         string
	TrackingBranch string // e.g., "origin/main"
	Ahead          int
	Behind         int
	Staged         []GitFile
	Modified       []GitFile
	Untracked      []GitFile
	Conflicted     []GitFile
}

// GitFile represents a file in git status
type GitFile struct {
	Path string
	Code string
}

func init() {
	rootCmd.AddCommand(gitCmd)

	// Global flags for git command
	gitCmd.PersistentFlags().StringVarP(&gitDirectory, "directory", "C", "", "Run git in specified directory")
	gitCmd.PersistentFlags().StringVar(&gitDir, "git-dir", "", "Set the .git directory path")
	gitCmd.PersistentFlags().StringVar(&gitWorkTree, "work-tree", "", "Set the working tree path")
	gitCmd.PersistentFlags().BoolVar(&gitNoPager, "no-pager", false, "Disable pager")
	gitCmd.PersistentFlags().BoolVar(&gitNoOptLocks, "no-optional-locks", false, "Skip optional locks")
	gitCmd.PersistentFlags().BoolVar(&gitBare, "bare", false, "Treat repository as bare")
	gitCmd.PersistentFlags().BoolVar(&gitLiteralPaths, "literal-pathspecs", false, "Treat pathspecs literally")
	gitCmd.PersistentFlags().StringArrayVarP(&gitConfigOpts, "config", "c", nil, "Set git config option (key=value)")

	// Add subcommands
	gitCmd.AddCommand(gitStatusCmd)
	gitCmd.AddCommand(gitDiffCmd)
	gitCmd.AddCommand(gitLogCmd)
	gitCmd.AddCommand(gitShowCmd)
	gitCmd.AddCommand(gitAddCmd)
	gitCmd.AddCommand(gitCommitCmd)
	gitCmd.AddCommand(gitPushCmd)
	gitCmd.AddCommand(gitPullCmd)
	gitCmd.AddCommand(gitBranchCmd)
	gitCmd.AddCommand(gitFetchCmd)
	gitCmd.AddCommand(gitStashCmd)
	gitCmd.AddCommand(gitWorktreeCmd)

	// Git log specific flags
	gitLogCmd.Flags().IntVarP(&gitLogCount, "number", "n", 0, "Number of commits to show")
}

// runGitStatus executes git status with porcelain parsing
func runGitStatus() (string, error) {
	// Get porcelain output for parsing
	porcelainCmd := buildGitCmd("status", "--porcelain", "-b")
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
			status.Branch, status.TrackingBranch, status.Ahead, status.Behind = parseBranchLine(line)
			continue
		}

		if len(line) < 4 {
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

// parseBranchLine extracts branch name, tracking branch, and ahead/behind counts
func parseBranchLine(line string) (branch, tracking string, ahead, behind int) {
	// Remove "## " prefix
	line = strings.TrimPrefix(line, "## ")

	// Check for detached HEAD
	if strings.HasPrefix(line, "HEAD detached") {
		return "HEAD detached", "", 0, 0
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
		return branch, "", ahead, behind
	}

	// Normal branch with tracking: "main...origin/main [ahead 2, behind 1]"
	parts := strings.SplitN(line, "...", 2)
	branch = strings.Fields(parts[0])[0] // Handle "main (no branch)"

	if len(parts) > 1 {
		// Extract tracking branch (remove [ahead/behind] if present)
		trackingPart := parts[1]
		if strings.Contains(trackingPart, "[") {
			abParts := strings.SplitN(trackingPart, "[", 2)
			tracking = strings.TrimSpace(abParts[0])
			ahead, behind = parseAheadBehind(abParts[1])
		} else {
			tracking = strings.TrimSpace(trackingPart)
		}
	}

	return branch, tracking, ahead, behind
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
	// Ultra-compact mode: ASCII-only, inline format
	if ultraCompact {
		return formatStatusUltraCompact(status)
	}

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

// formatStatusUltraCompact returns ultra-compact ASCII-only output
func formatStatusUltraCompact(status *GitStatus) string {
	var buf strings.Builder

	// Branch line with tracking
	branch := status.Branch
	if status.TrackingBranch != "" {
		buf.WriteString(fmt.Sprintf("* %s...%s\n", branch, status.TrackingBranch))
	} else if status.Ahead > 0 || status.Behind > 0 {
		// Show tracking with ahead/behind
		buf.WriteString(fmt.Sprintf("* %s [a%d b%d]\n", branch, status.Ahead, status.Behind))
	} else {
		buf.WriteString(fmt.Sprintf("* %s\n", branch))
	}

	// Staged files: section header + max 3 files
	if len(status.Staged) > 0 {
		buf.WriteString(fmt.Sprintf("+ Staged: %d files\n", len(status.Staged)))
		for i, f := range status.Staged {
			if i >= 3 {
				buf.WriteString(fmt.Sprintf("   ... +%d more\n", len(status.Staged)-3))
				break
			}
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	// Modified files
	if len(status.Modified) > 0 {
		buf.WriteString(fmt.Sprintf("~ Modified: %d files\n", len(status.Modified)))
		for i, f := range status.Modified {
			if i >= 3 {
				buf.WriteString(fmt.Sprintf("   ... +%d more\n", len(status.Modified)-3))
				break
			}
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	// Untracked files
	if len(status.Untracked) > 0 {
		buf.WriteString(fmt.Sprintf("? Untracked: %d files\n", len(status.Untracked)))
		for i, f := range status.Untracked {
			if i >= 3 {
				buf.WriteString(fmt.Sprintf("   ... +%d more\n", len(status.Untracked)-3))
				break
			}
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	// Conflicted files
	if len(status.Conflicted) > 0 {
		buf.WriteString(fmt.Sprintf("! Conflicted: %d files\n", len(status.Conflicted)))
		for i, f := range status.Conflicted {
			if i >= 3 {
				buf.WriteString(fmt.Sprintf("   ... +%d more\n", len(status.Conflicted)-3))
				break
			}
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	// Clean state
	if len(status.Staged) == 0 && len(status.Modified) == 0 &&
		len(status.Untracked) == 0 && len(status.Conflicted) == 0 {
		buf.WriteString("clean\n")
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
	statArgs := append([]string{"--stat"}, args...)
	statCmd := buildGitCmd("diff", statArgs...)
	var statOut bytes.Buffer
	statCmd.Stdout = &statOut
	statCmd.Run() // Ignore error, diff may have no changes

	// Run git diff for content
	diffArgs := append([]string{}, args...)
	diffCmd := buildGitCmd("diff", diffArgs...)
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

// filterDiff filters diff hunks to max 30 lines
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
	// Check if user specified -n or --count
	hasCount := false
	for _, arg := range args {
		if arg == "-n" || strings.HasPrefix(arg, "-n") || strings.HasPrefix(arg, "--count") {
			hasCount = true
			break
		}
	}

	// Check if user specified custom format
	hasFormat := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "--format") || strings.HasPrefix(arg, "--pretty") {
			hasFormat = true
			break
		}
	}

	// Build log args with compact format by default
	var logArgs []string
	
	if !hasFormat && !fullOutput {
		// Use compact format with timestamp and author
		logArgs = []string{"--format=%h %s (%ar) <%an>"}
	}
	
	// Add user args
	logArgs = append(logArgs, args...)
	
	// Add default count if not specified
	if !hasCount && len(logArgs) > 0 {
		logArgs = append(logArgs, "-n", "20")
	}

	cmd := buildGitCmd("log", logArgs...)
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

// ============================================================================
// Additional Git Subcommands
// ============================================================================

var gitShowCmd = &cobra.Command{
	Use:   "show [args...]",
	Short: "Show commit or object (filtered)",
	Long: `Show git show with output filtering:
- Compact summary first (hash + subject)
- Diff limited to 30 lines per hunk`,
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitShow(args, verbose)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(output)
		recordCommand("git show", output, output, execTime, true)
	},
}

func runGitShow(args []string, verboseLevel int) (string, error) {
	// Check for blob show (rev:path) or stat-only
	for _, arg := range args {
		if strings.Contains(arg, ":") && !strings.HasPrefix(arg, "-") {
			// Blob show - pass through directly
			showCmd := buildGitCmd("show", args...)
			out, err := showCmd.Output()
			if err != nil {
				return "", fmt.Errorf("git show failed: %w", err)
			}
			return string(out), nil
		}
	}

	// Get summary
	summaryArgs := []string{"--no-patch", "--pretty=format:%h %s (%ar) <%an>"}
	summaryArgs = append(summaryArgs, args...)
	summaryCmd := buildGitCmd("show", summaryArgs...)
	var summaryOut bytes.Buffer
	summaryCmd.Stdout = &summaryOut
	summaryCmd.Run()

	// Get stat
	statArgs := []string{"--stat", "--pretty=format:"}
	statArgs = append(statArgs, args...)
	statCmd := buildGitCmd("show", statArgs...)
	var statOut bytes.Buffer
	statCmd.Stdout = &statOut
	statCmd.Run()

	// Get diff
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

	return result.String(), nil
}

var gitAddCmd = &cobra.Command{
	Use:   "add [args...]",
	Short: "Add files to staging (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitAdd(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(output)
		recordCommand("git add", output, output, execTime, true)
	},
}

func runGitAdd(args []string) (string, error) {
	addArgs := args
	if len(addArgs) == 0 {
		addArgs = []string{"."}
	}

	addCmd := buildGitCmd("add", addArgs...)
	output, err := addCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git add failed: %w\n%s", err, output)
	}

	// Check staged files
	statCmd := buildGitCmd("diff", "--cached", "--stat", "--shortstat")
	statOut, _ := statCmd.Output()
	stat := strings.TrimSpace(string(statOut))

	if stat == "" {
		return "ok (nothing to add)", nil
	}

	// Extract short stat
	lines := strings.Split(stat, "\n")
	lastLine := lines[len(lines)-1]
	return fmt.Sprintf("ok ✓ %s", lastLine), nil
}

var gitCommitCmd = &cobra.Command{
	Use:   "commit [args...]",
	Short: "Commit changes (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitCommit(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(output)
		recordCommand("git commit", output, output, execTime, true)
	},
}

func runGitCommit(args []string) (string, error) {
	commitCmd := buildGitCmd("commit", args...)
	output, err := commitCmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		if strings.Contains(outStr, "nothing to commit") {
			return "ok (nothing to commit)", nil
		}
		return "", fmt.Errorf("git commit failed: %w\n%s", err, outStr)
	}

	// Extract commit hash from output like "[main abc1234] message"
	for _, line := range strings.Split(outStr, "\n") {
		if strings.HasPrefix(line, "[") {
			// Parse "[branch hash] message"
			parts := strings.Fields(line)
			for _, p := range parts {
				if len(p) >= 7 && isHexHash(p) {
					return fmt.Sprintf("ok ✓ %s", p[:7]), nil
				}
			}
		}
	}

	return "ok ✓", nil
}

func isHexHash(s string) bool {
	if len(s) < 7 {
		return false
	}
	for _, c := range s[:7] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

var gitPushCmd = &cobra.Command{
	Use:   "push [args...]",
	Short: "Push commits (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitPush(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(output)
		recordCommand("git push", output, output, execTime, true)
	},
}

func runGitPush(args []string) (string, error) {
	pushCmd := buildGitCmd("push", args...)
	output, err := pushCmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		return "", fmt.Errorf("git push failed: %w\n%s", err, outStr)
	}

	if strings.Contains(outStr, "Everything up-to-date") {
		return "ok (up-to-date)", nil
	}

	// Extract branch reference from "->" lines
	for _, line := range strings.Split(outStr, "\n") {
		if strings.Contains(line, "->") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "->" && i+1 < len(parts) {
					return fmt.Sprintf("ok ✓ %s", parts[i+1]), nil
				}
			}
		}
	}

	return "ok ✓", nil
}

var gitPullCmd = &cobra.Command{
	Use:   "pull [args...]",
	Short: "Pull commits (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitPull(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(output)
		recordCommand("git pull", output, output, execTime, true)
	},
}

func runGitPull(args []string) (string, error) {
	pullCmd := buildGitCmd("pull", args...)
	output, err := pullCmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		return "", fmt.Errorf("git pull failed: %w\n%s", err, outStr)
	}

	if strings.Contains(outStr, "Already up to date") || strings.Contains(outStr, "Already up-to-date") {
		return "ok (up-to-date)", nil
	}

	// Parse file changes
	files := 0
	insertions := 0
	deletions := 0

	for _, line := range strings.Split(outStr, "\n") {
		if strings.Contains(line, "file") && strings.Contains(line, "changed") {
			for _, part := range strings.Split(line, ",") {
				part = strings.TrimSpace(part)
				if strings.Contains(part, "file") {
					fmt.Sscanf(part, "%d", &files)
				} else if strings.Contains(part, "insertion") {
					fmt.Sscanf(part, "%d", &insertions)
				} else if strings.Contains(part, "deletion") {
					fmt.Sscanf(part, "%d", &deletions)
				}
			}
		}
	}

	if files > 0 {
		return fmt.Sprintf("ok ✓ %d files +%d -%d", files, insertions, deletions), nil
	}

	return "ok ✓", nil
}

var gitBranchCmd = &cobra.Command{
	Use:   "branch [args...]",
	Short: "List or manage branches (compact output)",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitBranch(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(output)
		recordCommand("git branch", output, output, execTime, true)
	},
}

func runGitBranch(args []string) (string, error) {
	// Check for write operations
	hasAction := false
	for _, arg := range args {
		if arg == "-d" || arg == "-D" || arg == "-m" || arg == "-M" || arg == "-c" || arg == "-C" {
			hasAction = true
			break
		}
	}

	// Check for positional args (branch creation)
	hasPositional := false
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			hasPositional = true
			break
		}
	}

	hasListFlag := false
	for _, arg := range args {
		if arg == "-a" || arg == "--all" || arg == "-r" || arg == "--remotes" || arg == "--list" {
			hasListFlag = true
			break
		}
	}

	// Write operation or branch creation
	if hasAction || (hasPositional && !hasListFlag) {
		branchCmd := buildGitCmd("branch", args...)
		output, err := branchCmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git branch failed: %w\n%s", err, output)
		}
		return "ok ✓", nil
	}

	// List mode
	listArgs := []string{"-a", "--no-color"}
	listArgs = append(listArgs, args...)
	branchCmd := buildGitCmd("branch", listArgs...)
	output, err := branchCmd.Output()
	if err != nil {
		return "", fmt.Errorf("git branch failed: %w", err)
	}

	return filterBranchOutput(string(output)), nil
}

func filterBranchOutput(output string) string {
	var current string
	var local []string
	var remote []string

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "* ") {
			current = strings.TrimPrefix(line, "* ")
		} else if strings.HasPrefix(line, "remotes/origin/") {
			branch := strings.TrimPrefix(line, "remotes/origin/")
			if !strings.HasPrefix(branch, "HEAD ") {
				remote = append(remote, branch)
			}
		} else {
			local = append(local, line)
		}
	}

	var result []string
	result = append(result, fmt.Sprintf("* %s", current))

	for _, b := range local {
		result = append(result, fmt.Sprintf("  %s", b))
	}

	// Remote-only branches
	remoteOnly := []string{}
	for _, r := range remote {
		found := false
		if r == current {
			found = true
		}
		for _, l := range local {
			if r == l {
				found = true
				break
			}
		}
		if !found {
			remoteOnly = append(remoteOnly, r)
		}
	}

	if len(remoteOnly) > 0 {
		result = append(result, fmt.Sprintf("  remote-only (%d):", len(remoteOnly)))
		for i, b := range remoteOnly {
			if i >= 10 {
				result = append(result, fmt.Sprintf("    ... +%d more", len(remoteOnly)-10))
				break
			}
			result = append(result, fmt.Sprintf("    %s", b))
		}
	}

	return strings.Join(result, "\n")
}

var gitFetchCmd = &cobra.Command{
	Use:   "fetch [args...]",
	Short: "Fetch from remote (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitFetch(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(output)
		recordCommand("git fetch", output, output, execTime, true)
	},
}

func runGitFetch(args []string) (string, error) {
	fetchCmd := buildGitCmd("fetch", args...)
	output, err := fetchCmd.CombinedOutput()
	outStr := string(output)

	if err != nil {
		return "", fmt.Errorf("git fetch failed: %w\n%s", err, outStr)
	}

	// Count new refs
	newRefs := 0
	for _, line := range strings.Split(outStr, "\n") {
		if strings.Contains(line, "->") || strings.Contains(line, "[new") {
			newRefs++
		}
	}

	if newRefs > 0 {
		return fmt.Sprintf("ok fetched (%d new refs)", newRefs), nil
	}

	return "ok fetched", nil
}

var gitStashCmd = &cobra.Command{
	Use:   "stash [subcommand] [args...]",
	Short: "Stash changes (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitStash(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(output)
		recordCommand("git stash", output, output, execTime, true)
	},
}

func runGitStash(args []string) (string, error) {
	subCmd := ""
	remainingArgs := args

	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		subCmd = args[0]
		remainingArgs = args[1:]
	}

	switch subCmd {
	case "list":
		stashCmd := buildGitCmd("stash", "list")
		output, err := stashCmd.Output()
		if err != nil {
			return "", fmt.Errorf("git stash list failed: %w", err)
		}

		if strings.TrimSpace(string(output)) == "" {
			return "No stashes", nil
		}

		return filterStashList(string(output)), nil

	case "show":
		showArgs := []string{"show", "-p"}
		showArgs = append(showArgs, remainingArgs...)
		stashCmd := buildGitCmd("stash", showArgs...)
		output, err := stashCmd.Output()
		if err != nil {
			return "", fmt.Errorf("git stash show failed: %w", err)
		}

		if strings.TrimSpace(string(output)) == "" {
			return "Empty stash", nil
		}

		return filterDiff(string(output)), nil

	case "pop", "apply", "drop", "push":
		stashArgs := append([]string{subCmd}, remainingArgs...)
		stashCmd := buildGitCmd("stash", stashArgs...)
		output, err := stashCmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git stash %s failed: %w\n%s", subCmd, err, output)
		}
		return fmt.Sprintf("ok stash %s", subCmd), nil

	default:
		// Default: git stash (push)
		stashCmd := buildGitCmd("stash", args...)
		output, err := stashCmd.CombinedOutput()
		outStr := string(output)

		if err != nil {
			return "", fmt.Errorf("git stash failed: %w\n%s", err, outStr)
		}

		if strings.Contains(outStr, "No local changes") {
			return "ok (nothing to stash)", nil
		}

		return "ok stashed", nil
	}
}

func filterStashList(output string) string {
	var result []string
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		// Format: "stash@{0}: WIP on main: abc1234 commit message"
		colonIdx := strings.Index(line, ": ")
		if colonIdx == -1 {
			result = append(result, line)
			continue
		}

		index := line[:colonIdx]
		rest := line[colonIdx+2:]

		// Strip "WIP on branch:" prefix
		if secondColon := strings.Index(rest, ": "); secondColon != -1 {
			rest = rest[secondColon+2:]
		}

		result = append(result, fmt.Sprintf("%s: %s", index, strings.TrimSpace(rest)))
	}

	return strings.Join(result, "\n")
}

var gitWorktreeCmd = &cobra.Command{
	Use:   "worktree [args...]",
	Short: "Manage worktrees (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		output, err := runGitWorktree(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(output)
		recordCommand("git worktree", output, output, execTime, true)
	},
}

func runGitWorktree(args []string) (string, error) {
	// Check for action commands
	hasAction := false
	for _, arg := range args {
		if arg == "add" || arg == "remove" || arg == "prune" || arg == "lock" || arg == "unlock" || arg == "move" {
			hasAction = true
			break
		}
	}

	if hasAction {
		worktreeCmd := buildGitCmd("worktree", args...)
		output, err := worktreeCmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git worktree failed: %w\n%s", err, output)
		}
		return "ok ✓", nil
	}

	// Default: list
	worktreeCmd := buildGitCmd("worktree", "list")
	output, err := worktreeCmd.Output()
	if err != nil {
		return "", fmt.Errorf("git worktree list failed: %w", err)
	}

	return filterWorktreeList(string(output)), nil
}

func filterWorktreeList(output string) string {
	home := os.Getenv("HOME")

	var result []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "/path/to/worktree  abc1234 [branch]"
		parts := strings.Fields(line)
		if len(parts) >= 3 {
			path := parts[0]
			hash := parts[1]
			branch := strings.Join(parts[2:], " ")

			// Shorten home path
			if home != "" && strings.HasPrefix(path, home) {
				path = "~" + path[len(home):]
			}

			result = append(result, fmt.Sprintf("%s %s %s", path, hash, branch))
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
