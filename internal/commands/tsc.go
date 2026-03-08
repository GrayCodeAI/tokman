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

var tscCmd = &cobra.Command{
	Use:   "tsc [args...]",
	Short: "TypeScript compiler with filtered output",
	Long: `TypeScript compiler with token-optimized output.

Groups errors by file and shows error code summaries.

Examples:
  tokman tsc
  tokman tsc --noEmit
  tokman tsc -p tsconfig.build.json`,
	RunE: runTsc,
}

func init() {
	rootCmd.AddCommand(tscCmd)
}

type TsError struct {
	File          string
	Line          string
	Col           string
	Severity      string
	Code          string
	Message       string
	ContextLines  []string
}

func runTsc(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	// Try tsc directly first, fallback to npx
	tscPath, err := exec.LookPath("tsc")
	if err != nil {
		tscPath = "" // Will use npx
	}

	var c *exec.Cmd
	if tscPath != "" {
		c = exec.Command(tscPath, args...)
	} else {
		npxArgs := append([]string{"tsc"}, args...)
		c = exec.Command("npx", npxArgs...)
	}
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err = c.Run()
	output := stdout.String() + stderr.String()

	filtered := filterTscOutput(output)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("tsc %s", strings.Join(args, " ")), "tokman tsc", originalTokens, filteredTokens)

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", originalTokens-filteredTokens)
	}

	// Preserve tsc exit code for CI/CD compatibility
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		os.Exit(1)
	}
	return nil
}

// TSC error pattern: src/file.ts(12,5): error TS2322: Type 'string' is not assignable to type 'number'.
var tscErrorRegex = regexp.MustCompile(`^(.+?)\((\d+),(\d+)\):\s+(error|warning)\s+(TS\d+):\s+(.+)$`)

func filterTscOutput(output string) string {
	var errors []TsError
	lines := strings.Split(output, "\n")

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		matches := tscErrorRegex.FindStringSubmatch(line)
		if matches != nil {
			err := TsError{
				File:     matches[1],
				Line:     matches[2],
				Col:      matches[3],
				Severity: matches[4],
				Code:     matches[5],
				Message:  matches[6],
			}

			// Capture continuation lines (indented context from tsc)
			for i+1 < len(lines) {
				next := lines[i+1]
				trimmed := strings.TrimSpace(next)
				if trimmed != "" && (strings.HasPrefix(next, "  ") || strings.HasPrefix(next, "\t")) && !tscErrorRegex.MatchString(next) {
					err.ContextLines = append(err.ContextLines, trimmed)
					i++
				} else {
					break
				}
			}
			errors = append(errors, err)
		}
	}

	if len(errors) == 0 {
		if strings.Contains(output, "Found 0 errors") {
			return "✓ TypeScript: No errors found\n"
		}
		return "TypeScript compilation completed\n"
	}

	// Group by file
	byFile := make(map[string][]TsError)
	for _, err := range errors {
		byFile[err.File] = append(byFile[err.File], err)
	}

	// Count by error code for summary
	byCode := make(map[string]int)
	for _, err := range errors {
		byCode[err.Code]++
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("TypeScript: %d errors in %d files\n", len(errors), len(byFile)))
	result.WriteString("═══════════════════════════════════════\n")

	// Top error codes summary
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
		errors []TsError
	}
	var filesSorted []fileErrors
	for file, errs := range byFile {
		filesSorted = append(filesSorted, fileErrors{file, errs})
	}
	sort.Slice(filesSorted, func(i, j int) bool {
		return len(filesSorted[i].errors) > len(filesSorted[j].errors)
	})

	// Show every error per file — no limits
	for _, fe := range filesSorted {
		result.WriteString(fmt.Sprintf("%s (%d errors)\n", fe.file, len(fe.errors)))
		for _, err := range fe.errors {
			msg := truncate(err.Message, 120)
			result.WriteString(fmt.Sprintf("  L%s: %s %s\n", err.Line, err.Code, msg))
			for _, ctx := range err.ContextLines {
				result.WriteString(fmt.Sprintf("    %s\n", truncate(ctx, 120)))
			}
		}
		result.WriteString("\n")
	}

	return result.String()
}
