package vcs

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var gitFetchCmd = &cobra.Command{
	Use:   "fetch [args...]",
	Short: "Fetch from remote (compact output)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return shared.ExecuteAndRecord("git fetch", func() (string, string, error) {
			return runGitFetch(args)
		})
	},
}

func runGitFetch(args []string) (string, string, error) {
	fetchCmd := buildGitCmd("fetch", args...)
	output, err := fetchCmd.CombinedOutput()
	raw := string(output)

	if err != nil {
		return raw, "", fmt.Errorf("git fetch failed: %w\n%s", err, raw)
	}

	newRefs := 0
	for _, line := range strings.Split(raw, "\n") {
		if strings.Contains(line, "->") || strings.Contains(line, "[new") {
			newRefs++
		}
	}

	if newRefs > 0 {
		return raw, fmt.Sprintf("ok fetched (%d new refs)", newRefs), nil
	}

	return raw, "ok fetched", nil
}
