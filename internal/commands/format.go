package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var formatCmd = &cobra.Command{
	Use:   "format [args...]",
	Short: "Universal format checker (prettier, black, ruff format)",
	Long: `Auto-detect and run the appropriate formatter for your project.

Detects formatter from project files:
- .prettierrc, .prettierrc.json → prettier
- pyproject.toml, setup.cfg → black/ruff format
- go.mod → gofmt

Examples:
  tokman format --check .
  tokman format --write .`,
	DisableFlagParsing: true,
	RunE:               runFormat,
}

func init() {
	rootCmd.AddCommand(formatCmd)
}

func runFormat(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		args = []string{"--check", "."}
	}

	// Detect formatter
	formatter := detectFormatter()
	if verbose {
		fmt.Fprintf(os.Stderr, "Detected formatter: %s\n", formatter)
	}

	var execCmd *exec.Cmd
	switch formatter {
	case "prettier":
		execCmd = exec.Command("prettier", args...)
	case "black":
		execCmd = exec.Command("black", args...)
	case "ruff":
		execCmd = exec.Command("ruff", append([]string{"format"}, args...)...)
	case "gofmt":
		execCmd = exec.Command("gofmt", args...)
	default:
		// Fallback to prettier
		execCmd = exec.Command("prettier", args...)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Running: %s %s\n", formatter, strings.Join(args, " "))
	}

	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterFormatOutput(raw, formatter)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("%s %s", formatter, strings.Join(args, " ")), "tokman format", originalTokens, filteredTokens)

	return err
}

func detectFormatter() string {
	// Check for prettier config
	if _, err := os.Stat(".prettierrc"); err == nil {
		return "prettier"
	}
	if _, err := os.Stat(".prettierrc.json"); err == nil {
		return "prettier"
	}
	if _, err := os.Stat(".prettierrc.js"); err == nil {
		return "prettier"
	}

	// Check for Python formatters
	if _, err := os.Stat("pyproject.toml"); err == nil {
		// Check if ruff is configured
		content, _ := os.ReadFile("pyproject.toml")
		if strings.Contains(string(content), "[tool.ruff") {
			return "ruff"
		}
		return "black"
	}
	if _, err := os.Stat("setup.cfg"); err == nil {
		return "black"
	}

	// Check for Go
	if _, err := os.Stat("go.mod"); err == nil {
		return "gofmt"
	}

	// Check for package.json (prettier is common)
	if _, err := os.Stat("package.json"); err == nil {
		return "prettier"
	}

	return "prettier"
}

func filterFormatOutput(raw, formatter string) string {
	if raw == "" {
		return "✅ All files formatted correctly"
	}

	lines := strings.Split(raw, "\n")
	var files []string
	var errors []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Detect files that need formatting
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") || strings.Contains(lower, "failed") {
			errors = append(errors, truncateLine(line, 100))
		} else if line != "" && !strings.HasPrefix(line, "Checking") {
			files = append(files, truncateLine(line, 80))
		}
	}

	var result []string

	if len(errors) > 0 {
		result = append(result, fmt.Sprintf("❌ Errors (%d):", len(errors)))
		for _, e := range errors {
			result = append(result, fmt.Sprintf("   %s", e))
		}
	}

	if len(files) > 0 {
		result = append(result, fmt.Sprintf("📝 Files (%d):", len(files)))
		for i, f := range files {
			if i >= 20 {
				result = append(result, fmt.Sprintf("   ... +%d more", len(files)-20))
				break
			}
			result = append(result, fmt.Sprintf("   %s", f))
		}
	}

	if len(result) == 0 {
		return "✅ All files formatted correctly"
	}
	return strings.Join(result, "\n")
}
