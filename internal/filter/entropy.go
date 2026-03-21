package filter

import (
	"math"
	"strings"
	"sync"
)

// Cached token frequencies (initialized once, shared across all EntropyFilter instances)
var (
	cachedTokenFrequencies map[string]float64
	tokenFrequenciesOnce   sync.Once
)

// EntropyFilter implements Selective Context compression (Mila/Guerin et al., 2023).
// Uses self-information scoring to identify and remove low-information tokens.
//
// Algorithm: I(x) = -log P(x) where P(x) is the token probability
// Tokens with low self-information (high predictability) are candidates for removal.
//
// Research Results: 2-3x compression while preserving semantic content.
type EntropyFilter struct {
	// Token frequency table (could be learned from corpus)
	frequencies map[string]float64
	totalTokens float64

	// Threshold for entropy-based pruning
	entropyThreshold float64
}

// NewEntropyFilter creates a new entropy-based filter
func NewEntropyFilter() *EntropyFilter {
	return NewEntropyFilterWithThreshold(2.0)
}

// NewEntropyFilterWithThreshold creates an entropy filter with custom threshold.
// T34: Configurable entropy threshold.
func NewEntropyFilterWithThreshold(threshold float64) *EntropyFilter {
	return &EntropyFilter{
		frequencies:      initTokenFrequencies(),
		totalTokens:      1000000, // Normalized corpus size
		entropyThreshold: threshold,
	}
}

// initTokenFrequencies returns common token frequencies (cached via sync.Once)
// T31-T32: Expanded from ~150 to 500+ entries including code-specific tokens.
func initTokenFrequencies() map[string]float64 {
	tokenFrequenciesOnce.Do(func() {
		cachedTokenFrequencies = buildTokenFrequencies()
	})
	return cachedTokenFrequencies
}

