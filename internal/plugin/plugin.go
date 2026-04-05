// Package plugin provides an extensible plugin system for TokMan.
// Currently supports Go plugins. WASM plugin support is planned.
//
// TODO: Add WASM plugin support using github.com/tetratelabs/wazero
// See: https://github.com/tetratelabs/wazero
package plugin

import (
	"fmt"
	"sync"
)

// Plugin represents a TokMan plugin that can process text.
type Plugin interface {
	// Name returns the plugin's unique identifier.
	Name() string

	// Version returns the plugin version string.
	Version() string

	// Apply processes input text and returns the result.
	// Returns the processed text and number of tokens saved.
	Apply(input string) (output string, tokensSaved int, err error)

	// Close releases any resources held by the plugin.
	Close() error
}

// PluginType represents the type of plugin.
type PluginType string

const (
	// PluginTypeGo is a native Go plugin.
	PluginTypeGo PluginType = "go"

	// PluginTypeWASM is a WebAssembly plugin (planned).
	// TODO: Implement WASM support with wazero runtime.
	PluginTypeWASM PluginType = "wasm"

	// PluginTypeLua is a Lua script plugin (planned).
	PluginTypeLua PluginType = "lua"
)

// PluginInfo contains metadata about a plugin.
type PluginInfo struct {
	Name        string     `json:"name"`
	Version     string     `json:"version"`
	Type        PluginType `json:"type"`
	Path        string     `json:"path"`
	Description string     `json:"description,omitempty"`
	Author      string     `json:"author,omitempty"`
	Enabled     bool       `json:"enabled"`
}

// Registry manages loaded plugins.
type Registry struct {
	mu      sync.RWMutex
	plugins map[string]Plugin
	info    map[string]PluginInfo
}

// NewRegistry creates a new plugin registry.
func NewRegistry() *Registry {
	return &Registry{
		plugins: make(map[string]Plugin),
		info:    make(map[string]PluginInfo),
	}
}

// Register adds a plugin to the registry.
func (r *Registry) Register(p Plugin, info PluginInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := p.Name()
	if _, exists := r.plugins[name]; exists {
		return fmt.Errorf("plugin %q already registered", name)
	}

	r.plugins[name] = p
	r.info[name] = info
	return nil
}

// Get retrieves a plugin by name.
func (r *Registry) Get(name string) (Plugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.plugins[name]
	return p, ok
}

// List returns info about all registered plugins.
func (r *Registry) List() []PluginInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]PluginInfo, 0, len(r.info))
	for _, info := range r.info {
		result = append(result, info)
	}
	return result
}

// Unregister removes a plugin from the registry.
func (r *Registry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.plugins[name]
	if !ok {
		return fmt.Errorf("plugin %q not found", name)
	}

	if err := p.Close(); err != nil {
		return fmt.Errorf("close plugin %q: %w", name, err)
	}

	delete(r.plugins, name)
	delete(r.info, name)
	return nil
}

// Close closes all plugins and clears the registry.
func (r *Registry) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var errs []error
	for name, p := range r.plugins {
		if err := p.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close %q: %w", name, err))
		}
	}

	r.plugins = make(map[string]Plugin)
	r.info = make(map[string]PluginInfo)

	if len(errs) > 0 {
		return fmt.Errorf("errors closing plugins: %v", errs)
	}
	return nil
}

// DefaultRegistry is the global plugin registry.
var DefaultRegistry = NewRegistry()

// --- Future WASM Support ---
//
// TODO: Implement WASMPlugin using wazero:
//
// type WASMPlugin struct {
//     runtime wazero.Runtime
//     module  wazero.CompiledModule
//     name    string
//     version string
// }
//
// func LoadWASM(path string) (*WASMPlugin, error) {
//     ctx := context.Background()
//     r := wazero.NewRuntime(ctx)
//     
//     wasm, err := os.ReadFile(path)
//     if err != nil {
//         return nil, err
//     }
//     
//     module, err := r.CompileModule(ctx, wasm)
//     if err != nil {
//         return nil, err
//     }
//     
//     return &WASMPlugin{
//         runtime: r,
//         module:  module,
//     }, nil
// }
