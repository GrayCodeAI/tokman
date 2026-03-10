package filter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// PluginFilter represents a user-defined filter loaded from config.
type PluginFilter struct {
	name         string
	description  string
	patterns     []*regexp.Regexp
	replacements map[string]string
	enabled      bool
}

// PluginConfig defines the structure for filter plugins in config.
type PluginConfig struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Enabled      bool              `json:"enabled"`
	Patterns     []string          `json:"patterns"`
	Replacements map[string]string `json:"replacements"`
	Mode         string            `json:"mode"` // "hide" or "replace"
}

// NewPluginFilter creates a filter from plugin config.
func NewPluginFilter(cfg PluginConfig) (*PluginFilter, error) {
	pf := &PluginFilter{
		name:         cfg.Name,
		description:  cfg.Description,
		replacements: cfg.Replacements,
		enabled:      cfg.Enabled,
	}

	// Compile patterns
	for _, p := range cfg.Patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", p, err)
		}
		pf.patterns = append(pf.patterns, re)
	}

	return pf, nil
}

func (f *PluginFilter) Name() string {
	return f.name
}

func (f *PluginFilter) Description() string {
	return f.description
}

func (f *PluginFilter) Apply(input string, mode Mode) (string, int) {
	if !f.enabled {
		return input, 0
	}

	output := input
	for _, pattern := range f.patterns {
		// Check if there's a replacement for this pattern
		patternStr := pattern.String()
		if replacement, ok := f.replacements[patternStr]; ok {
			output = pattern.ReplaceAllString(output, replacement)
		} else {
			// Default: hide matching lines
			output = pattern.ReplaceAllString(output, "")
		}
	}

	saved := CalculateTokensSaved(input, output)
	return output, saved
}

// PluginManager handles loading and managing filter plugins.
type PluginManager struct {
	pluginsDir string
	filters    []*PluginFilter
}

// NewPluginManager creates a plugin manager.
func NewPluginManager(pluginsDir string) *PluginManager {
	return &PluginManager{
		pluginsDir: pluginsDir,
		filters:    make([]*PluginFilter, 0),
	}
}

// LoadPlugins loads all plugin configurations from the plugins directory.
func (pm *PluginManager) LoadPlugins() error {
	if pm.pluginsDir == "" {
		return nil
	}

	// Check if directory exists
	if _, err := os.Stat(pm.pluginsDir); os.IsNotExist(err) {
		return nil
	}

	// Load all .json files
	entries, err := os.ReadDir(pm.pluginsDir)
	if err != nil {
		return fmt.Errorf("failed to read plugins dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		path := filepath.Join(pm.pluginsDir, entry.Name())
		if err := pm.loadPlugin(path); err != nil {
			// Log warning but continue
			fmt.Fprintf(os.Stderr, "Warning: failed to load plugin %s: %v\n", entry.Name(), err)
		}
	}

	return nil
}

func (pm *PluginManager) loadPlugin(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var cfg PluginConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	filter, err := NewPluginFilter(cfg)
	if err != nil {
		return err
	}

	pm.filters = append(pm.filters, filter)
	return nil
}

// GetFilters returns all loaded plugin filters.
func (pm *PluginManager) GetFilters() []*PluginFilter {
	return pm.filters
}

// AddFilter manually adds a plugin filter.
func (pm *PluginManager) AddFilter(filter *PluginFilter) {
	pm.filters = append(pm.filters, filter)
}

// ExamplePluginConfigs returns example plugin configurations.
func ExamplePluginConfigs() []PluginConfig {
	return []PluginConfig{
		{
			Name:        "hide-npm-warnings",
			Description: "Hide npm deprecation warnings",
			Enabled:     true,
			Patterns:    []string{`npm WARN deprecated`},
			Mode:        "hide",
		},
		{
			Name:        "shorten-paths",
			Description: "Shorten long file paths",
			Enabled:     true,
			Patterns:    []string{`/home/[^/]+/`},
			Replacements: map[string]string{
				`/home/[^/]+/`: "~/",
			},
			Mode: "replace",
		},
		{
			Name:        "hide-test-output",
			Description: "Hide verbose test output lines",
			Enabled:     true,
			Patterns: []string{
				`=== RUN   Test`,
				`--- PASS: Test`,
				`--- FAIL: Test`,
			},
			Mode: "hide",
		},
	}
}

// SaveExamplePlugins saves example plugin configs to the specified directory.
func SaveExamplePlugins(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	examples := ExamplePluginConfigs()
	for _, cfg := range examples {
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return err
		}

		filename := fmt.Sprintf("%s.json", cfg.Name)
		path := filepath.Join(dir, filename)
		if err := os.WriteFile(path, data, 0644); err != nil {
			return err
		}
	}

	return nil
}
