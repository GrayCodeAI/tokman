package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var aliasCmd = &cobra.Command{
	Use:   "alias",
	Short: "Manage command aliases for common filter+command combos",
	Long: `Create shorthand aliases for frequently used tokman commands.

Examples:
  tokman alias set gs "git status"
  tokman alias set dl "docker logs --tail 50"
  tokman alias list`,
}

var aliasListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all aliases",
	RunE:  runAliasList,
}

var aliasSetCmd = &cobra.Command{
	Use:   "set <name> <command...>",
	Short: "Create or update an alias",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runAliasSet,
}

var aliasRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an alias",
	Args:  cobra.ExactArgs(1),
	RunE:  runAliasRemove,
}

func init() {
	aliasCmd.AddCommand(aliasListCmd)
	aliasCmd.AddCommand(aliasSetCmd)
	aliasCmd.AddCommand(aliasRemoveCmd)
	rootCmd.AddCommand(aliasCmd)
}

func getAliasPath() string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	dir := filepath.Join(home, ".config", "tokman")
	os.MkdirAll(dir, 0755)
	return filepath.Join(dir, "aliases.txt")
}

func loadAliases() map[string]string {
	aliases := make(map[string]string)
	path := getAliasPath()
	if path == "" {
		return aliases
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return aliases
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			aliases[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return aliases
}

func saveAliases(aliases map[string]string) error {
	path := getAliasPath()
	if path == "" {
		return fmt.Errorf("cannot determine config path")
	}
	var lines []string
	for k, v := range aliases {
		lines = append(lines, fmt.Sprintf("%s=%s", k, v))
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0644)
}

func runAliasList(cmd *cobra.Command, args []string) error {
	aliases := loadAliases()
	if len(aliases) == 0 {
		fmt.Println("No aliases configured.")
		fmt.Println("Create one with: tokman alias set <name> <command>")
		return nil
	}
	fmt.Println("Aliases:")
	for name, command := range aliases {
		fmt.Printf("  %s → %s\n", name, command)
	}
	return nil
}

func runAliasSet(cmd *cobra.Command, args []string) error {
	name := args[0]
	command := strings.Join(args[1:], " ")
	aliases := loadAliases()
	aliases[name] = command
	if err := saveAliases(aliases); err != nil {
		return err
	}
	fmt.Printf("Alias '%s' → '%s' created.\n", name, command)
	return nil
}

func runAliasRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	aliases := loadAliases()
	if _, ok := aliases[name]; !ok {
		return fmt.Errorf("alias '%s' not found", name)
	}
	delete(aliases, name)
	if err := saveAliases(aliases); err != nil {
		return err
	}
	fmt.Printf("Alias '%s' removed.\n", name)
	return nil
}
