package plugin

import (
	"testing"
)

// mockPlugin implements Plugin for testing.
type mockPlugin struct {
	name    string
	version string
	closed  bool
}

func (m *mockPlugin) Name() string    { return m.name }
func (m *mockPlugin) Version() string { return m.version }
func (m *mockPlugin) Apply(input string) (string, int, error) {
	// Simple mock: return input unchanged
	return input, 0, nil
}
func (m *mockPlugin) Close() error {
	m.closed = true
	return nil
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	p := &mockPlugin{name: "test-plugin", version: "1.0.0"}
	info := PluginInfo{
		Name:    "test-plugin",
		Version: "1.0.0",
		Type:    PluginTypeGo,
		Enabled: true,
	}

	if err := r.Register(p, info); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Should fail on duplicate
	if err := r.Register(p, info); err == nil {
		t.Error("Register() should fail for duplicate plugin")
	}
}

func TestRegistry_Get(t *testing.T) {
	r := NewRegistry()

	p := &mockPlugin{name: "test-plugin", version: "1.0.0"}
	info := PluginInfo{Name: "test-plugin", Type: PluginTypeGo}
	r.Register(p, info)

	got, ok := r.Get("test-plugin")
	if !ok {
		t.Fatal("Get() returned false for registered plugin")
	}
	if got.Name() != "test-plugin" {
		t.Errorf("Get() name = %v, want %v", got.Name(), "test-plugin")
	}

	_, ok = r.Get("nonexistent")
	if ok {
		t.Error("Get() should return false for nonexistent plugin")
	}
}

func TestRegistry_List(t *testing.T) {
	r := NewRegistry()

	p1 := &mockPlugin{name: "plugin-1", version: "1.0"}
	p2 := &mockPlugin{name: "plugin-2", version: "2.0"}

	r.Register(p1, PluginInfo{Name: "plugin-1", Type: PluginTypeGo})
	r.Register(p2, PluginInfo{Name: "plugin-2", Type: PluginTypeGo})

	list := r.List()
	if len(list) != 2 {
		t.Errorf("List() returned %d plugins, want 2", len(list))
	}
}

func TestRegistry_Unregister(t *testing.T) {
	r := NewRegistry()

	p := &mockPlugin{name: "test-plugin", version: "1.0.0"}
	r.Register(p, PluginInfo{Name: "test-plugin", Type: PluginTypeGo})

	if err := r.Unregister("test-plugin"); err != nil {
		t.Fatalf("Unregister() error = %v", err)
	}

	if !p.closed {
		t.Error("Unregister() should close the plugin")
	}

	_, ok := r.Get("test-plugin")
	if ok {
		t.Error("Get() should return false after Unregister()")
	}
}

func TestRegistry_Close(t *testing.T) {
	r := NewRegistry()

	p1 := &mockPlugin{name: "plugin-1"}
	p2 := &mockPlugin{name: "plugin-2"}

	r.Register(p1, PluginInfo{Name: "plugin-1", Type: PluginTypeGo})
	r.Register(p2, PluginInfo{Name: "plugin-2", Type: PluginTypeGo})

	if err := r.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if !p1.closed || !p2.closed {
		t.Error("Close() should close all plugins")
	}

	if len(r.List()) != 0 {
		t.Error("Close() should clear the registry")
	}
}

func TestPluginType_Constants(t *testing.T) {
	// Verify constants exist
	if PluginTypeGo != "go" {
		t.Errorf("PluginTypeGo = %v, want 'go'", PluginTypeGo)
	}
	if PluginTypeWASM != "wasm" {
		t.Errorf("PluginTypeWASM = %v, want 'wasm'", PluginTypeWASM)
	}
	if PluginTypeLua != "lua" {
		t.Errorf("PluginTypeLua = %v, want 'lua'", PluginTypeLua)
	}
}
