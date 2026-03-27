package filtercmd

// watch_cmd.go implements Task #162: --watch mode for continuous compression.
// Usage:
//
//	tokman watch <file>
//	tokman watch --mode aggressive --interval 2s file.txt
//	tokman watch --outdir compressed/ file.txt

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var watchCmd = &cobra.Command{
	Use:   "watch <file>",
	Short: "Watch a file and re-compress on changes",
	Long: `Poll a file for modification time changes and re-compress it each time.
Compressed output is written to stdout or --outdir.

Examples:
  tokman watch input.txt
  tokman watch --mode aggressive --interval 2s input.txt
  tokman watch --outdir out/ input.txt`,
	Args: cobra.ExactArgs(1),
	RunE: runWatch,
}

var (
	watchMode     string
	watchInterval time.Duration
	watchOutDir   string
)

func init() {
	watchCmd.Flags().StringVar(&watchMode, "mode", "minimal", "Compression mode: none|minimal|aggressive")
	watchCmd.Flags().DurationVar(&watchInterval, "interval", time.Second, "Poll interval (e.g. 1s, 500ms)")
	watchCmd.Flags().StringVar(&watchOutDir, "outdir", "", "Output directory (default: stdout)")
	registry.Add(func() { registry.Register(watchCmd) })
}

func runWatch(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	mode := filter.ModeMinimal
	switch strings.ToLower(watchMode) {
	case "none":
		mode = filter.ModeNone
	case "aggressive":
		mode = filter.ModeAggressive
	}

	if watchOutDir != "" {
		if err := os.MkdirAll(watchOutDir, 0750); err != nil {
			return fmt.Errorf("watch: mkdir %s: %w", watchOutDir, err)
		}
	}

	// Get initial mtime so we don't compress on startup.
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("watch: stat %s: %w", filePath, err)
	}
	lastMod := info.ModTime()

	// Set up signal handling for graceful exit.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(watchInterval)
	defer ticker.Stop()

	var totalEvents int
	var totalOrigTokens, totalFinalTokens int

	fmt.Fprintf(os.Stderr, "[tokman watch] Watching %s (interval: %s)\n", filePath, watchInterval)

	for {
		select {
		case <-sigCh:
			// Print summary and exit.
			fmt.Fprintf(os.Stderr, "\n[tokman watch] Stopped. Summary:\n")
			fmt.Fprintf(os.Stderr, "  Events:       %d\n", totalEvents)
			fmt.Fprintf(os.Stderr, "  Total input:  %d tokens\n", totalOrigTokens)
			fmt.Fprintf(os.Stderr, "  Total output: %d tokens\n", totalFinalTokens)
			if totalOrigTokens > 0 {
				saved := totalOrigTokens - totalFinalTokens
				pct := float64(saved) / float64(totalOrigTokens) * 100
				fmt.Fprintf(os.Stderr, "  Saved:        %s tokens (%.1f%%)\n",
					formatSaved(saved), pct)
			}
			return nil

		case <-ticker.C:
			info, err := os.Stat(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[tokman watch] stat error: %v\n", err)
				continue
			}

			if !info.ModTime().After(lastMod) {
				continue
			}
			lastMod = info.ModTime()

			fmt.Fprintf(os.Stderr, "Recompressing %s...\n", filePath)

			raw, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[tokman watch] read error: %v\n", err)
				continue
			}

			input := string(raw)
			if input == "" {
				continue
			}

			pipeline := filter.NewPipelineCoordinator(filter.PipelineConfig{
				Mode:             mode,
				NgramEnabled:     true,
				EnableCompaction: true,
			})
			result, stats := pipeline.Process(input)

			origTokens := core.EstimateTokens(input)
			totalEvents++
			totalOrigTokens += origTokens
			totalFinalTokens += stats.FinalTokens

			if watchOutDir != "" {
				outPath := filepath.Join(watchOutDir, filepath.Base(filePath))
				if err := os.WriteFile(outPath, []byte(result), 0600); err != nil {
					fmt.Fprintf(os.Stderr, "[tokman watch] write error: %v\n", err)
				}
			} else {
				fmt.Print(result)
			}
		}
	}
}
