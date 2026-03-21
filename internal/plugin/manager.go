package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

// Plugin is the interface that all TokMan plugins must implement
type Plugin interface {
	// Name returns the plugin name
	Name() string

	// Version returns the plugin version
	Version() string

	// Description returns a human-readable description
	Description() string

	// Filters returns the filters provided by this plugin
	Filters() []filter.Filter

	// Commands returns additional CLI commands (optional)
	Commands() []Command

	// Init initializes the plugin (called once on load)
	Init(config map[string]interface{}) error

	// Cleanup releases resources (called on unload)
	Cleanup() error
}

// Command represents an additional CLI command from a plugin
type Command struct {
	Name        string
	Description string
	Handler     func(args []string) error
}

// Manager manages plugin lifecycle
type Manager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin

	// Plugin directories in search order
	pluginDirs []string

	// Loaded .so plugin handles for cleanup
	handles map[string]*plugin.Plugin
}

// NewManager creates a new plugin manager
func NewManager(pluginDirs ...string) *Manager {
	if len(pluginDirs) == 0 {
		// Default plugin directories
		home, _ := os.UserHomeDir()
		pluginDirs = []string{
			filepath.Join(home, ".config", "tokman", "plugins", "native"),
			"/usr/local/lib/tokman/plugins",
			"/usr/lib/tokman/plugins",
		}
	}

	return &Manager{
		plugins:    make(map[string]Plugin),
		pluginDirs: pluginDirs,
		handles:    make(map[string]*plugin.Plugin),
	}
}

// LoadAll loads all plugins from configured directories
func (m *Manager) LoadAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, dir := range m.pluginDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			continue
		}

		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())

			// Load Go .so plugins
			if strings.HasSuffix(entry.Name(), ".so") {
				if err := m.loadGoPlugin(path); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to load plugin %s: %v\n", path, err)
				}
			}

			// WASM plugins handled separately via wasm.go
			if strings.HasSuffix(entry.Name(), ".wasm") {
				if err := m.loadWasmPlugin(path); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to load WASM plugin %s: %v\n", path, err)
				}
			}
		}
	}

	return nil
}

// loadGoPlugin loads a Go .so plugin
func (m *Manager) loadGoPlugin(path string) error {
	// Open the .so file
	handle, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look for the Plugin symbol
	sym, err := handle.Lookup("Plugin")
	if err != nil {
		return fmt.Errorf("plugin missing 'Plugin' symbol: %w", err)
	}

	p, ok := sym.(Plugin)
	if !ok {
		return fmt.Errorf("plugin symbol is not a Plugin interface")
	}

	// Initialize the plugin
	if err := p.Init(nil); err != nil {
		return fmt.Errorf("plugin init failed: %w", err)
	}

	// Store the plugin
	name := p.Name()
	m.plugins[name] = p
	m.handles[name] = handle

	return nil
}

// loadWasmPlugin loads a WASM plugin (delegates to wasm.go)
func (m *Manager) loadWasmPlugin(path string) error {
	wasmPlugin, err := LoadWasmPlugin(path)
	if err != nil {
		return err
	}

	m.plugins[wasmPlugin.Name()] = wasmPlugin
	return nil
}

// GetPlugin retrieves a loaded plugin by name
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.plugins[name]
	return p, ok
}

// AllPlugins returns all loaded plugins
func (m *Manager) AllPlugins() []Plugin {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugins := make([]Plugin, 0, len(m.plugins))
	for _, p := range m.plugins {
		plugins = append(plugins, p)
	}
	return plugins
}

// AllFilters returns all filters from all loaded plugins
func (m *Manager) AllFilters() []filter.Filter {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var filters []filter.Filter
	for _, p := range m.plugins {
		filters = append(filters, p.Filters()...)
	}
	return filters
}

// Unload unloads a plugin by name
func (m *Manager) Unload(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	p, ok := m.plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}

	// Call cleanup
	if err := p.Cleanup(); err != nil {
		return fmt.Errorf("cleanup failed: %w", err)
	}

	// Remove from maps
	delete(m.plugins, name)
	delete(m.handles, name)

	return nil
}

// UnloadAll unloads all plugins
func (m *Manager) UnloadAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error
	for name, p := range m.plugins {
		if err := p.Cleanup(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", name, err))
		}
		delete(m.plugins, name)
		delete(m.handles, name)
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors during unload: %v", errs)
	}
	return nil
}

// PluginInfo returns information about a loaded plugin
type PluginInfo struct {
	Name        string
	Version     string
	Description string
	FilterCount int
	Type        string // "native" or "wasm"
}

// ListPlugins returns info about all loaded plugins
func (m *Manager) ListPlugins() []PluginInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var infos []PluginInfo
	for name, p := range m.plugins {
		_, isWasm := p.(*WasmPlugin)
		pType := "native"
		if isWasm {
			pType = "wasm"
		}

		infos = append(infos, PluginInfo{
			Name:        name,
			Version:     p.Version(),
			Description: p.Description(),
			FilterCount: len(p.Filters()),
			Type:        pType,
		})
	}
	return infos
}
