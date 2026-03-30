// Package tomlfilter provides TOML-based declarative filtering for command output.
// This enables users to define filters without writing Go code.
package tomlfilter

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

// TomlFilter represents a single filter definition from a TOML file.
type TomlFilter struct {
	// Name is the filter identifier (from TOML table name)
	Name string `toml:"-"`

	// Description is a human-readable description of the filter
	Description string `toml:"description"`

	// MatchCommand is a regex pattern to match against the full command string
	MatchCommand string `toml:"match_command"`

	// StripANSI removes ANSI escape codes before processing
	StripANSI bool `toml:"strip_ansi"`

	// StripLinesMatching drops lines matching any of these regex patterns
	StripLinesMatching []string `toml:"strip_lines_matching"`

	// KeepLinesMatching keeps only lines matching at least one of these regex patterns
	KeepLinesMatching []string `toml:"keep_lines_matching"`

	// Replace applies regex substitutions
	Replace []ReplaceRule `toml:"replace"`

	// MatchOutput applies short-circuit rules based on output content
	MatchOutput []MatchRule `toml:"match_output"`

	// TruncateLinesAt truncates lines longer than N characters
	TruncateLinesAt int `toml:"truncate_lines_at"`

	// MaxLines keeps only the first N lines after filtering
	MaxLines int `toml:"max_lines"`

	// TailLines keeps only the last N lines (applied after other filters)
	TailLines int `toml:"tail_lines"`

	// OnEmpty is the fallback message when filtered output is empty
	OnEmpty string `toml:"on_empty"`

	// Compiled regex patterns (cached for performance)
	matchCommandRegex     *regexp.Regexp
	stripLinesRegex       []*regexp.Regexp
	keepLinesRegex        []*regexp.Regexp
	replaceRegex          []*regexp.Regexp
	matchOutputRegex      []*regexp.Regexp
	ansiStripRegex        *regexp.Regexp
}

// ReplaceRule defines a regex substitution rule.
type ReplaceRule struct {
	Pattern     string `toml:"pattern"`
	Replacement string `toml:"replacement"`
}

// MatchRule defines a short-circuit rule based on output matching.
type MatchRule struct {
	Pattern  string `toml:"pattern"`
	Message  string `toml:"message"`
}

// FilterTest represents an inline test case in a TOML filter.
type FilterTest struct {
	Name     string `toml:"name"`
	Input    string `toml:"input"`
	Expected string `toml:"expected"`
}

// TomlFilterFile represents the structure of a TOML filter file.
type TomlFilterFile struct {
	Filters map[string]TomlFilter `toml:"filters"`
	Tests   map[string][]FilterTest `toml:"tests"`
}

// Registry holds loaded TOML filters and provides matching functionality.
type Registry struct {
	filters []*TomlFilter
	byName  map[string]*TomlFilter
}

// NewRegistry creates an empty filter registry.
func NewRegistry() *Registry {
	return &Registry{
		filters: make([]*TomlFilter, 0),
		byName:  make(map[string]*TomlFilter),
	}
}

// compileRegex compiles all regex patterns in the filter for efficient matching.
func (f *TomlFilter) compileRegex() error {
	var err error

	// Compile match_command regex
	if f.MatchCommand != "" {
		f.matchCommandRegex, err = regexp.Compile(f.MatchCommand)
		if err != nil {
			return fmt.Errorf("invalid match_command regex: %w", err)
		}
	}

	// Compile strip_lines_matching regexes
	for _, pattern := range f.StripLinesMatching {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid strip_lines_matching regex %q: %w", pattern, err)
		}
		f.stripLinesRegex = append(f.stripLinesRegex, re)
	}

	// Compile keep_lines_matching regexes
	for _, pattern := range f.KeepLinesMatching {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return fmt.Errorf("invalid keep_lines_matching regex %q: %w", pattern, err)
		}
		f.keepLinesRegex = append(f.keepLinesRegex, re)
	}

	// Compile replace regexes
	for _, rule := range f.Replace {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return fmt.Errorf("invalid replace regex %q: %w", rule.Pattern, err)
		}
		f.replaceRegex = append(f.replaceRegex, re)
	}

	// Compile match_output regexes
	for _, rule := range f.MatchOutput {
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return fmt.Errorf("invalid match_output regex %q: %w", rule.Pattern, err)
		}
		f.matchOutputRegex = append(f.matchOutputRegex, re)
	}

	// Compile ANSI strip regex
	f.ansiStripRegex = regexp.MustCompile("\x1b\\[[0-9;]*[a-zA-Z]")

	return nil
}

// Matches checks if this filter should be applied to the given command.
func (f *TomlFilter) Matches(command string) bool {
	if f.matchCommandRegex == nil {
		return false
	}
	return f.matchCommandRegex.MatchString(command)
}

