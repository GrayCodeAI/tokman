package config

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/spf13/viper"
)

// Config represents the main configuration structure.
type Config struct {
	Tracking TrackingConfig `mapstructure:"tracking"`
	Filter   FilterConfig   `mapstructure:"filter"`
	Hooks    HooksConfig    `mapstructure:"hooks"`
}

// TrackingConfig controls token tracking behavior.
type TrackingConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	DatabasePath string `mapstructure:"database_path"`
	Telemetry    bool   `mapstructure:"telemetry"`
}

// FilterConfig controls output filtering behavior.
type FilterConfig struct {
	NoiseDirs []string `mapstructure:"noise_dirs"`
	Mode      string   `mapstructure:"mode"` // "minimal" or "aggressive"
}

// HooksConfig controls shell hook behavior.
type HooksConfig struct {
	ExcludedCommands []string `mapstructure:"excluded_commands"`
}

// Defaults returns the default configuration.
func Defaults() *Config {
	return &Config{
		Tracking: TrackingConfig{
			Enabled:      true,
			DatabasePath: "",
			Telemetry:    false,
		},
		Filter: FilterConfig{
			NoiseDirs: []string{
				".git",
				"node_modules",
				"target",
				"__pycache__",
				".venv",
				"vendor",
				".idea",
				".vscode",
				"dist",
				"build",
			},
			Mode: "minimal",
		},
		Hooks: HooksConfig{
			ExcludedCommands: []string{},
		},
	}
}

// Load reads configuration from file and environment.
func Load(cfgFile string) (*Config, error) {
	cfg := Defaults()

	// Set up viper
	viper.SetConfigType("toml")

	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("$HOME/.config/tokman")
		viper.SetConfigName("config")
	}

	// Environment variable overrides
	viper.AutomaticEnv()
	viper.SetEnvPrefix("TOKMAN")

	// Read config file if it exists
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
		// Config file not found, use defaults
		return cfg, nil
	}

	// Unmarshal into config struct
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadFromFile reads a TOML configuration file directly.
func LoadFromFile(path string) (*Config, error) {
	cfg := Defaults()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save writes the configuration to a TOML file.
func (c *Config) Save(path string) error {
	// Ensure directory exists
	if err := os.MkdirAll(path[:len(path)-len("/config.toml")], 0755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(c)
}

// GetDatabasePath returns the effective database path.
func (c *Config) GetDatabasePath() string {
	if c.Tracking.DatabasePath != "" {
		return c.Tracking.DatabasePath
	}
	return DatabasePath()
}
