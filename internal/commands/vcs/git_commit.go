package vcs

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var gitCommitCmd = &cobra.Command{
	Use:   "commit [args...]",
	Short: "Commit changes (compact output)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return shared.ExecuteAndRecord("git commit", func() (string, string, error) {
			return runGitCommit(args)
		})
	},
}

func runGitCommit(args []string) (string, string, error) {
	commitCmd := buildGitCmd("commit", args...)
	output, err := commitCmd.CombinedOutput()
	raw := string(output)

	if err != nil {
		if strings.Contains(raw, "nothing to commit") {
			return raw, "ok (nothing to commit)", nil
		}
		return raw, "", fmt.Errorf("git commit failed: %w\n%s", err, raw)
	}

	for _, line := range strings.Split(raw, "\n") {
		if strings.HasPrefix(line, "[") {
			parts := strings.Fields(line)
			for _, p := range parts {
				if len(p) >= 7 && isHexHash(p) {
					return raw, fmt.Sprintf("ok %s", p[:7]), nil
				}
			}
		}
	}

	return raw, "ok", nil
}
