package system

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/config"
)

var (
	cleanDays       int
	cleanAll        bool
	cleanTee        bool
	cleanReversible bool
)

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean up old tracking data and caches",
	Long: `Remove old tracking records, tee files, and reversible compression entries.

Examples:
  tokman clean           # Clean data older than 30 days
  tokman clean -d 7      # Clean data older than 7 days
  tokman clean --all     # Remove all tracking data
  tokman clean --tee     # Remove all tee files`,
	RunE: runClean,
}

func init() {
	cleanCmd.Flags().IntVarP(&cleanDays, "days", "d", 30, "remove data older than N days")
	cleanCmd.Flags().BoolVar(&cleanAll, "all", false, "remove all tracking data")
	cleanCmd.Flags().BoolVar(&cleanTee, "tee", false, "remove all tee files")
	cleanCmd.Flags().BoolVar(&cleanReversible, "reversible", false, "remove reversible entries")
	registry.Add(func() { registry.Register(cleanCmd) })
}

func runClean(cmd *cobra.Command, args []string) error {
	totalRemoved := 0

	// Clean tracking data
	if !cleanTee && !cleanReversible || cleanAll {
		tracker, err := shared.OpenTracker()
		if err == nil {
			defer tracker.Close()
			if cleanAll {
				removed, _ := tracker.CleanupWithRetention(0)
				totalRemoved += int(removed)
				fmt.Printf("  Removed %d tracking records\n", removed)
			} else {
				removed, _ := tracker.CleanupWithRetention(cleanDays)
				totalRemoved += int(removed)
				fmt.Printf("  Removed %d tracking records (older than %d days)\n", removed, cleanDays)
			}

			// Vacuum database
			if err := tracker.Vacuum(); err == nil {
				fmt.Println("  Database vacuumed")
			}

			// Show database size
			if size, err := tracker.DatabaseSize(); err == nil {
				sizeMB := float64(size) / 1024 / 1024
				fmt.Printf("  Database size: %.1fMB\n", sizeMB)
			}
		}
	}

	// Clean tee files
	if cleanTee || cleanAll {
		teeDir := cleanTeeDir()
		if entries, err := os.ReadDir(teeDir); err == nil {
			cleaned := 0
			for _, e := range entries {
				if !e.IsDir() {
					if err := os.Remove(filepath.Join(teeDir, e.Name())); err != nil {
						fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", e.Name(), err)
					} else {
						cleaned++
					}
				}
			}
			fmt.Printf("  Removed %d tee files\n", cleaned)
			totalRemoved += cleaned
		}
	}

	// Clean reversible entries
	if cleanReversible || cleanAll {
		revDir := cleanReversibleDir()
		if entries, err := os.ReadDir(revDir); err == nil {
			cutoff := time.Now().AddDate(0, 0, -cleanDays)
			removed := 0
			for _, e := range entries {
				if cleanAll {
					if err := os.Remove(filepath.Join(revDir, e.Name())); err != nil {
						fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", e.Name(), err)
					} else {
						removed++
					}
					continue
				}

				if info, err := e.Info(); err == nil && info.ModTime().Before(cutoff) {
					if err := os.Remove(filepath.Join(revDir, e.Name())); err != nil {
						fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", e.Name(), err)
					} else {
						removed++
					}
				}
			}
			fmt.Printf("  Removed %d reversible entries\n", removed)
			totalRemoved += removed
		}
	}

	fmt.Printf("\nTotal items removed: %d\n", totalRemoved)
	return nil
}

func cleanTeeDir() string {
	return filepath.Join(config.DataPath(), "tee")
}

func cleanReversibleDir() string {
	return filepath.Join(config.DataPath(), "reversible")
}
