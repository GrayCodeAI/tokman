package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var undoCmd = &cobra.Command{
	Use:   "undo",
	Short: "Show original output from last tee save",
	Long: `Display the most recent tee file containing original unfiltered output.
Tee files are saved when commands fail or produce suspicious results.

Use --list to see all available tee files.`,
	RunE: runUndo,
}

var undoList bool

func init() {
	undoCmd.Flags().BoolVarP(&undoList, "list", "l", false, "list all tee files")
	rootCmd.AddCommand(undoCmd)
}

func runUndo(cmd *cobra.Command, args []string) error {
	teeDir := getTeeDirFromHome()
	if teeDir == "" {
		return fmt.Errorf("cannot determine tee directory")
	}

	entries, err := os.ReadDir(teeDir)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("no tee files found (tee directory: %s)", teeDir)
	}

	// Sort by name (timestamp prefix) descending
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() > entries[j].Name()
	})

	if undoList {
		fmt.Printf("Tee files in %s:\n\n", teeDir)
		for _, e := range entries {
			info, _ := e.Info()
			size := ""
			if info != nil {
				size = fmt.Sprintf(" (%d bytes)", info.Size())
			}
			fmt.Printf("  %s%s\n", e.Name(), size)
		}
		return nil
	}

	// Show the most recent tee file
	latest := entries[0]
	path := filepath.Join(teeDir, latest.Name())
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read tee file: %w", err)
	}

	content := string(data)
	// Parse tee file format: command slug from filename
	parts := strings.SplitN(latest.Name(), "_", 2)
	cmdName := ""
	if len(parts) > 1 {
		cmdName = strings.TrimSuffix(parts[1], ".log")
		cmdName = strings.ReplaceAll(cmdName, "_", " ")
	}

	fmt.Printf("Last tee save: %s\n", latest.Name())
	if cmdName != "" {
		fmt.Printf("Command: %s\n", cmdName)
	}
	fmt.Printf("File: %s\n\n", path)
	fmt.Println(content)

	return nil
}

func getTeeDirFromHome() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(homeDir, ".local", "share", "tokman", "tee")
}
