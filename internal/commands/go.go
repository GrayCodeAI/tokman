package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var goCmd = &cobra.Command{
	Use:   "go [args...]",
	Short: "Go commands with compact output",
	Long: `Execute Go commands with token-optimized output.

Provides compact output for test, build, vet, and other go commands.

Examples:
  tokman go test ./...
  tokman go build ./...
  tokman go vet ./...`,
	DisableFlagParsing: true,
	RunE:               runGo,
}

func init() {
	rootCmd.AddCommand(goCmd)
}

func runGo(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		args = []string{"help"}
	}

	// Route to specialized handlers
	switch args[0] {
	case "test":
		return runGoTestCmd(args[1:])
	case "build":
		return runGoBuildCmd(args[1:])
	case "vet":
		return runGoVet(args[1:])
	default:
		return runGoPassthrough(args)
	}
}

func runGoTestCmd(args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: go test %s\n", strings.Join(args, " "))
	}

	// Use -json for structured output
	jsonArgs := append([]string{"test", "-json"}, args...)
	execCmd := exec.Command("go", jsonArgs...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGoTestOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("go test %s", strings.Join(args, " ")), "tokman go test", originalTokens, filteredTokens)

	return err
}

func runGoBuildCmd(args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: go build %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("go", append([]string{"build"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGoBuildOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("go build %s", strings.Join(args, " ")), "tokman go build", originalTokens, filteredTokens)

	return err
}

func runGoVet(args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: go vet %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("go", append([]string{"vet"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterGoVetOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("go vet %s", strings.Join(args, " ")), "tokman go vet", originalTokens, filteredTokens)

	return err
}

func runGoPassthrough(args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: go %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("go", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	// Basic filtering
	filtered := filterGoOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("go %s", strings.Join(args, " ")), "tokman go", originalTokens, filteredTokens)

	return err
}

// Filter functions

type GoTestEvent struct {
	Time    string `json:"Time"`
	Action  string `json:"Action"`
	Package string `json:"Package"`
	Test    string `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
	Output  string `json:"Output"`
}

func filterGoTestOutput(raw string) string {
	var passed, failed, skipped int
	var failures []string
	var packageResults = make(map[string][]string)

	for _, line := range strings.Split(raw, "\n") {
		if line == "" {
			continue
		}

		var event GoTestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		switch event.Action {
		case "pass":
			if event.Test == "" {
				// Package pass
				packageResults[event.Package] = append(packageResults[event.Package], "✅ PASS")
			} else {
				passed++
			}
		case "fail":
			if event.Test == "" {
				// Package fail
				packageResults[event.Package] = append(packageResults[event.Package], "❌ FAIL")
			} else {
				failed++
				failures = append(failures, fmt.Sprintf("%s.%s", event.Package, event.Test))
			}
		case "skip":
			skipped++
		}
	}

	var result []string
	result = append(result, "📋 Go Test Results:")
	result = append(result, fmt.Sprintf("   ✅ %d passed", passed))
	if failed > 0 {
		result = append(result, fmt.Sprintf("   ❌ %d failed", failed))
	}
	if skipped > 0 {
		result = append(result, fmt.Sprintf("   ⏭️  %d skipped", skipped))
	}

	// Package summary
	if len(packageResults) > 0 {
		result = append(result, "")
		result = append(result, "Packages:")
		for pkg, status := range packageResults {
			result = append(result, fmt.Sprintf("   %s: %s", pkg, strings.Join(status, ", ")))
		}
	}

	if len(failures) > 0 {
		result = append(result, "")
		result = append(result, "Failures:")
		for i, f := range failures {
			if i >= 10 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(failures)-10))
				break
			}
			result = append(result, fmt.Sprintf("   • %s", f))
		}
	}

	return strings.Join(result, "\n")
}

func filterGoBuildOutput(raw string) string {
	if raw == "" {
		return "✅ Build successful"
	}

	lines := strings.Split(raw, "\n")
	var errors []string
	var warnings []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") {
			errors = append(errors, truncateLine(line, 100))
		} else if strings.Contains(lower, "warning") {
			warnings = append(warnings, truncateLine(line, 100))
		}
	}

	var result []string
	if len(errors) > 0 {
		result = append(result, fmt.Sprintf("❌ Errors (%d):", len(errors)))
		for _, e := range errors {
			result = append(result, fmt.Sprintf("   %s", e))
		}
	}

	if len(warnings) > 0 {
		result = append(result, fmt.Sprintf("⚠️  Warnings (%d):", len(warnings)))
		for _, w := range warnings {
			result = append(result, fmt.Sprintf("   %s", w))
		}
	}

	if len(result) == 0 && raw != "" {
		// No errors/warnings detected, but output exists
		return raw
	}

	if len(result) == 0 {
		return "✅ Build successful"
	}
	return strings.Join(result, "\n")
}

func filterGoVetOutput(raw string) string {
	if raw == "" {
		return "✅ No vet issues found"
	}

	lines := strings.Split(raw, "\n")
	var issues []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			issues = append(issues, truncateLine(line, 100))
		}
	}

	if len(issues) == 0 {
		return "✅ No vet issues found"
	}

	var result []string
	result = append(result, fmt.Sprintf("⚠️  Vet Issues (%d):", len(issues)))
	for i, issue := range issues {
		if i >= 15 {
			result = append(result, fmt.Sprintf("   ... +%d more", len(issues)-15))
			break
		}
		result = append(result, fmt.Sprintf("   %s", issue))
	}
	return strings.Join(result, "\n")
}

func filterGoOutput(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, truncateLine(line, 120))
		}
	}

	if len(result) > 30 {
		return strings.Join(result[:30], "\n") + fmt.Sprintf("\n... (%d more lines)", len(result)-30)
	}
	return strings.Join(result, "\n")
}
