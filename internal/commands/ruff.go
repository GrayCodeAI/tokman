package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var ruffCmd = &cobra.Command{
	Use:   "ruff [args...]",
	Short: "Ruff linter/formatter with filtered output",
	Long: `Ruff linter/formatter with token-optimized output.

Groups issues by rule and file for easy scanning.

Examples:
  tokman ruff check .
  tokman ruff format --check .
  tokman ruff check --fix src/`,
	RunE: runRuff,
}

func init() {
	rootCmd.AddCommand(ruffCmd)
}

type RuffLocation struct {
	Row    int `json:"row"`
	Column int `json:"column"`
}

type RuffFix struct {
	Applicability string `json:"applicability"`
}

type RuffDiagnostic struct {
	Code        string       `json:"code"`
	Message     string       `json:"message"`
	Location    RuffLocation `json:"location"`
	Filename    string       `json:"filename"`
	Fix         *RuffFix     `json:"fix"`
}

func runRuff(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	// Detect subcommand: check, format, or version
	isCheck := len(args) == 0 || args[0] == "check" || (!strings.HasPrefix(args[0], "-") && args[0] != "format" && args[0] != "version")
	isFormat := len(args) > 0 && args[0] == "format"

	c := exec.Command("ruff")

	if isCheck {
		// Force JSON output for check command
		hasOutputFormat := false
		for _, arg := range args {
			if strings.HasPrefix(arg, "--output-format") {
				hasOutputFormat = true
				break
			}
		}

		c.Args = append(c.Args, "check")
		if !hasOutputFormat {
			c.Args = append(c.Args, "--output-format=json")
		}

		// Add user arguments (skip "check" if it was the first arg)
		startIdx := 0
		if len(args) > 0 && args[0] == "check" {
			startIdx = 1
		}
		c.Args = append(c.Args, args[startIdx:]...)

		// Default to current directory if no path specified
		hasPath := false
		for _, arg := range args[startIdx:] {
			if !strings.HasPrefix(arg, "-") && !strings.Contains(arg, "=") {
				hasPath = true
				break
			}
		}
		if !hasPath {
			c.Args = append(c.Args, ".")
		}
	} else {
		c.Args = append(c.Args, args...)
	}

	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	var filtered string
	if isCheck && strings.TrimSpace(stdout.String()) != "" {
		filtered = filterRuffCheckJSON(stdout.String())
	} else if isFormat {
		filtered = filterRuffFormat(output)
	} else {
		filtered = strings.TrimSpace(output)
	}

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("ruff %s", strings.Join(args, " ")), "tokman ruff", originalTokens, filteredTokens)

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

