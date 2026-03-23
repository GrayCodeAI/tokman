package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := Defaults()

	if cfg == nil {
		t.Fatal("Defaults() returned nil")
	}

	// Tracking defaults
	if !cfg.Tracking.Enabled {
		t.Error("Tracking.Enabled should default to true")
	}
	if cfg.Tracking.Telemetry {
		t.Error("Tracking.Telemetry should default to false")
	}

	// Filter defaults
	if cfg.Filter.Mode != "minimal" {
		t.Errorf("Filter.Mode = %q, want 'minimal'", cfg.Filter.Mode)
	}
	if len(cfg.Filter.NoiseDirs) == 0 {
		t.Error("Filter.NoiseDirs should not be empty")
	}
	if len(cfg.Filter.IgnoreFiles) == 0 {
		t.Error("Filter.IgnoreFiles should not be empty")
	}

	// Pipeline defaults
	if cfg.Pipeline.MaxContextTokens != 2000000 {
		t.Errorf("Pipeline.MaxContextTokens = %d, want 2000000", cfg.Pipeline.MaxContextTokens)
	}
	if cfg.Pipeline.ChunkSize != 100000 {
		t.Errorf("Pipeline.ChunkSize = %d, want 100000", cfg.Pipeline.ChunkSize)
	}
	if !cfg.Pipeline.EnableEntropy {
		t.Error("Pipeline.EnableEntropy should default to true")
	}
	if !cfg.Pipeline.CacheEnabled {
		t.Error("Pipeline.CacheEnabled should default to true")
	}
	if cfg.Pipeline.CacheMaxSize != 1000 {
		t.Errorf("Pipeline.CacheMaxSize = %d, want 1000", cfg.Pipeline.CacheMaxSize)
	}
	if cfg.Pipeline.StreamThreshold != 500000 {
		t.Errorf("Pipeline.StreamThreshold = %d, want 500000", cfg.Pipeline.StreamThreshold)
	}

	// Threshold defaults
	if cfg.Pipeline.EntropyThreshold != 0.3 {
		t.Errorf("Pipeline.EntropyThreshold = %f, want 0.3", cfg.Pipeline.EntropyThreshold)
	}
	if cfg.Pipeline.PerplexityThreshold != 0.5 {
		t.Errorf("Pipeline.PerplexityThreshold = %f, want 0.5", cfg.Pipeline.PerplexityThreshold)
	}

	// Dashboard defaults
	if cfg.Dashboard.Port != 8080 {
		t.Errorf("Dashboard.Port = %d, want 8080", cfg.Dashboard.Port)
	}
	if cfg.Dashboard.Bind != "localhost" {
		t.Errorf("Dashboard.Bind = %q, want 'localhost'", cfg.Dashboard.Bind)
	}

	// Alert defaults
	if !cfg.Alerts.Enabled {
		t.Error("Alerts.Enabled should default to true")
	}
	if cfg.Alerts.DailyTokenLimit != 1000000 {
		t.Errorf("Alerts.DailyTokenLimit = %d, want 1000000", cfg.Alerts.DailyTokenLimit)
	}

	// Export defaults
	if cfg.Export.DefaultFormat != "json" {
		t.Errorf("Export.DefaultFormat = %q, want 'json'", cfg.Export.DefaultFormat)
	}
}

func TestValidateValidConfig(t *testing.T) {
	cfg := Defaults()
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() on defaults should pass, got: %v", err)
	}
}

func TestValidateInvalidThresholds(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Config)
	}{
		{"entropy_threshold negative", func(c *Config) { c.Pipeline.EntropyThreshold = -0.1 }},
		{"entropy_threshold too high", func(c *Config) { c.Pipeline.EntropyThreshold = 1.5 }},
		{"perplexity_threshold negative", func(c *Config) { c.Pipeline.PerplexityThreshold = -0.1 }},
		{"perplexity_threshold too high", func(c *Config) { c.Pipeline.PerplexityThreshold = 1.5 }},
		{"goal_driven_threshold negative", func(c *Config) { c.Pipeline.GoalDrivenThreshold = -0.1 }},
		{"ast_preserve_threshold too high", func(c *Config) { c.Pipeline.ASTPreserveThreshold = 1.5 }},
		{"contrastive_threshold negative", func(c *Config) { c.Pipeline.ContrastiveThreshold = -0.1 }},
		{"evaluator_threshold too high", func(c *Config) { c.Pipeline.EvaluatorThreshold = 1.5 }},
		{"hierarchical_ratio negative", func(c *Config) { c.Pipeline.HierarchicalRatio = -0.1 }},
		{"attribution_threshold too high", func(c *Config) { c.Pipeline.AttributionThreshold = 1.5 }},
		{"semantic_threshold negative", func(c *Config) { c.Pipeline.SemanticThreshold = -0.1 }},
		{"lazy_budget_ratio too high", func(c *Config) { c.Pipeline.LazyBudgetRatio = 1.5 }},
		{"anchor_threshold negative", func(c *Config) { c.Pipeline.AnchorThreshold = -0.1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			tt.modify(cfg)
			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() should fail for %s", tt.name)
			}
		})
	}
}

