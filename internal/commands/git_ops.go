package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var gitAddCmd = &cobra.Command{
	Use:   "add [args...]",
	Short: "Add files to staging (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		raw, filtered, err := runGitAdd(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(filtered)
		recordCommand("git add", raw, filtered, execTime, true)
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

var gitCommitCmd = &cobra.Command{
	Use:   "commit [args...]",
	Short: "Commit changes (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		raw, filtered, err := runGitCommit(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(filtered)
		recordCommand("git commit", raw, filtered, execTime, true)
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

var gitPushCmd = &cobra.Command{
	Use:   "push [args...]",
	Short: "Push commits (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		raw, filtered, err := runGitPush(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(filtered)
		recordCommand("git push", raw, filtered, execTime, true)
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

var gitPullCmd = &cobra.Command{
	Use:   "pull [args...]",
	Short: "Pull commits (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		raw, filtered, err := runGitPull(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(filtered)
		recordCommand("git pull", raw, filtered, execTime, true)
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
		return raw, fmt.Sprintf("ok %d files +%d -%d", files, insertions, deletions), nil
	}

	return raw, "ok", nil
}

var gitBranchCmd = &cobra.Command{
	Use:                "branch [args...]",
	Short:              "List or manage branches (compact output)",
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		raw, filtered, err := runGitBranch(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(filtered)
		recordCommand("git branch", raw, filtered, execTime, true)
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

var gitFetchCmd = &cobra.Command{
	Use:   "fetch [args...]",
	Short: "Fetch from remote (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		raw, filtered, err := runGitFetch(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(filtered)
		recordCommand("git fetch", raw, filtered, execTime, true)
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

var gitStashCmd = &cobra.Command{
	Use:   "stash [subcommand] [args...]",
	Short: "Stash changes (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		raw, filtered, err := runGitStash(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(filtered)
		recordCommand("git stash", raw, filtered, execTime, true)
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

var gitWorktreeCmd = &cobra.Command{
	Use:   "worktree [args...]",
	Short: "Manage worktrees (compact output)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		raw, filtered, err := runGitWorktree(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(filtered)
		recordCommand("git worktree", raw, filtered, execTime, true)
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
