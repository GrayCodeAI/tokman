package vcs

import (
	"os/exec"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
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
  -c, --config <key=value>    Set git config option`,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	TraverseChildren:   true,
}

// buildGitCmd builds a git command with global flags prepended
func buildGitCmd(subCmd string, args ...string) *exec.Cmd {
	gitArgs := []string{}

	// Add global flags
	if gitDirectory != "" {
		gitArgs = append(gitArgs, "-C", gitDirectory)
	}
	if gitDir != "" {
		gitArgs = append(gitArgs, "--git-dir", gitDir)
	}
	if gitWorkTree != "" {
		gitArgs = append(gitArgs, "--work-tree", gitWorkTree)
	}
	if gitNoPager {
		gitArgs = append(gitArgs, "--no-pager")
	}
	if gitNoOptLocks {
		gitArgs = append(gitArgs, "--no-optional-locks")
	}
	if gitBare {
		gitArgs = append(gitArgs, "--bare")
	}
	if gitLiteralPaths {
		gitArgs = append(gitArgs, "--literal-pathspecs")
	}
	for _, opt := range gitConfigOpts {
		gitArgs = append(gitArgs, "-c", opt)
	}

	gitArgs = append(gitArgs, subCmd)
	gitArgs = append(gitArgs, args...)

	return exec.Command("git", gitArgs...)
}

// extractGitArgs filters tokman-specific flags from args, leaving only git-compatible ones
func extractGitArgs(args []string) []string {
	var gitArgs []string
	skipNext := false
	for i, arg := range args {
		if skipNext {
			skipNext = false
			continue
		}
		// Skip tokman-specific flags that take values
		if arg == "--query" || arg == "--budget" || arg == "--preset" ||
			arg == "--output" || arg == "-o" ||
			arg == "--compaction-threshold" || arg == "--compaction-preserve" ||
			arg == "--compaction-max-tokens" {
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
			arg == "--skip-env" ||
			arg == "--quiet" ||
			arg == "-q" ||
			arg == "--json" ||
			arg == "--compaction" ||
			arg == "--compaction-snapshot" ||
			arg == "--compaction-auto-detect" {
			continue
		}
		// Skip tokman flags with values
		if strings.HasPrefix(arg, "--query=") ||
			strings.HasPrefix(arg, "--config=") ||
			strings.HasPrefix(arg, "-c=") ||
			strings.HasPrefix(arg, "--budget=") ||
			strings.HasPrefix(arg, "--preset=") ||
			strings.HasPrefix(arg, "--output=") ||
			strings.HasPrefix(arg, "-o=") ||
			strings.HasPrefix(arg, "--compaction-threshold=") ||
			strings.HasPrefix(arg, "--compaction-preserve=") ||
			strings.HasPrefix(arg, "--compaction-max-tokens=") {
			continue
		}
		// Skip tokman flags with values (positional form)
		if (arg == "--budget" || arg == "--preset" || arg == "--output" || arg == "-o" ||
			arg == "--compaction-threshold" || arg == "--compaction-preserve" ||
			arg == "--compaction-max-tokens") && i+1 < len(args) {
			skipNext = true
			continue
		}
		gitArgs = append(gitArgs, arg)
	}
	return gitArgs
}

// gray returns a gray-colored string
func gray(s string) string {
	dim := color.New(color.FgHiBlack).SprintFunc()
	return dim(s)
}

// isHexHash checks if the first 7 characters of a string are hex digits
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

func init() {
	registry.Add(func() { registry.Register(gitCmd) })

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
