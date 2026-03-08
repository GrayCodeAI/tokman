package commands

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/filter"
	"github.com/GrayCodeAI/tokman/internal/tracking"
)

var treeCmd = &cobra.Command{
	Use:   "tree [args...]",
	Short: "Directory tree with token-optimized output",
	Long: `Proxy to native tree with token-optimized output.

Supports all native tree flags like -L, -d, -a.
Filters noise directories (node_modules, .git, target, etc.) for cleaner output.

Examples:
  tokman tree
  tokman tree -L 2
  tokman tree -a -I 'node_modules|.git'`,
	RunE: runTree,
}

func init() {
	rootCmd.AddCommand(treeCmd)
}

func runTree(cmd *cobra.Command, args []string) error {
	timer := tracking.Start()

	// Build tree command
	treeArgs := append([]string{}, args...)
	
	c := exec.Command("tree", treeArgs...)
	
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	
	err := c.Run()
	output := stdout.String()
	if output == "" && stderr.Len() > 0 {
		output = stderr.String()
	}

	// Apply filtering
	engine := filter.NewEngine(filter.ModeMinimal)
	filtered, tokensSaved := engine.Process(output)

	fmt.Print(filtered)

	// Track
	originalTokens := filter.EstimateTokens(output)
	filteredTokens := filter.EstimateTokens(filtered)
	timer.Track(fmt.Sprintf("tree %s", strings.Join(args, " ")), "tokman tree", originalTokens, filteredTokens)

	if verbose && tokensSaved > 0 {
		fmt.Fprintf(os.Stderr, "Tokens saved: %d\n", tokensSaved)
	}

	if err != nil {
		return err
	}
	return nil
}