func filterRuffCheckJSON(output string) string {
	var diagnostics []RuffDiagnostic
	if err := json.Unmarshal([]byte(output), &diagnostics); err != nil {
		return fmt.Sprintf("Ruff check (JSON parse failed: %s)\n%s", err, truncate(output, 500))
	}

	if len(diagnostics) == 0 {
		return "✓ Ruff: No issues found\n"
	}

	totalIssues := len(diagnostics)
	fixableCount := 0
	for _, d := range diagnostics {
		if d.Fix != nil {
			fixableCount++
		}
	}

	// Count unique files
	fileSet := make(map[string]bool)
	for _, d := range diagnostics {
		fileSet[d.Filename] = true
	}
	totalFiles := len(fileSet)

	// Group by rule code
	byRule := make(map[string]int)
	for _, d := range diagnostics {
		byRule[d.Code]++
	}

	// Group by file
	byFile := make(map[string]int)
	for _, d := range diagnostics {
		byFile[d.Filename]++
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Ruff: %d issues in %d files", totalIssues, totalFiles))
	if fixableCount > 0 {
		result.WriteString(fmt.Sprintf(" (%d fixable)", fixableCount))
	}
	result.WriteString("\n")
	result.WriteString("═══════════════════════════════════════\n")

	// Show top rules
	type ruleCount struct {
		code  string
		count int
	}
	var ruleCounts []ruleCount
	for code, count := range byRule {
		ruleCounts = append(ruleCounts, ruleCount{code, count})
	}
	sort.Slice(ruleCounts, func(i, j int) bool {
		return ruleCounts[i].count > ruleCounts[j].count
	})

	if len(ruleCounts) > 0 {
		result.WriteString("Top rules:\n")
		for i := 0; i < 10 && i < len(ruleCounts); i++ {
			result.WriteString(fmt.Sprintf("  %s (%dx)\n", ruleCounts[i].code, ruleCounts[i].count))
		}
		result.WriteString("\n")
	}

	// Show top files
	type fileCount struct {
		file  string
		count int
	}
	var fileCounts []fileCount
	for file, count := range byFile {
		fileCounts = append(fileCounts, fileCount{file, count})
	}
	sort.Slice(fileCounts, func(i, j int) bool {
		return fileCounts[i].count > fileCounts[j].count
	})

	result.WriteString("Top files:\n")
	for i := 0; i < 10 && i < len(fileCounts); i++ {
		shortPath := compactPath(fileCounts[i].file)
		result.WriteString(fmt.Sprintf("  %s (%d issues)\n", shortPath, fileCounts[i].count))

		// Show top 3 rules in this file
		fileRules := make(map[string]int)
		for _, d := range diagnostics {
			if d.Filename == fileCounts[i].file {
				fileRules[d.Code]++
			}
		}
		var fr []ruleCount
		for code, count := range fileRules {
			fr = append(fr, ruleCount{code, count})
		}
		sort.Slice(fr, func(i, j int) bool { return fr[i].count > fr[j].count })

		for j := 0; j < 3 && j < len(fr); j++ {
			result.WriteString(fmt.Sprintf("    %s (%d)\n", fr[j].code, fr[j].count))
		}
	}

	if len(fileCounts) > 10 {
		result.WriteString(fmt.Sprintf("\n... +%d more files\n", len(fileCounts)-10))
	}

	if fixableCount > 0 {
		result.WriteString(fmt.Sprintf("\n💡 Run `ruff check --fix` to auto-fix %d issues\n", fixableCount))
	}

	return result.String()
}

func filterRuffFormat(output string) string {
	var filesToFormat []string
	filesChecked := 0

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		// Count "would reformat" lines (check mode)
		if strings.Contains(lower, "would reformat:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 {
				filesToFormat = append(filesToFormat, strings.TrimSpace(parts[1]))
			}
		}

		// Count total checked files
		if strings.Contains(lower, "left unchanged") {
			parts := strings.Split(trimmed, ",")
			for _, part := range parts {
				if strings.Contains(strings.ToLower(part), "left unchanged") {
					words := strings.Fields(part)
					for i, word := range words {
						if (word == "file" || word == "files") && i > 0 {
							if n, _ := fmt.Sscanf(words[i-1], "%d", &filesChecked); n == 1 {
								break
							}
						}
					}
				}
			}
		}
	}

	outputLower := strings.ToLower(output)

	// Check if all files are formatted
	if len(filesToFormat) == 0 && strings.Contains(outputLower, "left unchanged") {
		return "✓ Ruff format: All files formatted correctly\n"
	}

	var result strings.Builder

	if strings.Contains(outputLower, "would reformat") {
		if len(filesToFormat) == 0 {
			result.WriteString("✓ Ruff format: All files formatted correctly\n")
		} else {
			result.WriteString(fmt.Sprintf("Ruff format: %d files need formatting\n", len(filesToFormat)))
			result.WriteString("═══════════════════════════════════════\n")

			for i := 0; i < 10 && i < len(filesToFormat); i++ {
				result.WriteString(fmt.Sprintf("%d. %s\n", i+1, compactPath(filesToFormat[i])))
			}

			if len(filesToFormat) > 10 {
				result.WriteString(fmt.Sprintf("\n... +%d more files\n", len(filesToFormat)-10))
			}

			if filesChecked > 0 {
				result.WriteString(fmt.Sprintf("\n✓ %d files already formatted\n", filesChecked))
			}

			result.WriteString("\n💡 Run `ruff format` to format these files\n")
		}
	} else {
		result.WriteString(strings.TrimSpace(output) + "\n")
	}

	return result.String()
}

func compactPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")

	if idx := strings.LastIndex(path, "/src/"); idx != -1 {
		return "src/" + path[idx+5:]
	}
	if idx := strings.LastIndex(path, "/lib/"); idx != -1 {
		return "lib/" + path[idx+5:]
	}
	if idx := strings.LastIndex(path, "/tests/"); idx != -1 {
		return "tests/" + path[idx+7:]
	}
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		return path[idx+1:]
	}
	return path
}
