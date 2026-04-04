package vcs

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var gitWorktreeCmd = &cobra.Command{
	Use:   "worktree [args...]",
	Short: "Manage worktrees (compact output)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return shared.ExecuteAndRecord("git worktree", func() (string, string, error) {
			return runGitWorktree(args)
		})
	},
}

func runGitWorktree(args []string) (string, string, error) {
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
		raw := string(output)
		if err != nil {
			return raw, "", fmt.Errorf("git worktree failed: %w\n%s", err, output)
		}
		return raw, "ok", nil
	}

	worktreeCmd := buildGitCmd("worktree", "list")
	output, err := worktreeCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("git worktree list failed: %w", err)
	}

	raw := string(output)
	return raw, filterWorktreeList(raw), nil
}

func filterWorktreeList(output string) string {
	home := os.Getenv("HOME")

	var result []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.Fields(line)
		if len(parts) >= 3 {
			path := parts[0]
			hash := parts[1]
			branch := strings.Join(parts[2:], " ")

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
