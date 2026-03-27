package filter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ASTSkeletonFilter extracts code skeletons from source files by replacing
// large function bodies with "..." placeholders. This preserves the API
// surface (signatures, types, interfaces) while discarding implementation
// details that are irrelevant for most LLM reasoning tasks.
//
// Language support: Go, Python, JavaScript/TypeScript, Java, Rust, C/C++.
// Falls back gracefully on unrecognized content (pass-through).
//
// Min body lines before eliding: 8 (minimal), 4 (aggressive).
type ASTSkeletonFilter struct{}

var (
	// Go: func name(args) (returns) {
	goFuncRe = regexp.MustCompile(`(?m)^(\s*(?:func\s+\([^)]*\)\s+\w+|func\s+\w+)\s*\([^)]*\)[^{]*\{)\s*$`)
	// Python: def name(args):  /  async def name(args):
	pyFuncRe = regexp.MustCompile(`(?m)^(\s*(?:async\s+)?def\s+\w+\s*\([^)]*\)[^:]*:)\s*$`)
	// JS/TS: function name(args) {  /  name(args) {  /  name = (args) => {
	jsFuncRe = regexp.MustCompile(`(?m)^(\s*(?:(?:export\s+)?(?:async\s+)?function\s+\w+|(?:export\s+)?(?:async\s+)?\w+\s*[:=]\s*(?:async\s+)?(?:function\s*)?)\s*\([^)]*\)[^{]*\{)\s*$`)
	// Java/C#: [modifiers] returnType name(args) {
	javaFuncRe = regexp.MustCompile(`(?m)^(\s*(?:(?:public|private|protected|static|final|override|virtual|abstract)\s+)*\w[\w<>\[\]]*\s+\w+\s*\([^)]*\)\s*(?:throws\s+\w+(?:,\s*\w+)*)?\s*\{)\s*$`)
	// Rust: fn name(args) -> ret {
	rustFuncRe = regexp.MustCompile(`(?m)^(\s*(?:pub\s+)?(?:async\s+)?fn\s+\w+[^{]*\{)\s*$`)

	// Code detection
	codeHintRe = regexp.MustCompile(`(?:func |def |function |class |struct |impl |interface |enum )\w+`)
)

// NewASTSkeletonFilter creates a new AST skeleton extraction filter.
func NewASTSkeletonFilter() *ASTSkeletonFilter {
	return &ASTSkeletonFilter{}
}

// Name returns the filter name.
func (f *ASTSkeletonFilter) Name() string {
	return "ast_skeleton"
}

// Apply extracts a code skeleton by eliding large function bodies.
func (f *ASTSkeletonFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	// Only apply to code-like content
	if !codeHintRe.MatchString(input) {
		return input, 0
	}

	minBodyLines := 8
	if mode == ModeAggressive {
		minBodyLines = 4
	}

	original := core.EstimateTokens(input)
	output := f.elideBodies(input, minBodyLines)

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// elideBodyPatterns lists the opening-brace/colon patterns per language.
var elideBodyPatterns = []*regexp.Regexp{
	goFuncRe,
	javaFuncRe,
	rustFuncRe,
	jsFuncRe,
}

// elideBodies scans for function definitions and elides their bodies when large.
func (f *ASTSkeletonFilter) elideBodies(input string, minBodyLines int) string {
	lines := strings.Split(input, "\n")
	result := make([]string, 0, len(lines))
	i := 0

	for i < len(lines) {
		line := lines[i]

		// Check for Python (indent-based) function start
		if pyFuncRe.MatchString(line) {
			sig, bodyStart, bodyLen := f.extractPythonBody(lines, i)
			if bodyLen >= minBodyLines {
				result = append(result, sig)
				result = append(result, f.bodyPlaceholder(bodyLen, detectIndent(lines[i])))
				i = bodyStart + bodyLen
				continue
			}
		}

		// Check for brace-delimited function start
		matchedBrace := false
		for _, re := range elideBodyPatterns {
			if re.MatchString(line) {
				bodyStart, bodyLen := f.extractBraceBody(lines, i)
				if bodyLen >= minBodyLines {
					result = append(result, line)
					result = append(result, f.bodyPlaceholder(bodyLen, detectIndent(lines[i])+"  "))
					// Closing brace
					closeIdx := bodyStart + bodyLen
					if closeIdx < len(lines) {
						result = append(result, lines[closeIdx])
						i = closeIdx + 1
					} else {
						i = closeIdx
					}
					matchedBrace = true
					break
				}
			}
		}
		if matchedBrace {
			continue
		}

		result = append(result, line)
		i++
	}

	return strings.Join(result, "\n")
}

// extractBraceBody finds the body of a brace-delimited function starting at lineIdx.
// Returns (firstBodyLine, bodyLineCount) — bodyLineCount excludes closing brace.
func (f *ASTSkeletonFilter) extractBraceBody(lines []string, startIdx int) (int, int) {
	// Opening brace is on startIdx
	depth := 0
	for _, ch := range lines[startIdx] {
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
		}
	}

	bodyStart := startIdx + 1
	i := bodyStart
	for i < len(lines) && depth > 0 {
		for _, ch := range lines[i] {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
			}
		}
		if depth > 0 {
			i++
		}
	}
	// i is now the index of the closing brace
	return bodyStart, i - bodyStart
}

// extractPythonBody finds the body of an indent-based Python function.
// Returns (signature line, firstBodyLine, bodyLineCount).
func (f *ASTSkeletonFilter) extractPythonBody(lines []string, startIdx int) (string, int, int) {
	sig := lines[startIdx]
	indent := detectIndent(sig)

	// Body is lines with indent > sig indent
	bodyStart := startIdx + 1
	i := bodyStart
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			i++
			continue
		}
		lineIndent := detectIndent(lines[i])
		if len(lineIndent) <= len(indent) {
			break
		}
		i++
	}
	return sig, bodyStart, i - bodyStart
}

// bodyPlaceholder returns a single-line placeholder for an elided function body.
func (f *ASTSkeletonFilter) bodyPlaceholder(lineCount int, indent string) string {
	return fmt.Sprintf("%s// ... [%d lines elided] ...", indent, lineCount)
}

// detectIndent returns the leading whitespace of a line.
func detectIndent(line string) string {
	for i, ch := range line {
		if ch != ' ' && ch != '\t' {
			return line[:i]
		}
	}
	return line
}
