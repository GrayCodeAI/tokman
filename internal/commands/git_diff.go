package commands

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var gitDiffCmd = &cobra.Command{
	Use:   "diff [args...]",
	Short: "Show changes (filtered)",
	Long: `Show git diff with output filtering:
- Stats summary first
- Diff hunks limited to 30 lines each
- ANSI colors stripped`,
	FParseErrWhitelist: cobra.FParseErrWhitelist{UnknownFlags: true},
	Run: func(cmd *cobra.Command, args []string) {
		startTime := time.Now()
		raw, filtered, err := runGitDiff(args)
		execTime := time.Since(startTime).Milliseconds()

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(filtered)

		if err := recordCommand("git diff", raw, filtered, execTime, true); err != nil && verbose > 0 {
			fmt.Fprintf(os.Stderr, "Warning: failed to record: %v\n", err)
		}
	},
}

func runGitDiff(args []string) (string, string, error) {
	// Capture raw stat output
	statArgs := append([]string{"--stat"}, args...)
	statCmd := buildGitCmd("diff", statArgs...)
	var statOut bytes.Buffer
	statCmd.Stdout = &statOut
	statCmd.Run()

	// Capture raw diff output
	diffArgs := append([]string{}, args...)
	diffCmd := buildGitCmd("diff", diffArgs...)
	var diffOut bytes.Buffer
	diffCmd.Stdout = &diffOut
	if err := diffCmd.Run(); err != nil {
		return "", "", fmt.Errorf("git diff failed: %w", err)
	}

	rawDiff := diffOut.String()
	raw := rawDiff
	if statOut.Len() > 0 {
		raw = statOut.String() + "\n" + rawDiff
	}

	filtered := filterDiff(rawDiff)
	if statOut.Len() > 0 {
		filtered = statOut.String() + "\n" + filtered
	}

	return raw, filtered, nil
}

func filterDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var result []string
	hunkLineCount := 0
	inHunk := false
	maxHunkLines := 30
	maxTotalLines := 500
	currentFile := ""
	added := 0
	removed := 0

	for _, line := range lines {
		if len(result) >= maxTotalLines {
			result = append(result, gray("... (more changes truncated)"))
			break
		}

		if strings.HasPrefix(line, "diff --git") {
			// Per-file summary for previous file
			if currentFile != "" && (added > 0 || removed > 0) {
				result = append(result, fmt.Sprintf("  +%d -%d", added, removed))
			}
			// Extract filename from "diff --git a/path b/path"
			parts := strings.SplitN(line, " b/", 2)
			if len(parts) == 2 {
				currentFile = parts[1]
			} else {
				currentFile = "unknown"
			}
			result = append(result, "", currentFile)
			added = 0
			removed = 0
			inHunk = false
			continue
		}

		if strings.HasPrefix(line, "@@") {
			inHunk = true
			hunkLineCount = 0
			// Compact hunk header: "  @@ -1,3 +1,4 @@"
			parts := strings.SplitN(line, "@@", 3)
			if len(parts) >= 2 {
				result = append(result, fmt.Sprintf("  @@ %s @@", strings.TrimSpace(parts[1])))
			} else {
				result = append(result, fmt.Sprintf("  %s", line))
			}
			continue
		}

		if strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") {
			inHunk = false
			continue
		}

		if inHunk {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				added++
				if hunkLineCount < maxHunkLines {
					result = append(result, fmt.Sprintf("  %s", line))
					hunkLineCount++
				}
			} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
				removed++
				if hunkLineCount < maxHunkLines {
					result = append(result, fmt.Sprintf("  %s", line))
					hunkLineCount++
				}
			} else if hunkLineCount < maxHunkLines && !strings.HasPrefix(line, "\\") {
				if hunkLineCount > 0 {
					result = append(result, fmt.Sprintf("  %s", line))
					hunkLineCount++
				}
			}

			if hunkLineCount == maxHunkLines {
				result = append(result, gray("  ... (truncated)"))
				hunkLineCount++
			}
		}
	}

	// Per-file summary for last file
	if currentFile != "" && (added > 0 || removed > 0) {
		result = append(result, fmt.Sprintf("  +%d -%d", added, removed))
	}

	return strings.Join(result, "\n")
}
