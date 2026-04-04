package vcs

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var gitAddCmd = &cobra.Command{
	Use:   "add [args...]",
	Short: "Add files to staging (compact output)",
	RunE: func(cmd *cobra.Command, args []string) error {
		return shared.ExecuteAndRecord("git add", func() (string, string, error) {
			return runGitAdd(args)
		})
	},
}

func runGitAdd(args []string) (string, string, error) {
	addArgs := args
	if len(addArgs) == 0 {
		addArgs = []string{"."}
	}

	addCmd := buildGitCmd("add", addArgs...)
	output, err := addCmd.CombinedOutput()
	raw := string(output)
	if err != nil {
		return raw, "", fmt.Errorf("git add failed: %w\n%s", err, output)
	}

	statCmd := buildGitCmd("diff", "--cached", "--stat", "--shortstat")
	statOut, _ := statCmd.Output()
	stat := strings.TrimSpace(string(statOut))

	if stat == "" {
		return raw, "ok (nothing to add)", nil
	}

	lines := strings.Split(stat, "\n")
	lastLine := lines[len(lines)-1]
	return raw, fmt.Sprintf("ok %s", lastLine), nil
}
