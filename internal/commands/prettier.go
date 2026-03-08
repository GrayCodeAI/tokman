package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var prettierCmd = &cobra.Command{
	Use:   "prettier [args...]",
	Short: "Prettier formatter with filtered output",
	Long: `Prettier formatter with token-optimized output.

Shows files that need formatting in check mode.

Examples:
  tokman prettier --check .
  tokman prettier --write src/
  tokman prettier --check "**/*.{ts,tsx}"`,
	RunE: runPrettier,
}

func init() {
	rootCmd.AddCommand(prettierCmd)
}

func runPrettier(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	// Use package manager to run prettier
	prettierPath, err := exec.LookPath("prettier")
	if err != nil {
		prettierPath = "" // Will use npx
	}

	var c *exec.Cmd
	if prettierPath != "" {
		c = exec.Command(prettierPath, args...)
	} else {
		npxArgs := append([]string{"prettier"}, args...)
		c = exec.Command("npx", npxArgs...)
	}
	c.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr

	err = c.Run()
	output := stdout.String() + stderr.String()

	// Handle case where prettier not installed or no output
	hasOutput := strings.TrimSpace(stdout.String()) != ""
	if !hasOutput && err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			fmt.Fprintln(os.Stderr, "Error: prettier not found or produced no output")
		} else {
			fmt.Fprintln(os.Stderr, msg)
		}
		return err
	}

	filtered := filterPrettierOutput(output)

	fmt.Print(filtered)

	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("prettier %s", strings.Join(args, " ")), "tokman prettier", originalTokens, filteredTokens)

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

func filterPrettierOutput(output string) string {
	// Empty output means prettier didn't run
	if strings.TrimSpace(output) == "" {
		return "Error: prettier produced no output\n"
	}

	var filesToFormat []string
	filesChecked := 0
	isCheckMode := true

	for _, line := range strings.Split(output, "\n") {
		trimmed := strings.TrimSpace(line)

		// Detect check mode
		if strings.Contains(trimmed, "Checking formatting") {
			isCheckMode = true
		}

		// Count files that need formatting (check mode)
		if trimmed != "" &&
			!strings.HasPrefix(trimmed, "Checking") &&
			!strings.HasPrefix(trimmed, "All matched") &&
			!strings.HasPrefix(trimmed, "Code style") &&
			!strings.Contains(trimmed, "[warn]") &&
			!strings.Contains(trimmed, "[error]") &&
			(isSourceFile(trimmed)) {
			filesToFormat = append(filesToFormat, trimmed)
		}

		// Count total files checked
		if strings.Contains(trimmed, "All matched files use Prettier") {
			parts := strings.Fields(trimmed)
			if len(parts) > 0 {
				if count, _ := parseInt(parts[0]); count > 0 {
					filesChecked = count
				}
			}
		}
	}

	// Check if all files are formatted
	if len(filesToFormat) == 0 && strings.Contains(output, "All matched files use Prettier") {
		return "✓ Prettier: All files formatted correctly\n"
	}

	// Check if files were written (write mode)
	if strings.Contains(output, "modified") || strings.Contains(output, "formatted") {
		isCheckMode = false
	}

	var result strings.Builder

	if isCheckMode {
		if len(filesToFormat) == 0 {
			result.WriteString("✓ Prettier: All files formatted correctly\n")
		} else {
			result.WriteString(fmt.Sprintf("Prettier: %d files need formatting\n", len(filesToFormat)))
			result.WriteString("═══════════════════════════════════════\n")

			for i, file := range filesToFormat {
				if i >= 10 {
					break
				}
				result.WriteString(fmt.Sprintf("%d. %s\n", i+1, file))
			}

			if len(filesToFormat) > 10 {
				result.WriteString(fmt.Sprintf("\n... +%d more files\n", len(filesToFormat)-10))
			}

			if filesChecked > 0 {
				result.WriteString(fmt.Sprintf("\n✓ %d files already formatted\n", filesChecked-len(filesToFormat)))
			}
		}
	} else {
		// Write mode: show what was formatted
		result.WriteString(fmt.Sprintf("✓ Prettier: %d files formatted\n", len(filesToFormat)))
	}

	return result.String()
}

func isSourceFile(path string) bool {
	exts := []string{".ts", ".tsx", ".js", ".jsx", ".json", ".md", ".css", ".scss", ".html", ".yaml", ".yml"}
	for _, ext := range exts {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}

func parseInt(s string) (int, bool) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err == nil
}
