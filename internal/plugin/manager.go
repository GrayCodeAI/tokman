package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

// Metadata provides basic plugin identification.
type Metadata interface {
	Name() string
	Version() string
	Description() string
}

// FilterProvider provides compression filters.
type FilterProvider interface {
	Filters() []filter.Filter
}

// CommandProvider provides CLI commands.
type CommandProvider interface {
	Commands() []Command
}

// Lifecycle manages plugin initialization and cleanup.
type Lifecycle interface {
	Init(config map[string]any) error
	Cleanup() error
}

// Plugin is the composition of all plugin interfaces.
// Any type implementing Metadata + Lifecycle can be a minimal plugin.
// Full plugins also implement FilterProvider and/or CommandProvider.
type Plugin interface {
	Metadata
	Lifecycle
	// Optional interfaces checked at runtime via type assertions:
	// - FilterProvider (if Filters() is implemented)
	// - CommandProvider (if Commands() is implemented)
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
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
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
	// Verify plugin checksum before loading
	if err := m.verifyPlugin(path); err != nil {
		return err
	}

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
	// Verify plugin checksum before loading
	if err := m.verifyPlugin(path); err != nil {
		return err
	}

	wasmPlugin, err := LoadWasmPlugin(path)
	if err != nil {
		return err
	}

	m.plugins[wasmPlugin.Name()] = wasmPlugin
	return nil
}

// verifyPlugin checks the plugin file's SHA-256 hash against a .sha256 sidecar file
func (m *Manager) verifyPlugin(path string) error {
	hashFile := path + ".sha256"
	if _, err := os.Stat(hashFile); os.IsNotExist(err) {
		// No hash file - log warning but allow (backward compatible)
		fmt.Fprintf(os.Stderr, "Warning: no checksum file for plugin %s (run 'tokman plugin verify')\n", path)
		return nil
	}
	// Read expected hash
	expectedBytes, err := os.ReadFile(hashFile)
	if err != nil {
		return fmt.Errorf("failed to read plugin checksum: %w", err)
	}
	expected := strings.TrimSpace(string(expectedBytes))

	// Compute actual hash
	actual, err := computeFileHash(path)
	if err != nil {
		return fmt.Errorf("failed to compute plugin hash: %w", err)
	}

	if expected != actual {
		return fmt.Errorf("plugin checksum mismatch: expected %s, got %s", expected[:16], actual[:16])
	}
	return nil
}

func computeFileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
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
		if fp, ok := p.(FilterProvider); ok {
			filters = append(filters, fp.Filters()...)
		}
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
	// Collect plugins under lock, then release before calling Cleanup
	// to avoid deadlock if Cleanup tries to acquire the same mutex.
	m.mu.Lock()
	type entry struct {
		name string
		p    Plugin
	}
	items := make([]entry, 0, len(m.plugins))
	for name, p := range m.plugins {
		items = append(items, entry{name, p})
	}
	// Clear maps while holding lock
	m.plugins = make(map[string]Plugin)
	m.handles = make(map[string]*plugin.Plugin)
	m.mu.Unlock()

	var errs []error
	for _, item := range items {
		if err := item.p.Cleanup(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", item.name, err))
		}
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

		filterCount := 0
		if fp, ok := p.(FilterProvider); ok {
			filterCount = len(fp.Filters())
		}

		infos = append(infos, PluginInfo{
			Name:        name,
			Version:     p.Version(),
			Description: p.Description(),
			FilterCount: filterCount,
			Type:        pType,
		})
	}
	return infos
}
