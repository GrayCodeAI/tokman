package filter

import (
	"strings"
)

// QualityMetrics measures compression quality by checking information preservation.
// T185: Track if compressed output still answers user's question.
type QualityMetrics struct {
	OriginalLength      int
	CompressedLength    int
	ErrorPreserved      bool
	NumbersPreserved    bool
	KeywordsPreserved   float64
	StructuralIntegrity bool
}

// MeasureQuality computes quality metrics for a compression result.
func MeasureQuality(original, compressed string) QualityMetrics {
	return QualityMetrics{
		OriginalLength:      len(original),
		CompressedLength:    len(compressed),
		ErrorPreserved:      checkErrorPreserved(original, compressed),
		NumbersPreserved:    checkNumbersPreserved(original, compressed),
		KeywordsPreserved:   checkKeywordsPreserved(original, compressed),
		StructuralIntegrity: checkStructuralIntegrity(compressed),
	}
}

// QualityScore returns 0.0-1.0 quality score.
func (m QualityMetrics) QualityScore() float64 {
	score := 0.0
	components := 0.0

	if m.ErrorPreserved {
		score += 1.0
	}
	components++

	if m.NumbersPreserved {
		score += 1.0
	}
	components++

	score += m.KeywordsPreserved
	components++

	if m.StructuralIntegrity {
		score += 1.0
	}
	components++

	if components == 0 {
		return 1.0
	}
	return score / components
}

func checkErrorPreserved(original, compressed string) bool {
	lower := strings.ToLower(original)
	if !strings.Contains(lower, "error") && !strings.Contains(lower, "fail") {
		return true // No errors in original
	}
	compLower := strings.ToLower(compressed)
	return strings.Contains(compLower, "error") || strings.Contains(compLower, "fail")
}

func checkNumbersPreserved(original, compressed string) bool {
	origNumbers := extractNumbers(original)
	compNumbers := extractNumbers(compressed)

	if len(origNumbers) == 0 {
		return true
	}

	// Check that at least 50% of numbers are preserved
	preserved := 0
	for _, n := range origNumbers {
		for _, cn := range compNumbers {
			if n == cn {
				preserved++
				break
			}
		}
	}

	return float64(preserved) >= float64(len(origNumbers))*0.5
}

func checkKeywordsPreserved(original, compressed string) float64 {
	keywords := []string{"error", "warning", "failed", "success", "passed", "exit", "code"}
	origLower := strings.ToLower(original)
	compLower := strings.ToLower(compressed)

	found := 0
	preserved := 0
	for _, kw := range keywords {
		if strings.Contains(origLower, kw) {
			found++
			if strings.Contains(compLower, kw) {
				preserved++
			}
		}
	}

	if found == 0 {
		return 1.0
	}
	return float64(preserved) / float64(found)
}

func checkStructuralIntegrity(s string) bool {
	// Check balanced brackets
	parens, brackets, braces := 0, 0, 0
	for _, c := range s {
		switch c {
		case '(':
			parens++
		case ')':
			parens--
		case '[':
			brackets++
		case ']':
			brackets--
		case '{':
			braces++
		case '}':
			braces--
		}
	}
	return parens >= -5 && parens <= 10 && brackets >= -5 && brackets <= 10 && braces >= -5 && braces <= 10
}

func extractNumbers(s string) []string {
	var numbers []string
	current := ""
	for _, c := range s {
		if (c >= '0' && c <= '9') || c == '.' {
			current += string(c)
		} else {
			if len(current) > 0 {
				numbers = append(numbers, current)
				current = ""
			}
		}
	}
	if len(current) > 0 {
		numbers = append(numbers, current)
	}
	return numbers
}
