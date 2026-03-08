package filter

import (
	"regexp"
	"strings"
)

// CommentFilter strips comments from source code.
type CommentFilter struct {
	patterns map[string]*regexp.Regexp
}

// NewCommentFilter creates a new comment filter.
func NewCommentFilter() *CommentFilter {
	return &CommentFilter{
		patterns: CommentPatterns,
	}
}

// Name returns the filter name.
func (f *CommentFilter) Name() string {
	return "comment"
}

// Apply strips comments and returns token savings.
func (f *CommentFilter) Apply(input string, mode Mode) (string, int) {
	// Detect language
	lang := DetectLanguage(input)
	
	// Get pattern for language
	pattern, ok := f.patterns[lang]
	if !ok {
		// Use a generic multi-line comment pattern
		pattern = regexp.MustCompile(`(?m)//.*$|/\*[\s\S]*?\*/|#.*$`)
	}
	
	original := len(input)
	
	// Remove comments but preserve structure
	output := f.removeComments(input, pattern, lang)
	
	bytesSaved := original - len(output)
	tokensSaved := bytesSaved / 4
	
	return output, tokensSaved
}

// removeComments removes comments while preserving line structure.
func (f *CommentFilter) removeComments(input string, pattern *regexp.Regexp, lang string) string {
	lines := strings.Split(input, "\n")
	var result []string
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		// Skip pure comment lines
		if f.isPureComment(trimmed, lang) {
			continue
		}
		
		// Remove inline comments
		cleaned := f.removeInlineComment(line, lang)
		result = append(result, cleaned)
	}
	
	return strings.Join(result, "\n")
}

// isPureComment checks if a line is only a comment.
func (f *CommentFilter) isPureComment(line string, lang string) bool {
	if line == "" {
		return false
	}
	
	switch lang {
	case "python", "sh", "bash", "yaml", "toml", "dockerfile", "makefile", "ruby":
		return strings.HasPrefix(line, "#")
	case "go", "rust", "javascript", "typescript", "java", "c", "cpp":
		return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "/*")
	default:
		return strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#")
	}
}

// removeInlineComment removes inline comments from a line.
func (f *CommentFilter) removeInlineComment(line string, lang string) string {
	switch lang {
	case "python", "sh", "bash", "yaml", "toml":
		// Don't remove # inside strings
		if idx := strings.Index(line, "#"); idx > 0 {
			// Simple heuristic: check if we're inside a string
			before := line[:idx]
			singleQuotes := strings.Count(before, "'")
			doubleQuotes := strings.Count(before, "\"")
			if singleQuotes%2 == 0 && doubleQuotes%2 == 0 {
				return strings.TrimSpace(line[:idx])
			}
		}
		return line
	
	case "go", "rust", "javascript", "typescript", "java", "c", "cpp":
		// Remove // comments not in strings
		if idx := strings.Index(line, "//"); idx > 0 {
			before := line[:idx]
			doubleQuotes := strings.Count(before, "\"")
			if doubleQuotes%2 == 0 {
				return strings.TrimSpace(line[:idx])
			}
		}
		return line
	
	default:
		return line
	}
}

// StripComments is a utility function to strip comments from code.
func StripComments(code string, lang string) string {
	filter := NewCommentFilter()
	output, _ := filter.Apply(code, ModeMinimal)
	return output
}
