package commands

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:   "marketplace",
	Short: "Browse and install community TOML filters",
	Long:  `Search, install, and manage community-contributed TOML filters.`,
}

var marketplaceSearchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search available filters",
	Args:  cobra.ExactArgs(1),
	RunE:  runMarketplaceSearch,
}

var marketplaceInstallCmd = &cobra.Command{
	Use:   "install <filter-name>",
	Short: "Install a community filter",
	Args:  cobra.ExactArgs(1),
	RunE:  runMarketplaceInstall,
}

var marketplaceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed community filters",
	RunE:  runMarketplaceList,
}

func init() {
	marketplaceCmd.AddCommand(marketplaceSearchCmd)
	marketplaceCmd.AddCommand(marketplaceInstallCmd)
	marketplaceCmd.AddCommand(marketplaceListCmd)
	rootCmd.AddCommand(marketplaceCmd)
}

// CommunityFilters is a registry of known community filters.
var CommunityFilters = map[string]string{
	"jest":       "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/jest.toml",
	"vitest":     "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/vitest.toml",
	"playwright": "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/playwright.toml",
	"cypress":    "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/cypress.toml",
	"mocha":      "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/mocha.toml",
	"eslint":     "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/eslint.toml",
	"biome":      "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/biome.toml",
	"swc":        "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/swc.toml",
	"webpack":    "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/webpack.toml",
	"vite":       "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/vite.toml",
	"rollup":     "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/rollup.toml",
	"trivy":      "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/trivy.toml",
	"snyk":       "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/snyk.toml",
	"opentofu":   "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/opentofu.toml",
	"pulumi":     "https://raw.githubusercontent.com/GrayCodeAI/tokman/main/filters/pulumi.toml",
}

func runMarketplaceSearch(cmd *cobra.Command, args []string) error {
	query := strings.ToLower(args[0])
	fmt.Printf("Searching for filters matching '%s'...\n\n", query)

	found := 0
	for name, url := range CommunityFilters {
		if strings.Contains(strings.ToLower(name), query) {
			fmt.Printf("  %-20s %s\n", name, url)
			found++
		}
	}

	if found == 0 {
		fmt.Println("No filters found. Try a different search term.")
		fmt.Println("\nAvailable filters:")
		for name := range CommunityFilters {
			fmt.Printf("  %s\n", name)
		}
	}

	return nil
}

func runMarketplaceInstall(cmd *cobra.Command, args []string) error {
	name := args[0]
	url, ok := CommunityFilters[name]
	if !ok {
		return fmt.Errorf("unknown filter: %s (use 'tokman marketplace search' to find filters)", name)
	}

	// Download filter
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download filter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("filter not found (HTTP %d)", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read filter: %w", err)
	}

	// Save to user filters directory
	home, _ := os.UserHomeDir()
	filterDir := filepath.Join(home, ".config", "tokman", "filters")
	os.MkdirAll(filterDir, 0755)

	filterPath := filepath.Join(filterDir, name+".toml")
	if err := os.WriteFile(filterPath, content, 0644); err != nil {
		return fmt.Errorf("failed to save filter: %w", err)
	}

	fmt.Printf("Installed '%s' to %s\n", name, filterPath)
	return nil
}

func runMarketplaceList(cmd *cobra.Command, args []string) error {
	home, _ := os.UserHomeDir()
	filterDir := filepath.Join(home, ".config", "tokman", "filters")

	entries, err := os.ReadDir(filterDir)
	if err != nil || len(entries) == 0 {
		fmt.Println("No community filters installed.")
		fmt.Println("Install with: tokman marketplace install <name>")
		return nil
	}

	fmt.Println("Installed community filters:")
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".toml") {
			name := strings.TrimSuffix(e.Name(), ".toml")
			fmt.Printf("  %s\n", name)
		}
	}
	return nil
}
