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

var npxCmd = &cobra.Command{
	Use:   "npx [args...]",
	Short: "npx with intelligent routing to specialized filters",
	Long: `Execute npx with intelligent command routing.

Routes common tools to specialized filters:
- tsc, typescript → tokman tsc
- eslint → tokman lint
- prettier → tokman prettier
- prisma → specialized prisma filter
- next → tokman next

Examples:
  tokman npx tsc --noEmit
  tokman npx eslint src/
  tokman npx prisma generate`,
	DisableFlagParsing: true,
	RunE:               runNpx,
}

func init() {
	rootCmd.AddCommand(npxCmd)
}

func runNpx(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("npx requires a command argument")
	}

	// Intelligent routing: delegate to specialized filters
	switch args[0] {
	case "tsc", "typescript":
		return runTscCommand(args[1:])
	case "eslint":
		return runLintCommand(args[1:])
	case "prettier":
		return runPrettierCommand(args[1:])
	case "prisma":
		return runPrismaCommand(args[1:])
	case "next":
		return runNextCommand(args[1:])
	default:
		return runNpxPassthrough(args)
	}
}

func runTscCommand(args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: npx tsc %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("npx", append([]string{"tsc"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterTscOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("npx tsc %s", strings.Join(args, " ")), "tokman npx tsc", originalTokens, filteredTokens)

	return err
}

func runLintCommand(args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: npx eslint %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("npx", append([]string{"eslint"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterEslintJSON(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("npx eslint %s", strings.Join(args, " ")), "tokman npx eslint", originalTokens, filteredTokens)

	return err
}

func runPrettierCommand(args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: npx prettier %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("npx", append([]string{"prettier"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterPrettierOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("npx prettier %s", strings.Join(args, " ")), "tokman npx prettier", originalTokens, filteredTokens)

	return err
}

func runPrismaCommand(args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: npx prisma %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("npx", append([]string{"prisma"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterPrismaOutputCompact(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("npx prisma %s", strings.Join(args, " ")), "tokman npx prisma", originalTokens, filteredTokens)

	return err
}

func runNextCommand(args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: npx next %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("npx", append([]string{"next"}, args...)...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterNextOutputCompact(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("npx next %s", strings.Join(args, " ")), "tokman npx next", originalTokens, filteredTokens)

	return err
}

func runNpxPassthrough(args []string) error {
	timer := tracking.Start()

	if verbose > 0 {
		fmt.Fprintf(os.Stderr, "Running: npx %s\n", strings.Join(args, " "))
	}

	execCmd := exec.Command("npx", args...)
	output, err := execCmd.CombinedOutput()
	raw := string(output)

	filtered := filterNpmOutput(raw)
	fmt.Println(filtered)

	originalTokens := filter.EstimateTokens(raw)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("npx %s", strings.Join(args, " ")), "tokman npx", originalTokens, filteredTokens)

	return err
}
