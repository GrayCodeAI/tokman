package vcs

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var gitPushCmd = &cobra.Command{
	Use:   "push [args...]",
	Short: "Push commits (compact output)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return shared.ExecuteAndRecord("git push", func() (string, string, error) {
			return runGitPush(args)
		})
	},
}

func runGitPush(args []string) (string, string, error) {
	pushCmd := buildGitCmd("push", args...)
	output, err := pushCmd.CombinedOutput()
	raw := string(output)

	if err != nil {
		return raw, "", fmt.Errorf("git push failed: %w\n%s", err, raw)
	}

	if strings.Contains(raw, "Everything up-to-date") {
		return raw, "ok (up-to-date)", nil
	}

	for _, line := range strings.Split(raw, "\n") {
		if strings.Contains(line, "->") {
			parts := strings.Fields(line)
			for i, p := range parts {
				if p == "->" && i+1 < len(parts) {
					return raw, fmt.Sprintf("ok %s", parts[i+1]), nil
				}
			}
		}
	}

	return raw, "ok", nil
}
