package vcs

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var gitStashCmd = &cobra.Command{
	Use:   "stash [subcommand] [args...]",
	Short: "Stash changes (compact output)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return shared.ExecuteAndRecord("git stash", func() (string, string, error) {
			return runGitStash(args)
		})
	},
}

func runGitStash(args []string) (string, string, error) {
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
		raw := string(output)
		if err != nil {
			return raw, "", fmt.Errorf("git stash list failed: %w", err)
		}

		if strings.TrimSpace(raw) == "" {
			return raw, "No stashes", nil
		}

		return raw, filterStashList(raw), nil

	case "show":
		showArgs := []string{"show", "-p"}
		showArgs = append(showArgs, remainingArgs...)
		stashCmd := buildGitCmd("stash", showArgs...)
		output, err := stashCmd.Output()
		raw := string(output)
		if err != nil {
			return raw, "", fmt.Errorf("git stash show failed: %w", err)
		}

		if strings.TrimSpace(raw) == "" {
			return raw, "Empty stash", nil
		}

		return raw, filterDiff(raw), nil

	case "pop", "apply", "drop", "push":
		stashArgs := append([]string{subCmd}, remainingArgs...)
		stashCmd := buildGitCmd("stash", stashArgs...)
		output, err := stashCmd.CombinedOutput()
		raw := string(output)
		if err != nil {
			return raw, "", fmt.Errorf("git stash %s failed: %w\n%s", subCmd, err, output)
		}
		return raw, fmt.Sprintf("ok stash %s", subCmd), nil

	default:
		stashCmd := buildGitCmd("stash", args...)
		output, err := stashCmd.CombinedOutput()
		raw := string(output)

		if err != nil {
			return raw, "", fmt.Errorf("git stash failed: %w\n%s", err, raw)
		}

		if strings.Contains(raw, "No local changes") {
			return raw, "ok (nothing to stash)", nil
		}

		return raw, "ok stashed", nil
	}
}

func filterStashList(output string) string {
	var result []string
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}

		colonIdx := strings.Index(line, ": ")
		if colonIdx == -1 {
			result = append(result, line)
			continue
		}

		index := line[:colonIdx]
		rest := line[colonIdx+2:]

		if secondColon := strings.Index(rest, ": "); secondColon != -1 {
			rest = rest[secondColon+2:]
		}

		result = append(result, fmt.Sprintf("%s: %s", index, strings.TrimSpace(rest)))
	}

	return strings.Join(result, "\n")
}
