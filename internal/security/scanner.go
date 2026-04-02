package security

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/utils"
)

// Scanner performs security scanning on content.
// Inspired by clawshield and token-lens.
type Scanner struct {
	rules []Rule
}

// Rule represents a security scanning rule.
type Rule struct {
	Name        string
	Pattern     *regexp.Regexp
	Severity    string
	Description string
}

// Finding represents a security finding.
type Finding struct {
	Rule     string
	Severity string
	Message  string
	Line     int
	Match    string
}

// NewScanner creates a new security scanner.
func NewScanner() *Scanner {
	return &Scanner{
		rules: buildDefaultRules(),
	}
}

// Scan scans content for security issues.
func (s *Scanner) Scan(content string) []Finding {
	var findings []Finding
	lines := strings.Split(content, "\n")

	for lineNum, line := range lines {
		for _, rule := range s.rules {
			if match := rule.Pattern.FindString(line); match != "" {
				findings = append(findings, Finding{
					Rule:     rule.Name,
					Severity: rule.Severity,
					Message:  rule.Description,
					Line:     lineNum + 1,
					Match:    match,
				})
			}
		}
	}
	return findings
}

// RedactPII redacts personally identifiable information from content.
func RedactPII(content string) string {
	result := content

	// Email
	emailRe := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	result = emailRe.ReplaceAllString(result, "[EMAIL_REDACTED]")

	// Phone numbers
	phoneRe := regexp.MustCompile(`\b\d{3}[-.\s]?\d{3}[-.\s]?\d{4}\b`)
	result = phoneRe.ReplaceAllString(result, "[PHONE_REDACTED]")

	// SSN
	ssnRe := regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`)
	result = ssnRe.ReplaceAllString(result, "[SSN_REDACTED]")

	// Credit cards
	ccRe := regexp.MustCompile(`\b(?:\d{4}[-\s]?){3}\d{4}\b`)
	result = ccRe.ReplaceAllString(result, "[CC_REDACTED]")

	return result
}

// DetectSecrets checks for potential secrets in content.
func DetectSecrets(content string) []Finding {
	var findings []Finding
	lines := strings.Split(content, "\n")

	secretPatterns := map[string]*regexp.Regexp{
		"api_key":      regexp.MustCompile(`(?i)(?:api[_-]?key|apikey)\s*[:=]\s*['"]?[a-zA-Z0-9]{20,}`),
		"password":     regexp.MustCompile(`(?i)(?:password|passwd|pwd)\s*[:=]\s*['"]?[^\s'"]{8,}`),
		"token":        regexp.MustCompile(`(?i)(?:token|secret|auth)\s*[:=]\s*['"]?[a-zA-Z0-9]{20,}`),
		"private_key":  regexp.MustCompile(`-----BEGIN (?:RSA |EC |DSA )?PRIVATE KEY-----`),
		"aws_key":      regexp.MustCompile(`(?:AKIA|ABIA|ACCA|ASIA)[A-Z0-9]{16}`),
		"github_token": regexp.MustCompile(`ghp_[a-zA-Z0-9]{36}`),
	}

	for lineNum, line := range lines {
		for name, pattern := range secretPatterns {
			if match := pattern.FindString(line); match != "" {
				findings = append(findings, Finding{
					Rule:     name,
					Severity: "critical",
					Message:  "Potential secret detected: " + name,
					Line:     lineNum + 1,
					Match:    match[:utils.Min(len(match), 50)],
				})
			}
		}
	}
	return findings
}

// DetectPromptInjection checks for prompt injection attempts.
func DetectPromptInjection(content string) []Finding {
	var findings []Finding
	injectionPatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)ignore\s+(?:all\s+)?(?:previous|prior|above)\s+(?:instructions|rules|prompts)`),
		regexp.MustCompile(`(?i)(?:you\s+are\s+now|act\s+as|pretend\s+to\s+be)\s+(?:a|an|the)`),
		regexp.MustCompile(`(?i)(?:system|developer|admin)\s*(?:prompt|instruction|mode)`),
		regexp.MustCompile(`(?i)DAN\s+mode`),
		regexp.MustCompile(`(?i)jailbreak`),
		regexp.MustCompile(`(?i)disregard\s+(?:all|any|the)\s+(?:previous|prior)`),
	}

	for i, pattern := range injectionPatterns {
		if match := pattern.FindString(content); match != "" {
			findings = append(findings, Finding{
				Rule:     "prompt_injection_" + itoa(i),
				Severity: "high",
				Message:  "Potential prompt injection detected",
				Line:     0,
				Match:    match[:utils.Min(len(match), 80)],
			})
		}
	}
	return findings
}

func buildDefaultRules() []Rule {
	return []Rule{
		{
			Name:        "sql_injection",
			Pattern:     regexp.MustCompile(`(?i)(?:SELECT|INSERT|UPDATE|DELETE|DROP|UNION|ALTER)\s+.*(?:FROM|INTO|TABLE|WHERE)`),
			Severity:    "high",
			Description: "Potential SQL injection pattern",
		},
		{
			Name:        "xss",
			Pattern:     regexp.MustCompile(`(?i)<script[^>]*>|javascript:|on(?:error|load|click)\s*=`),
			Severity:    "high",
			Description: "Potential XSS pattern",
		},
		{
			Name:        "path_traversal",
			Pattern:     regexp.MustCompile(`\.\./\.\./|\.\.\\..\\`),
			Severity:    "medium",
			Description: "Potential path traversal",
		},
		{
			Name:        "ssrf",
			Pattern:     regexp.MustCompile(`(?i)(?:https?://)(?:localhost|127\.0\.0\.1|0\.0\.0\.0|169\.254\.169\.254|metadata\.google)`),
			Severity:    "high",
			Description: "Potential SSRF target",
		},
	}
}

func itoa(n int) string {
	return string(rune('0' + n))
}
