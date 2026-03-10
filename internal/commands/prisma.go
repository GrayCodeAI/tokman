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

var prismaCmd = &cobra.Command{
	Use:   "prisma [args...]",
	Short: "Prisma commands with compact output (no ASCII art)",
	Long: `Execute Prisma commands with token-optimized output.

Strips ASCII art banners and provides compact summaries.

Examples:
  tokman prisma generate
  tokman prisma migrate dev
  tokman prisma db push`,
	DisableFlagParsing: true,
	RunE:               runPrisma,
}

func init() {
	rootCmd.AddCommand(prismaCmd)
}

func runPrisma(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	if len(args) == 0 {
		args = []string{"--help"}
	}

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: prisma %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("prisma", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterPrismaOutputCompact(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("prisma %s", strings.Join(args, " ")), "tokman prisma", originalTokens, filteredTokens)

	return err
}

func filterPrismaOutputCompact(raw string) string {
	lines := strings.Split(raw, "\n")
	var result []string
	var inBanner bool

	for _, line := range lines {
		// Skip ASCII art banners
		if strings.Contains(line, "│") || strings.Contains(line, "─") ||
			strings.Contains(line, "╭") || strings.Contains(line, "╰") ||
			strings.Contains(line, "▼") || strings.Contains(line, "▲") {
			inBanner = true
			continue
		}

		// Skip empty lines after banners
		if inBanner && strings.TrimSpace(line) == "" {
			inBanner = false
			continue
		}

		inBanner = false
		line = strings.TrimSpace(line)

		// Skip progress indicators
		if strings.HasPrefix(line, "✔") || strings.HasPrefix(line, "✅") {
			continue
		}

		// Keep meaningful output
		if line != "" {
			// Truncate long lines but keep important info
			result = append(result, truncateLine(line, 100))
		}
	}

	// Compact output
	if len(result) == 0 {
		return "✅ Prisma command completed"
	}

	// Remove consecutive empty lines
	var compact []string
	for i, line := range result {
		if i > 0 && line == "" && result[i-1] == "" {
			continue
		}
		compact = append(compact, line)
	}

	if len(compact) > 20 {
		return strings.Join(compact[:20], "\n") + fmt.Sprintf("\n... (%d more lines)", len(compact)-20)
	}
	return strings.Join(compact, "\n")
}
