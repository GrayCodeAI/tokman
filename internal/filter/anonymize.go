package filter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// AnonymizeFilter replaces PII and sensitive identifiers with
// stable pseudonyms, preserving referential integrity within a document.
// For example, all occurrences of "john@example.com" become "[email-1]",
// and "jane@example.com" becomes "[email-2]", so relationships are preserved.
// Task #183: Content anonymization mode.
type AnonymizeFilter struct {
	// ReplaceCredentials controls whether credentials are replaced (default true).
	ReplaceCredentials bool
	// ReplacePII controls whether PII (emails, SSNs, phones) are replaced.
	ReplacePII bool
	// UseStablePseudonyms uses sequential IDs for referential integrity.
	UseStablePseudonyms bool
}

// NewAnonymizeFilter creates an anonymizer with all categories enabled.
func NewAnonymizeFilter() *AnonymizeFilter {
	return &AnonymizeFilter{
		ReplaceCredentials:  true,
		ReplacePII:          true,
		UseStablePseudonyms: true,
	}
}

// Name returns the filter name.
func (f *AnonymizeFilter) Name() string { return "anonymize" }

var (
	anonEmailRe   = regexp.MustCompile(`\b[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}\b`)
	anonPhoneRe   = regexp.MustCompile(`\b(?:\+1[-.\s]?)?\(?(\d{3})\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`)
	anonSSNRe     = regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	anonIPv4Re    = regexp.MustCompile(`\b(?:\d{1,3}\.){3}\d{1,3}\b`)
	anonJWTRe     = regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)
	anonAPIKeyRe  = regexp.MustCompile(`(?i)(?:api[_-]?key|apikey|auth[_-]?token|access[_-]?token)\s*[:=]\s*["']?([A-Za-z0-9_\-]{20,})["']?`)
	anonAWSKeyRe  = regexp.MustCompile(`(?:AKIA|ASIA|AROA)[0-9A-Z]{16}`)
)

// anonymizeState holds substitution maps for a single Apply call.
type anonymizeState struct {
	emails   map[string]string
	phones   map[string]string
	ssns     map[string]string
	ips      map[string]string
	counters map[string]int
}

func newAnonymizeState() *anonymizeState {
	return &anonymizeState{
		emails:   make(map[string]string),
		phones:   make(map[string]string),
		ssns:     make(map[string]string),
		ips:      make(map[string]string),
		counters: make(map[string]int),
	}
}

func (s *anonymizeState) pseudonym(category, original string, m map[string]string) string {
	if p, ok := m[original]; ok {
		return p
	}
	s.counters[category]++
	p := fmt.Sprintf("[%s-%d]", category, s.counters[category])
	m[original] = p
	return p
}

// Apply anonymizes PII and credentials in the input.
func (f *AnonymizeFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := input
	state := newAnonymizeState()

	if f.ReplacePII {
		// Emails
		output = anonEmailRe.ReplaceAllStringFunc(output, func(m string) string {
			return state.pseudonym("email", strings.ToLower(m), state.emails)
		})
		// Phone numbers
		output = anonPhoneRe.ReplaceAllStringFunc(output, func(m string) string {
			return state.pseudonym("phone", m, state.phones)
		})
		// SSNs
		output = anonSSNRe.ReplaceAllStringFunc(output, func(m string) string {
			return state.pseudonym("ssn", m, state.ssns)
		})
		// IP addresses (only in aggressive mode to avoid false positives in version strings)
		if mode == ModeAggressive {
			output = anonIPv4Re.ReplaceAllStringFunc(output, func(m string) string {
				return state.pseudonym("ip", m, state.ips)
			})
		}
	}

	if f.ReplaceCredentials {
		// JWT tokens
		output = anonJWTRe.ReplaceAllString(output, "[jwt-token]")
		// AWS keys
		output = anonAWSKeyRe.ReplaceAllString(output, "[aws-key]")
		// Generic API keys
		output = anonAPIKeyRe.ReplaceAllStringFunc(output, func(m string) string {
			// Preserve the key name, redact the value
			idx := strings.IndexAny(m, "=:")
			if idx < 0 {
				return "[api-key-value]"
			}
			return m[:idx+1] + " [api-key-value]"
		})
	}

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return output, 0 // Still return anonymized output even if no tokens saved
	}
	return output, saved
}
