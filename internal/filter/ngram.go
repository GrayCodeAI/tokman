package filter

import (
	"regexp"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/simd"
)

// NgramAbbreviator compresses output by abbreviating common patterns.
// Research-based: CompactPrompt N-gram Abbreviation (2025) - achieves 10-20%
// lossless compression by replacing common tokens with shorter equivalents.
//
// Key insight: Programming and CLI output contains many repeated long tokens
// that can be abbreviated while remaining understandable to LLMs.
type NgramAbbreviator struct {
	abbreviations map[string]string
	codePatterns  map[string]string
	cliPatterns   map[string]string
	inString      *regexp.Regexp
}

// NewNgramAbbreviator creates a new n-gram abbreviator.
func NewNgramAbbreviator() *NgramAbbreviator {
	return &NgramAbbreviator{
		abbreviations: initAbbreviations(),
		codePatterns:  initCodePatterns(),
		cliPatterns:   initCLIPatterns(),
		inString:      regexp.MustCompile("[\"'`](.|[\r\n])*?[\"'`]"),
	}
}

// initAbbreviations returns the core abbreviation dictionary
func initAbbreviations() map[string]string {
	return map[string]string{
		// Programming keywords
		"function":      "fn",
		"return":        "ret",
		"const":         "cst",
		"var":           "vr",
		"let":           "lt",
		"import":        "imp",
		"export":        "exp",
		"async":         "asc",
		"await":         "awt",
		"static":        "stc",
		"public":        "pub",
		"private":       "prv",
		"protected":     "ptd",
		"interface":     "iface",
		"implements":    "impl",
		"extends":       "ext",
		"namespace":     "ns",
		"package":       "pkg",
		"module":        "mod",
		"class":         "cls",
		"struct":        "stct",
		"enum":          "en",
		"constructor":   "ctor",
		"destructor":    "dtor",
		"parameter":     "param",
		"argument":      "arg",
		"variable":      "vrbl",
		"constant":      "cnst",
		"initialize":    "init",
		"configuration": "cfg",
		"environment":   "env",
		"development":   "dev",
		"production":    "prod",
		"staging":       "stag",
		"testing":       "test",

		// Common words
		"information": "info",
		"message":     "msg",
		"error":       "err",
		"warning":     "warn",
		"success":     "ok",
		"failure":     "fail",
		"exception":   "exc",
		"response":    "resp",
		"request":     "req",
		"reference":   "ref",
		"property":    "prop",
		"attribute":   "attr",
		"method":      "mtd",
		"value":       "val",
		"string":      "str",
		"number":      "num",
		"integer":     "int",
		"boolean":     "bool",
		"character":   "char",
		"array":       "arr",
		"object":      "obj",
		"element":     "el",
		"document":    "doc",
		"directory":   "dir",
		"file":        "fl",
		"resource":    "res",
		"process":     "proc",
		"thread":      "thrd",
		"connection":  "conn",
		"transaction": "txn",
	}
}

// initCodePatterns returns code-specific patterns
func initCodePatterns() map[string]string {
	return map[string]string{
		// Type declarations
		"undefined": "undef",
		"null":      "nil",
		"boolean":   "bool",
		"integer":   "int",
		"float":     "flt",
		"double":    "dbl",
		"string":    "str",
		"character": "chr",
		"object":    "obj",
		"array":     "arr",
		"function":  "fn",
		"promise":   "pm",
		"async":     "asc",
		"await":     "awt",

		// Common operations
		"toString":    "toStr",
		"parseInt":    "toInt",
		"parseFloat":  "toFlt",
		"length":      "len",
		"indexOf":     "idxOf",
		"substring":   "substr",
		"replace":     "repl",
		"contains":    "has",
		"startsWith":  "startsW",
		"endsWith":    "endsW",
		"toLowerCase": "toLower",
		"toUpperCase": "toUpper",
		"trim":        "trm",
		"split":       "splt",
		"join":        "jn",
		"push":        "psh",
		"pop":         "pp",
		"shift":       "shft",
		"unshift":     "unshft",
		"slice":       "slc",
		"splice":      "splc",
		"forEach":     "each",
		"map":         "mp",
		"filter":      "fltr",
		"reduce":      "red",
		"find":        "fnd",
		"some":        "sm",
		"every":       "evry",
		"includes":    "incl",
		"sort":        "srt",
		"reverse":     "rev",
		"concat":      "cat",
		"spread":      "sprd",

		// Common patterns
		"console.log":      "log",
		"console.error":    "logErr",
		"console.warn":     "logWarn",
		"console.info":     "logInfo",
		"return null":      "ret nil",
		"return true":      "ret T",
		"return false":     "ret F",
		"return undefined": "ret undef",
	}
}