// buildTokenFrequencies builds the token frequency map
func buildTokenFrequencies() map[string]float64 {
	return map[string]float64{
		// ============ ENGLISH COMMON WORDS ============
		// Very common tokens (high frequency = low entropy = candidates for removal)
		"the":      50000,
		"a":        30000,
		"an":       15000,
		"is":       25000,
		"are":      20000,
		"was":      18000,
		"were":     12000,
		"be":       15000,
		"been":     10000,
		"being":    8000,
		"have":     20000,
		"has":      18000,
		"had":      15000,
		"do":       18000,
		"does":     12000,
		"did":      10000,
		"will":     15000,
		"would":    12000,
		"could":    10000,
		"should":   8000,
		"may":      10000,
		"might":    8000,
		"must":     7000,
		"can":      15000,
		"to":       40000,
		"of":       35000,
		"in":       30000,
		"for":      25000,
		"on":       20000,
		"with":     18000,
		"at":       18000,
		"by":       15000,
		"from":     15000,
		"as":       20000,
		"into":     10000,
		"through":  8000,
		"during":   7000,
		"before":   8000,
		"after":    9000,
		"above":    6000,
		"below":    5000,
		"between":  7000,
		"under":    6000,
		"again":    7000,
		"further":  6000,
		"then":     12000,
		"once":     8000,
		"here":     10000,
		"there":    12000,
		"when":     15000,
		"where":    12000,
		"why":      8000,
		"how":      12000,
		"all":      15000,
		"each":     10000,
		"few":      6000,
		"more":     12000,
		"most":     10000,
		"other":    12000,
		"some":     12000,
		"such":     10000,
		"no":       15000,
		"nor":      5000,
		"not":      20000,
		"only":     10000,
		"own":      8000,
		"same":     8000,
		"so":       18000,
		"than":     12000,
		"too":      10000,
		"very":     10000,
		"just":     15000,
		"and":      45000,
		"but":      20000,
		"or":       20000,
		"if":       18000,
		"because":  10000,
		"until":    7000,
		"while":    9000,
		"although": 6000,
		"though":   7000,
		"this":     25000,
		"that":     30000,
		"these":    12000,
		"those":    10000,
		"what":     15000,
		"which":    18000,
		"who":      15000,
		"whom":     5000,
		"it":       30000,
		"its":      12000,
		"they":     20000,
		"them":     15000,
		"their":    18000,
		"we":       20000,
		"us":       12000,
		"our":      15000,
		"you":      25000,
		"your":     18000,
		"he":       18000,
		"him":      12000,
		"his":      15000,
		"she":      15000,
		"her":      15000,
		"hers":     6000,
		"i":        35000,
		"me":       15000,
		"my":       18000,
		"myself":   7000,

		// ============ PROGRAMMING KEYWORDS (Language-specific) ============
		// Go
		"func": 25000, "package": 20000, "import": 25000, "return": 30000, "const": 15000, "struct": 18000,
		"interface": 15000, "map": 12000, "chan": 8000, "defer": 12000,
		"select": 8000, "case": 12000, "switch": 10000, "range": 12000,
		"break": 8000, "else": 18000, "nil": 15000, "float64": 15000,
		"byte": 10000, "make": 18000, "append": 15000, "cap": 8000,
		"close": 10000, "copy": 8000, "panic": 10000, "recover": 8000,
		"println": 10000, "fmt": 20000, "main": 18000, "init": 12000,
		"err": 25000, "ok": 18000,
		// Python
		"def": 25000, "class": 18000, "yield": 8000, "lambda": 8000,
		"pass": 10000, "raise": 10000, "except": 10000, "global": 6000,
		"nonlocal": 4000, "assert": 10000, "del": 6000, "None": 18000,
		"self": 20000, "str": 15000, "list": 18000, "dict": 15000,
		"tuple": 10000, "super": 8000, "__init__": 12000, "__name__": 8000,
		// JavaScript/TypeScript
		"function": 20000, "let": 20000, "arrow": 6000, "export": 15000,
		"typeof": 10000, "instanceof": 8000, "NaN": 8000,
		"console": 18000, "require": 12000, "module": 10000, "exports": 10000,
		"throw": 12000, "catch": 12000, "Promise": 10000, "resolve": 10000,
		"reject": 8000, "Array": 12000, "Object": 12000, "String": 12000,
		"Number": 10000, "Boolean": 10000, "Map": 8000, "Set": 8000,
		"Symbol": 6000, "prototype": 8000, "constructor": 8000, "extends": 10000,
		"implements": 6000, "enum": 8000, "namespace": 10000, "declare": 8000,
		"readonly": 6000,
		// Rust
		"fn": 20000, "pub": 18000, "mod": 12000, "use": 18000,
		"impl": 15000, "trait": 12000, "match": 15000, "Some": 12000,
		"Ok": 12000, "Err": 12000, "Result": 12000, "Option": 12000,
		"Vec": 12000, "Box": 8000, "Rc": 6000, "Arc": 6000,
		"mut": 18000, "ref": 8000, "move": 8000,
		"Self": 12000, "crate": 10000, "unsafe": 8000, "dyn": 8000,
		"macro": 8000, "derive": 10000,
		// C/C++
		"void": 15000, "char": 12000, "short": 8000, "long": 10000,
		"double": 10000, "unsigned": 10000, "signed": 6000, "static": 15000,
		"extern": 8000, "inline": 8000, "volatile": 6000, "register": 4000,
		"sizeof": 10000, "typedef": 10000, "union": 6000, "include": 18000,
		"define": 12000, "ifdef": 8000, "ifndef": 8000, "endif": 8000,
		"pragma": 6000, "NULL": 12000, "nullptr": 8000, "std": 15000,
		"cout": 10000, "cin": 8000, "endl": 10000, "using": 15000,
		"template": 12000, "public": 15000, "private": 12000, "protected": 10000,
		"virtual": 10000, "override": 8000, "final": 8000, "constexpr": 6000,
		"noexcept": 6000,

		// ============ COMMON PROGRAMMING SYMBOLS ============
		"=": 30000, "==": 18000, "!=": 15000, ">=": 12000, "<=": 12000,
		"->": 15000, "=>": 12000, "::": 10000, ":=": 8000,
		"+=": 12000, "-=": 10000, "*=": 8000, "/=": 8000,
		"&&": 15000, "||": 12000, "++": 10000, "--": 10000,
		"//": 25000, "/*": 12000, "*/": 12000,

		// ============ BUILD/CLI TOOL WORDS ============
		"npm": 18000, "install": 18000, "run": 20000, "build": 20000,
		"start": 15000, "dev": 15000, "watch": 10000,
		"clean": 10000, "lint": 12000, "format": 10000, "check": 12000,
		"update": 12000, "upgrade": 8000, "remove": 10000, "add": 18000,
		"config": 12000, "set": 15000, "get": 15000,
		"show": 12000, "status": 15000, "diff": 12000,
		"commit": 15000, "push": 12000, "pull": 12000, "fetch": 10000,
		"merge": 12000, "rebase": 10000, "checkout": 12000, "branch": 12000,
		"stash": 8000, "tag": 8000, "remote": 10000,

		// ============ DOCKER/K8S WORDS ============
		"docker": 18000, "container": 15000, "image": 15000, "volume": 10000,
		"network": 10000, "port": 12000, "service": 12000, "pod": 10000,
		"deploy": 12000, "replica": 8000, "kubectl": 12000, "apply": 12000,
		"create": 15000, "describe": 10000, "exec": 12000, "logs": 15000,
		"port-forward": 8000,

		// ============ FILE/PATH WORDS ============
		"file": 20000, "path": 15000, "dir": 12000, "directory": 10000,
		"src": 15000, "lib": 12000, "bin": 10000, "dist": 10000,
		"node_modules": 8000, "target": 10000, "vendor": 8000,
		"json": 15000, "yaml": 10000, "toml": 8000, "xml": 8000,
		"txt": 8000, "md": 8000, "js": 12000, "ts": 12000,
		"py": 12000, "rs": 10000, "rb": 8000,

		// ============ LOG/OUTPUT WORDS ============
		"info": 15000, "warn": 12000, "warning": 12000,
		"debug": 15000, "trace": 10000, "fatal": 8000, "critical": 8000,
		"success": 12000, "failed": 15000, "done": 12000,
		"completed": 10000, "started": 10000, "running": 10000,
		"stopped": 8000, "skipped": 8000, "ignored": 6000,
		"passed": 12000, "pending": 8000,

		// ============ HTTP WORDS ============
		"GET": 15000, "POST": 15000, "PUT": 12000, "DELETE": 12000,
		"PATCH": 8000, "HEAD": 8000, "OPTIONS": 6000,
		"http": 12000, "https": 12000, "localhost": 10000, "api": 15000,
		"response": 12000, "request": 12000, "header": 10000,
		"body": 10000, "code": 18000,

		// ============ DATA WORDS ============
		"key": 15000, "value": 15000, "name": 18000, "id": 18000,
		"index": 12000, "count": 12000, "size": 12000, "length": 10000,
		"width": 8000, "height": 8000, "data": 18000, "settings": 10000,
		"options": 12000, "params": 12000, "args": 12000, "flags": 10000,
		"input": 15000, "output": 15000, "result": 15000,

		// ============ MISC CODE WORDS ============
		"null": 15000, "empty": 10000, "invalid": 10000, "missing": 10000,
		"enabled": 10000, "disabled": 8000, "optional": 8000, "required": 10000,
		"deprecated": 6000, "example": 10000, "sample": 8000, "mock": 8000,
		"setup": 10000, "teardown": 6000,
	}
}

