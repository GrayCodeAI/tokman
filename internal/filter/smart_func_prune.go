package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// SmartFuncPruner selectively strips function bodies based on importance.
// Unlike BodyFilter (which strips all bodies in aggressive mode), this filter
// preserves short functions, constructors, error handlers, and interface
// implementations — stripping only large, low-information bodies.
//
// Heuristics used:
//   - Bodies shorter than MinBodyLines are kept (already concise)
//   - Constructor-named functions (New*, Make*, Init*) are kept
//   - Functions containing only return/panic/error statements are kept
//   - Bodies longer than MinBodyLines are replaced with a /* N lines */ stub
//
// Task #138: Smart function body pruning.
type SmartFuncPruner struct {
	// MinBodyLines is the minimum body line count before pruning is considered.
	// Default: 8. Bodies shorter than this are never stripped.
	MinBodyLines int
	// MaxKeptLines caps how many body lines to keep (0 = respect MinBodyLines only).
	MaxKeptLines int
}

// NewSmartFuncPruner creates a SmartFuncPruner with sensible defaults.
func NewSmartFuncPruner() *SmartFuncPruner {
	return &SmartFuncPruner{MinBodyLines: 8}
}

// Name returns the filter name.
func (f *SmartFuncPruner) Name() string { return "smart_func_prune" }

// Apply selectively prunes function bodies in code blocks.
func (f *SmartFuncPruner) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}
	if !looksLikeCode(input) {
		return input, 0
	}

	originalTokens := core.EstimateTokens(input)

	lang := DetectLanguageFromInput(input)
	var output string
	switch lang {
	case LangGo:
		output = f.pruneGo(input)
	case LangRust:
		output = f.pruneRust(input)
	case LangPython:
		output = f.prunePython(input)
	case LangJavaScript, LangTypeScript:
		output = f.pruneJS(input)
	default:
		if mode == ModeAggressive {
			output = f.pruneGeneric(input)
		} else {
			return input, 0
		}
	}

	saved := originalTokens - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// importantFuncRe matches constructor-like and critical function names.
var importantFuncRe = regexp.MustCompile(`(?i)^func\s+(?:\([^)]+\)\s+)?(New|Make|Init|Close|Open|Error|String|MarshalJSON|UnmarshalJSON|ServeHTTP)\w*\s*\(`)

func (f *SmartFuncPruner) pruneGo(input string) string {
	lines := strings.Split(input, "\n")
	result := make([]string, 0, len(lines))
	i := 0

	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		if goFuncRe.MatchString(trimmed) {
			// Collect the complete function (signature + body)
			sig, bodyStart, bodyLines, end := collectGoFunc(lines, i)
			funcLen := end - i

			shouldPrune := funcLen > f.MinBodyLines &&
				!importantFuncRe.MatchString(line) &&
				!isSimpleBody(bodyLines)

			if shouldPrune {
				// Emit signature + placeholder
				result = append(result, sig...)
				if bodyStart >= 0 {
					result = append(result, "{ /* "+itoa(len(bodyLines))+" lines pruned */ }")
				}
			} else {
				result = append(result, lines[i:end]...)
			}
			i = end
			continue
		}

		result = append(result, line)
		i++
	}

	return strings.Join(result, "\n")
}

// collectGoFunc collects lines of a Go function starting at start.
// Returns: signature lines, bodyStart index (or -1), body lines, end index.
func collectGoFunc(lines []string, start int) (sig []string, bodyStart int, bodyLines []string, end int) {
	i := start
	depth := 0
	bodyStart = -1

	for i < len(lines) {
		line := lines[i]
		opens := strings.Count(line, "{")
		closes := strings.Count(line, "}")

		if bodyStart < 0 && opens > 0 {
			bodyStart = i
			depth += opens - closes
			if depth <= 0 {
				// single-line function
				sig = []string{line}
				return sig, bodyStart, nil, i + 1
			}
			// signature ends here, body starts
			sig = lines[start : i+1]
			i++
			break
		}
		i++
	}

	// Now collect body lines
	for i < len(lines) {
		line := lines[i]
		depth += strings.Count(line, "{")
		depth -= strings.Count(line, "}")
		if depth <= 0 {
			end = i + 1
			return sig, bodyStart, lines[bodyStart+1 : i], end
		}
		bodyLines = append(bodyLines, line)
		i++
	}

	return sig, bodyStart, bodyLines, i
}

// isSimpleBody returns true when the body is trivially simple:
// only return/panic/error/assign/call lines — worth keeping.
func isSimpleBody(lines []string) bool {
	if len(lines) == 0 {
		return true
	}
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || t == "}" {
			continue
		}
		if strings.HasPrefix(t, "return ") ||
			strings.HasPrefix(t, "panic(") ||
			strings.HasPrefix(t, "err ") ||
			strings.HasPrefix(t, "if err") ||
			t == "return nil" ||
			t == "return" {
			continue
		}
		return false
	}
	return true
}

// pruneRust delegates to the existing BodyFilter in aggressive mode.
func (f *SmartFuncPruner) pruneRust(input string) string {
	bf := NewBodyFilter()
	out := bf.stripRustBodies(input)
	return out
}

// prunePython applies smart pruning for Python def bodies.
var pyDefRe = regexp.MustCompile(`^(\s*)def\s+(\w+)\s*\(`)
var pyImportantRe = regexp.MustCompile(`^(\s*)def\s+(__(init|new|repr|str|enter|exit|call|len|iter|next)__)`)

func (f *SmartFuncPruner) prunePython(input string) string {
	lines := strings.Split(input, "\n")
	result := make([]string, 0, len(lines))
	i := 0

	for i < len(lines) {
		line := lines[i]
		if !pyDefRe.MatchString(line) || pyImportantRe.MatchString(line) {
			result = append(result, line)
			i++
			continue
		}

		// Measure indentation level
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		bodyLines := []string{}
		j := i + 1
		for j < len(lines) {
			bl := lines[j]
			if bl == "" {
				j++
				continue
			}
			blIndent := len(bl) - len(strings.TrimLeft(bl, " \t"))
			if blIndent <= indent && strings.TrimSpace(bl) != "" {
				break
			}
			bodyLines = append(bodyLines, bl)
			j++
		}

		if len(bodyLines) > f.MinBodyLines {
			result = append(result, line)
			result = append(result, strings.Repeat(" ", indent+4)+
				"pass  # "+itoa(len(bodyLines))+" lines pruned")
		} else {
			result = append(result, lines[i:j]...)
		}
		i = j
	}

	return strings.Join(result, "\n")
}

// pruneJS applies smart pruning for JavaScript/TypeScript function bodies.
func (f *SmartFuncPruner) pruneJS(input string) string {
	bf := NewBodyFilter()
	out := bf.stripJSBodies(input)
	return out
}

// pruneGeneric strips large brace-delimited blocks for unknown languages.
func (f *SmartFuncPruner) pruneGeneric(input string) string {
	bf := NewBodyFilter()
	out := bf.stripGenericBodies(input)
	return out
}
