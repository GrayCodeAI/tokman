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

var lintCmd = &cobra.Command{
	Use:   "lint [linter] [args...]",
	Short: "Universal linter with filtered output",
	Long: `Universal linter with token-optimized output.

Auto-detects and runs the appropriate linter, grouping issues by rule and file.

Supported linters:
  - eslint (JS/TS, via npm/npx)
  - ruff (Python, via pip)
  - pylint (Python, via pip)
  - mypy (Python, via pip)
  - biome (JS/TS, via npm/npx)

Examples:
  tokman lint .
  tokman lint eslint src/
  tokman lint ruff check .
  tokman lint mypy src/`,
	RunE: runLint,
}

func init() {
	rootCmd.AddCommand(lintCmd)
}

// ESLint JSON structures
type EslintMessage struct {
	RuleId   string `json:"ruleId"`
	Severity int    `json:"severity"`
	Message  string `json:"message"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
}

type EslintResult struct {
	FilePath     string         `json:"filePath"`
	Messages     []EslintMessage `json:"messages"`
	ErrorCount   int            `json:"errorCount"`
	WarningCount int            `json:"warningCount"`
}

// Pylint JSON structures
type PylintDiagnostic struct {
	Type      string `json:"type"`
	Module    string `json:"module"`
	Obj       string `json:"obj"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Path      string `json:"path"`
	Symbol    string `json:"symbol"`
	Message   string `json:"message"`
	MessageId string `json:"message-id"`
}

var pythonLinters = map[string]bool{
	"ruff":    true,
	"pylint":  true,
	"mypy":    true,
	"flake8":  true,
}

