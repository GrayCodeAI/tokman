package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/config"
)

var configDiffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Show differences from default configuration",
	Long:  `Compare current configuration against defaults and show what has been changed.`,
	RunE:  runConfigDiff,
}

func init() {
	configCmd.AddCommand(configDiffCmd)
}

func runConfigDiff(cmd *cobra.Command, args []string) error {
	defaults := config.Defaults()

	// Load user config
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "tokman", "config.toml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("No config file found. Using defaults.")
		fmt.Printf("Config path: %s\n", configPath)
		return nil
	}

	userCfg, err := config.LoadFromFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Config diff: %s vs defaults\n\n", configPath)

	hasDiff := false

	// Compare pipeline settings
	hasDiff = diffStruct("pipeline", &defaults.Pipeline, &userCfg.Pipeline) || hasDiff

	// Compare filter settings
	hasDiff = diffStruct("filter", &defaults.Filter, &userCfg.Filter) || hasDiff

	// Compare tracking settings
	hasDiff = diffStruct("tracking", &defaults.Tracking, &userCfg.Tracking) || hasDiff

	if !hasDiff {
		fmt.Println("No differences from defaults.")
	}

	return nil
}

func diffStruct(prefix string, defaults, current interface{}) bool {
	hasDiff := false
	dv := reflect.ValueOf(defaults).Elem()
	cv := reflect.ValueOf(current).Elem()

	for i := 0; i < dv.NumField(); i++ {
		df := dv.Field(i)
		cf := cv.Field(i)
		fieldName := dv.Type().Field(i).Tag.Get("mapstructure")
		if fieldName == "" {
			fieldName = dv.Type().Field(i).Name
		}

		if !reflect.DeepEqual(df.Interface(), cf.Interface()) {
			fmt.Printf("  %s.%s: %v → %v\n", prefix, fieldName, df.Interface(), cf.Interface())
			hasDiff = true
		}
	}
	return hasDiff
}
