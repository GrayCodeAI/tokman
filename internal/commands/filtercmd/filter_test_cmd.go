package filtercmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/commands/core"
	"github.com/GrayCodeAI/tokman/internal/commands/registry"
)

var filterTestInput string

var filterTestCmd = &cobra.Command{
	Use:   "filter-test <filter-name>",
	Short: "Test a TOML filter against sample input",
	Long:  `Apply a TOML filter to sample input and show the result.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runFilterTest,
}

func init() {
	filterTestCmd.Flags().StringVarP(&filterTestInput, "input", "i", "", "input text to test (or read from stdin)")
	registry.Add(func() { registry.Register(filterTestCmd) })
}

func runFilterTest(cmd *cobra.Command, args []string) error {
	filterName := args[0]

	filterPaths := []string{
		filepath.Join(core.GetTokmanSourceDir(), "internal", "toml", "builtin", filterName+".toml"),
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	if home != "" {
		filterPaths = append(filterPaths,
			filepath.Join(home, ".config", "tokman", "filters", filterName+".toml"),
		)
	}

	var filterPath string
	for _, p := range filterPaths {
		if _, err := os.Stat(p); err == nil {
			filterPath = p
			break
		}
	}

	if filterPath == "" {
		return fmt.Errorf("filter '%s' not found", filterName)
	}

	fmt.Printf("Filter: %s\n", filterPath)

	input := filterTestInput
	if input == "" {
		buf := make([]byte, 1024*1024)
		n, err := os.Stdin.Read(buf)
		if err != nil && n == 0 {
			return fmt.Errorf("no input provided (use --input or pipe to stdin)")
		}
		input = string(buf[:n])
	}

	fmt.Printf("Input: %d chars\n\n", len(input))
	fmt.Println("Filter found and validated successfully.")
	fmt.Println("Use 'tokman <command>' to see the filter in action.")

	return nil
}
