package filtercmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

// compressCmd implements Task #187: compression pipeline as a Unix pipe filter.
// Usage:
//
//	echo "some text" | tokman compress
//	cat big-output.txt | tokman compress --mode aggressive
//	tokman compress --file input.txt --stats
var compressCmd = &cobra.Command{
	Use:   "compress [file]",
	Short: "Compress text via stdin or file (Unix pipe filter)",
	Long: `Read text from stdin (or a file) and write compressed output to stdout.
Designed for use in Unix pipelines:

  cat output.txt | tokman compress | llm-tool
  tokman compress --mode aggressive < big_file.txt`,
	RunE: runCompress,
}

var (
	compressMode   string
	compressStats  bool
	compressFile   string
	compressBudget int
)

func init() {
	compressCmd.Flags().StringVar(&compressMode, "mode", "minimal", "Compression mode: none|minimal|aggressive")
	compressCmd.Flags().BoolVar(&compressStats, "stats", false, "Print compression stats to stderr")
	compressCmd.Flags().StringVar(&compressFile, "file", "", "Input file (default: stdin)")
	compressCmd.Flags().IntVar(&compressBudget, "budget", 0, "Target token budget (0 = no limit)")
	registry.Add(func() { registry.Register(compressCmd) })
}

func runCompress(cmd *cobra.Command, args []string) error {
	// Determine input source: positional arg > --file > stdin
	inputFile := compressFile
	if len(args) > 0 {
		inputFile = args[0]
	}

	var raw []byte
	var err error
	if inputFile != "" {
		raw, err = os.ReadFile(inputFile)
		if err != nil {
			return fmt.Errorf("compress: read %s: %w", inputFile, err)
		}
	} else {
		raw, err = io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("compress: read stdin: %w", err)
		}
	}

	input := string(raw)
	if input == "" {
		return nil
	}

	mode := filter.ModeMinimal
	switch strings.ToLower(compressMode) {
	case "none":
		mode = filter.ModeNone
	case "aggressive":
		mode = filter.ModeAggressive
	}

	budget := compressBudget
	if budget == 0 {
		budget = shared.GetTokenBudget()
	}

	pipeline := filter.NewPipelineCoordinator(filter.PipelineConfig{
		Mode:                mode,
		QueryIntent:         shared.GetQueryIntent(),
		Budget:              budget,
		SessionTracking:     false,
		NgramEnabled:        true,
		EnableCompaction:    true,
		EnableAttribution:   true,
		EnableH2O:           true,
		EnableAttentionSink: true,
	})

	result, stats := pipeline.Process(input)

	// Write compressed output to stdout
	fmt.Print(result)

	// Optionally print stats to stderr
	if compressStats {
		origTokens := core.EstimateTokens(input)
		pct := stats.ReductionPercent
		fmt.Fprintf(os.Stderr, "[tokman] %d → %d tokens (%.1f%% reduction, saved %s)\n",
			origTokens,
			stats.FinalTokens,
			pct,
			formatSaved(origTokens-stats.FinalTokens),
		)
	}

	return nil
}

// formatSaved returns a human-readable token count.
func formatSaved(n int) string {
	if n < 0 {
		return "0"
	}
	if n >= 1000 {
		return strconv.FormatFloat(float64(n)/1000, 'f', 1, 64) + "K"
	}
	return strconv.Itoa(n)
}
