package filter

import (
	"regexp"
	"strings"
)

// SecretDetector detects and redacts secrets, tokens, and sensitive data.
// This is a security-critical filter that runs FIRST in the pipeline to ensure
// no secrets leak through compression into LLM context windows.
//
// Detects: API keys, tokens, passwords, private keys, connection strings,
// JWT tokens, AWS credentials, GitHub tokens, and common secret patterns.
type SecretDetector struct {
	config   SecretConfig
	patterns []secretPattern
}

// SecretConfig holds configuration for secret detection
type SecretConfig struct {
	// Enabled controls whether detection is active
	Enabled bool

	// RedactionMode: "full" replaces with [REDACTED], "partial" shows first/last 4 chars
	RedactionMode string

	// CustomPatterns additional regex patterns to detect
	CustomPatterns []string
}

// secretPattern defines a detection pattern
type secretPattern struct {
	name    string
	pattern *regexp.Regexp
	replace string
}

// defaultSecretConfig returns default configuration
func defaultSecretConfig() SecretConfig {
	return SecretConfig{
		Enabled:       true,
		RedactionMode: "partial",
	}
}

// newSecretDetector creates a new secret detector
func newSecretDetector() *SecretDetector {
	return &SecretDetector{
		config:   defaultSecretConfig(),
		patterns: initSecretPatterns(),
	}
}

// Name returns the filter name
func (s *SecretDetector) Name() string {
	return "secret_detector"
}

// Apply detects and redacts secrets in the input
func (s *SecretDetector) Apply(input string, mode Mode) (string, int) {
	if !s.config.Enabled {
		return input, 0
	}

	output := input
	redacted := 0

	for _, p := range s.patterns {
		matches := p.pattern.FindAllString(output, -1)
		if len(matches) > 0 {
			for _, match := range matches {
				replacement := s.redact(match, p.name)
				output = strings.Replace(output, match, replacement, 1)
				redacted++
			}
		}
	}

	return output, redacted
}

// IsSecret checks if input contains secrets (without redacting)
func (s *SecretDetector) IsSecret(input string) bool {
	for _, p := range s.patterns {
		if p.pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// redact creates a redacted version of a secret
func (s *SecretDetector) redact(secret, patternName string) string {
	if s.config.RedactionMode == "full" {
		return "[" + patternName + ":REDACTED]"
	}

	// Partial: show first 4 and last 4 chars
	if len(secret) <= 8 {
		return "[" + patternName + ":REDACTED]"
	}
	return secret[:4] + "..." + secret[len(secret)-4:] + " [" + patternName + "]"
}

// initSecretPatterns initializes detection patterns
func initSecretPatterns() []secretPattern {
	return []secretPattern{
		// AWS Access Key
		{"AWS_KEY", regexp.MustCompile(`AKIA[0-9A-Z]{16}`), "[AWS_KEY:REDACTED]"},

		// AWS Secret Key
		{"AWS_SECRET", regexp.MustCompile(`(?i)aws.{0,20}secret.{0,20}['"]([A-Za-z0-9/+=]{40})['"]`), "[AWS_SECRET:REDACTED]"},

		// GitHub Token
		{"GITHUB_TOKEN", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{36,255}`), "[GITHUB_TOKEN:REDACTED]"},

		// GitHub Fine-grained Token
		{"GITHUB_FINEGRAINED", regexp.MustCompile(`github_pat_[A-Za-z0-9_]{22,255}`), "[GITHUB_FINEGRAINED:REDACTED]"},

		// Generic API Key patterns
		{"API_KEY", regexp.MustCompile(`(?i)(api[_-]?key|apikey|api[_-]?secret)['"=: ]+['"]?([A-Za-z0-9_\-]{20,})['"]?`), "[API_KEY:REDACTED]"},

		// Generic Token patterns
		{"TOKEN", regexp.MustCompile(`(?i)(token|bearer|auth)['"=: ]+['"]?([A-Za-z0-9_\-\.]{20,})['"]?`), "[TOKEN:REDACTED]"},

		// Private Key
		{"PRIVATE_KEY", regexp.MustCompile(`-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`), "[PRIVATE_KEY:REDACTED]"},

		// JWT Token
		{"JWT", regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`), "[JWT:REDACTED]"},

		// Generic Secret
		{"SECRET", regexp.MustCompile(`(?i)(secret|password|passwd|pwd)['"=: ]+['"]?([^\s'"]{8,})['"]?`), "[SECRET:REDACTED]"},

		// Connection String
		{"CONN_STRING", regexp.MustCompile(`(?i)(mongodb|postgres|mysql|redis|amqp|mssql)://[^\s]+:[^\s]+@[^\s]+`), "[CONN_STRING:REDACTED]"},

		// Slack Token
		{"SLACK_TOKEN", regexp.MustCompile(`xox[bporas]-[0-9]{10,}-[0-9]{10,}-[a-zA-Z0-9]{24,}`), "[SLACK_TOKEN:REDACTED]"},

		// Stripe Key
		{"STRIPE_KEY", regexp.MustCompile(`(?:sk|pk)_(?:live|test)_[A-Za-z0-9]{20,}`), "[STRIPE_KEY:REDACTED]"},

		// Google API Key
		{"GOOGLE_KEY", regexp.MustCompile(`AIza[0-9A-Za-z_-]{35}`), "[GOOGLE_KEY:REDACTED]"},

		// OpenAI Key
		{"OPENAI_KEY", regexp.MustCompile(`sk-[A-Za-z0-9]{20,}`), "[OPENAI_KEY:REDACTED]"},

		// Anthropic Key
		{"ANTHROPIC_KEY", regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{20,}`), "[ANTHROPIC_KEY:REDACTED]"},

		// .env style secrets
		{"ENV_SECRET", regexp.MustCompile(`(?m)^[A-Z_]+(_KEY|_SECRET|_TOKEN|_PASSWORD)=.{8,}`), "[ENV_SECRET:REDACTED]"},

		// Base64 encoded secrets (long strings)
		{"BASE64_SECRET", regexp.MustCompile(`[A-Za-z0-9+/]{40,}={0,2}`), "[BASE64_SECRET:REDACTED]"},
	}
}
