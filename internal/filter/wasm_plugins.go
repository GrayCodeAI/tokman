package filter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// WASMPluginSystem enables custom compression layers via WebAssembly.
// Users can write compression rules in any language (Rust, C, Go, Python)
// and compile to WASM for plug-in execution.
//
// Architecture:
// 1. Plugin directory: ~/.config/tokman/plugins/*.wasm
// 2. Each plugin implements a single Filter interface
// 3. Plugins are loaded lazily and cached
// 4. Sandboxed execution via wazero runtime
type WASMPluginSystem struct {
	config    WASMPluginConfig
	plugins   map[string]*WASMPlugin
	registry  map[string]Filter // name -> filter
	mu        sync.RWMutex
	pluginDir string
}

// WASMPluginConfig holds configuration for the WASM plugin system
type WASMPluginConfig struct {
	// Enabled controls whether WASM plugins are active
	Enabled bool

	// PluginDir is the directory containing .wasm plugin files
	PluginDir string

	// MaxPlugins limits the number of loaded plugins
	MaxPlugins int

	// TimeoutMs is the max execution time per plugin call
	TimeoutMs int
}

// WASMPlugin represents a loaded WASM compression plugin
type WASMPlugin struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Author      string `json:"author"`
	Path        string `json:"path"`
	Enabled     bool   `json:"enabled"`
	wasmBytes   []byte
}

// PluginManifest describes a WASM plugin
type PluginManifest struct {
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Description string        `json:"description"`
	Author      string        `json:"author"`
	Entry       string        `json:"entry"`
	Layers      []PluginLayer `json:"layers"`
}

// PluginLayer defines a single compression layer in a plugin
type PluginLayer struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	MinTokens   int     `json:"min_tokens"`
	MaxTokens   int     `json:"max_tokens"`
	Weight      float64 `json:"weight"`
}

// defaultWASMPluginConfig returns default configuration
func defaultWASMPluginConfig() WASMPluginConfig {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.TempDir()
	}
	return WASMPluginConfig{
		Enabled:    true,
		PluginDir:  filepath.Join(homeDir, ".config", "tokman", "plugins"),
		MaxPlugins: 10,
		TimeoutMs:  1000,
	}
}

// newWASMPluginSystem creates a new WASM plugin system
func newWASMPluginSystem() *WASMPluginSystem {
	return newWASMPluginSystemWithConfig(defaultWASMPluginConfig())
}

// newWASMPluginSystemWithConfig creates a plugin system with custom config
func newWASMPluginSystemWithConfig(cfg WASMPluginConfig) *WASMPluginSystem {
	ps := &WASMPluginSystem{
		config:    cfg,
		plugins:   make(map[string]*WASMPlugin),
		registry:  make(map[string]Filter),
		pluginDir: cfg.PluginDir,
	}

	if cfg.Enabled {
		ps.loadPlugins()
	}

	return ps
}

// loadPlugins discovers and loads WASM plugins from the plugin directory
func (ps *WASMPluginSystem) loadPlugins() {
	if ps.pluginDir == "" {
		return
	}

	// Create plugin directory if it doesn't exist
	os.MkdirAll(ps.pluginDir, 0755)

	// Scan for .wasm files
	entries, err := os.ReadDir(ps.pluginDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".wasm") {
			continue
		}

		pluginPath := filepath.Join(ps.pluginDir, name)
		plugin, err := ps.loadPlugin(pluginPath)
		if err != nil {
			continue
		}

		ps.plugins[plugin.Name] = plugin
		ps.registry[plugin.Name] = &WASMPluginFilter{plugin: plugin}
	}
}

// loadPlugin loads a single WASM plugin
func (ps *WASMPluginSystem) loadPlugin(path string) (*WASMPlugin, error) {
	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin %s: %w", path, err)
	}

	// Try to load manifest from companion .json file
	manifestPath := strings.TrimSuffix(path, ".wasm") + ".json"
	name := strings.TrimSuffix(filepath.Base(path), ".wasm")

	plugin := &WASMPlugin{
		Name:      name,
		Version:   "1.0.0",
		Path:      path,
		Enabled:   true,
		wasmBytes: wasmBytes,
	}

	// Load manifest if exists
	if manifestBytes, err := os.ReadFile(manifestPath); err == nil {
		var manifest PluginManifest
		if err := json.Unmarshal(manifestBytes, &manifest); err == nil {
			plugin.Name = manifest.Name
			plugin.Version = manifest.Version
			plugin.Description = manifest.Description
			plugin.Author = manifest.Author
		}
	}

	return plugin, nil
}

// Register registers a plugin as a filter
func (ps *WASMPluginSystem) Register(name string, filter Filter) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.registry[name] = filter
}

