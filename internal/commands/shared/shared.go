package shared

// Package shared provides global state, utilities, and the fallback handler
// for CLI command processing.
//
// This file re-exports from split files for backward compatibility:
//   - flags.go: Global flags and thread-safe accessors
//   - config.go: Configuration loading and caching
//   - executor.go: Command execution and recording
//   - fallback.go: TOML-based fallback handler
//   - utils.go: Utility functions (truncation, sanitization, etc.)
//
// The package has been refactored to reduce coupling:
//   - flags.go has NO external tokman dependencies
//   - utils.go has NO external tokman dependencies
//   - config.go depends only on internal/config
//   - executor.go depends on core, tracking, tee, config
//   - fallback.go has the most dependencies (used by root command)

// Backward compatibility: SetConfig alias for SetFlags
// Deprecated: Use SetFlags instead
func SetConfig(cfg FlagConfig) {
	SetFlags(cfg)
}
