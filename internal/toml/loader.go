package toml

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

//go:embed builtin/*.toml
var builtinFilters embed.FS

// Loader handles loading TOML filters from multiple sources
type Loader struct {
	parser        *Parser
	trustedPaths  map[string]bool
	trustedFile   string
	mu            sync.RWMutex
}

// LoaderConfig configures the filter loader
type LoaderConfig struct {
	ConfigDir  string // Base config directory (default: ~/.config/tokman)
	DataDir    string // Base data directory (default: ~/.local/share/tokman)
	ProjectDir string // Current project directory
}

// NewLoader creates a new filter loader
func NewLoader(configDir string) *Loader {
	if configDir == "" {
		homeDir, _ := os.UserHomeDir()
		configDir = filepath.Join(homeDir, ".config", "tokman")
	}

	trustedFile := filepath.Join(configDir, "trusted.json")

	return &Loader{
		parser:       NewParser(),
		trustedPaths: make(map[string]bool),
		trustedFile:  trustedFile,
	}
}

// LoadAll loads filters from all sources with priority order:
// 1. Project-local: .tokman/filters.toml (requires trust)
// 2. User-global: ~/.config/tokman/filters.toml
// 3. Built-in: embedded filters
func (l *Loader) LoadAll(projectDir string) (*FilterRegistry, error) {
	registry := NewFilterRegistry()

	// Load built-in filters first (lowest priority)
	if err := l.loadBuiltin(registry); err != nil {
		return nil, fmt.Errorf("failed to load builtin filters: %w", err)
	}

	// Load user-global filters
	homeDir, _ := os.UserHomeDir()
	userFiltersPath := filepath.Join(homeDir, ".config", "tokman", "filters.toml")
	if _, err := os.Stat(userFiltersPath); err == nil {
		if err := registry.LoadFile(userFiltersPath); err != nil {
			// Log warning but don't fail
			fmt.Fprintf(os.Stderr, "Warning: failed to load user filters: %v\n", err)
		}
	}

	// Load project-local filters (highest priority)
	if projectDir != "" {
		projectFiltersPath := filepath.Join(projectDir, ".tokman", "filters.toml")
		if _, err := os.Stat(projectFiltersPath); err == nil {
			// Check if project is trusted
			if l.IsTrusted(projectDir) {
				if err := registry.LoadFile(projectFiltersPath); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to load project filters: %v\n", err)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Warning: project filters not loaded (run 'tokman trust' to enable)\n")
			}
		}
	}

	return registry, nil
}

// loadBuiltin loads embedded built-in filters
func (l *Loader) loadBuiltin(registry *FilterRegistry) error {
	entries, err := builtinFilters.ReadDir("builtin")
	if err != nil {
		return fmt.Errorf("failed to read builtin filters: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}

		content, err := builtinFilters.ReadFile(filepath.Join("builtin", entry.Name()))
		if err != nil {
			return fmt.Errorf("failed to read builtin filter %s: %w", entry.Name(), err)
		}

		filter, err := l.parser.ParseContent(content, "builtin/"+entry.Name())
		if err != nil {
			return fmt.Errorf("failed to parse builtin filter %s: %w", entry.Name(), err)
		}

		if err := filter.Validate(); err != nil {
			return fmt.Errorf("invalid builtin filter %s: %w", entry.Name(), err)
		}

		registry.filters[entry.Name()] = filter
	}

	return nil
}

// TrustProject marks a project directory as trusted for filter loading
func (l *Loader) TrustProject(projectPath string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Resolve to absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve project path: %w", err)
	}

	l.trustedPaths[absPath] = true

	// Persist to file
	return l.saveTrusted()
}

// UntrustProject removes a project from the trusted list
func (l *Loader) UntrustProject(projectPath string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("failed to resolve project path: %w", err)
	}

	delete(l.trustedPaths, absPath)

	return l.saveTrusted()
}

// IsTrusted checks if a project directory is trusted
func (l *Loader) IsTrusted(projectPath string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()

	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return false
	}

	return l.trustedPaths[absPath]
}

// LoadTrusted loads the trusted projects file
func (l *Loader) LoadTrusted() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := os.ReadFile(l.trustedFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist yet
		}
		return fmt.Errorf("failed to read trusted file: %w", err)
	}

	// Parse simple line-based format
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			l.trustedPaths[line] = true
		}
	}

	return nil
}

// saveTrusted saves the trusted projects to file
func (l *Loader) saveTrusted() error {
	// Ensure directory exists
	dir := filepath.Dir(l.trustedFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	var lines []string
	lines = append(lines, "# TokMan Trusted Projects")
	lines = append(lines, "# Generated by 'tokman trust' command")
	lines = append(lines, "")

	for path := range l.trustedPaths {
		lines = append(lines, path)
	}

	content := strings.Join(lines, "\n")
	return os.WriteFile(l.trustedFile, []byte(content), 0600)
}

// ListTrusted returns all trusted project paths
func (l *Loader) ListTrusted() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	paths := make([]string, 0, len(l.trustedPaths))
	for path := range l.trustedPaths {
		paths = append(paths, path)
	}
	return paths
}

// GlobalFilterLoader is the default loader instance
var GlobalFilterLoader *Loader
var loaderOnce sync.Once

// GetLoader returns the global filter loader (singleton)
func GetLoader() *Loader {
	loaderOnce.Do(func() {
		GlobalFilterLoader = NewLoader("")
		GlobalFilterLoader.LoadTrusted()
	})
	return GlobalFilterLoader
}
