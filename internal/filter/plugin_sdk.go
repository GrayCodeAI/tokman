package filter

import "strings"

// ExamplePlugin is a template showing how to implement SDKFilter.
// Copy this template, fill in your logic, and register via WrapSDKFilter.
const ExamplePlugin = `
// Example: implement SDKFilter and register it with WrapSDKFilter.
//
// package myplugin
//
// import "github.com/GrayCodeAI/tokman/internal/filter"
//
// type MyFilter struct{}
//
// func (f *MyFilter) Name() string        { return "my-filter" }
// func (f *MyFilter) Description() string { return "My custom filter" }
// func (f *MyFilter) SupportedModes() []string { return []string{"none", "minimal", "aggressive"} }
// func (f *MyFilter) Apply(input string, mode string) (string, int) {
//     // Your filtering logic here
//     return input, 0
// }
//
// // Register with an engine:
// // engine.AddFilter(filter.WrapSDKFilter(&MyFilter{}))
`

// SDKFilter is the interface external plugin authors must implement.
// It mirrors Filter but uses plain string types for portability.
type SDKFilter interface {
	// Name returns the filter's unique name.
	Name() string
	// Description returns a human-readable description of the filter.
	Description() string
	// SupportedModes returns the list of modes this filter supports.
	SupportedModes() []string
	// Apply processes input in the given mode and returns the result and
	// the number of tokens saved.
	Apply(input string, mode string) (string, int)
}

// sdkFilterAdapter wraps an SDKFilter so it satisfies the internal Filter interface.
type sdkFilterAdapter struct {
	inner SDKFilter
}

// WrapSDKFilter wraps an SDKFilter as a standard Filter suitable for use
// in a filter Engine.
func WrapSDKFilter(f SDKFilter) Filter {
	return &sdkFilterAdapter{inner: f}
}

func (a *sdkFilterAdapter) Name() string {
	return a.inner.Name()
}

func (a *sdkFilterAdapter) Apply(input string, mode Mode) (string, int) {
	return a.inner.Apply(input, string(mode))
}

// PluginSDK is the entry point for external plugin authors. It provides
// helpers for manifest validation and filter wrapping.
type PluginSDK struct{}

// NewPluginSDK returns an initialised PluginSDK.
func NewPluginSDK() *PluginSDK {
	return &PluginSDK{}
}

// Wrap is a convenience method on PluginSDK that calls WrapSDKFilter.
func (s *PluginSDK) Wrap(f SDKFilter) Filter {
	return WrapSDKFilter(f)
}

// Validate is a convenience method on PluginSDK that calls ValidateManifest.
func (s *PluginSDK) Validate(m PluginManifestV2) []string {
	return ValidateManifest(m)
}

// PluginManifestV2 describes a plugin for registration and discovery.
type PluginManifestV2 struct {
	Name           string   `json:"name"`
	Version        string   `json:"version"`
	Description    string   `json:"description"`
	Author         string   `json:"author"`
	SupportedModes []string `json:"supported_modes"`
	MinSDKVersion  string   `json:"min_sdk_version"`
}

// ValidateManifest validates a PluginManifestV2 and returns a slice of
// human-readable error strings. An empty slice indicates the manifest is valid.
func ValidateManifest(m PluginManifestV2) []string {
	var errs []string

	if strings.TrimSpace(m.Name) == "" {
		errs = append(errs, "manifest: Name must not be empty")
	}
	if strings.TrimSpace(m.Version) == "" {
		errs = append(errs, "manifest: Version must not be empty")
	}
	if strings.TrimSpace(m.Author) == "" {
		errs = append(errs, "manifest: Author must not be empty")
	}
	if strings.TrimSpace(m.MinSDKVersion) == "" {
		errs = append(errs, "manifest: MinSDKVersion must not be empty")
	}
	if len(m.SupportedModes) == 0 {
		errs = append(errs, "manifest: SupportedModes must not be empty")
	} else {
		validModes := map[string]bool{"none": true, "minimal": true, "aggressive": true}
		for _, mode := range m.SupportedModes {
			if !validModes[mode] {
				errs = append(errs, "manifest: unknown mode "+mode+" (valid: none, minimal, aggressive)")
			}
		}
	}

	return errs
}