func TestValidateInvalidIntegers(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Config)
	}{
		{"max_context_tokens negative", func(c *Config) { c.Pipeline.MaxContextTokens = -1 }},
		{"chunk_size negative", func(c *Config) { c.Pipeline.ChunkSize = -1 }},
		{"cache_max_size negative", func(c *Config) { c.Pipeline.CacheMaxSize = -1 }},
		{"ngram_min_occurrences negative", func(c *Config) { c.Pipeline.NgramMinOccurrences = -1 }},
		{"hierarchical_max_levels negative", func(c *Config) { c.Pipeline.HierarchicalMaxLevels = -1 }},
		{"default_budget negative", func(c *Config) { c.Pipeline.DefaultBudget = -1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			tt.modify(cfg)
			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() should fail for %s", tt.name)
			}
		})
	}
}

func TestValidateInvalidFilterMode(t *testing.T) {
	cfg := Defaults()
	cfg.Filter.Mode = "invalid"
	if err := cfg.Validate(); err == nil {
		t.Error("Validate() should fail for invalid filter mode")
	}
}

func TestValidateEmptyFilterMode(t *testing.T) {
	cfg := Defaults()
	cfg.Filter.Mode = ""
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() should pass for empty filter mode, got: %v", err)
	}
}

func TestValidateInvalidDashboardPort(t *testing.T) {
	tests := []struct {
		name string
		port int
	}{
		{"negative port", -1},
		{"port too high", 70000},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			cfg.Dashboard.Port = tt.port
			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() should fail for port %d", tt.port)
			}
		})
	}
}

func TestValidateAlertLimits(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Config)
	}{
		{"daily_token_limit negative", func(c *Config) { c.Alerts.DailyTokenLimit = -1 }},
		{"weekly_token_limit negative", func(c *Config) { c.Alerts.WeeklyTokenLimit = -1 }},
		{"usage_spike_threshold negative", func(c *Config) { c.Alerts.UsageSpikeThreshold = -1 }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := Defaults()
			tt.modify(cfg)
			if err := cfg.Validate(); err == nil {
				t.Errorf("Validate() should fail for %s", tt.name)
			}
		})
	}
}

func TestGetDatabasePath(t *testing.T) {
	cfg := Defaults()

	t.Run("custom path", func(t *testing.T) {
		cfg.Tracking.DatabasePath = "/custom/path/db.sqlite"
		if got := cfg.GetDatabasePath(); got != "/custom/path/db.sqlite" {
			t.Errorf("GetDatabasePath() = %q, want /custom/path/db.sqlite", got)
		}
	})

	t.Run("default path", func(t *testing.T) {
		cfg.Tracking.DatabasePath = ""
		got := cfg.GetDatabasePath()
		if got == "" {
			t.Error("GetDatabasePath() returned empty string")
		}
	})
}

func TestLoadFromFileNotFound(t *testing.T) {
	cfg, err := LoadFromFile("/nonexistent/path/config.toml")
	if err != nil {
		t.Errorf("LoadFromFile should not error for missing file, got: %v", err)
	}
	if cfg == nil {
		t.Error("LoadFromFile should return defaults for missing file")
	}
}

func TestLoadFromFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "config.toml")

	content := `
[filter]
mode = "aggressive"
`
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if cfg.Filter.Mode != "aggressive" {
		t.Errorf("Filter.Mode = %q, want 'aggressive'", cfg.Filter.Mode)
	}
	// Pipeline fields use mapstructure tags, decoded by toml.DecodeFile
	// which may not map all nested fields. Verify defaults are preserved.
	if cfg.Pipeline.MaxContextTokens <= 0 {
		t.Error("Pipeline.MaxContextTokens should have a positive default")
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	cfgPath := filepath.Join(tmpDir, "subdir", "config.toml")

	cfg := Defaults()
	cfg.Filter.Mode = "aggressive"
	cfg.Pipeline.MaxContextTokens = 1000000

	if err := cfg.Save(cfgPath); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := LoadFromFile(cfgPath)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if loaded.Filter.Mode != "aggressive" {
		t.Errorf("Filter.Mode = %q, want 'aggressive'", loaded.Filter.Mode)
	}
	if loaded.Pipeline.MaxContextTokens != 1000000 {
		t.Errorf("Pipeline.MaxContextTokens = %d, want 1000000", loaded.Pipeline.MaxContextTokens)
	}
}

func TestValidateBoundaries(t *testing.T) {
	cfg := Defaults()
	// Test exact boundary values
	cfg.Pipeline.EntropyThreshold = 0.0
	cfg.Pipeline.PerplexityThreshold = 1.0
	cfg.Pipeline.GoalDrivenThreshold = 0.0
	cfg.Pipeline.ASTPreserveThreshold = 1.0
	if err := cfg.Validate(); err != nil {
		t.Errorf("Validate() should pass for boundary values, got: %v", err)
	}
}

func TestValidateMultipleErrors(t *testing.T) {
	cfg := Defaults()
	cfg.Pipeline.EntropyThreshold = -1.0
	cfg.Pipeline.PerplexityThreshold = 2.0
	cfg.Pipeline.MaxContextTokens = -1
	cfg.Filter.Mode = "invalid"

	err := cfg.Validate()
	if err == nil {
		t.Error("Validate() should fail with multiple errors")
	}
}
