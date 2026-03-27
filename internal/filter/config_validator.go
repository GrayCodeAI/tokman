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

// ValidatePipelineConfig validates a PipelineConfig struct and returns any
// errors or warnings. Checks for: unknown modes, out-of-range thresholds,
// conflicting settings, and unreachable budget values.
func ValidatePipelineConfig(cfg PipelineConfig) ValidationResult {
	v := &ConfigValidator{}

	// Validate mode
	switch cfg.Mode {
	case ModeNone, ModeMinimal, ModeAggressive:
	default:
		v.errors = append(v.errors,
			fmt.Sprintf("invalid mode %q — must be one of: none, minimal, aggressive", cfg.Mode))
	}

	// Budget range
	if cfg.Budget < 0 {
		v.errors = append(v.errors, "budget must be ≥ 0 (0 = unlimited)")
	}
	if cfg.Budget > 0 && cfg.Budget < 50 {
		v.warnings = append(v.warnings,
			fmt.Sprintf("budget %d is very small — content may be over-compressed", cfg.Budget))
	}

	// Threshold ranges: 0.0–1.0
	thresholds := map[string]float64{
		"AttributionThreshold":   cfg.AttributionThreshold,
		"SketchBudgetRatio":      cfg.SketchBudgetRatio,
		"LazyDecayRate":          cfg.LazyDecayRate,
		"SemanticAnchorRatio":    cfg.SemanticAnchorRatio,
		"AgentKnowledgeRetention": cfg.AgentKnowledgeRetention,
		"AgentHistoryPrune":      cfg.AgentHistoryPrune,
		"QuestionAwareThreshold": cfg.QuestionAwareThreshold,
		"DensityTargetRatio":     cfg.DensityTargetRatio,
		"DensityThreshold":       cfg.DensityThreshold,
		"TFIDFThreshold":         cfg.TFIDFThreshold,
		"DynamicRatioBase":       cfg.DynamicRatioBase,
		"SemanticChunkThreshold": cfg.SemanticChunkThreshold,
		"SketchHeavyHitter":      cfg.SketchHeavyHitter,
	}
	for name, val := range thresholds {
		if val != 0 && (val < 0.0 || val > 1.0) {
			v.errors = append(v.errors,
				fmt.Sprintf("%s = %.3f is out of range [0.0, 1.0]", name, val))
		}
	}

	// Positive-only sizes
	sizes := map[string]int{
		"H2OSinkSize":             cfg.H2OSinkSize,
		"H2ORecentSize":           cfg.H2ORecentSize,
		"H2OHeavyHitterSize":      cfg.H2OHeavyHitterSize,
		"AttentionSinkCount":      cfg.AttentionSinkCount,
		"AttentionRecentCount":    cfg.AttentionRecentCount,
		"MetaTokenWindow":         cfg.MetaTokenWindow,
		"MetaTokenMinSize":        cfg.MetaTokenMinSize,
		"SketchMaxSize":           cfg.SketchMaxSize,
		"LazyBaseBudget":          cfg.LazyBaseBudget,
		"LazyRevivalBudget":       cfg.LazyRevivalBudget,
		"SemanticAnchorSpacing":   cfg.SemanticAnchorSpacing,
		"AgentConsolidationMax":   cfg.AgentConsolidationMax,
		"MaxReflectionLoops":      cfg.MaxReflectionLoops,
		"CompactionThreshold":     cfg.CompactionThreshold,
		"CompactionMaxTokens":     cfg.CompactionMaxTokens,
	}
	for name, val := range sizes {
		if val < 0 {
			v.errors = append(v.errors, fmt.Sprintf("%s = %d must be ≥ 0", name, val))
		}
	}

	// SemanticChunkMethod: must be one of known values when set
	switch cfg.SemanticChunkMethod {
	case "", "auto", "code", "text", "mixed":
	default:
		v.errors = append(v.errors,
			fmt.Sprintf("SemanticChunkMethod %q is unknown — use: auto, code, text, mixed", cfg.SemanticChunkMethod))
	}

	// Conflicting settings
	if cfg.EnableAgentMemory && cfg.AgentKnowledgeRetention == 0 && cfg.AgentHistoryPrune == 0 {
		v.warnings = append(v.warnings,
			"EnableAgentMemory is true but AgentKnowledgeRetention and AgentHistoryPrune are both 0 — defaults will be used")
	}

	return v.result()
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
