package vcs

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
)

var gitBranchCmd = &cobra.Command{
	Use:                "branch [args...]",
	Short:              "List or manage branches (compact output)",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	RunE: func(cmd *cobra.Command, args []string) error {
		return shared.ExecuteAndRecord("git branch", func() (string, string, error) {
			return runGitBranch(args)
		})
	},
}

func runGitBranch(args []string) (string, string, error) {
	hasAction := false
	for _, arg := range args {
		if arg == "-d" || arg == "-D" || arg == "-m" || arg == "-M" || arg == "-c" || arg == "-C" {
			hasAction = true
			break
		}
	}

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

	if hasAction || (hasPositional && !hasListFlag) {
		branchCmd := buildGitCmd("branch", args...)
		output, err := branchCmd.CombinedOutput()
		raw := string(output)
		if err != nil {
			return raw, "", fmt.Errorf("git branch failed: %w\n%s", err, output)
		}
		return raw, "ok", nil
	}

	listArgs := []string{"-a", "--no-color"}
	listArgs = append(listArgs, args...)
	branchCmd := buildGitCmd("branch", listArgs...)
	output, err := branchCmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("git branch failed: %w", err)
	}

	raw := string(output)
	return raw, filterBranchOutput(raw), nil
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
