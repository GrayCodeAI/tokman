package sessioncmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/config"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var snapshotDir string

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Save and manage compression snapshots",
	Long:  `Save snapshots of compressed outputs for later comparison.`,
}

var snapshotSaveCmd = &cobra.Command{
	Use:   "save <name>",
	Short: "Save a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE:  runSnapshotSave,
}

var snapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved snapshots",
	RunE:  runSnapshotList,
}

var snapshotShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE:  runSnapshotShow,
}

func init() {
	snapshotCmd.AddCommand(snapshotSaveCmd)
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotShowCmd)
	registry.Add(func() { registry.Register(snapshotCmd) })
}

func sanitizeSnapshotName(name string) string {
	// Remove null bytes
	name = strings.ReplaceAll(name, "\x00", "")
	// Remove path traversal attempts
	name = strings.ReplaceAll(name, "..", "")
	// Remove path separators
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	// Take only the basename
	name = filepath.Base(name)
	// Final safety: reject empty names
	if name == "" || name == "." {
		return "unnamed"
	}
	return name
}

func getSnapshotDir() string {
	dir := filepath.Join(config.DataPath(), "snapshots")
	if err := os.MkdirAll(dir, 0700); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to create directory: %v\n", err)
	}
	return dir
}

func runSnapshotSave(cmd *cobra.Command, args []string) error {
	name := sanitizeSnapshotName(args[0])
	store := filter.NewReversibleStore()
	entries, err := store.ListRecent(1)
	if err != nil {
		return fmt.Errorf("failed to list recent compressions: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("no recent compression to snapshot")
	}

	entry := entries[0]
	snapshotFile := filepath.Join(getSnapshotDir(), name+".txt")
	content := fmt.Sprintf("# Snapshot: %s\n# Command: %s\n# Time: %s\n# Original: %d chars\n# Compressed: %d chars\n\n%s\n",
		name, entry.Command, time.Now().Format(time.RFC3339),
		len(entry.Original), len(entry.Compressed), entry.Compressed)

	if err := os.WriteFile(snapshotFile, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Printf("Snapshot '%s' saved to %s\n", name, snapshotFile)
	return nil
}

func runSnapshotList(cmd *cobra.Command, args []string) error {
	entries, err := os.ReadDir(getSnapshotDir())
	if err != nil || len(entries) == 0 {
		fmt.Println("No snapshots saved.")
		return nil
	}

	fmt.Println("Saved Snapshots:")
	for _, e := range entries {
		if !e.IsDir() {
			info, err := e.Info()
			if err != nil {
				fmt.Printf("  %s (unable to read info)\n", e.Name())
				continue
			}
			fmt.Printf("  %s (%s, %d bytes)\n", e.Name(), info.ModTime().Format("01-02 15:04"), info.Size())
		}
	}
	return nil
}

func runSnapshotShow(cmd *cobra.Command, args []string) error {
	name := sanitizeSnapshotName(args[0])
	path := filepath.Join(getSnapshotDir(), name+".txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("snapshot '%s' not found", name)
	}
	fmt.Print(string(data))
	return nil
}
