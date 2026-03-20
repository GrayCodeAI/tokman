package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var filterCreateMatch string
var filterCreateDesc string

var filterCreateCmd = &cobra.Command{
	Use:   "filter-create <name>",
	Short: "Create a new TOML filter template",
	Long:  `Generate a TOML filter template for a command.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runFilterCreate,
}

func init() {
	filterCreateCmd.Flags().StringVarP(&filterCreateMatch, "match", "m", "", "command pattern to match")
	filterCreateCmd.Flags().StringVarP(&filterCreateDesc, "description", "d", "", "filter description")
	rootCmd.AddCommand(filterCreateCmd)
}

func runFilterCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	match := filterCreateMatch
	desc := filterCreateDesc

	if match == "" {
		match = name
	}
	if desc == "" {
		desc = name + " output filter"
	}

	template := fmt.Sprintf(`[[rules]]
match = "%s *"
description = "%s"

[rules.filter]
# Keep error/warning lines
keep_lines_matching = [
    ".*error.*",
    ".*warning.*",
    ".*Error.*",
    ".*Warning.*",
]

# Strip verbose output
strip_lines_matching = [
    "^\\s*$",
    "^\\s*\\[.*\\].*",
]

# Max output lines
max_lines = 100

[rules.filter.on_empty]
return = "%s: completed successfully"
`, match, desc, name)

	// Save to user filters directory
	home, _ := os.UserHomeDir()
	filterDir := filepath.Join(home, ".config", "tokman", "filters")
	os.MkdirAll(filterDir, 0755)

	filterPath := filepath.Join(filterDir, name+".toml")
	if _, err := os.Stat(filterPath); err == nil {
		return fmt.Errorf("filter '%s' already exists at %s", name, filterPath)
	}

	if err := os.WriteFile(filterPath, []byte(template), 0644); err != nil {
		return err
	}

	fmt.Printf("Created filter: %s\n", filterPath)
	fmt.Println("\nEdit the filter to customize:")
	fmt.Printf("  %s\n", filterPath)
	fmt.Println("\nTest it with:")
	fmt.Printf("  tokman filter-test %s\n", name)

	return nil
}
