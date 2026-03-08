package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var mypyCmd = &cobra.Command{
	Use:   "mypy [args...]",
	Short: "Mypy type checker with filtered output",
	Long: `Mypy type checker with token-optimized output.

Groups errors by file and shows error code summaries.

Examples:
  tokman mypy src/
  tokman mypy --strict .
  tokman mypy -p mypackage`,
	RunE: runMypy,
}

func init() {
	rootCmd.AddCommand(mypyCmd)
}

type MypyError struct {
	File         string
	Line         int
	Code         string
	Message      string
	ContextLines []string
}

// mypy pattern: file.py:12: error: Message [error-code]
// or file.py:12:5: error: Message [error-code]
var mypyDiagRegex = regexp.MustCompile(`^(.+?):(\d+)(?::\d+)?: (error|warning|note): (.+?)(?:\s+\[(.+)\])?$`)

func runMypy(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	// Try mypy directly, fallback to python -m mypy
	mypyPath, err := exec.LookPath("mypy")
	if err != nil {
		mypyPath = "" // Will use python -m mypy
	}

	var c *exec.Cmd
	if mypyPath != "" {
		c = exec.Command(mypyPath, args...)
	} else {
		pyArgs := append([]string{"-m", "mypy"}, args...)
		c = exec.Command("python3", pyArgs...)
	}
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err = c.Run()
	output := stdout.String() + stderr.String()

	// Strip ANSI codes
	clean := stripANSI(output)

	filtered := filterMypyOutput(clean)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("mypy %s", strings.Join(args, " ")), "tokman mypy", originalTokens, filteredTokens)

	if verbose {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", originalTokens-filteredTokens)
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	return nil
}

func filterMypyOutput(output string) string {
	lines := strings.Split(output, "\n")
	var errors []MypyError
	var filelessLines []string

	for i := 0; i < len(lines); i++ {
		line := lines[i]

		// Skip mypy's own summary line
		if strings.HasPrefix(line, "Found ") && strings.Contains(line, " error") {
			continue
		}
		// Skip "Success: no issues found"
		if strings.HasPrefix(line, "Success:") {
			continue
		}

		matches := mypyDiagRegex.FindStringSubmatch(line)
		if matches != nil {
			severity := matches[3]
			file := matches[1]
			lineNum := parseIntOrZero(matches[2])
			message := matches[4]
			code := ""
			if len(matches) > 5 {
				code = matches[5]
			}

			if severity == "note" {
				// Attach note to preceding error if same file
				if len(errors) > 0 {
					last := &errors[len(errors)-1]
					if last.File == file {
						last.ContextLines = append(last.ContextLines, message)
						continue
					}
				}
				// Standalone note -- display as fileless
				filelessLines = append(filelessLines, line)
				continue
			}

			myErr := MypyError{
				File:    file,
				Line:    lineNum,
				Code:    code,
				Message: message,
			}

			// Capture continuation note lines
			for i+1 < len(lines) {
				nextMatches := mypyDiagRegex.FindStringSubmatch(lines[i+1])
				if nextMatches != nil && nextMatches[3] == "note" && nextMatches[1] == file {
					myErr.ContextLines = append(myErr.ContextLines, nextMatches[4])
					i++
					continue
				}
				break
			}

			errors = append(errors, myErr)
		} else if strings.Contains(line, "error:") && strings.TrimSpace(line) != "" {
			// File-less error (config errors, import errors)
			filelessLines = append(filelessLines, line)
		}
	}

	// No errors at all
	if len(errors) == 0 && len(filelessLines) == 0 {
		if strings.Contains(output, "Success: no issues found") || strings.Contains(output, "no issues found") {
			return "mypy: No issues found\n"
		}
		return "mypy: No issues found\n"
	}

	// Group by file
	byFile := make(map[string][]MypyError)
	for _, err := range errors {
		byFile[err.File] = append(byFile[err.File], err)
	}

	// Count by error code
	byCode := make(map[string]int)
	for _, err := range errors {
		if err.Code != "" {
			byCode[err.Code]++
		}
	}

	var result strings.Builder

	// File-less errors first
	for _, line := range filelessLines {
		result.WriteString(line + "\n")
	}
	if len(filelessLines) > 0 && len(errors) > 0 {
		result.WriteString("\n")
	}

	if len(errors) > 0 {
		result.WriteString(fmt.Sprintf("mypy: %d errors in %d files\n", len(errors), len(byFile)))
		result.WriteString("═══════════════════════════════════════\n")

		// Top error codes summary (only when 2+ distinct codes)
		type codeCount struct {
			code  string
			count int
		}
		var codeCounts []codeCount
		for code, count := range byCode {
			codeCounts = append(codeCounts, codeCount{code, count})
		}
		sort.Slice(codeCounts, func(i, j int) bool {
			return codeCounts[i].count > codeCounts[j].count
		})

		if len(codeCounts) > 1 {
			var codesStr []string
			for i := 0; i < 5 && i < len(codeCounts); i++ {
				codesStr = append(codesStr, fmt.Sprintf("%s (%dx)", codeCounts[i].code, codeCounts[i].count))
			}
			result.WriteString(fmt.Sprintf("Top codes: %s\n\n", strings.Join(codesStr, ", ")))
		}

		// Files sorted by error count (most errors first)
		type fileErrors struct {
			file   string
			errors []MypyError
		}
		var filesSorted []fileErrors
		for file, errs := range byFile {
			filesSorted = append(filesSorted, fileErrors{file, errs})
		}
		sort.Slice(filesSorted, func(i, j int) bool {
			return len(filesSorted[i].errors) > len(filesSorted[j].errors)
		})

		for _, fe := range filesSorted {
			result.WriteString(fmt.Sprintf("%s (%d errors)\n", fe.file, len(fe.errors)))

			for _, err := range fe.errors {
				if err.Code == "" {
					result.WriteString(fmt.Sprintf("  L%d: %s\n", err.Line, truncate(err.Message, 120)))
				} else {
					result.WriteString(fmt.Sprintf("  L%d: [%s] %s\n", err.Line, err.Code, truncate(err.Message, 120)))
				}
				for _, ctx := range err.ContextLines {
					result.WriteString(fmt.Sprintf("    %s\n", truncate(ctx, 120)))
				}
			}
			result.WriteString("\n")
		}
	}

	return result.String()
}

func parseIntOrZero(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}

func stripANSI(s string) string {
	// Simple ANSI escape code stripper
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(s, "")
}
