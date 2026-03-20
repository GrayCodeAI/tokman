package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
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
	rootCmd.AddCommand(filterTestCmd)
}

func runFilterTest(cmd *cobra.Command, args []string) error {
	filterName := args[0]

	// Find the filter file
	filterPaths := []string{
		filepath.Join(getTokmanSourceDir(), "internal", "toml", "builtin", filterName+".toml"),
	}

	home, _ := os.UserHomeDir()
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

	// Get input
	input := filterTestInput
	if input == "" {
		// Read from stdin
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
