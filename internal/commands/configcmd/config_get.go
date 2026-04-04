package configcmd

import (
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/spf13/cobra"

	"github.com/GrayCodeAI/tokman/internal/config"
)

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE:  runConfigGet,
}

func init() {
	configCmd.AddCommand(configGetCmd)
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	envKey := "TOKMAN_" + strings.ToUpper(strings.NewReplacer(".", "_", "-", "_").Replace(key))
	if val := os.Getenv(envKey); val != "" {
		fmt.Printf("%s = %s (from env)\n", key, val)
		return nil
	}

	configPath := effectiveConfigPath()
	cfg := config.Defaults()
	source := "default"
	if _, err := os.Stat(configPath); err == nil {
		loadedCfg, loadErr := config.LoadFromFile(configPath)
		if loadErr != nil {
			return fmt.Errorf("failed to read config file %s: %w", configPath, loadErr)
		}
		cfg = loadedCfg
		source = "file"
	}

	value, ok := lookupConfigValue(cfg, key)
	if !ok {
		fmt.Printf("%s = (not set)\n", key)
		return nil
	}

	if source == "file" {
		fmt.Printf("%s = %s\n", key, value)
		return nil
	}
	fmt.Printf("%s = %s (default)\n", key, value)
	return nil
}

func lookupConfigValue(cfg *config.Config, key string) (string, bool) {
	value := reflect.ValueOf(cfg)
	if !value.IsValid() || value.Kind() != reflect.Ptr || value.IsNil() {
		return "", false
	}
	current := value.Elem()
	for _, part := range strings.Split(key, ".") {
		field, ok := findConfigField(current, part)
		if !ok {
			return "", false
		}
		current = indirectValue(field)
		if !current.IsValid() {
			return "", false
		}
	}
	return fmt.Sprintf("%v", current.Interface()), true
}

func findConfigField(v reflect.Value, key string) (reflect.Value, bool) {
	if v.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	want := normalizeConfigKey(key)
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		names := []string{fieldType.Name}
		if tag := fieldType.Tag.Get("mapstructure"); tag != "" {
			names = append(names, tag)
		}
		for _, name := range names {
			if normalizeConfigKey(name) == want {
				return v.Field(i), true
			}
		}
	}

	return reflect.Value{}, false
}

func indirectValue(v reflect.Value) reflect.Value {
	for v.IsValid() && v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return reflect.Value{}
		}
		v = v.Elem()
	}
	return v
}

func normalizeConfigKey(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "")
	s = strings.ReplaceAll(s, "-", "")
	return s
}
