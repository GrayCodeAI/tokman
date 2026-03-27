package filtercmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/core"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

// batchCmd implements Task #173: multi-file batch compression.
// It compresses one or more files (or a directory glob) in parallel,
// writing results to stdout or an output directory.
var batchCmd = &cobra.Command{
	Use:   "batch <file|glob...>",
	Short: "Compress multiple files in parallel",
	Long: `Compress one or more files (or glob patterns) through the token-reduction pipeline.

Examples:
  tokman batch *.txt
  tokman batch --outdir compressed/ src/**/*.go
  tokman batch --stats file1.txt file2.txt file3.txt`,
	Args: cobra.MinimumNArgs(1),
	RunE: runBatch,
}

var (
	batchMode    string
	batchOutDir  string
	batchStats   bool
	batchWorkers int
	batchInPlace bool
)

func init() {
	batchCmd.Flags().StringVar(&batchMode, "mode", "minimal", "Compression mode: none|minimal|aggressive")
	batchCmd.Flags().StringVar(&batchOutDir, "outdir", "", "Output directory (default: stdout)")
	batchCmd.Flags().BoolVar(&batchStats, "stats", false, "Print per-file stats to stderr")
	batchCmd.Flags().IntVar(&batchWorkers, "workers", 4, "Parallel workers")
	batchCmd.Flags().BoolVar(&batchInPlace, "in-place", false, "Overwrite input files (use with care)")
	registry.Add(func() { registry.Register(batchCmd) })
}

type batchResult struct {
	path          string
	compressed    string
	origTokens    int
	finalTokens   int
	err           error
}

func runBatch(cmd *cobra.Command, args []string) error {
	// Expand globs and collect all file paths
	var paths []string
	for _, arg := range args {
		matches, err := filepath.Glob(arg)
		if err != nil {
			return fmt.Errorf("batch: glob %q: %w", arg, err)
		}
		if matches == nil {
			// Not a glob — treat as literal path
			paths = append(paths, arg)
		} else {
			for _, m := range matches {
				info, err := os.Stat(m)
				if err != nil {
					continue
				}
				if info.IsDir() {
					// Walk directory
					_ = filepath.WalkDir(m, func(p string, d fs.DirEntry, err error) error {
						if err == nil && !d.IsDir() {
							paths = append(paths, p)
						}
						return nil
					})
				} else {
					paths = append(paths, m)
				}
			}
		}
	}

	if len(paths) == 0 {
		return fmt.Errorf("batch: no files found")
	}

	mode := filter.ModeMinimal
	switch strings.ToLower(batchMode) {
	case "none":
		mode = filter.ModeNone
	case "aggressive":
		mode = filter.ModeAggressive
	}

	// Worker pool
	workers := batchWorkers
	if workers < 1 {
		workers = 1
	}
	jobs := make(chan string, len(paths))
	results := make(chan batchResult, len(paths))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			pipeline := filter.NewPipelineCoordinator(filter.PipelineConfig{
				Mode:             mode,
				QueryIntent:      shared.GetQueryIntent(),
				Budget:           shared.GetTokenBudget(),
				NgramEnabled:     true,
				EnableCompaction: true,
			})
			for path := range jobs {
				raw, err := os.ReadFile(path)
				if err != nil {
					results <- batchResult{path: path, err: err}
					continue
				}
				input := string(raw)
				orig := core.EstimateTokens(input)
				compressed, stats := pipeline.Process(input)
				results <- batchResult{
					path:        path,
					compressed:  compressed,
					origTokens:  orig,
					finalTokens: stats.FinalTokens,
				}
			}
		}()
	}

	for _, p := range paths {
		jobs <- p
	}
	close(jobs)
	wg.Wait()
	close(results)

	// Setup output dir if needed
	if batchOutDir != "" {
		if err := os.MkdirAll(batchOutDir, 0750); err != nil {
			return fmt.Errorf("batch: mkdir %s: %w", batchOutDir, err)
		}
	}

	var totalOrig, totalFinal, errCount int
	for r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "[tokman batch] error: %s: %v\n", r.path, r.err)
			errCount++
			continue
		}

		totalOrig += r.origTokens
		totalFinal += r.finalTokens

		switch {
		case batchInPlace:
			if err := os.WriteFile(r.path, []byte(r.compressed), 0600); err != nil {
				fmt.Fprintf(os.Stderr, "[tokman batch] write error: %s: %v\n", r.path, err)
				errCount++
			}
		case batchOutDir != "":
			outPath := filepath.Join(batchOutDir, filepath.Base(r.path))
			if err := os.WriteFile(outPath, []byte(r.compressed), 0600); err != nil {
				fmt.Fprintf(os.Stderr, "[tokman batch] write error: %s: %v\n", outPath, err)
				errCount++
			}
		default:
			// Stdout with file markers
			fmt.Printf("=== %s ===\n%s\n", r.path, r.compressed)
		}

		if batchStats {
			saved := r.origTokens - r.finalTokens
			var pct float64
			if r.origTokens > 0 {
				pct = float64(saved) / float64(r.origTokens) * 100
			}
			fmt.Fprintf(os.Stderr, "[tokman] %-40s %d→%d tokens (%.1f%% reduction)\n",
				r.path, r.origTokens, r.finalTokens, pct)
		}
	}

	if batchStats {
		saved := totalOrig - totalFinal
		var pct float64
		if totalOrig > 0 {
			pct = float64(saved) / float64(totalOrig) * 100
		}
		fmt.Fprintf(os.Stderr, "[tokman] TOTAL: %d files, %d→%d tokens (%.1f%% reduction)\n",
			len(paths)-errCount, totalOrig, totalFinal, pct)
	}

	if errCount > 0 {
		return fmt.Errorf("batch: %d file(s) failed", errCount)
	}
	return nil
}
