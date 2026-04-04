package vcs

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var gitPullCmd = &cobra.Command{
	Use:   "pull [args...]",
	Short: "Pull commits (compact output)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return shared.ExecuteAndRecord("git pull", func() (string, string, error) {
			return runGitPull(args)
		})
	},
}

func runGitPull(args []string) (string, string, error) {
	pullCmd := buildGitCmd("pull", args...)
	output, err := pullCmd.CombinedOutput()
	raw := string(output)

	if err != nil {
		return raw, "", fmt.Errorf("git pull failed: %w\n%s", err, raw)
	}

	if strings.Contains(raw, "Already up to date") || strings.Contains(raw, "Already up-to-date") {
		return raw, "ok (up-to-date)", nil
	}

	files := 0
	insertions := 0
	deletions := 0

	for _, line := range strings.Split(raw, "\n") {
		if strings.Contains(line, "file") && strings.Contains(line, "changed") {
			for _, part := range strings.Split(line, ",") {
				part = strings.TrimSpace(part)
				if strings.Contains(part, "file") {
					if _, err = fmt.Sscanf(part, "%d", &files); err != nil {
						files = 0
					}
				} else if strings.Contains(part, "insertion") {
					if _, err = fmt.Sscanf(part, "%d", &insertions); err != nil {
						insertions = 0
					}
				} else if strings.Contains(part, "deletion") {
					if _, err = fmt.Sscanf(part, "%d", &deletions); err != nil {
						deletions = 0
					}
				}
			}
		}
	}

	if files > 0 {
		return raw, fmt.Sprintf("ok %d files +%d -%d", files, insertions, deletions), nil
	}

	return raw, "ok", nil
}
