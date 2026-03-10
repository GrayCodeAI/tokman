package config

import (
	"os"

	"github.com/BurntSushi/toml"
	"github.com/spf13/viper"
)

// Config represents the main configuration structure.
type Config struct {
	Tracking  TrackingConfig  `mapstructure:"tracking"`
	Filter    FilterConfig    `mapstructure:"filter"`
	Hooks     HooksConfig     `mapstructure:"hooks"`
	Dashboard DashboardConfig `mapstructure:"dashboard"`
	Alerts    AlertsConfig    `mapstructure:"alerts"`
	Export    ExportConfig    `mapstructure:"export"`
}

// TrackingConfig controls token tracking behavior.
type TrackingConfig struct {
	Enabled      bool   `mapstructure:"enabled"`
	DatabasePath string `mapstructure:"database_path"`
	Telemetry    bool   `mapstructure:"telemetry"`
}

// FilterConfig controls output filtering behavior.
type FilterConfig struct {
	NoiseDirs   []string `mapstructure:"noise_dirs"`
	IgnoreFiles []string `mapstructure:"ignore_files"` // File patterns to ignore (e.g., "*.lock", "*.min.js")
	Mode        string   `mapstructure:"mode"`         // "minimal" or "aggressive"
	MaxWidth    int      `mapstructure:"max_width"`    // Max display width (0 = auto)
}

// HooksConfig controls shell hook behavior.
type HooksConfig struct {
	ExcludedCommands []string `mapstructure:"excluded_commands"`
	AuditDir         string   `mapstructure:"audit_dir"` // Directory for hook audit logs
	TeeDir           string   `mapstructure:"tee_dir"`   // Directory for failure tee logs
}

// DashboardConfig controls dashboard behavior.
type DashboardConfig struct {
	Port           int    `mapstructure:"port"`
	Bind           string `mapstructure:"bind"`
	UpdateInterval int    `mapstructure:"update_interval"`
	Theme          string `mapstructure:"theme"`
	EnableExport   bool   `mapstructure:"enable_export"`
}

// AlertsConfig controls alert thresholds.
type AlertsConfig struct {
	Enabled             bool    `mapstructure:"enabled"`
	DailyTokenLimit     int64   `mapstructure:"daily_token_limit"`
	WeeklyTokenLimit    int64   `mapstructure:"weekly_token_limit"`
	UsageSpikeThreshold float64 `mapstructure:"usage_spike_threshold"`
}

// ExportConfig controls export behavior.
type ExportConfig struct {
	DefaultFormat    string `mapstructure:"default_format"`
	IncludeTimestamp bool   `mapstructure:"include_timestamps"`
	MaxRecords       int    `mapstructure:"max_records"`
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
			IgnoreFiles: []string{
				"*.lock",
				"*.min.js",
				"*.min.css",
				"*.map",
				"package-lock.json",
				"yarn.lock",
				"pnpm-lock.yaml",
				"Cargo.lock",
				"poetry.lock",
			},
			Mode:     "minimal",
			MaxWidth: 0,
		},
		Hooks: HooksConfig{
			ExcludedCommands: []string{},
			AuditDir:         "",
			TeeDir:           "",
		},
		Dashboard: DashboardConfig{
			Port:           8080,
			Bind:           "localhost",
			UpdateInterval: 30000,
			Theme:          "dark",
			EnableExport:   true,
		},
		Alerts: AlertsConfig{
			Enabled:             true,
			DailyTokenLimit:     1000000,
			WeeklyTokenLimit:    5000000,
			UsageSpikeThreshold: 2.0,
		},
		Export: ExportConfig{
			DefaultFormat:    "json",
			IncludeTimestamp: true,
			MaxRecords:       0,
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

	// Environment variable aliases (for compatibility)
	if val := os.Getenv("TOKMAN_DB_PATH"); val != "" {
		viper.SetDefault("tracking.database_path", val)
	}
	if val := os.Getenv("TOKMAN_TELEMETRY_DISABLED"); val != "" {
		viper.SetDefault("tracking.telemetry", val == "false")
	}
	if val := os.Getenv("TOKMAN_AUDIT_DIR"); val != "" {
		viper.SetDefault("hooks.audit_dir", val)
	}
	if val := os.Getenv("TOKMAN_TEE_DIR"); val != "" {
		viper.SetDefault("hooks.tee_dir", val)
	}
	if val := os.Getenv("TOKMAN_TEE"); val != "" {
		viper.SetDefault("hooks.tee_enabled", val == "true" || val == "1")
	}
	if val := os.Getenv("TOKMAN_HOOK_AUDIT"); val != "" {
		viper.SetDefault("hooks.audit_enabled", val == "true" || val == "1")
	}

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
