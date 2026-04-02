package toml

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/BurntSushi/toml"
)

// SchemaVersion is the expected TOML filter schema version
const SchemaVersion = 1

// TOMLFilterRule represents a single TOML filter rule configuration
type TOMLFilterRule struct {
	MatchCommand       string            `toml:"match_command"`
	StripANSI          bool              `toml:"strip_ansi"`
	Replace            []ReplaceRule     `toml:"replace"`
	MatchOutput        []MatchOutputRule `toml:"match_output"`
	StripLinesMatching []string          `toml:"strip_lines_matching"`
	KeepLinesMatching  []string          `toml:"keep_lines_matching"`
	TruncateLinesAt    int               `toml:"truncate_lines_at"`
	Head               int               `toml:"head"`
	Tail               int               `toml:"tail"`
	MaxLines           int               `toml:"max_lines"`
	OnEmpty            string            `toml:"on_empty"`
}

// ReplaceRule defines a regex replacement rule
type ReplaceRule struct {
	Pattern     string `toml:"pattern"`
	Replacement string `toml:"replacement"`
}

// MatchOutputRule defines a short-circuit match rule
// If pattern matches, returns message immediately (short-circuit).
// If unless is set and also matches, the rule is skipped (prevents swallowing errors).
type MatchOutputRule struct {
	Pattern  string `toml:"pattern"`
	Message  string `toml:"message"`
	Priority int    `toml:"priority"`
	Unless   string `toml:"unless"` // Optional: if this pattern also matches, skip the rule
}

// TOMLFilter represents a parsed TOML filter file
type TOMLFilter struct {
	SchemaVersion int                       `toml:"schema_version"`
	Filters       map[string]TOMLFilterRule `toml:"-"`
	RawContent    map[string]any            `toml:"-"`
}

// Parser handles parsing TOML filter files
type Parser struct {
	filtersDir string
}

// NewParser creates a new TOML filter parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseFile parses a single TOML filter file
func (p *Parser) ParseFile(path string) (*TOMLFilter, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read filter file %s: %w", path, err)
	}

	return p.ParseContent(content, path)
}

// ParseContent parses TOML filter content
func (p *Parser) ParseContent(content []byte, source string) (*TOMLFilter, error) {
	var raw map[string]any
	if err := toml.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse TOML from %s: %w", source, err)
	}

	filter := &TOMLFilter{
		Filters:    make(map[string]TOMLFilterRule),
		RawContent: raw,
	}

	// Extract schema version
	if sv, ok := raw["schema_version"].(int64); ok {
		filter.SchemaVersion = int(sv)
	} else if sv, ok := raw["schema_version"].(int); ok {
		filter.SchemaVersion = sv
	} else {
		filter.SchemaVersion = SchemaVersion
	}

	// Parse individual filter configurations
	for name, val := range raw {
		if name == "schema_version" || name == "tests" {
			continue
		}

		if filterMap, ok := val.(map[string]any); ok {
			// Check if this is a nested "filters" section (e.g., [filters.apt])
			// In this case, the map contains filter names as keys
			if name == "filters" {
				for nestedName, nestedVal := range filterMap {
					if nestedFilterMap, ok := nestedVal.(map[string]any); ok {
						cfg, err := parseFilterRule(nestedFilterMap)
						if err != nil {
							return nil, fmt.Errorf("failed to parse filter %q: %w", nestedName, err)
						}
						filter.Filters[nestedName] = cfg
					}
				}
				continue
			}

			cfg, err := parseFilterRule(filterMap)
			if err != nil {
				return nil, fmt.Errorf("failed to parse filter %q: %w", name, err)
			}
			filter.Filters[name] = cfg
		}
	}

	return filter, nil
}

