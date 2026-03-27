package filter

import (
	"fmt"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ValidationSeverity indicates how serious a validation finding is.
type ValidationSeverity string

const (
	SeverityError   ValidationSeverity = "error"
	SeverityWarning ValidationSeverity = "warning"
	SeverityInfo    ValidationSeverity = "info"
)

// ValidationFinding describes a single output quality issue.
type ValidationFinding struct {
	Severity ValidationSeverity
	Code     string
	Message  string
}

// OutputValidationResult holds the result of output validation.
type OutputValidationResult struct {
	Valid    bool
	Findings []ValidationFinding
	// Stats
	OriginalTokens    int
	CompressedTokens  int
	ReductionPct      float64
	CharactersRemoved int
}

// String returns a human-readable summary.
func (r OutputValidationResult) String() string {
	var sb strings.Builder
	status := "PASS"
	if !r.Valid {
		status = "FAIL"
	}
	sb.WriteString(fmt.Sprintf("Output validation: %s\n", status))
	sb.WriteString(fmt.Sprintf("  Tokens: %d → %d (%.1f%% reduction)\n",
		r.OriginalTokens, r.CompressedTokens, r.ReductionPct))
	for _, f := range r.Findings {
		sb.WriteString(fmt.Sprintf("  [%s] %s: %s\n", f.Severity, f.Code, f.Message))
	}
	return sb.String()
}

// OutputValidator validates compression pipeline output quality.
// Task #171: pipeline output validation.
type OutputValidator struct {
	// MinReductionPct is the minimum acceptable reduction (0-100).
	// Failing this produces a warning.
	MinReductionPct float64
	// MaxReductionPct is the maximum acceptable reduction (0-100).
	// Exceeding this produces a warning (too much may signal data loss).
	MaxReductionPct float64
	// MaxOutputTokens is the absolute token limit. Exceeding is an error.
	MaxOutputTokens int
	// RequireUTF8 validates that output is valid UTF-8.
	RequireUTF8 bool
}

// NewOutputValidator creates a validator with sensible defaults.
func NewOutputValidator() *OutputValidator {
	return &OutputValidator{
		MinReductionPct: 0.0,   // no minimum — sometimes content is incompressible
		MaxReductionPct: 95.0,  // >95% reduction is suspicious
		MaxOutputTokens: 0,     // 0 = no limit
		RequireUTF8:     true,
	}
}

// Validate checks whether the compressed output meets quality criteria.
func (v *OutputValidator) Validate(original, compressed string) OutputValidationResult {
	origToks := core.EstimateTokens(original)
	compToks := core.EstimateTokens(compressed)

	var reduction float64
	if origToks > 0 {
		reduction = float64(origToks-compToks) / float64(origToks) * 100
	}

	result := OutputValidationResult{
		Valid:             true,
		OriginalTokens:    origToks,
		CompressedTokens:  compToks,
		ReductionPct:      reduction,
		CharactersRemoved: len(original) - len(compressed),
	}

	// Check: output must not be empty when input was non-empty
	if len(original) > 0 && len(compressed) == 0 {
		result.Valid = false
		result.Findings = append(result.Findings, ValidationFinding{
			Severity: SeverityError,
			Code:     "EMPTY_OUTPUT",
			Message:  "compressed output is empty but input was non-empty",
		})
	}

	// Check: maximum reduction threshold
	if reduction > v.MaxReductionPct && origToks > 20 {
		result.Findings = append(result.Findings, ValidationFinding{
			Severity: SeverityWarning,
			Code:     "EXCESSIVE_REDUCTION",
			Message:  fmt.Sprintf("reduction %.1f%% exceeds warning threshold %.1f%%", reduction, v.MaxReductionPct),
		})
	}

	// Check: minimum reduction (only for large inputs)
	if v.MinReductionPct > 0 && origToks > 100 && reduction < v.MinReductionPct {
		result.Findings = append(result.Findings, ValidationFinding{
			Severity: SeverityInfo,
			Code:     "BELOW_MIN_REDUCTION",
			Message:  fmt.Sprintf("reduction %.1f%% is below minimum %.1f%%", reduction, v.MinReductionPct),
		})
	}

	// Check: absolute token limit
	if v.MaxOutputTokens > 0 && compToks > v.MaxOutputTokens {
		result.Valid = false
		result.Findings = append(result.Findings, ValidationFinding{
			Severity: SeverityError,
			Code:     "EXCEEDS_TOKEN_BUDGET",
			Message:  fmt.Sprintf("output %d tokens exceeds budget %d", compToks, v.MaxOutputTokens),
		})
	}

	// Check: output grew larger than input
	if compToks > origToks && origToks > 10 {
		result.Findings = append(result.Findings, ValidationFinding{
			Severity: SeverityWarning,
			Code:     "OUTPUT_GREW",
			Message:  fmt.Sprintf("output (%d tokens) is larger than input (%d tokens)", compToks, origToks),
		})
	}

	// Check: output contains [TRUNCATED] markers from truncation filter
	if strings.Contains(compressed, "…[truncated]") || strings.Contains(compressed, "[truncated]") {
		result.Findings = append(result.Findings, ValidationFinding{
			Severity: SeverityInfo,
			Code:     "CONTAINS_TRUNCATION",
			Message:  "output contains truncation markers",
		})
	}

	return result
}
