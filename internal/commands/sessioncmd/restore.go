package sessioncmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/filter"
)

var restoreList bool

var restoreCmd = &cobra.Command{
	Use:   "restore [hash]",
	Short: "Restore compressed output to original",
	Long: `Restore a previously compressed output to its original form.

When compression is run with --reversible, the original output is stored
and can be retrieved using the hash prefix shown during compression.

Examples:
  tokman restore abc123        # Restore by hash prefix
  tokman restore --list        # List recent reversible entries
  tokman restore abc123 --show # Show both original and compressed`,
	Args: cobra.MaximumNArgs(1),
	RunE: runRestore,
}

func init() {
	restoreCmd.Flags().BoolVarP(&restoreList, "list", "l", false, "List recent reversible entries")
	registry.Add(func() { registry.Register(restoreCmd) })
}

func runRestore(cmd *cobra.Command, args []string) error {
	store := filter.NewReversibleStore()

	if restoreList {
		entries, err := store.ListRecent(20)
		if err != nil || len(entries) == 0 {
			fmt.Println("No reversible entries found.")
			return nil
		}

		fmt.Println("Recent Reversible Entries:")
		fmt.Println("==========================")
		for _, e := range entries {
			fmt.Printf("\n  Hash:    %s\n", e.Hash)
			fmt.Printf("  Command: %s\n", e.Command)
			fmt.Printf("  Time:    %s\n", e.Timestamp.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Mode:    %s\n", e.Mode)
			fmt.Printf("  Saved:   %d tokens\n", calculateSaved(e))
		}
		return nil
	}

	if len(args) == 0 {
		return fmt.Errorf("requires hash argument or --list flag")
	}

	hash := args[0]
	entry, err := store.Restore(hash)
	if err != nil {
		return fmt.Errorf("failed to restore: %w", err)
	}

	fmt.Println("=== Original Output ===")
	fmt.Println(entry.Original)
	fmt.Println("\n=== Compression Info ===")
	fmt.Printf("  Hash:      %s\n", entry.Hash)
	fmt.Printf("  Command:   %s\n", entry.Command)
	fmt.Printf("  Time:      %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("  Mode:      %s\n", entry.Mode)
	fmt.Printf("  Budget:    %d\n", entry.Budget)
	fmt.Printf("  Token Saved: %d\n", calculateSaved(*entry))

	return nil
}

func calculateSaved(e filter.StoredEntry) int {
	orig := filter.EstimateTokens(e.Original)
	comp := filter.EstimateTokens(e.Compressed)
	return orig - comp
}