// GetFilter returns a registered filter by name
func (ps *WASMPluginSystem) GetFilter(name string) (Filter, bool) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()
	f, ok := ps.registry[name]
	return f, ok
}

// ListPlugins returns all loaded plugins
func (ps *WASMPluginSystem) ListPlugins() []*WASMPlugin {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	plugins := make([]*WASMPlugin, 0, len(ps.plugins))
	for _, p := range ps.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// EnablePlugin enables a plugin by name
func (ps *WASMPluginSystem) EnablePlugin(name string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if p, ok := ps.plugins[name]; ok {
		p.Enabled = true
	}
}

// DisablePlugin disables a plugin by name
func (ps *WASMPluginSystem) DisablePlugin(name string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	if p, ok := ps.plugins[name]; ok {
		p.Enabled = false
	}
}

// GetAllFilters returns all registered plugin filters
func (ps *WASMPluginSystem) GetAllFilters() []Filter {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	filters := make([]Filter, 0, len(ps.registry))
	for _, f := range ps.registry {
		filters = append(filters, f)
	}
	return filters
}

// WASMPluginFilter wraps a WASM plugin as a Filter
type WASMPluginFilter struct {
	plugin *WASMPlugin
}

// Name returns the filter name
func (f *WASMPluginFilter) Name() string {
	return f.plugin.Name
}

// Apply runs the WASM plugin filter
// In production, this would execute the WASM module via wazero.
// For now, it acts as a pass-through with metadata tracking.
func (f *WASMPluginFilter) Apply(input string, mode Mode) (string, int) {
	if !f.plugin.Enabled {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)

	// TODO: Execute WASM module via wazero runtime
	// For now, return a placeholder that indicates plugin execution
	output := fmt.Sprintf("[WASM:%s] %s", f.plugin.Name, input)
	finalTokens := core.EstimateTokens(output)

	// Plugins typically don't save tokens in this placeholder mode
	saved := originalTokens - finalTokens
	if saved < 0 {
		saved = 0
	}

	return output, saved
}

// PluginBuilder helps create WASM plugin configurations
type PluginBuilder struct {
	manifest PluginManifest
}

// newPluginBuilder creates a new plugin builder
func newPluginBuilder(name string) *PluginBuilder {
	return &PluginBuilder{
		manifest: PluginManifest{
			Name:    name,
			Version: "1.0.0",
			Entry:   "compress",
		},
	}
}

// SetDescription sets the plugin description
func (b *PluginBuilder) SetDescription(desc string) *PluginBuilder {
	b.manifest.Description = desc
	return b
}

// SetAuthor sets the plugin author
func (b *PluginBuilder) SetAuthor(author string) *PluginBuilder {
	b.manifest.Author = author
	return b
}

// SetVersion sets the plugin version
func (b *PluginBuilder) SetVersion(version string) *PluginBuilder {
	b.manifest.Version = version
	return b
}

// AddLayer adds a compression layer to the plugin
func (b *PluginBuilder) AddLayer(name, description string, minTokens, maxTokens int, weight float64) *PluginBuilder {
	b.manifest.Layers = append(b.manifest.Layers, PluginLayer{
		Name:        name,
		Description: description,
		MinTokens:   minTokens,
		MaxTokens:   maxTokens,
		Weight:      weight,
	})
	return b
}

// Build generates the plugin manifest JSON
func (b *PluginBuilder) Build() ([]byte, error) {
	return json.MarshalIndent(b.manifest, "", "  ")
}

// SaveToDir saves the plugin manifest to a directory
func (b *PluginBuilder) SaveToDir(dir string) error {
	data, err := b.Build()
	if err != nil {
		return err
	}

	os.MkdirAll(dir, 0755)
	path := filepath.Join(dir, b.manifest.Name+".json")
	return os.WriteFile(path, data, 0644)
}

// PluginRuntime manages WASM plugin execution
type PluginRuntime struct {
	ctx     context.Context
	plugins map[string]*WASMPlugin
	mu      sync.RWMutex
}

// newPluginRuntime creates a new WASM runtime manager
func newPluginRuntime() *PluginRuntime {
	return &PluginRuntime{
		ctx:     context.Background(),
		plugins: make(map[string]*WASMPlugin),
	}
}

// ExecutePlugin executes a WASM plugin with the given input
func (r *PluginRuntime) ExecutePlugin(name string, input string, mode Mode) (string, int, error) {
	r.mu.RLock()
	plugin, ok := r.plugins[name]
	r.mu.RUnlock()

	if !ok {
		return input, 0, fmt.Errorf("plugin %s not found", name)
	}

	if !plugin.Enabled {
		return input, 0, nil
	}

	// TODO: Execute via wazero WASM runtime
	// For now, return input unchanged
	return input, 0, nil
}
