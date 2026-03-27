package filter

import (
	"regexp"
	"strings"
)

// SensitivityCategory identifies a class of sensitive content.
type SensitivityCategory string

const (
	SensitivityCredentials SensitivityCategory = "credentials"
	SensitivityPII         SensitivityCategory = "pii"
	SensitivityFinancial   SensitivityCategory = "financial"
	SensitivityMedical     SensitivityCategory = "medical"
	SensitivityNone        SensitivityCategory = ""
)

// SensitivityMatch records a detected sensitive pattern.
type SensitivityMatch struct {
	Category SensitivityCategory
	Pattern  string // the regex pattern name that fired
	LineNum  int    // 1-indexed
	Snippet  string // up to 40 chars around the match (redacted)
}

// SensitivityClassifier detects sensitive content using pattern matching and
// entropy analysis. No LLM required. Runs at pre-compression to warn and
// optionally redact sensitive data.
type SensitivityClassifier struct {
	// MaxSnippetLen controls how many chars of context are shown in snippets.
	MaxSnippetLen int
}

type sensitivityRule struct {
	name     string
	category SensitivityCategory
	re       *regexp.Regexp
}

var sensitivityRules = []sensitivityRule{
	// Credentials
	{"aws-access-key", SensitivityCredentials,
		regexp.MustCompile(`(?i)(?:AKIA|ASIA|AROA)[0-9A-Z]{16}`)},
	{"aws-secret-key", SensitivityCredentials,
		regexp.MustCompile(`(?i)aws[_-]?(?:secret|access)[_-]?key\s*[:=]\s*\S{20,}`)},
	{"private-key-header", SensitivityCredentials,
		regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA )?PRIVATE KEY-----`)},
	{"jwt-token", SensitivityCredentials,
		regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)},
	{"generic-api-key", SensitivityCredentials,
		regexp.MustCompile(`(?i)(?:api[_-]?key|apikey|auth[_-]?token|access[_-]?token)\s*[:=]\s*["']?[A-Za-z0-9_\-]{20,}["']?`)},
	{"password-in-config", SensitivityCredentials,
		regexp.MustCompile(`(?i)(?:password|passwd|secret)\s*[:=]\s*["']?[^\s"']{8,}["']?`)},
	{"github-token", SensitivityCredentials,
		regexp.MustCompile(`(?:ghp|gho|ghu|ghs|ghr)_[A-Za-z0-9]{36}`)},

	// PII
	{"email-address", SensitivityPII,
		regexp.MustCompile(`\b[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}\b`)},
	{"us-ssn", SensitivityPII,
		regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)},
	{"phone-number", SensitivityPII,
		regexp.MustCompile(`\b(?:\+1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`)},
	{"credit-card", SensitivityFinancial,
		regexp.MustCompile(`\b(?:4\d{3}|5[1-5]\d{2}|6011|3[47]\d{2})[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}\b`)},

	// Financial
	{"bank-account-iban", SensitivityFinancial,
		regexp.MustCompile(`\b[A-Z]{2}\d{2}[A-Z0-9]{11,30}\b`)},

	// Medical
	{"medical-record", SensitivityMedical,
		regexp.MustCompile(`(?i)(?:patient[_\s]?id|mrn|medical[_\s]?record[_\s]?number)\s*[:=]\s*\w+`)},
	{"hipaa-diagnosis", SensitivityMedical,
		regexp.MustCompile(`(?i)(?:diagnosis|icd-10|icd10|patient[_\s]?name)\s*[:=]\s*\S+`)},
}

// NewSensitivityClassifier creates a new sensitivity classifier.
func NewSensitivityClassifier() *SensitivityClassifier {
	return &SensitivityClassifier{MaxSnippetLen: 40}
}

// Classify scans input and returns all detected sensitive matches.
func (c *SensitivityClassifier) Classify(input string) []SensitivityMatch {
	lines := strings.Split(input, "\n")
	var matches []SensitivityMatch

	for lineNum, line := range lines {
		for _, rule := range sensitivityRules {
			if rule.re.MatchString(line) {
				snippet := redactSnippet(line, rule.re, c.MaxSnippetLen)
				matches = append(matches, SensitivityMatch{
					Category: rule.category,
					Pattern:  rule.name,
					LineNum:  lineNum + 1,
					Snippet:  snippet,
				})
				break // one match per line per category
			}
		}
	}
	return matches
}

// HasSensitiveContent returns true if any sensitive patterns are found.
func (c *SensitivityClassifier) HasSensitiveContent(input string) bool {
	for _, rule := range sensitivityRules {
		if rule.re.MatchString(input) {
			return true
		}
	}
	return false
}

// Categories returns the unique categories of sensitive content found.
func (c *SensitivityClassifier) Categories(input string) []SensitivityCategory {
	found := make(map[SensitivityCategory]bool)
	for _, rule := range sensitivityRules {
		if rule.re.MatchString(input) {
			found[rule.category] = true
		}
	}
	result := make([]SensitivityCategory, 0, len(found))
	for cat := range found {
		result = append(result, cat)
	}
	return result
}

// redactSnippet returns a brief redacted snippet showing where the match occurred.
func redactSnippet(line string, re *regexp.Regexp, maxLen int) string {
	loc := re.FindStringIndex(line)
	if loc == nil {
		return ""
	}
	start := loc[0] - 10
	if start < 0 {
		start = 0
	}
	end := loc[1] + 10
	if end > len(line) {
		end = len(line)
	}
	snippet := line[start:end]
	// Replace the match with [REDACTED]
	snippet = re.ReplaceAllString(snippet, "[REDACTED]")
	if len(snippet) > maxLen {
		snippet = snippet[:maxLen] + "..."
	}
	return snippet
}