func runLint(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	// Strip package manager prefixes (npx, bunx, pnpm exec, yarn)
	args = stripPmPrefix(args)

	// Detect linter
	linter, explicit := detectLinter(args)
	linterArgs := args
	if explicit && len(args) > 0 {
		linterArgs = args[1:]
	}

	// Build command
	var c *exec.Cmd
	if pythonLinters[linter] {
		c = exec.Command(linter)
	} else {
		c = packageManagerExec(linter)
	}

	// Add format flags based on linter
	switch linter {
	case "eslint":
		c.Args = append(c.Args, "-f", "json")
	case "ruff":
		hasOutputFormat := false
		for _, arg := range linterArgs {
			if strings.HasPrefix(arg, "--output-format") {
				hasOutputFormat = true
				break
			}
		}
		c.Args = append(c.Args, "check")
		if !hasOutputFormat {
			c.Args = append(c.Args, "--output-format=json")
		}
	case "pylint":
		hasOutputFormat := false
		for _, arg := range linterArgs {
			if strings.HasPrefix(arg, "--output-format") {
				hasOutputFormat = true
				break
			}
		}
		if !hasOutputFormat {
			c.Args = append(c.Args, "--output-format=json2")
		}
	case "mypy":
		// mypy uses default text output
	}

	// Add user arguments
	startIdx := 0
	if linter == "ruff" && len(linterArgs) > 0 && linterArgs[0] == "check" {
		startIdx = 1
	}
	for _, arg := range linterArgs[startIdx:] {
		// Skip --output-format if we already added it
		if (linter == "ruff" || linter == "pylint") && strings.HasPrefix(arg, "--output-format") {
			continue
		}
		c.Args = append(c.Args, arg)
	}

	// Default to current directory if no path specified
	if _, ok := pythonLinters[linter]; ok || linter == "eslint" {
		hasPath := false
		for _, arg := range linterArgs[startIdx:] {
			if !strings.HasPrefix(arg, "-") && !strings.Contains(arg, "=") {
				hasPath = true
				break
			}
		}
		if !hasPath {
			c.Args = append(c.Args, ".")
		}
	}

	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err := c.Run()
	output := stdout.String() + stderr.String()

	// Filter based on linter
	var filtered string
	switch linter {
	case "eslint":
		filtered = filterEslintJSON(stdout.String())
	case "ruff":
		if strings.TrimSpace(stdout.String()) != "" {
			filtered = filterRuffCheckJSON(stdout.String())
		} else {
			filtered = "✓ Ruff: No issues found\n"
		}
	case "pylint":
		filtered = filterPylintJSON(stdout.String())
	case "mypy":
		filtered = filterMypyOutput(stripANSI(output))
	default:
		filtered = filterGenericLint(output)
	}

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("%s %s", linter, strings.Join(args, " ")), fmt.Sprintf("tokman lint %s", linter), originalTokens, filteredTokens)

	if verbose > 0 {
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

func stripPmPrefix(args []string) []string {
	pmNames := map[string]bool{"npx": true, "bunx": true, "pnpm": true, "yarn": true}
	skip := 0
	for _, arg := range args {
		if pmNames[arg] || arg == "exec" {
			skip++
		} else {
			break
		}
	}
	if skip >= len(args) {
		return nil
	}
	return args[skip:]
}

func detectLinter(args []string) (string, bool) {
	isPathOrFlag := len(args) == 0 || strings.HasPrefix(args[0], "-") || strings.Contains(args[0], "/") || strings.Contains(args[0], ".")

	if isPathOrFlag {
		return "eslint", false
	}
	return args[0], true
}

func packageManagerExec(tool string) *exec.Cmd {
	// Try direct tool first
	if path, err := exec.LookPath(tool); err == nil {
		return exec.Command(path)
	}
	// Fallback to npx
	return exec.Command("npx", tool)
}

func filterEslintJSON(output string) string {
	var results []EslintResult
	if err := json.Unmarshal([]byte(output), &results); err != nil {
		return fmt.Sprintf("ESLint output (JSON parse failed: %s)\n%s", err, truncate(output, 500))
	}

	// Count total issues
	totalErrors := 0
	totalWarnings := 0
	for _, r := range results {
		totalErrors += r.ErrorCount
		totalWarnings += r.WarningCount
	}

	filesWithIssues := 0
	for _, r := range results {
		if len(r.Messages) > 0 {
			filesWithIssues++
		}
	}

	if totalErrors == 0 && totalWarnings == 0 {
		return "✓ ESLint: No issues found\n"
	}

	// Group messages by rule
	byRule := make(map[string]int)
	for _, r := range results {
		for _, msg := range r.Messages {
			if msg.RuleId != "" {
				byRule[msg.RuleId]++
			}
		}
	}

	// Group by file
	type fileResult struct {
		path    string
		count   int
		messages []EslintMessage
	}
	var byFile []fileResult
	for _, r := range results {
		if len(r.Messages) > 0 {
			byFile = append(byFile, fileResult{r.FilePath, len(r.Messages), r.Messages})
		}
	}
	sort.Slice(byFile, func(i, j int) bool {
		return byFile[i].count > byFile[j].count
	})

	var result strings.Builder
	result.WriteString(fmt.Sprintf("ESLint: %d errors, %d warnings in %d files\n", totalErrors, totalWarnings, filesWithIssues))
	result.WriteString("═══════════════════════════════════════\n")

	// Show top rules
	type ruleCount struct {
		rule  string
		count int
	}
	var ruleCounts []ruleCount
	for rule, count := range byRule {
		ruleCounts = append(ruleCounts, ruleCount{rule, count})
	}
	sort.Slice(ruleCounts, func(i, j int) bool {
		return ruleCounts[i].count > ruleCounts[j].count
	})

	if len(ruleCounts) > 0 {
		result.WriteString("Top rules:\n")
		for i := 0; i < 10 && i < len(ruleCounts); i++ {
			result.WriteString(fmt.Sprintf("  %s (%dx)\n", ruleCounts[i].rule, ruleCounts[i].count))
		}
		result.WriteString("\n")
	}

	// Show top files
	result.WriteString("Top files:\n")
	for i := 0; i < 10 && i < len(byFile); i++ {
		shortPath := compactPath(byFile[i].path)
		result.WriteString(fmt.Sprintf("  %s (%d issues)\n", shortPath, byFile[i].count))

		// Show top 3 rules in this file
		fileRules := make(map[string]int)
		for _, msg := range byFile[i].messages {
			if msg.RuleId != "" {
				fileRules[msg.RuleId]++
			}
		}
		var fr []ruleCount
		for rule, count := range fileRules {
			fr = append(fr, ruleCount{rule, count})
		}
		sort.Slice(fr, func(i, j int) bool { return fr[i].count > fr[j].count })

		for j := 0; j < 3 && j < len(fr); j++ {
			result.WriteString(fmt.Sprintf("    %s (%d)\n", fr[j].rule, fr[j].count))
		}
	}

	if len(byFile) > 10 {
		result.WriteString(fmt.Sprintf("\n... +%d more files\n", len(byFile)-10))
	}

	return result.String()
}

func filterPylintJSON(output string) string {
	var diagnostics []PylintDiagnostic
	if err := json.Unmarshal([]byte(output), &diagnostics); err != nil {
		return fmt.Sprintf("Pylint output (JSON parse failed: %s)\n%s", err, truncate(output, 500))
	}

	if len(diagnostics) == 0 {
		return "✓ Pylint: No issues found\n"
	}

	// Count by type
	errors := 0
	warnings := 0
	conventions := 0
	refactors := 0
	for _, d := range diagnostics {
		switch d.Type {
		case "error":
			errors++
		case "warning":
			warnings++
		case "convention":
			conventions++
		case "refactor":
			refactors++
		}
	}

	// Count unique files
	fileSet := make(map[string]bool)
	for _, d := range diagnostics {
		fileSet[d.Path] = true
	}
	totalFiles := len(fileSet)

	// Group by symbol (rule code)
	bySymbol := make(map[string]int)
	for _, d := range diagnostics {
		key := fmt.Sprintf("%s (%s)", d.Symbol, d.MessageId)
		bySymbol[key]++
	}

	// Group by file
	byFile := make(map[string]int)
	for _, d := range diagnostics {
		byFile[d.Path]++
	}

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

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Pylint: %d issues in %d files\n", len(diagnostics), totalFiles))

	if errors > 0 || warnings > 0 {
		result.WriteString(fmt.Sprintf("  %d errors, %d warnings", errors, warnings))
		if conventions > 0 || refactors > 0 {
			result.WriteString(fmt.Sprintf(", %d conventions, %d refactors", conventions, refactors))
		}
		result.WriteString("\n")
	}

	result.WriteString("═══════════════════════════════════════\n")

	// Show top symbols (rules)
	type symbolCount struct {
		symbol string
		count  int
	}
	var symbolCounts []symbolCount
	for symbol, count := range bySymbol {
		symbolCounts = append(symbolCounts, symbolCount{symbol, count})
	}
	sort.Slice(symbolCounts, func(i, j int) bool {
		return symbolCounts[i].count > symbolCounts[j].count
	})

	if len(symbolCounts) > 0 {
		result.WriteString("Top rules:\n")
		for i := 0; i < 10 && i < len(symbolCounts); i++ {
			result.WriteString(fmt.Sprintf("  %s (%dx)\n", symbolCounts[i].symbol, symbolCounts[i].count))
		}
		result.WriteString("\n")
	}

	// Show top files
	result.WriteString("Top files:\n")
	for i := 0; i < 10 && i < len(fileCounts); i++ {
		shortPath := compactPath(fileCounts[i].file)
		result.WriteString(fmt.Sprintf("  %s (%d issues)\n", shortPath, fileCounts[i].count))

		// Show top 3 rules in this file
		fileSymbols := make(map[string]int)
		for _, d := range diagnostics {
			if d.Path == fileCounts[i].file {
				key := fmt.Sprintf("%s (%s)", d.Symbol, d.MessageId)
				fileSymbols[key]++
			}
		}
		var fs []symbolCount
		for symbol, count := range fileSymbols {
			fs = append(fs, symbolCount{symbol, count})
		}
		sort.Slice(fs, func(i, j int) bool { return fs[i].count > fs[j].count })

		for j := 0; j < 3 && j < len(fs); j++ {
			result.WriteString(fmt.Sprintf("    %s (%d)\n", fs[j].symbol, fs[j].count))
		}
	}

	if len(fileCounts) > 10 {
		result.WriteString(fmt.Sprintf("\n... +%d more files\n", len(fileCounts)-10))
	}

	return result.String()
}

func filterGenericLint(output string) string {
	var warnings, errors int
	var issues []string

	for _, line := range strings.Split(output, "\n") {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "warning") {
			warnings++
			issues = append(issues, line)
		}
		if strings.Contains(lineLower, "error") && !strings.Contains(lineLower, "0 error") {
			errors++
			issues = append(issues, line)
		}
	}

	if errors == 0 && warnings == 0 {
		return "✓ Lint: No issues found\n"
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("Lint: %d errors, %d warnings\n", errors, warnings))
	result.WriteString("═══════════════════════════════════════\n")

	for i := 0; i < 20 && i < len(issues); i++ {
		result.WriteString(fmt.Sprintf("%s\n", truncate(issues[i], 100)))
	}

	if len(issues) > 20 {
		result.WriteString(fmt.Sprintf("\n... +%d more issues\n", len(issues)-20))
	}

	return result.String()
}
