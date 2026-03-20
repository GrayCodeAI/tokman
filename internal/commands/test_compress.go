package commands

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var testAll bool

var compressTestCmd = &cobra.Command{
	Use:   "compress-test [command...]",
	Short: "Test compression quality on a command",
	Long:  `Run a command and verify compression preserves critical information.`,
	RunE:  runCompressTest,
}

func init() {
	compressTestCmd.Flags().BoolVar(&testAll, "all", false, "test all presets")
	rootCmd.AddCommand(compressTestCmd)
}

func runCompressTest(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: tokman test <command> [args...]")
	}

	// Execute command
	exePath, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", args[0])
	}

	execCmd := exec.Command(exePath, args[1:]...)
	output, _ := execCmd.CombinedOutput()
	rawOutput := string(output)

	originalTokens := core.EstimateTokens(rawOutput)

	fmt.Printf("Testing: %s\n", strings.Join(args, " "))
	fmt.Printf("Original: ~%d tokens\n\n", originalTokens)

	equiv := filter.NewSemanticEquivalence()

	presets := []filter.PipelinePreset{filter.PresetFast, filter.PresetBalanced, filter.PresetFull}

	passed := 0
	failed := 0

	for _, preset := range presets {
		cfg := filter.PresetConfig(preset, filter.ModeMinimal)
		pipeline := filter.NewPipelineCoordinator(cfg)
		compressed, stats := pipeline.Process(rawOutput)
		report := equiv.Check(rawOutput, compressed)

		status := "PASS"
		if !report.IsGood() {
			status = "FAIL"
			failed++
		} else {
			passed++
		}

		fmt.Printf("  %s %-12s saved=%d (%.1f%%) quality=%.0f%% errors=%v nums=%v\n",
			status, preset, stats.TotalSaved, stats.ReductionPercent,
			report.Score*100, report.ErrorPreserved, report.NumbersPreserved)
	}

	fmt.Printf("\nResults: %d passed, %d failed\n", passed, failed)

	if failed > 0 {
		return fmt.Errorf("compression quality test failed")
	}
	return nil
}
