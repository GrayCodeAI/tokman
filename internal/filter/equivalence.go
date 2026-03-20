package filter

import (
	"strings"
)

// SemanticEquivalence checks if compressed output preserves meaning.
// R16: Verify no critical information was lost during compression.
type SemanticEquivalence struct{}

// NewSemanticEquivalence creates a checker.
func NewSemanticEquivalence() *SemanticEquivalence {
	return &SemanticEquivalence{}
}

// Check returns an equivalence report.
func (s *SemanticEquivalence) Check(original, compressed string) EquivalenceReport {
	return EquivalenceReport{
		ErrorPreserved:     checkErrorsPreserved(original, compressed),
		NumbersPreserved:   checkCriticalNumbers(original, compressed),
		URLsPreserved:      checkURLsPreserved(original, compressed),
		FilePathsPreserved: checkPathsPreserved(original, compressed),
		ExitCodesPreserved: checkExitCodes(original, compressed),
		Score:              computeEquivalenceScore(original, compressed),
	}
}

// EquivalenceReport holds the semantic check results.
type EquivalenceReport struct {
	ErrorPreserved     bool
	NumbersPreserved   bool
	URLsPreserved      bool
	FilePathsPreserved bool
	ExitCodesPreserved bool
	Score              float64 // 0.0-1.0
}

// IsGood returns true if the compression preserved critical information.
func (r EquivalenceReport) IsGood() bool {
	return r.Score >= 0.7 && r.ErrorPreserved
}

func checkErrorsPreserved(original, compressed string) bool {
	lower := strings.ToLower(original)
	if !strings.Contains(lower, "error") && !strings.Contains(lower, "fail") {
		return true
	}
	compLower := strings.ToLower(compressed)
	return strings.Contains(compLower, "error") || strings.Contains(compLower, "fail")
}

func checkCriticalNumbers(original, compressed string) bool {
	// Extract numbers that look like exit codes, line numbers, etc.
	origNums := extractCriticalNumbers(original)
	compNums := extractCriticalNumbers(compressed)

	if len(origNums) == 0 {
		return true
	}

	preserved := 0
	for _, n := range origNums {
		for _, cn := range compNums {
			if n == cn {
				preserved++
				break
			}
		}
	}
	return float64(preserved) >= float64(len(origNums))*0.5
}

func checkURLsPreserved(original, compressed string) bool {
	urls := extractURLs(original)
	if len(urls) == 0 {
		return true
	}
	for _, url := range urls {
		if !strings.Contains(compressed, url) {
			return false
		}
	}
	return true
}

func checkPathsPreserved(original, compressed string) bool {
	paths := extractPaths(original)
	if len(paths) == 0 {
		return true
	}
	preserved := 0
	for _, p := range paths {
		if strings.Contains(compressed, p) {
			preserved++
		}
	}
	return float64(preserved) >= float64(len(paths))*0.5
}

func checkExitCodes(original, compressed string) bool {
	// Check for "exit code" patterns
	lower := strings.ToLower(original)
	if !strings.Contains(lower, "exit code") && !strings.Contains(lower, "exit:") {
		return true
	}
	compLower := strings.ToLower(compressed)
	return strings.Contains(compLower, "exit code") || strings.Contains(compLower, "exit:")
}

func computeEquivalenceScore(original, compressed string) float64 {
	score := 0.0
	components := 0.0

	if checkErrorsPreserved(original, compressed) {
		score += 1.0
	}
	components++

	if checkCriticalNumbers(original, compressed) {
		score += 1.0
	}
	components++

	if checkURLsPreserved(original, compressed) {
		score += 1.0
	}
	components++

	if checkPathsPreserved(original, compressed) {
		score += 1.0
	}
	components++

	if checkExitCodes(original, compressed) {
		score += 1.0
	}
	components++

	if components == 0 {
		return 1.0
	}
	return score / components
}

func extractCriticalNumbers(s string) []string {
	var nums []string
	current := ""
	for _, c := range s {
		if c >= '0' && c <= '9' {
			current += string(c)
		} else {
			if len(current) > 0 && len(current) <= 5 { // Exit codes, line numbers
				nums = append(nums, current)
			}
			current = ""
		}
	}
	if len(current) > 0 && len(current) <= 5 {
		nums = append(nums, current)
	}
	return nums
}

func extractURLs(s string) []string {
	var urls []string
	for _, prefix := range []string{"http://", "https://"} {
		idx := 0
		for {
			start := strings.Index(s[idx:], prefix)
			if start < 0 {
				break
			}
			start += idx
			end := start + len(prefix)
			for end < len(s) && s[end] != ' ' && s[end] != '\n' && s[end] != ')' && s[end] != '>' {
				end++
			}
			urls = append(urls, s[start:end])
			idx = end
		}
	}
	return urls
}

func extractPaths(s string) []string {
	var paths []string
	for _, line := range strings.Split(s, "\n") {
		for _, prefix := range []string{"/", "./", "../", "~/"} {
			if strings.Contains(line, prefix) {
				// Simple heuristic: extract path-like strings
				words := strings.Fields(line)
				for _, w := range words {
					if strings.HasPrefix(w, prefix) || strings.Contains(w, "/") {
						if len(w) > 3 && len(w) < 200 {
							paths = append(paths, w)
						}
					}
				}
			}
		}
	}
	return paths
}
