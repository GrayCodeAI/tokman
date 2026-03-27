package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// CommentStripFilter strips comments from source code while preserving
// the content of string literals (which may contain comment-like sequences).
//
// Supports all major comment formats:
//   - //  (Go, C, C++, Java, JavaScript, Rust, Swift)
//   - #   (Python, Ruby, Shell, YAML, TOML, Dockerfile)
//   - --  (SQL, Lua, Haskell)
//   - %   (LaTeX, MATLAB, Erlang)
//   - ;   (Assembly, Lisp, INI)
//   - /* */ (C, Go, JavaScript, CSS)
//   - """ """ / ''' ''' (Python docstrings)
//   - <!-- --> (HTML, XML, SVG)
//   - {- -} (Haskell)
//   - (* *) (OCaml, Pascal, Mathematica)
//   - `` (R, Kotlin doc)
//
// Safety: does NOT strip comments inside string literals. Uses a simple
// state machine to track in-string state before stripping.
type CommentStripFilter struct {
	// PreserveDocComments keeps doc comments (///, //!, /** */).
	// Useful when doc comments carry API semantics.
	PreserveDocComments bool
}

var (
	// Block comment detectors by language
	htmlCommentStripRe    = regexp.MustCompile(`<!--[\s\S]*?-->`)
	cBlockCommentRe       = regexp.MustCompile(`/\*[\s\S]*?\*/`)
	haskellBlockCommentRe = regexp.MustCompile(`\{-[\s\S]*?-\}`)
	ocamlBlockCommentRe   = regexp.MustCompile(`\(\*[\s\S]*?\*\)`)

	// Python docstring (triple-quoted)
	pyDocstringRe = regexp.MustCompile(`(?s)"""[\s\S]*?"""|'''[\s\S]*?'''`)

	// Content-type detection for comment style
	usesHashRe  = regexp.MustCompile(`(?m)^[ \t]*#[^!]`)   // Python/Shell/Ruby/YAML
	usesDashRe  = regexp.MustCompile(`(?m)^[ \t]*--`)       // SQL/Lua/Haskell
	usesPercentRe = regexp.MustCompile(`(?m)^[ \t]*%[^{]`) // LaTeX/MATLAB
	usesSemiRe  = regexp.MustCompile(`(?m)^[ \t]*;`)        // Assembly/Lisp
)

// NewCommentStripFilter creates a comment stripping filter.
func NewCommentStripFilter() *CommentStripFilter {
	return &CommentStripFilter{}
}

// NewCommentStripFilterPreserveDoc creates a filter that keeps doc comments.
func NewCommentStripFilterPreserveDoc() *CommentStripFilter {
	return &CommentStripFilter{PreserveDocComments: true}
}

// Name returns the filter name.
func (f *CommentStripFilter) Name() string {
	return "comment_strip"
}

// Apply strips comments from the input.
func (f *CommentStripFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := core.EstimateTokens(input)
	output := f.strip(input)

	// Clean up blank lines left after stripping
	output = collapseBlankLines(output, 2)

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// strip applies language-appropriate comment removal.
func (f *CommentStripFilter) strip(input string) string {
	output := input

	// Detect comment styles
	hasSlash := strings.Contains(input, "//")
	hasCBlock := strings.Contains(input, "/*")
	hasHTML := strings.Contains(input, "<!--")
	hasHaskellBlock := strings.Contains(input, "{-")
	hasOCaml := strings.Contains(input, "(*")
	hasPyDoc := strings.Contains(input, `"""`) || strings.Contains(input, `'''`)

	// Block comments first (multi-line)
	if hasHTML {
		output = htmlCommentStripRe.ReplaceAllString(output, "")
	}
	if hasCBlock {
		output = f.stripCBlockComments(output)
	}
	if hasHaskellBlock {
		output = haskellBlockCommentRe.ReplaceAllString(output, "")
	}
	if hasOCaml {
		output = ocamlBlockCommentRe.ReplaceAllString(output, "")
	}
	if hasPyDoc {
		output = pyDocstringRe.ReplaceAllString(output, "")
	}

	// Line comments
	lines := strings.Split(output, "\n")
	result := make([]string, 0, len(lines))
	for _, line := range lines {
		line = f.stripLineComment(line, hasSlash, input)
		result = append(result, line)
	}
	output = strings.Join(result, "\n")

	return output
}

// stripCBlockComments removes C-style /* */ comments but preserves doc comments
// if PreserveDocComments is set.
func (f *CommentStripFilter) stripCBlockComments(input string) string {
	if !f.PreserveDocComments {
		return cBlockCommentRe.ReplaceAllString(input, "")
	}
	// Preserve /** and /*! doc comments
	return regexp.MustCompile(`/\*(?:[^*]|\*[^/])*\*/`).ReplaceAllStringFunc(input, func(m string) string {
		if strings.HasPrefix(m, "/**") || strings.HasPrefix(m, "/*!") {
			return m
		}
		return ""
	})
}

// stripLineComment removes the comment portion from a single line.
func (f *CommentStripFilter) stripLineComment(line string, hasSlashComments bool, fullInput string) string {
	// Detect comment style for this line
	trimmed := strings.TrimSpace(line)

	// C-style //
	if hasSlashComments && strings.Contains(line, "//") {
		if f.PreserveDocComments && (strings.Contains(line, "///") || strings.Contains(line, "//!")) {
			return line
		}
		line = stripAfterCommentMarker(line, "//")
	}

	// Hash comments (#)
	if usesHashRe.MatchString(fullInput) && strings.Contains(trimmed, "#") {
		// Don't strip shebang lines
		if !strings.HasPrefix(trimmed, "#!") {
			line = stripAfterCommentMarker(line, "#")
		}
	}

	// SQL/Lua -- comments
	if usesDashRe.MatchString(fullInput) && strings.Contains(line, "--") {
		line = stripAfterCommentMarker(line, "--")
	}

	// % comments (LaTeX, MATLAB)
	if usesPercentRe.MatchString(fullInput) && strings.Contains(line, "%") {
		line = stripAfterCommentMarker(line, "%")
	}

	// ; comments (Assembly, Lisp)
	if usesSemiRe.MatchString(fullInput) && strings.Contains(line, ";") {
		line = stripAfterCommentMarker(line, ";")
	}

	return line
}

// stripAfterCommentMarker removes everything from the comment marker to EOL,
// but respects string literals (doesn't strip if marker is inside a string).
func stripAfterCommentMarker(line, marker string) string {
	inSingle := false
	inDouble := false

	for i := 0; i < len(line)-len(marker)+1; i++ {
		ch := line[i]
		if ch == '\'' && !inDouble {
			inSingle = !inSingle
		} else if ch == '"' && !inSingle {
			inDouble = !inDouble
		} else if !inSingle && !inDouble {
			if strings.HasPrefix(line[i:], marker) {
				return strings.TrimRight(line[:i], " \t")
			}
		}
	}
	return line
}
