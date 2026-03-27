package filter

import (
	"math"
	"regexp"
	"sort"
	"strings"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ImportanceScoringFilter scores each line by estimated information value and
// prunes the lowest-scoring lines to hit a target keep fraction.
//
// Scoring combines four signals (all sub-millisecond, no LLM):
//  1. TF-IDF: lines whose rare words appear frequently in this document score higher
//  2. Positional bias: first 10% and last 10% of content score higher (topic sentences, conclusions)
//  3. Structural cues: function signatures, class decls, headings, error lines get a boost
//  4. Density: prefer lines with a high ratio of non-trivial tokens to length
//
// KeepFraction: fraction of lines to retain. Default 0.6 (minimal), 0.4 (aggressive).
type ImportanceScoringFilter struct {
	KeepFraction float64 // 0.0–1.0
}

var (
	// Lines that are structurally important regardless of content
	importantLineRe = regexp.MustCompile(
		`(?i)^[\s#/*]*(?:` +
			`func\s+|def\s+|class\s+|struct\s+|interface\s+|impl\s+|trait\s+|` + // function/type declarations
			`enum\s+|type\s+|const\s+\w+\s*=|var\s+\w+|let\s+\w+|` + // bindings
			`TODO|FIXME|HACK|BUG|NOTE|IMPORTANT|WARNING|ERROR|PANIC|` + // markers
			`@[A-Za-z]|#\[|^\s*\d+\.\s|^\s*[-*]\s` + // annotations, list items
			`)`,
	)
	// Trivial lines: braces alone, blank lines, short separators
	trivialLineRe = regexp.MustCompile(`^[\s{}()\[\];,|]*$`)
)

// NewImportanceScoringFilter creates a filter with default keep fraction.
func NewImportanceScoringFilter() *ImportanceScoringFilter {
	return &ImportanceScoringFilter{KeepFraction: 0.6}
}

// NewImportanceScoringFilterWithFraction creates a filter with a custom keep fraction.
func NewImportanceScoringFilterWithFraction(frac float64) *ImportanceScoringFilter {
	return &ImportanceScoringFilter{KeepFraction: frac}
}

// Name returns the filter name.
func (f *ImportanceScoringFilter) Name() string {
	return "importance_scoring"
}

// Apply scores and prunes low-importance lines.
func (f *ImportanceScoringFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	lines := strings.Split(input, "\n")
	if len(lines) < 15 {
		return input, 0
	}

	keepFrac := f.KeepFraction
	if keepFrac <= 0 {
		keepFrac = 0.6
	}
	if mode == ModeAggressive {
		keepFrac *= 0.67 // ≈ 0.4 for default
	}
	keepFrac = clampFloat(keepFrac, 0.1, 1.0)

	original := core.EstimateTokens(input)

	// Build document-level word frequency (for IDF approximation)
	wordDocFreq := buildWordDocFreq(lines)
	totalLines := float64(len(lines))

	scores := make([]float64, len(lines))
	for i, line := range lines {
		scores[i] = f.scoreLine(line, i, len(lines), wordDocFreq, totalLines)
	}

	// Determine threshold: keep top keepFrac by score
	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	sort.Float64s(sorted)
	keepCount := int(math.Ceil(float64(len(lines)) * keepFrac))
	if keepCount < 5 {
		keepCount = 5
	}
	threshold := 0.0
	if cutIdx := len(sorted) - keepCount; cutIdx >= 0 {
		threshold = sorted[cutIdx]
	}

	// Build output: keep high-score lines; insert omission markers
	keep := make([]bool, len(lines))
	for i, s := range scores {
		if s >= threshold {
			keep[i] = true
			// Keep immediate neighbours for context
			if i > 0 {
				keep[i-1] = true
			}
			if i+1 < len(lines) {
				keep[i+1] = true
			}
		}
	}
	// Always keep first and last lines
	keep[0] = true
	keep[len(lines)-1] = true

	var result []string
	skipped := 0
	for i, line := range lines {
		if keep[i] {
			if skipped > 0 {
				result = append(result, "... ["+itoa(skipped)+" low-importance lines omitted] ...")
				skipped = 0
			}
			result = append(result, line)
		} else {
			skipped++
		}
	}
	if skipped > 0 {
		result = append(result, "... ["+itoa(skipped)+" low-importance lines omitted] ...")
	}

	output := strings.Join(result, "\n")
	saved := original - core.EstimateTokens(output)
	if saved <= 0 {
		return input, 0
	}
	return output, saved
}

// scoreLine computes an importance score for a single line.
func (f *ImportanceScoringFilter) scoreLine(line string, idx, total int, wordDocFreq map[string]int, totalLines float64) float64 {
	// Trivial lines score near zero
	if trivialLineRe.MatchString(line) {
		return 0.01
	}

	// Structural boost
	structural := 0.0
	if importantLineRe.MatchString(line) {
		structural = 1.0
	}

	// TF-IDF score: sum of IDF weights for words in this line
	words := strings.Fields(strings.ToLower(line))
	tfidf := 0.0
	for _, w := range words {
		w = strings.Trim(w, ".,;:!?\"'()[]{}*/\\")
		if len(w) < 3 {
			continue
		}
		df := wordDocFreq[w]
		if df == 0 {
			df = 1
		}
		// IDF: log(N / df) — rare words score higher
		idf := math.Log(totalLines / float64(df))
		tfidf += idf
	}
	if len(words) > 0 {
		tfidf /= float64(len(words)) // normalize by line length
	}

	// Positional bias: first and last 10% score higher
	pos := float64(idx) / totalLines
	positional := 0.0
	if pos < 0.1 || pos > 0.9 {
		positional = 0.5
	} else if pos < 0.2 || pos > 0.8 {
		positional = 0.2
	}

	// Density: ratio of non-trivial tokens to line length
	density := 0.0
	if len(line) > 0 {
		nonTrivial := 0
		for _, w := range words {
			if len(w) >= 3 {
				nonTrivial++
			}
		}
		density = float64(nonTrivial) / math.Max(float64(len(words)), 1)
	}

	return tfidf + structural*2.0 + positional + density*0.5
}

// buildWordDocFreq builds a map of word → number of lines containing that word.
func buildWordDocFreq(lines []string) map[string]int {
	freq := make(map[string]int, 256)
	for _, line := range lines {
		seen := make(map[string]bool)
		for _, w := range strings.Fields(strings.ToLower(line)) {
			w = strings.Trim(w, ".,;:!?\"'()[]{}*/\\")
			if len(w) >= 3 && !seen[w] {
				freq[w]++
				seen[w] = true
			}
		}
	}
	return freq
}

func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// itoa converts an int to a decimal string (avoids strconv import duplication).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + itoa(-n)
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append(digits, byte('0'+n%10))
		n /= 10
	}
	// reverse
	for i, j := 0, len(digits)-1; i < j; i, j = i+1, j-1 {
		digits[i], digits[j] = digits[j], digits[i]
	}
	return string(digits)
}
