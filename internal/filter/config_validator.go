package filter

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

// ConfigValidator validates TokMan configuration files before runtime.
// Catches invalid TOML, missing required fields, and conflicting settings.
type ConfigValidator struct {
	errors   []string
	warnings []string
}

// ValidationResult holds validation output
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// newConfigValidator creates a new config validator
func newConfigValidator() *ConfigValidator {
	return &ConfigValidator{}
}

// ValidateFile validates a TOML config file
func (v *ConfigValidator) ValidateFile(path string) ValidationResult {
	v.errors = nil
	v.warnings = nil

	// Check file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		v.errors = append(v.errors, fmt.Sprintf("config file not found: %s", path))
		return v.result()
	}

	// Parse TOML
	var config map[string]interface{}
	if _, err := toml.DecodeFile(path, &config); err != nil {
		v.errors = append(v.errors, fmt.Sprintf("invalid TOML: %s", err.Error()))
		return v.result()
	}

	// Validate sections
	v.validateFilter(config)
	v.validatePipeline(config)
	v.validateHooks(config)
	v.validateTracking(config)
	v.validateDashboard(config)

	return v.result()
}

// ValidateBytes validates raw TOML bytes
func (v *ConfigValidator) ValidateBytes(data []byte) ValidationResult {
	v.errors = nil
	v.warnings = nil

	var config map[string]interface{}
	if _, err := toml.Decode(string(data), &config); err != nil {
		v.errors = append(v.errors, fmt.Sprintf("invalid TOML: %s", err.Error()))
		return v.result()
	}

	v.validateFilter(config)
	v.validatePipeline(config)

	return v.result()
}

func (v *ConfigValidator) validateFilter(config map[string]interface{}) {
	filter, ok := config["filter"].(map[string]interface{})
	if !ok {
		return
	}

	// Check mode
	if mode, ok := filter["mode"].(string); ok {
		validModes := []string{"none", "minimal", "aggressive"}
		found := false
		for _, m := range validModes {
			if mode == m {
				found = true
				break
			}
		}
		if !found {
			v.errors = append(v.errors, fmt.Sprintf("invalid filter.mode: %s (valid: %v)", mode, validModes))
		}
	}
}

func (v *ConfigValidator) validatePipeline(config map[string]interface{}) {
	pipeline, ok := config["pipeline"].(map[string]interface{})
	if !ok {
		return
	}

	// Check max_context_tokens
	if maxTokens, ok := pipeline["max_context_tokens"].(int64); ok {
		if maxTokens < 1000 {
			v.warnings = append(v.warnings, "pipeline.max_context_tokens < 1000 may cause issues")
		}
		if maxTokens > 10000000 {
			v.warnings = append(v.warnings, "pipeline.max_context_tokens > 10M may use excessive memory")
		}
	}

	// Check budget
	if budget, ok := pipeline["budget"].(int64); ok {
		if budget < 100 {
			v.warnings = append(v.warnings, "pipeline.budget < 100 may over-compress content")
		}
	}
}

func (v *ConfigValidator) validateHooks(config map[string]interface{}) {
	hooks, ok := config["hooks"].(map[string]interface{})
	if !ok {
		return
	}

	if excluded, ok := hooks["excluded_commands"].([]interface{}); ok {
		for _, cmd := range excluded {
			if _, ok := cmd.(string); !ok {
				v.errors = append(v.errors, "hooks.excluded_commands must be an array of strings")
			}
		}
	}
}

func (v *ConfigValidator) validateTracking(config map[string]interface{}) {
	tracking, ok := config["tracking"].(map[string]interface{})
	if !ok {
		return
	}

	if enabled, ok := tracking["enabled"].(bool); ok && enabled {
		if dbPath, ok := tracking["database_path"].(string); ok {
			if dbPath == "" {
				v.errors = append(v.errors, "tracking.database_path is empty but tracking is enabled")
			}
		}
	}
}

func (v *ConfigValidator) validateDashboard(config map[string]interface{}) {
	dashboard, ok := config["dashboard"].(map[string]interface{})
	if !ok {
		return
	}

	if port, ok := dashboard["port"].(int64); ok {
		if port < 1 || port > 65535 {
			v.errors = append(v.errors, fmt.Sprintf("dashboard.port %d is invalid (1-65535)", port))
		}
	}
}

func (v *ConfigValidator) result() ValidationResult {
	return ValidationResult{
		Valid:    len(v.errors) == 0,
		Errors:   v.errors,
		Warnings: v.warnings,
	}
}

// String returns a formatted validation report
func (r ValidationResult) String() string {
	var sb strings.Builder

	if r.Valid {
		sb.WriteString("Config: VALID\n")
	} else {
		sb.WriteString("Config: INVALID\n")
	}

	if len(r.Errors) > 0 {
		sb.WriteString("\nErrors:\n")
		for _, e := range r.Errors {
			sb.WriteString("  - " + e + "\n")
		}
	}

	if len(r.Warnings) > 0 {
		sb.WriteString("\nWarnings:\n")
		for _, w := range r.Warnings {
			sb.WriteString("  - " + w + "\n")
		}
	}

	return sb.String()
}
