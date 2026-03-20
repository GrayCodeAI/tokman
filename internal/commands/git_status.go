package commands

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// GitStatus represents parsed git status output
type GitStatus struct {
	Branch         string
	TrackingBranch string // e.g., "origin/main"
	Ahead          int
	Behind         int
	Staged         []GitFile
	Modified       []GitFile
	Untracked      []GitFile
	Conflicted     []GitFile
}

// GitFile represents a file in git status
type GitFile struct {
	Path string
	Code string
}

var gitStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show working tree status (filtered)",
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		raw, filtered, err := runGitStatus()
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(filtered)

		if err := recordCommand("git status", raw, filtered, execTime, true); err != nil && verbose > 0 {
			fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", err)
		}
	},
}

func runGitStatus() (string, string, error) {
	// Capture raw git status for token tracking
	rawCmd := buildGitCmd("status")
	var rawOut bytes.Buffer
	rawCmd.Stdout = &rawOut
	rawCmd.Run()
	raw := rawOut.String()

	// Get porcelain data for compact formatting
	porcelainCmd := buildGitCmd("status", "--porcelain", "-b")
	var porcelainOut bytes.Buffer
	porcelainCmd.Stdout = &porcelainOut
	if err := porcelainCmd.Run(); err != nil {
		return raw, "", fmt.Errorf("git status failed: %w", err)
	}

	status := parsePorcelain(porcelainOut.String())
	return raw, formatStatus(status), nil
}

func parsePorcelain(output string) *GitStatus {
	status := &GitStatus{}
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for i, line := range lines {
		if i == 0 {
			status.Branch, status.TrackingBranch, status.Ahead, status.Behind = parseBranchLine(line)
			continue
		}

		if len(line) < 4 {
			continue
		}

		code := line[:2]
		filePath := strings.TrimSpace(line[3:])

		if strings.Contains(filePath, " -> ") {
			parts := strings.Split(filePath, " -> ")
			if len(parts) == 2 {
				filePath = parts[1]
			}
		}

		gitFile := GitFile{Path: filePath, Code: code}

		switch {
		case strings.Contains(code, "U"):
			status.Conflicted = append(status.Conflicted, gitFile)
		case code == "??":
			status.Untracked = append(status.Untracked, gitFile)
		case code == "!!":
			// Ignored - skip
		case code[0] != ' ' && code[0] != '?':
			status.Staged = append(status.Staged, gitFile)
			if code[1] != ' ' {
				status.Modified = append(status.Modified, gitFile)
			}
		case code[1] != ' ':
			status.Modified = append(status.Modified, gitFile)
		}
	}

	return status
}

func parseBranchLine(line string) (branch, tracking string, ahead, behind int) {
	line = strings.TrimPrefix(line, "## ")

	if strings.HasPrefix(line, "HEAD detached") {
		return "HEAD detached", "", 0, 0
	}

	if !strings.Contains(line, "...") {
		if strings.Contains(line, "[") {
			parts := strings.SplitN(line, "[", 2)
			branch = strings.TrimSpace(parts[0])
			ahead, behind = parseAheadBehind(parts[1])
		} else {
			branch = strings.TrimSpace(line)
		}
		return branch, "", ahead, behind
	}

	parts := strings.SplitN(line, "...", 2)
	branch = strings.Fields(parts[0])[0]

	if len(parts) > 1 {
		trackingPart := parts[1]
		if strings.Contains(trackingPart, "[") {
			abParts := strings.SplitN(trackingPart, "[", 2)
			tracking = strings.TrimSpace(abParts[0])
			ahead, behind = parseAheadBehind(abParts[1])
		} else {
			tracking = strings.TrimSpace(trackingPart)
		}
	}

	return branch, tracking, ahead, behind
}

func parseAheadBehind(s string) (ahead, behind int) {
	s = strings.TrimSuffix(s, "]")

	aheadRe := regexp.MustCompile(`ahead (\d+)`)
	if matches := aheadRe.FindStringSubmatch(s); len(matches) > 1 {
		ahead, _ = strconv.Atoi(matches[1])
	}

	behindRe := regexp.MustCompile(`behind (\d+)`)
	if matches := behindRe.FindStringSubmatch(s); len(matches) > 1 {
		behind, _ = strconv.Atoi(matches[1])
	}

	return ahead, behind
}

