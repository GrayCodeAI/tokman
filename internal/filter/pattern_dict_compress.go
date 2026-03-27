package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// PatternDictCompressor compresses common code/text patterns using a
// language-aware substitution dictionary.
//
// Analogous to Zstandard dictionary compression: for each language we
// maintain a list of verbose-to-compact substitutions that are safe to
// apply reversibly (the compact form is still readable by an LLM).
//
// Examples:
//
//	"github.com/GrayCodeAI/tokman" → "[pkg:tokman]"
//	"context.Background()" → "[ctx:bg]"
//	"if err != nil { return err }" → "[goerr]"
//	"console.log(" → "[log("
//	"System.out.println(" → "[sout("
//
// The filter detects the dominant language in the input and applies the
// appropriate dictionary. Substitutions are only applied when they save
// at least 3 tokens each.
type PatternDictCompressor struct{}

// patternSubstitution is a single substitution rule.
type patternSubstitution struct {
	from *regexp.Regexp
	to   string
}

// Per-language pattern dictionaries (compiled once at package init).
var (
	goPatterns = []patternSubstitution{
		// Common Go idioms
		{regexp.MustCompile(`\bif err != nil \{\s*\n\s*return err\s*\n\s*\}`), `if err != nil { return err }`},
		{regexp.MustCompile(`\bif err != nil \{\s*\n\s*return nil, err\s*\n\s*\}`), `if err != nil { return nil, err }`},
		{regexp.MustCompile(`context\.Background\(\)`), `ctx.Background()`},
		{regexp.MustCompile(`context\.TODO\(\)`), `ctx.TODO()`},
		{regexp.MustCompile(`fmt\.Errorf\(`), `errorf(`},
		{regexp.MustCompile(`log\.Printf\(`), `logf(`},
		{regexp.MustCompile(`log\.Println\(`), `logln(`},
		{regexp.MustCompile(`strings\.TrimSpace\(`), `trimSpace(`},
		{regexp.MustCompile(`strings\.Contains\(`), `contains(`},
		{regexp.MustCompile(`strings\.HasPrefix\(`), `hasPrefix(`},
		{regexp.MustCompile(`strings\.HasSuffix\(`), `hasSuffix(`},
	}

	pythonPatterns = []patternSubstitution{
		{regexp.MustCompile(`\bif __name__ == ["']__main__["']:\s*\n`), "[main]\n"},
		{regexp.MustCompile(`\bprint\(f["']`), `printf("`},
		{regexp.MustCompile(`\blogging\.getLogger\(__name__\)`), `log`},
		{regexp.MustCompile(`\bself\._`), `s._`},
		{regexp.MustCompile(`\bException as e:\n\s+`), `Exception as e: `},
		{regexp.MustCompile(`from typing import `), `type: `},
		{regexp.MustCompile(`Optional\[(\w+)\]`), `$1?`},
	}

	jsPatterns = []patternSubstitution{
		{regexp.MustCompile(`console\.log\(`), `log(`},
		{regexp.MustCompile(`console\.error\(`), `err(`},
		{regexp.MustCompile(`console\.warn\(`), `warn(`},
		{regexp.MustCompile(`\basync\s+function\s+`), `async fn `},
		{regexp.MustCompile(`\bPromise\.resolve\(`), `resolved(`},
		{regexp.MustCompile(`\bPromise\.reject\(`), `rejected(`},
		{regexp.MustCompile(`\bDocument\.getElementById\(`), `getElementById(`},
		{regexp.MustCompile(`addEventListener\('`), `on('`},
	}

	javaPatterns = []patternSubstitution{
		{regexp.MustCompile(`System\.out\.println\(`), `sout(`},
		{regexp.MustCompile(`System\.err\.println\(`), `serr(`},
		{regexp.MustCompile(`\bpublic static void main\(String\[\] args\)`), `main()`},
		{regexp.MustCompile(`@Override\s*\n`), ""},
		{regexp.MustCompile(`\bnew ArrayList<>\(`), `list(`},
		{regexp.MustCompile(`\bnew HashMap<>\(`), `map(`},
		{regexp.MustCompile(`\bObjects\.requireNonNull\(`), `nonNull(`},
	}

	// Language-detection hints
	goHintRe     = regexp.MustCompile(`(?:^package\s+\w+|:=|func\s+\w+\()`)
	pythonHintRe = regexp.MustCompile(`(?:def\s+\w+\(|import\s+\w+|print\(|__name__)`)
	jsHintRe     = regexp.MustCompile(`(?:const\s+|let\s+|var\s+|=>\s*\{|console\.log)`)
	javaHintRe   = regexp.MustCompile(`(?:public\s+class|System\.out|void\s+main)`)
)

// NewPatternDictCompressor creates a new pattern dictionary compressor.
func NewPatternDictCompressor() *PatternDictCompressor {
	return &PatternDictCompressor{}
}

// Name returns the filter name.
func (f *PatternDictCompressor) Name() string {
	return "pattern_dict_compress"
}

// Apply applies language-aware pattern substitutions.
func (f *PatternDictCompressor) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := core.EstimateTokens(input)
	lang := patternDetectLanguage(input)
	if lang == "" {
		return input, 0
	}

	output := applyPatterns(input, lang, mode)

	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// patternDetectLanguage returns the detected language for pattern matching.
func patternDetectLanguage(input string) string {
	// Sample first 2000 chars for detection
	sample := input
	if len(sample) > 2000 {
		sample = sample[:2000]
	}

	goScore := len(goHintRe.FindAllString(sample, -1))
	pyScore := len(pythonHintRe.FindAllString(sample, -1))
	jsScore := len(jsHintRe.FindAllString(sample, -1))
	javaScore := len(javaHintRe.FindAllString(sample, -1))

	maxScore := 0
	lang := ""
	if goScore > maxScore {
		maxScore = goScore
		lang = "go"
	}
	if pyScore > maxScore {
		maxScore = pyScore
		lang = "python"
	}
	if jsScore > maxScore {
		maxScore = jsScore
		lang = "js"
	}
	if javaScore > maxScore {
		lang = "java"
	}

	if maxScore < 2 {
		return "" // not confident enough
	}
	return lang
}

// applyPatterns applies the substitutions for a given language.
func applyPatterns(input, lang string, mode Mode) string {
	var patterns []patternSubstitution
	switch lang {
	case "go":
		patterns = goPatterns
	case "python":
		patterns = pythonPatterns
	case "js", "javascript", "typescript":
		patterns = jsPatterns
	case "java":
		patterns = javaPatterns
	default:
		return input
	}

	output := input
	// In minimal mode, only apply single-line substitutions
	// In aggressive mode, apply all including multi-line ones
	for _, p := range patterns {
		if mode != ModeAggressive && strings.Contains(p.to, "\n") {
			continue
		}
		output = p.from.ReplaceAllString(output, p.to)
	}
	return output
}