// Name returns the filter name
func (f *EntropyFilter) Name() string {
	return "entropy"
}

// Apply applies entropy-based filtering to remove low-information tokens
func (f *EntropyFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := len(input)

	// Process line by line to maintain structure
	lines := strings.Split(input, "\n")
	var result []string

	for _, line := range lines {
		processed := f.processLine(line, mode)
		result = append(result, processed)
	}

	output := strings.Join(result, "\n")
	saved := (original - len(output)) / 4 // Rough token estimate

	return output, saved
}

// processLine processes a single line with entropy filtering
func (f *EntropyFilter) processLine(line string, mode Mode) string {
	words := tokenize(line)
	if len(words) == 0 {
		return line
	}

	var result []string
	for _, word := range words {
		if f.shouldKeep(word, mode) {
			result = append(result, word)
		}
	}

	return strings.Join(result, " ")
}

// shouldKeep determines if a word should be kept based on entropy
func (f *EntropyFilter) shouldKeep(word string, mode Mode) bool {
	// Always keep non-stopwords
	wordLower := strings.ToLower(word)

	// Check if it's a known high-frequency token
	freq, exists := f.frequencies[wordLower]
	if !exists {
		// Unknown token - likely important, keep it
		return true
	}

	// Calculate self-information (entropy)
	// I(x) = -log P(x)
	probability := freq / f.totalTokens
	entropy := -math.Log2(probability)

	// Aggressive mode: higher threshold (remove more)
	threshold := f.entropyThreshold
	if mode == ModeAggressive {
		threshold = 4.0
	}

	// Keep words with high entropy (low frequency = more informative)
	return entropy >= threshold
}

// calculateEntropy calculates the self-information of a token
func (f *EntropyFilter) calculateEntropy(token string) float64 {
	freq, exists := f.frequencies[strings.ToLower(token)]
	if !exists {
		return 10.0 // High entropy for unknown tokens
	}

	probability := freq / f.totalTokens
	return -math.Log2(probability)
}

// SetThreshold allows customizing the entropy threshold
func (f *EntropyFilter) SetThreshold(threshold float64) {
	f.entropyThreshold = threshold
}
