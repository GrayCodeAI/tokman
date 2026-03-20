package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var changelogCmd = &cobra.Command{
	Use:   "changelog",
	Short: "Show changelog for current version",
	Long:  `Display the changelog entries for the current tokman version.`,
	RunE:  runChangelog,
}

var changelogAll bool

func init() {
	changelogCmd.Flags().BoolVarP(&changelogAll, "all", "a", false, "show all versions")
	rootCmd.AddCommand(changelogCmd)
}

func runChangelog(cmd *cobra.Command, args []string) error {
	// Find CHANGELOG.md
	paths := []string{
		"CHANGELOG.md",
		filepath.Join(getTokmanSourceDir(), "CHANGELOG.md"),
	}

	var content []byte
	var err error
	for _, p := range paths {
		content, err = os.ReadFile(p)
		if err == nil {
			break
		}
	}

	if err != nil {
		fmt.Println("No CHANGELOG.md found.")
		fmt.Println("Visit https://github.com/GrayCodeAI/tokman/releases for release notes.")
		return nil
	}

	if changelogAll {
		fmt.Print(string(content))
		return nil
	}

	// Show only the first version section
	lines := strings.Split(string(content), "\n")
	versionCount := 0
	for i, line := range lines {
		if strings.HasPrefix(line, "## ") {
			versionCount++
			if versionCount > 1 {
				// Print up to the next version header
				fmt.Print(strings.Join(lines[:i], "\n"))
				return nil
			}
		}
		fmt.Println(line)
		if i > 50 {
			// Limit output
			break
		}
	}

	return nil
}
