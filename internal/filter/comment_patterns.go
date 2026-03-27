package filter

import (
	"regexp"
)

// CommentPatterns represents comment structure for a language
type CommentPatterns struct {
	Line       string
	BlockStart string
	BlockEnd   string
	DocLine    string
	DocBlock   string
}

// commentPatternsForLang returns comment patterns for a language
func commentPatternsForLang(lang Language) CommentPatterns {
	switch lang {
	case LangRust:
		return CommentPatterns{
			Line:       "//",
			BlockStart: "/*",
			BlockEnd:   "*/",
			DocLine:    "///",
			DocBlock:   "/**",
		}
	case LangPython:
		return CommentPatterns{
			Line:       "#",
			BlockStart: `"""`,
			BlockEnd:   `"""`,
			DocBlock:   `"""`,
		}
	case LangJavaScript, LangTypeScript, LangGo, LangC, LangCpp, LangJava:
		return CommentPatterns{
			Line:       "//",
			BlockStart: "/*",
			BlockEnd:   "*/",
			DocBlock:   "/**",
		}
	case LangRuby:
		return CommentPatterns{
			Line:       "#",
			BlockStart: "=begin",
			BlockEnd:   "=end",
		}
	case LangShell:
		return CommentPatterns{
			Line: "#",
		}
	case LangSQL:
		return CommentPatterns{
			Line: "--",
		}
	default:
		return CommentPatterns{
			Line:       "//",
			BlockStart: "/*",
			BlockEnd:   "*/",
		}
	}
}

// cStyleCommentRe matches C/C++/Go/Java/Rust/JS/TS style comments.
var cStyleCommentRe = regexp.MustCompile(`(?m)^//.*$|/\*[\s\S]*?\*/`)

// commentFallbackRe is used when no language-specific pattern is found.
var commentFallbackRe = regexp.MustCompile(`(?m)^//.*$|/\*[\s\S]*?\*/|^#.*$`)

// CommentPatternsMap maps languages to their comment regex patterns
var CommentPatternsMap = map[Language]*regexp.Regexp{
	LangGo:         cStyleCommentRe,
	LangRust:       cStyleCommentRe,
	LangPython:     regexp.MustCompile(`(?m)^#.*$|"""[\s\S]*?"""|'''[\s\S]*?'''`),
	LangJavaScript: cStyleCommentRe,
	LangTypeScript: cStyleCommentRe,
	LangJava:       cStyleCommentRe,
	LangC:          cStyleCommentRe,
	LangCpp:        cStyleCommentRe,
	LangShell:      regexp.MustCompile(`(?m)^#.*$`),
	LangRuby:       regexp.MustCompile(`(?m)^#.*$|=begin[\s\S]*?=end`),
	LangSQL:        regexp.MustCompile(`(?m)^--.*$`),
}

// ImportPatterns for various languages
var ImportPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^use\s+`),
	regexp.MustCompile(`^import\s+`),
	regexp.MustCompile(`^from\s+\S+\s+import`),
	regexp.MustCompile(`^require\(`),
	regexp.MustCompile(`^import\s*\(`),
	regexp.MustCompile(`^import\s+"`),
	regexp.MustCompile(`#include\s*<`),
	regexp.MustCompile(`#include\s*"`),
	regexp.MustCompile(`^package\s+`),
}

// SignaturePatterns for aggressive filtering
var SignaturePatterns = []*regexp.Regexp{
	regexp.MustCompile(`^(pub\s+)?(async\s+)?fn\s+\w+`),
	regexp.MustCompile(`^(pub\s+)?struct\s+\w+`),
	regexp.MustCompile(`^(pub\s+)?enum\s+\w+`),
	regexp.MustCompile(`^(pub\s+)?trait\s+\w+`),
	regexp.MustCompile(`^(pub\s+)?type\s+\w+`),
	regexp.MustCompile(`^impl\s+`),
	regexp.MustCompile(`^func\s+(\([^)]+\)\s+)?\w+`),
	regexp.MustCompile(`^type\s+\w+\s+(struct|interface)`),
	regexp.MustCompile(`^type\s+\w+\s+\w+`),
	regexp.MustCompile(`^def\s+\w+`),
	regexp.MustCompile(`^async\s+def\s+\w+`),
	regexp.MustCompile(`^class\s+\w+`),
	regexp.MustCompile(`^function\s+\w+`),
	regexp.MustCompile(`^(export\s+)?(async\s+)?function\s*\w*`),
	regexp.MustCompile(`^(export\s+)?(default\s+)?class\s+\w+`),
	regexp.MustCompile(`^(export\s+)?const\s+\w+\s*=\s*(async\s+)?\([^)]*\)\s*=>`),
	regexp.MustCompile(`^interface\s+\w+`),
	regexp.MustCompile(`^type\s+\w+\s*=`),
	regexp.MustCompile(`^(public|private|protected)?\s*(static\s+)?(class|interface|enum)\s+\w+`),
	regexp.MustCompile(`^(public|private|protected)?\s*(static\s+)?(async\s+)?\w+\s+\w+\s*\(`),
}

// BlockDelimiters for brace tracking
var BlockDelimiters = map[rune]rune{
	'{': '}',
	'[': ']',
	'(': ')',
}

// TestResultPatterns
var TestResultPatterns = []*regexp.Regexp{
	regexp.MustCompile(`test result: (ok|FAILED|ignored)\.`),
	regexp.MustCompile(`(\d+) passed`),
	regexp.MustCompile(`(\d+) failed`),
	regexp.MustCompile(`(\d+) ignored`),
	regexp.MustCompile(`(\d+) skipped`),
	regexp.MustCompile(`PASS`),
	regexp.MustCompile(`FAIL`),
	regexp.MustCompile(`ok\s+\S+\s+[\d.]+s`),
}

// DiffHunkPattern
var DiffHunkPattern = regexp.MustCompile(`^@@\s+-\d+(?:,\d+)?\s+\+\d+(?:,\d+)?\s+@@`)

// LogTimestampPatterns
var LogTimestampPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}`),
	regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}`),
	regexp.MustCompile(`^\d{2}:\d{2}:\d{2}`),
}

// CommentFilter strips comments from source code.
type CommentFilter struct {
	patterns map[Language]*regexp.Regexp
}

// newCommentFilter creates a new comment filter.
func newCommentFilter() *CommentFilter {
	return &CommentFilter{
		patterns: CommentPatternsMap,
	}
}

// Name returns the filter name.
func (f *CommentFilter) Name() string {
	return "comment"
}

// Apply strips comments and returns token savings.
func (f *CommentFilter) Apply(input string, mode Mode) (string, int) {
	lang := DetectLanguageFromInput(input)

	pattern, ok := f.patterns[lang]
	if !ok {
		pattern = commentFallbackRe
	}

	original := len(input)
	output := pattern.ReplaceAllString(input, "")

	bytesSaved := original - len(output)
	tokensSaved := bytesSaved / 4

	return output, tokensSaved
}

// stripComments is a utility function to strip comments from code.
func stripComments(code string, lang Language) string {
	filter := newCommentFilter()
	output, _ := filter.Apply(code, ModeMinimal)
	return output
}