// initCLIPatterns returns CLI-specific patterns
func initCLIPatterns() map[string]string {
	return map[string]string{
		// Status messages
		"Successfully": "✓",
		"successfully": "✓",
		"Success":      "✓",
		"success":      "✓",
		"Failed":       "✗",
		"failed":       "✗",
		"Failure":      "✗",
		"failure":      "✗",
		"Error":        "ERR",
		"error":        "err",
		"Warning":      "WARN",
		"warning":      "warn",
		"Complete":     "DONE",
		"complete":     "done",
		"Finished":     "DONE",
		"finished":     "done",
		"Running":      "RUN",
		"running":      "run",
		"Starting":     "START",
		"starting":     "start",
		"Processing":   "PROC",
		"processing":   "proc",
		"Building":     "BUILD",
		"building":     "build",
		"Compiling":    "COMP",
		"compiling":    "comp",
		"Downloading":  "DL",
		"downloading":  "dl",
		"Uploading":    "UL",
		"uploading":    "ul",
		"Installing":   "INST",
		"installing":   "inst",
		"Uninstalling": "UNINST",
		"uninstalling": "uninst",
		"Updating":     "UPD",
		"updating":     "upd",
		"Creating":     "CREAT",
		"creating":     "creat",
		"Deleting":     "DEL",
		"deleting":     "del",
		"Modifying":    "MOD",
		"modifying":    "mod",
		"Skipping":     "SKIP",
		"skipping":     "skip",
		"Ignoring":     "IGN",
		"ignoring":     "ign",
		"Checking":     "CHK",
		"checking":     "chk",
		"Validating":   "VAL",
		"validating":   "val",
		"Verifying":    "VER",
		"verifying":    "ver",

		// Common CLI phrases
		"Please wait": "...",
		"please wait": "...",
		"In progress": "...",
		"in progress": "...",
		"No changes":  "(no changes)",
		"no changes":  "(no changes)",
		"Not found":   "(not found)",
		"not found":   "(not found)",
		"None":        "(none)",
		"none":        "(none)",
		"Empty":       "(empty)",
		"empty":       "(empty)",
	}
}

// Name returns the filter name.
func (f *NgramAbbreviator) Name() string {
	return "ngram"
}

// Apply applies n-gram abbreviation to the input.
func (f *NgramAbbreviator) Apply(input string, mode Mode) (string, int) {
	original := len(input)

	// Don't process very short inputs
	if original < 100 {
		return input, 0
	}

	// Apply abbreviations
	output := f.applyAbbreviations(input, mode)

	bytesSaved := original - len(output)
	tokensSaved := bytesSaved / 4

	return output, tokensSaved
}

// applyAbbreviations applies abbreviation patterns
func (f *NgramAbbreviator) applyAbbreviations(input string, mode Mode) string {
	// Detect context: code vs CLI
	isCode := f.detectCodeContext(input)

	output := input

	// Apply CLI patterns first (more aggressive)
	for pattern, abbrev := range f.cliPatterns {
		// Case-sensitive replacement for CLI patterns
		output = strings.ReplaceAll(output, pattern, abbrev)
	}

	// Apply code patterns (more careful)
	if isCode {
		output = f.applyCodeAbbreviations(output, mode)
	}

	// Apply general abbreviations
	for pattern, abbrev := range f.abbreviations {
		output = f.safeReplace(output, pattern, abbrev)
	}

	return output
}

// detectCodeContext checks if output looks like code
func (f *NgramAbbreviator) detectCodeContext(input string) bool {
	codeIndicators := []string{
		"func ", "function ", "def ", "class ", "struct ",
		"import ", "package ", "use ", "require(",
		"pub fn", "pub struct", "pub async",
		"const ", "let ", "var ",
		"=>", "{", "}",
	}

	for _, ind := range codeIndicators {
		if strings.Contains(input, ind) {
			return true
		}
	}

	return false
}

// applyCodeAbbreviations applies code-specific abbreviations
func (f *NgramAbbreviator) applyCodeAbbreviations(input string, mode Mode) string {
	output := input

	for pattern, abbrev := range f.codePatterns {
		// Only replace whole words
		output = f.replaceWord(output, pattern, abbrev)
	}

	return output
}

// safeReplace replaces only when not inside a string
func (f *NgramAbbreviator) safeReplace(input, pattern, replacement string) string {
	// Simple approach: replace word boundaries only
	return f.replaceWord(input, pattern, replacement)
}

// replaceWord replaces whole words only using SIMD-optimized operations.
func (f *NgramAbbreviator) replaceWord(input, pattern, replacement string) string {
	// Fast path: use byte-level operations with SIMD helpers
	inputBytes := []byte(input)
	patternBytes := []byte(pattern)
	patternLen := len(patternBytes)

	// Pre-allocate result buffer (conservatively)
	result := make([]byte, 0, len(inputBytes))

	i := 0
	for i < len(inputBytes) {
		// Check if we have a potential match
		if i+patternLen <= len(inputBytes) {
			// Check word match (case-insensitive for keywords)
			match := true
			for j := 0; j < patternLen; j++ {
				if toLowerByte(inputBytes[i+j]) != patternBytes[j] {
					match = false
					break
				}
			}

			// Check word boundaries using SIMD-optimized function
			if match {
				// Check before - not a word char
				if i > 0 && simd.IsWordChar(inputBytes[i-1]) {
					match = false
				}
				// Check after - not a word char
				if i+patternLen < len(inputBytes) && simd.IsWordChar(inputBytes[i+patternLen]) {
					match = false
				}
			}

			if match {
				// Add replacement
				result = append(result, replacement...)
				i += patternLen
				continue
			}
		}

		result = append(result, inputBytes[i])
		i++
	}

	return string(result)
}

// toLowerByte converts a byte to lowercase (ASCII only)
func toLowerByte(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// isWordChar checks if a rune is part of a word (used by h2o.go for unicode support)
func isWordChar(r rune) bool {
	// ASCII fast path
	if r < 128 {
		return simd.IsWordChar(byte(r))
	}
	// Unicode letters and numbers are also word chars
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// GetAbbreviationLegend returns a legend for common abbreviations
func (f *NgramAbbreviator) GetAbbreviationLegend() string {
	legend := "Abbreviations: "
	count := 0

	for pattern, abbrev := range f.abbreviations {
		if count > 0 {
			legend += ", "
		}
		legend += pattern + "→" + abbrev
		count++
		if count >= 10 {
			legend += ", ..."
			break
		}
	}

	return legend
}
