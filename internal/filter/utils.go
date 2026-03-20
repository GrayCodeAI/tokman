package filter

import (
	"regexp"
	"strings"
)

var tokenizeRe = regexp.MustCompile(`[\s\p{P}\p{S}]+`)

// cleanWord normalizes a word for matching (lowercase, trim punctuation).
func cleanWord(word string) string {
	return strings.ToLower(strings.TrimSpace(word))
}

// tokenize splits text into words, handling code and natural language
func tokenize(text string) []string {
	// Split on whitespace and punctuation, keeping words together
	// This is a simple tokenizer suitable for compression algorithms

	// Replace common separators with spaces
	words := tokenizeRe.Split(text, -1)

	// Filter empty strings
	var result []string
	for _, word := range words {
		word = strings.TrimSpace(word)
		if word != "" {
			result = append(result, word)
		}
	}

	return result
}
