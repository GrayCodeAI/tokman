package configcmd

// migrate_cmd.go implements Task #161: configuration migration tool.
// Usage:
//
//	tokman config migrate
//	tokman config migrate --from v1 --to v2
//	tokman config migrate --dry-run

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/GrayCodeAI/tokman/internal/commands/registry"
	"github.com/GrayCodeAI/tokman/internal/config"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate configuration file from an older format",
	Long: `Apply migration rules to upgrade a tokman config file to the current format.

Migration rules applied:
  token_limit  → budget
  filter_mode  → mode
  aggressive   → mode = "aggressive" (when true)

Use --dry-run to preview changes without writing them.`,
	RunE: runConfigMigrate,
}

var (
	migrateFrom   string
	migrateTo     string
	migrateDryRun bool
)

func init() {
	migrateCmd.Flags().StringVar(&migrateFrom, "from", "v1", "Source config version")
	migrateCmd.Flags().StringVar(&migrateTo, "to", "v2", "Target config version")
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "Print migrated config without writing")
	registry.Add(func() { registry.Register(migrateCmd) })
}

func runConfigMigrate(cmd *cobra.Command, args []string) error {
	// Determine config file path: prefer viper's resolved file, then default.
	cfgPath := viper.ConfigFileUsed()
	if cfgPath == "" {
		cfgPath = config.ConfigPath()
	}

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return fmt.Errorf("migrate: config file not found: %s", cfgPath)
	}

	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		return fmt.Errorf("migrate: read %s: %w", cfgPath, err)
	}

	original := string(raw)
	migrated, changes := applyMigrationRules(original)

	if len(changes) == 0 {
		fmt.Fprintf(os.Stderr, "migrate: no changes needed in %s\n", cfgPath)
		return nil
	}

	fmt.Fprintf(os.Stderr, "migrate: %d change(s) to apply in %s:\n", len(changes), cfgPath)
	for _, c := range changes {
		fmt.Fprintf(os.Stderr, "  - %s\n", c)
	}

	if migrateDryRun {
		fmt.Print(migrated)
		return nil
	}

	// Write migrated config back to the same path.
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0700); err != nil {
		return fmt.Errorf("migrate: mkdir: %w", err)
	}
	if err := os.WriteFile(cfgPath, []byte(migrated), 0600); err != nil {
		return fmt.Errorf("migrate: write %s: %w", cfgPath, err)
	}

	fmt.Fprintf(os.Stderr, "migrate: wrote migrated config to %s\n", cfgPath)
	return nil
}

// applyMigrationRules transforms v1 config keys to v2 names.
// Returns the migrated content and a human-readable list of applied changes.
func applyMigrationRules(content string) (string, []string) {
	var changes []string
	lines := strings.Split(content, "\n")
	out := make([]string, 0, len(lines))

	aggressiveTrue := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip comments and section headers unchanged.
		if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "[") || trimmed == "" {
			out = append(out, line)
			continue
		}

		// Parse key = value pairs.
		eqIdx := strings.Index(trimmed, "=")
		if eqIdx < 0 {
			out = append(out, line)
			continue
		}

		key := strings.TrimSpace(trimmed[:eqIdx])
		value := strings.TrimSpace(trimmed[eqIdx+1:])

		switch key {
		case "token_limit":
			// Rename token_limit → budget (preserving indentation).
			indent := leadingWhitespace(line)
			out = append(out, indent+"budget = "+value)
			changes = append(changes, fmt.Sprintf("renamed token_limit → budget (value: %s)", value))

		case "filter_mode":
			// Rename filter_mode → mode (preserving indentation).
			indent := leadingWhitespace(line)
			out = append(out, indent+"mode = "+value)
			changes = append(changes, fmt.Sprintf("renamed filter_mode → mode (value: %s)", value))

		case "aggressive":
			// If aggressive = true, record it and suppress the key; we'll
			// emit mode = "aggressive" after the loop if needed.
			stripped := strings.Trim(value, `"' `)
			if stripped == "true" || stripped == "1" {
				aggressiveTrue = true
				changes = append(changes, `converted aggressive = true → mode = "aggressive"`)
				// Drop this line from output.
			} else {
				// aggressive = false — just remove the obsolete key silently.
				changes = append(changes, "removed obsolete key: aggressive = false")
			}

		default:
			out = append(out, line)
		}
	}

	if aggressiveTrue {
		out = append(out, `mode = "aggressive"`)
	}

	return strings.Join(out, "\n"), changes
}

// leadingWhitespace returns the leading whitespace of s.
func leadingWhitespace(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[:i]
		}
	}
	return s
}