func formatStatus(status *GitStatus) string {
	if ultraCompact {
		return formatStatusUltraCompact(status)
	}

	var buf strings.Builder

	green := color.New(color.FgGreen).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()

	buf.WriteString(bold("📌 ") + cyan(status.Branch))
	if status.Ahead > 0 || status.Behind > 0 {
		buf.WriteString(" [")
		if status.Ahead > 0 {
			buf.WriteString(fmt.Sprintf("ahead %d", status.Ahead))
		}
		if status.Behind > 0 {
			if status.Ahead > 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(fmt.Sprintf("behind %d", status.Behind))
		}
		buf.WriteString("]")
	}
	buf.WriteString("\n")

	if len(status.Staged) > 0 {
		buf.WriteString(green("\n✅ Staged:\n"))
		for _, f := range status.Staged {
			statusChar := getStatusCode(f.Code[0])
			buf.WriteString(fmt.Sprintf("   %s %s\n", statusChar, f.Path))
		}
	}

	if len(status.Modified) > 0 {
		buf.WriteString(yellow("\n📝 Modified:\n"))
		for _, f := range status.Modified {
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	if len(status.Untracked) > 0 {
		buf.WriteString(red("\n❓ Untracked:\n"))
		maxShow := 10
		for i, f := range status.Untracked {
			if i >= maxShow {
				buf.WriteString(fmt.Sprintf("   ... and %d more\n", len(status.Untracked)-maxShow))
				break
			}
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	if len(status.Conflicted) > 0 {
		buf.WriteString(red("\n⚠️  Conflicted:\n"))
		for _, f := range status.Conflicted {
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	if len(status.Staged) == 0 && len(status.Modified) == 0 &&
		len(status.Untracked) == 0 && len(status.Conflicted) == 0 {
		buf.WriteString(green("\n✓ Clean working tree\n"))
	}

	return buf.String()
}

func formatStatusUltraCompact(status *GitStatus) string {
	var buf strings.Builder

	branch := status.Branch
	if status.TrackingBranch != "" {
		buf.WriteString(fmt.Sprintf("* %s...%s\n", branch, status.TrackingBranch))
	} else if status.Ahead > 0 || status.Behind > 0 {
		buf.WriteString(fmt.Sprintf("* %s [a%d b%d]\n", branch, status.Ahead, status.Behind))
	} else {
		buf.WriteString(fmt.Sprintf("* %s\n", branch))
	}

	if len(status.Staged) > 0 {
		buf.WriteString(fmt.Sprintf("+ Staged: %d files\n", len(status.Staged)))
		for i, f := range status.Staged {
			if i >= 3 {
				buf.WriteString(fmt.Sprintf("   ... +%d more\n", len(status.Staged)-3))
				break
			}
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	if len(status.Modified) > 0 {
		buf.WriteString(fmt.Sprintf("~ Modified: %d files\n", len(status.Modified)))
		for i, f := range status.Modified {
			if i >= 3 {
				buf.WriteString(fmt.Sprintf("   ... +%d more\n", len(status.Modified)-3))
				break
			}
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	if len(status.Untracked) > 0 {
		buf.WriteString(fmt.Sprintf("? Untracked: %d files\n", len(status.Untracked)))
		for i, f := range status.Untracked {
			if i >= 3 {
				buf.WriteString(fmt.Sprintf("   ... +%d more\n", len(status.Untracked)-3))
				break
			}
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	if len(status.Conflicted) > 0 {
		buf.WriteString(fmt.Sprintf("! Conflicted: %d files\n", len(status.Conflicted)))
		for i, f := range status.Conflicted {
			if i >= 3 {
				buf.WriteString(fmt.Sprintf("   ... +%d more\n", len(status.Conflicted)-3))
				break
			}
			buf.WriteString(fmt.Sprintf("   %s\n", f.Path))
		}
	}

	if len(status.Staged) == 0 && len(status.Modified) == 0 &&
		len(status.Untracked) == 0 && len(status.Conflicted) == 0 {
		buf.WriteString("clean\n")
	}

	return buf.String()
}

func getStatusCode(code byte) string {
	switch code {
	case 'M':
		return "M"
	case 'A':
		return "A"
	case 'D':
		return "D"
	case 'R':
		return "R"
	case 'C':
		return "C"
	default:
		return string(code)
	}
}
