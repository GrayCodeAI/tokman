package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Min returns the smaller of a and b.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger of a and b.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Abs returns the absolute value of x.
func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// Clamp returns x constrained to the range [min, max].
func Clamp(x, min, max int) int {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

// ShortenPath truncates a path to fit within maxLen characters.
// It preserves the end of the path and adds "..." prefix if truncated.
func ShortenPath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}

	// If maxLen is too small, just return truncated with ...
	if maxLen <= 3 {
		return "..." + path[len(path)-maxLen+3:]
	}

	// Try to preserve the filename and as much of the path as possible
	// Start from the end and work backwards
	truncated := "..." + path[len(path)-maxLen+3:]

	// Clean the path to use consistent separators
	truncated = filepath.Clean(truncated)

	return truncated
}

// FormatBytes converts a byte count to a human-readable string.
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
		TB = GB * 1024
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1fT", float64(bytes)/TB)
	case bytes >= GB:
		return fmt.Sprintf("%.1fG", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.1fM", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.1fK", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}

// FormatDuration converts milliseconds to a human-readable duration string.
func FormatDuration(ms int64) string {
	const (
		Second = 1000
		Minute = Second * 60
		Hour   = Minute * 60
	)

	switch {
	case ms >= Hour:
		hours := ms / Hour
		minutes := (ms % Hour) / Minute
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm", hours, minutes)
		}
		return fmt.Sprintf("%dh", hours)
	case ms >= Minute:
		minutes := ms / Minute
		seconds := (ms % Minute) / Second
		if seconds > 0 {
			return fmt.Sprintf("%dm %ds", minutes, seconds)
		}
		return fmt.Sprintf("%dm", minutes)
	case ms >= Second:
		seconds := float64(ms) / Second
		return fmt.Sprintf("%.1fs", seconds)
	default:
		return fmt.Sprintf("%dms", ms)
	}
}

// FormatTokens formats a token count with K/M/B suffixes.
func FormatTokens(n int) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// FormatTokens64 formats a uint64 token count with K/M/B suffixes.
func FormatTokens64(n uint64) string {
	if n >= 1_000_000_000 {
		return fmt.Sprintf("%.1fB", float64(n)/1_000_000_000)
	}
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}

// GetModelFamily extracts the model family from a model name.
// Returns the family identifier (e.g., "claude", "gpt", "gemini") or "other".
// Used by tracking and command execution for AI agent attribution.
func GetModelFamily(modelName string) string {
	if modelName == "" {
		return ""
	}
	modelLower := strings.ToLower(modelName)
	switch {
	case strings.Contains(modelLower, "claude"):
		return "claude"
	case strings.Contains(modelLower, "gpt") || strings.Contains(modelLower, "o1") || strings.Contains(modelLower, "o3"):
		return "gpt"
	case strings.Contains(modelLower, "gemini"):
		return "gemini"
	case strings.Contains(modelLower, "llama") || strings.Contains(modelLower, "meta"):
		return "llama"
	case strings.Contains(modelLower, "qwen"):
		return "qwen"
	case strings.Contains(modelLower, "deepseek"):
		return "deepseek"
	case strings.Contains(modelLower, "mistral") || strings.Contains(modelLower, "mixtral"):
		return "mistral"
	default:
		return "other"
	}
}
