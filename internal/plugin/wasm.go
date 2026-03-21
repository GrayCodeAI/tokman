package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

// WasmPlugin wraps a WASM module as a Plugin
type WasmPlugin struct {
	mu      sync.RWMutex
	name    string
	version string
	desc    string
	filters []filter.Filter
	ctx     context.Context
	runtime wazero.Runtime
	module  api.Module
	config  map[string]interface{}
}

// wasmFilter wraps a WASM filter function
type wasmFilter struct {
	name   string
	desc   string
	plugin *WasmPlugin
	fnName string
}

// LoadWasmPlugin loads a WASM plugin from file
func LoadWasmPlugin(path string) (*WasmPlugin, error) {
	wasmBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read WASM file: %w", err)
	}

	ctx := context.Background()

	// Create runtime
	rt := wazero.NewRuntime(ctx)

	// Instantiate WASI (required for most WASM modules)
	wasi_snapshot_preview1.MustInstantiate(ctx, rt)

	// Compile and instantiate the module
	compiled, err := rt.CompileModule(ctx, wasmBytes)
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("failed to compile WASM: %w", err)
	}

	module, err := rt.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		rt.Close(ctx)
		return nil, fmt.Errorf("failed to instantiate WASM: %w", err)
	}

	p := &WasmPlugin{
		ctx:     ctx,
		runtime: rt,
		module:  module,
	}

	// Read plugin metadata from WASM
	if err := p.loadMetadata(); err != nil {
		p.Cleanup()
		return nil, fmt.Errorf("failed to load metadata: %w", err)
	}

	return p, nil
}

// loadMetadata reads plugin metadata from WASM exports
func (p *WasmPlugin) loadMetadata() error {
	// Get plugin name
	if nameFn := p.module.ExportedFunction("plugin_name"); nameFn != nil {
		if result, err := nameFn.Call(p.ctx); err == nil && len(result) > 0 {
			p.name = p.readString(result[0])
		}
	}

	// Get plugin version
	if versionFn := p.module.ExportedFunction("plugin_version"); versionFn != nil {
		if result, err := versionFn.Call(p.ctx); err == nil && len(result) > 0 {
			p.version = p.readString(result[0])
		}
	}

	// Get plugin description
	if descFn := p.module.ExportedFunction("plugin_description"); descFn != nil {
		if result, err := descFn.Call(p.ctx); err == nil && len(result) > 0 {
			p.desc = p.readString(result[0])
		}
	}

	// Get filter definitions
	if filtersFn := p.module.ExportedFunction("plugin_filters"); filtersFn != nil {
		if result, err := filtersFn.Call(p.ctx); err == nil && len(result) > 0 {
			filterJSON := p.readString(result[0])
			var filterDefs []struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Function    string `json:"function"`
			}
			if err := json.Unmarshal([]byte(filterJSON), &filterDefs); err == nil {
				for _, def := range filterDefs {
					p.filters = append(p.filters, &wasmFilter{
						name:   def.Name,
						desc:   def.Description,
						plugin: p,
						fnName: def.Function,
					})
				}
			}
		}
	}

	// Set defaults if not provided
	if p.name == "" {
		p.name = "unknown"
	}
	if p.version == "" {
		p.version = "0.0.0"
	}

	return nil
}

// readString reads a string from WASM memory
func (p *WasmPlugin) readString(ptr uint64) string {
	mem := p.module.Memory()
	if mem == nil {
		return ""
	}

	// Read length-prefixed string (first 4 bytes = length)
	lenBuf, ok := mem.Read(uint32(ptr), 4)
	if !ok {
		return ""
	}
	length := uint32(lenBuf[0]) | uint32(lenBuf[1])<<8 | uint32(lenBuf[2])<<16 | uint32(lenBuf[3])<<24

	// Read string data
	if length > 4096 { // sanity limit
		length = 4096
	}
	data, ok := mem.Read(uint32(ptr)+4, length)
	if !ok {
		return ""
	}

	return string(data)
}

// writeString writes a string to WASM memory, returns pointer
func (p *WasmPlugin) writeString(s string) (uint32, error) {
	mem := p.module.Memory()
	if mem == nil {
		return 0, fmt.Errorf("no memory")
	}

	// Allocate in WASM (call malloc)
	malloc := p.module.ExportedFunction("malloc")
	if malloc == nil {
		return 0, fmt.Errorf("no malloc export")
	}

	data := []byte(s)
	length := uint32(len(data))

	// Allocate length + 4 bytes (for length prefix)
	result, err := malloc.Call(p.ctx, uint64(length+4))
	if err != nil {
		return 0, err
	}
	ptr := uint32(result[0])

	// Write length prefix
	lenBuf := []byte{
		byte(length),
		byte(length >> 8),
		byte(length >> 16),
		byte(length >> 24),
	}
	mem.Write(ptr, lenBuf)

	// Write string data
	mem.Write(ptr+4, data)

	return ptr, nil
}

// Name returns the plugin name
func (p *WasmPlugin) Name() string {
	return p.name
}

// Version returns the plugin version
func (p *WasmPlugin) Version() string {
	return p.version
}

// Description returns the plugin description
func (p *WasmPlugin) Description() string {
	return p.desc
}

// Filters returns the filters provided by this plugin
func (p *WasmPlugin) Filters() []filter.Filter {
	return p.filters
}

// Commands returns additional CLI commands (not supported for WASM)
func (p *WasmPlugin) Commands() []Command {
	return nil
}

// Init initializes the plugin
func (p *WasmPlugin) Init(config map[string]interface{}) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if config != nil {
		p.config = config
	}

	// Call WASM init if exported
	if initFn := p.module.ExportedFunction("plugin_init"); initFn != nil {
		// Pass config as JSON
		configJSON, _ := json.Marshal(config)
		ptr, err := p.writeString(string(configJSON))
		if err != nil {
			return err
		}
		_, err = initFn.Call(p.ctx, uint64(ptr))
		return err
	}

	return nil
}

// Cleanup releases WASM resources
func (p *WasmPlugin) Cleanup() error {
	if p.module != nil {
		p.module.Close(p.ctx)
	}
	if p.runtime != nil {
		p.runtime.Close(p.ctx)
	}
	return nil
}

// wasmFilter implementation

func (f *wasmFilter) Name() string {
	return f.name
}

func (f *wasmFilter) Description() string {
	return f.desc
}

func (f *wasmFilter) Enabled() bool {
	return true
}

func (f *wasmFilter) Apply(input string, mode filter.Mode) (string, int) {
	f.plugin.mu.RLock()
	defer f.plugin.mu.RUnlock()

	// Call WASM filter function
	filterFn := f.plugin.module.ExportedFunction(f.fnName)
	if filterFn == nil {
		return input, 0 // passthrough if function not found
	}

	// Write input to WASM memory
	ptr, err := f.plugin.writeString(input)
	if err != nil {
		return input, 0
	}

	// Write mode to WASM memory
	modePtr, err := f.plugin.writeString(string(mode))
	if err != nil {
		return input, 0
	}

	// Call filter with input and mode pointers
	result, err := filterFn.Call(f.plugin.ctx, uint64(ptr), uint64(modePtr))
	if err != nil {
		return input, 0
	}

	// Read output (result is pointer to output string)
	output := f.plugin.readString(result[0])

	// Calculate tokens saved
	tokensSaved := filter.EstimateTokens(input) - filter.EstimateTokens(output)
	if tokensSaved < 0 {
		tokensSaved = 0
	}

	return output, tokensSaved
}