// parseFilterRule parses a single filter rule configuration from raw map
func parseFilterRule(m map[string]any) (TOMLFilterRule, error) {
	cfg := TOMLFilterRule{
		TruncateLinesAt: 0,
		Head:            0,
		Tail:            0,
		MaxLines:        0,
	}

	if v, ok := m["match_command"].(string); ok {
		cfg.MatchCommand = v
	}
	if v, ok := m["strip_ansi"].(bool); ok {
		cfg.StripANSI = v
	}
	if v, ok := m["truncate_lines_at"].(int64); ok {
		cfg.TruncateLinesAt = int(v)
	} else if v, ok := m["truncate_lines_at"].(int); ok {
		cfg.TruncateLinesAt = v
	}
	// Support both naming conventions: head/tail (TokMan) and head_lines/tail_lines
	if v, ok := m["head"].(int64); ok {
		cfg.Head = int(v)
	} else if v, ok := m["head"].(int); ok {
		cfg.Head = v
	} else if v, ok := m["head_lines"].(int64); ok {
		cfg.Head = int(v)
	} else if v, ok := m["head_lines"].(int); ok {
		cfg.Head = v
	}
	if v, ok := m["tail"].(int64); ok {
		cfg.Tail = int(v)
	} else if v, ok := m["tail"].(int); ok {
		cfg.Tail = v
	} else if v, ok := m["tail_lines"].(int64); ok {
		cfg.Tail = int(v)
	} else if v, ok := m["tail_lines"].(int); ok {
		cfg.Tail = v
	}
	if v, ok := m["max_lines"].(int64); ok {
		cfg.MaxLines = int(v)
	} else if v, ok := m["max_lines"].(int); ok {
		cfg.MaxLines = v
	} else if v, ok := m["max_lines"].(map[string]any); ok {
		// Handle nested format: [filters.x.max_lines] value = 30
		if val, ok := v["value"].(int64); ok {
			cfg.MaxLines = int(val)
		} else if val, ok := v["value"].(int); ok {
			cfg.MaxLines = val
		}
	}
	if v, ok := m["on_empty"].(string); ok {
		cfg.OnEmpty = v
	}

	// Parse string arrays
	if v, ok := m["strip_lines_matching"].([]any); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				cfg.StripLinesMatching = append(cfg.StripLinesMatching, s)
			}
		}
	} else if v, ok := m["strip_lines_matching"].(map[string]any); ok {
		// Handle nested format: [filters.x.strip_lines_matching] patterns = [...]
		if patterns, ok := v["patterns"].([]any); ok {
			for _, item := range patterns {
				if s, ok := item.(string); ok {
					cfg.StripLinesMatching = append(cfg.StripLinesMatching, s)
				}
			}
		}
	}
	if v, ok := m["keep_lines_matching"].([]any); ok {
		for _, item := range v {
			if s, ok := item.(string); ok {
				cfg.KeepLinesMatching = append(cfg.KeepLinesMatching, s)
			}
		}
	}

	// Parse replace rules
	if v, ok := m["replace"].([]any); ok {
		for _, item := range v {
			if ruleMap, ok := item.(map[string]any); ok {
				rule := ReplaceRule{}
				if p, ok := ruleMap["pattern"].(string); ok {
					rule.Pattern = p
				}
				if r, ok := ruleMap["replacement"].(string); ok {
					rule.Replacement = r
				}
				cfg.Replace = append(cfg.Replace, rule)
			}
		}
	}

	// Parse match_output rules
	if v, ok := m["match_output"].([]any); ok {
		for _, item := range v {
			if ruleMap, ok := item.(map[string]any); ok {
				rule := MatchOutputRule{}
				if p, ok := ruleMap["pattern"].(string); ok {
					rule.Pattern = p
				}
				if m, ok := ruleMap["message"].(string); ok {
					rule.Message = m
				}
				if pr, ok := ruleMap["priority"].(int64); ok {
					rule.Priority = int(pr)
				} else if pr, ok := ruleMap["priority"].(int); ok {
					rule.Priority = pr
				}
				// Parse optional unless clause (prevents short-circuit if pattern matches)
				if u, ok := ruleMap["unless"].(string); ok {
					rule.Unless = u
				}
				cfg.MatchOutput = append(cfg.MatchOutput, rule)
			}
		}
	}

	return cfg, nil
}

// Validate validates the filter configuration
func (f *TOMLFilter) Validate() error {
	if f.SchemaVersion != SchemaVersion {
		return fmt.Errorf("unsupported schema version %d, expected %d", f.SchemaVersion, SchemaVersion)
	}

	for name, cfg := range f.Filters {
		if cfg.MatchCommand == "" {
			return fmt.Errorf("filter %q: match_command is required", name)
		}

		// Validate regex patterns
		if _, err := regexp.Compile(cfg.MatchCommand); err != nil {
			return fmt.Errorf("filter %q: invalid match_command regex: %w", name, err)
		}

		for _, rule := range cfg.StripLinesMatching {
			if _, err := regexp.Compile(rule); err != nil {
				return fmt.Errorf("filter %q: invalid strip_lines_matching regex: %w", name, err)
			}
		}

		for _, rule := range cfg.KeepLinesMatching {
			if _, err := regexp.Compile(rule); err != nil {
				return fmt.Errorf("filter %q: invalid keep_lines_matching regex: %w", name, err)
			}
		}

		for _, rule := range cfg.Replace {
			if _, err := regexp.Compile(rule.Pattern); err != nil {
				return fmt.Errorf("filter %q: invalid replace pattern: %w", name, err)
			}
		}

		for _, rule := range cfg.MatchOutput {
			if _, err := regexp.Compile(rule.Pattern); err != nil {
				return fmt.Errorf("filter %q: invalid match_output pattern: %w", name, err)
			}
			// Validate optional unless pattern
			if rule.Unless != "" {
				if _, err := regexp.Compile(rule.Unless); err != nil {
					return fmt.Errorf("filter %q: invalid match_output unless pattern: %w", name, err)
				}
			}
		}
	}

	return nil
}

// MatchesCommand checks if any filter matches the given command
func (f *TOMLFilter) MatchesCommand(command string) (string, *TOMLFilterRule, error) {
	for name, cfg := range f.Filters {
		matched, err := regexp.MatchString(cfg.MatchCommand, command)
		if err != nil {
			return "", nil, fmt.Errorf("filter %q: regex error: %w", name, err)
		}
		if matched {
			return name, &cfg, nil
		}
	}
	return "", nil, nil
}

// FilterRegistry holds all loaded TOML filters
type FilterRegistry struct {
	mu      sync.RWMutex
	filters map[string]*TOMLFilter
	parser  *Parser
}

// NewFilterRegistry creates a new filter registry
func NewFilterRegistry() *FilterRegistry {
	return &FilterRegistry{
		filters: make(map[string]*TOMLFilter),
		parser:  NewParser(),
	}
}

// LoadFile loads a single TOML filter file
func (r *FilterRegistry) LoadFile(path string) error {
	filter, err := r.parser.ParseFile(path)
	if err != nil {
		return err
	}

	if err := filter.Validate(); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.filters[filepath.Base(path)] = filter
	return nil
}

// FindMatchingFilter finds a filter that matches the given command
func (r *FilterRegistry) FindMatchingFilter(command string) (string, string, *TOMLFilterRule) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for filename, filter := range r.filters {
		if name, cfg, err := filter.MatchesCommand(command); err == nil && cfg != nil {
			return filename, name, cfg
		}
	}
	return "", "", nil
}

// Count returns the number of loaded filters
func (r *FilterRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.filters)
}