// Apply applies the filter to the input and returns the filtered output.
func (f *TomlFilter) Apply(input string) string {
	// Step 1: Strip ANSI codes if enabled
	if f.StripANSI && f.ansiStripRegex != nil {
		input = f.ansiStripRegex.ReplaceAllString(input, "")
	}

	// Step 2: Check match_output short-circuit rules
	for i, rule := range f.MatchOutput {
		if i < len(f.matchOutputRegex) && f.matchOutputRegex[i].MatchString(input) {
			return rule.Message
		}
	}

	// Step 3: Apply replace rules
	for i, rule := range f.Replace {
		if i < len(f.replaceRegex) {
			input = f.replaceRegex[i].ReplaceAllString(input, rule.Replacement)
		}
	}

	// Step 4: Process line by line
	lines := strings.Split(input, "\n")
	var filteredLines []string

	for _, line := range lines {
		// Apply truncate_lines_at
		if f.TruncateLinesAt > 0 && len(line) > f.TruncateLinesAt {
			line = line[:f.TruncateLinesAt]
		}

		// Check strip_lines_matching
		shouldStrip := false
		for _, re := range f.stripLinesRegex {
			if re.MatchString(line) {
				shouldStrip = true
				break
			}
		}
		if shouldStrip {
			continue
		}

		// Check keep_lines_matching (if any defined, line must match at least one)
		if len(f.keepLinesRegex) > 0 {
			shouldKeep := false
			for _, re := range f.keepLinesRegex {
				if re.MatchString(line) {
					shouldKeep = true
					break
				}
			}
			if !shouldKeep {
				continue
			}
		}

		filteredLines = append(filteredLines, line)
	}

	// Step 5: Apply max_lines truncation
	if f.MaxLines > 0 && len(filteredLines) > f.MaxLines {
		filteredLines = filteredLines[:f.MaxLines]
	}

	// Step 6: Apply tail_lines (last N lines)
	if f.TailLines > 0 && len(filteredLines) > f.TailLines {
		filteredLines = filteredLines[len(filteredLines)-f.TailLines:]
	}

	// Step 7: Build output
	output := strings.Join(filteredLines, "\n")
	output = strings.TrimSpace(output)

	// Step 8: Apply on_empty fallback
	if output == "" && f.OnEmpty != "" {
		return f.OnEmpty
	}

	return output
}

// LoadFilters loads all TOML filter files from the given directory.
func (r *Registry) LoadFilters(directory string) error {
	// Check if directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return fmt.Errorf("filter directory does not exist: %s", directory)
	}

	// Walk the directory and load .toml files
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-TOML files
		if info.IsDir() || !strings.HasSuffix(path, ".toml") {
			return nil
		}

		// Parse TOML file
		var filterFile TomlFilterFile
		if _, err := toml.DecodeFile(path, &filterFile); err != nil {
			return fmt.Errorf("failed to parse %s: %w", path, err)
		}

		// Add filters to registry
		for name, filter := range filterFile.Filters {
			filter.Name = name
			if err := filter.compileRegex(); err != nil {
				return fmt.Errorf("filter %q in %s: %w", name, path, err)
			}
			r.filters = append(r.filters, &filter)
			r.byName[name] = &filter
		}

		return nil
	})

	return err
}

// LoadFromEmbedded loads filters from embedded TOML content (for built-in filters).
func (r *Registry) LoadFromEmbedded(content string) error {
	var filterFile TomlFilterFile
	if _, err := toml.Decode(content, &filterFile); err != nil {
		return fmt.Errorf("failed to parse embedded TOML: %w", err)
	}

	for name, filter := range filterFile.Filters {
		filter.Name = name
		if err := filter.compileRegex(); err != nil {
			return fmt.Errorf("embedded filter %q: %w", name, err)
		}
		r.filters = append(r.filters, &filter)
		r.byName[name] = &filter
	}

	return nil
}

// MatchFilter finds a filter that matches the given command.
// Returns nil if no filter matches.
func (r *Registry) MatchFilter(command string) *TomlFilter {
	for _, filter := range r.filters {
		if filter.Matches(command) {
			return filter
		}
	}
	return nil
}

// GetFilter returns a filter by name, or nil if not found.
func (r *Registry) GetFilter(name string) *TomlFilter {
	return r.byName[name]
}

// AllFilters returns all loaded filters.
func (r *Registry) AllFilters() []*TomlFilter {
	return r.filters
}

// Count returns the number of loaded filters.
func (r *Registry) Count() int {
	return len(r.filters)
}

// ApplyToCommand finds a matching filter and applies it to the input.
// Returns the filtered output and true if a filter was applied,
// or the original input and false if no filter matched.
func (r *Registry) ApplyToCommand(command, input string) (string, bool) {
	filter := r.MatchFilter(command)
	if filter == nil {
		return input, false
	}
	return filter.Apply(input), true
}

// ValidateFilter validates a filter definition without adding it to the registry.
func ValidateFilter(name string, filter TomlFilter) error {
	filter.Name = name
	return filter.compileRegex()
}

// ReadFileLines reads a file and returns its lines.
func ReadFileLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
