package filter

import (
	"regexp"
	"strings"
)

var codeSymbolRe = regexp.MustCompile(`[{}\[\]();:]`)

// GoalDrivenFilter implements SWE-Pruner style compression (Shanghai Jiao Tong, 2025).
// Goal-driven line-level pruning using CRF-inspired scoring.
//
// Algorithm:
// 1. Parse goal/intent from query
// 2. Score each line for relevance to goal
// 3. Apply CRF-style sequential labeling for keep/prune decisions
// 4. Preserve structural coherence
//
// Research Results: Up to 14.8x compression for code contexts.
type GoalDrivenFilter struct {
	goal      string
	goalTerms map[string]float64
	mode      GoalMode

	// CRF-style transition weights
	keepKeepWeight   float64
	keepPruneWeight  float64
	pruneKeepWeight  float64
	prunePruneWeight float64
}

// GoalMode defines the goal-driven filtering mode
type GoalMode int

const (
	GoalModeDebug GoalMode = iota
	GoalModeReview
	GoalModeDeploy
	GoalModeSearch
	GoalModeBuild
	GoalModeTest
	GoalModeGeneric
)

// NewGoalDrivenFilter creates a new goal-driven filter
func NewGoalDrivenFilter(goal string) *GoalDrivenFilter {
	g := &GoalDrivenFilter{
		goal:             goal,
		goalTerms:        make(map[string]float64),
		mode:             parseGoalMode(goal),
		keepKeepWeight:   0.8, // Prefer keeping sequences
		keepPruneWeight:  0.3,
		pruneKeepWeight:  0.5,
		prunePruneWeight: 0.2,
	}

	g.extractGoalTerms()
	return g
}

// parseGoalMode determines the goal mode from the goal string
func parseGoalMode(goal string) GoalMode {
	goalLower := strings.ToLower(goal)

	if strings.Contains(goalLower, "debug") || strings.Contains(goalLower, "error") || strings.Contains(goalLower, "fix") {
		return GoalModeDebug
	}
	if strings.Contains(goalLower, "review") || strings.Contains(goalLower, "check") {
		return GoalModeReview
	}
	if strings.Contains(goalLower, "deploy") || strings.Contains(goalLower, "prod") || strings.Contains(goalLower, "release") {
		return GoalModeDeploy
	}
	if strings.Contains(goalLower, "search") || strings.Contains(goalLower, "find") || strings.Contains(goalLower, "grep") {
		return GoalModeSearch
	}
	if strings.Contains(goalLower, "build") || strings.Contains(goalLower, "compile") || strings.Contains(goalLower, "make") {
		return GoalModeBuild
	}
	if strings.Contains(goalLower, "test") {
		return GoalModeTest
	}

	return GoalModeGeneric
}

// extractGoalTerms extracts important terms from the goal
func (f *GoalDrivenFilter) extractGoalTerms() {
	// Tokenize goal
	words := tokenize(f.goal)

	for i, word := range words {
		wordLower := strings.ToLower(word)

		// Position weight: earlier words more important
		posWeight := 1.0 - float64(i)/float64(len(words)+1)

		// Length weight: longer words often more specific
		lenWeight := float64(len(word)) / 10.0
		if lenWeight > 1.0 {
			lenWeight = 1.0
		}

		f.goalTerms[wordLower] = posWeight * lenWeight
	}

	// Add mode-specific terms
	f.addModeSpecificTerms()
}

// addModeSpecificTerms adds terms based on the goal mode
func (f *GoalDrivenFilter) addModeSpecificTerms() {
	switch f.mode {
	case GoalModeDebug:
		f.goalTerms["error"] = 2.0
		f.goalTerms["warning"] = 1.8
		f.goalTerms["exception"] = 2.0
		f.goalTerms["trace"] = 1.5
		f.goalTerms["stack"] = 1.5
		f.goalTerms["fail"] = 2.0
		f.goalTerms["failed"] = 2.0
		f.goalTerms["panic"] = 2.0
		f.goalTerms["fatal"] = 2.0
		f.goalTerms["undefined"] = 1.8
		f.goalTerms["null"] = 1.5
		f.goalTerms["nil"] = 1.5

	case GoalModeReview:
		f.goalTerms["changed"] = 1.8
		f.goalTerms["modified"] = 1.8
		f.goalTerms["added"] = 1.5
		f.goalTerms["removed"] = 1.5
		f.goalTerms["deleted"] = 1.5
		f.goalTerms["update"] = 1.5
		f.goalTerms["fix"] = 1.8
		f.goalTerms["todo"] = 1.5
		f.goalTerms["fixme"] = 1.5

	case GoalModeDeploy:
		f.goalTerms["success"] = 1.8
		f.goalTerms["deployed"] = 2.0
		f.goalTerms["published"] = 1.8
		f.goalTerms["release"] = 1.5
		f.goalTerms["version"] = 1.3
		f.goalTerms["environment"] = 1.3
		f.goalTerms["production"] = 1.5

	case GoalModeSearch:
		f.goalTerms["found"] = 1.8
		f.goalTerms["match"] = 1.5
		f.goalTerms["result"] = 1.3
		f.goalTerms["file"] = 1.2

	case GoalModeBuild:
		f.goalTerms["building"] = 1.5
		f.goalTerms["compiling"] = 1.5
		f.goalTerms["success"] = 1.8
		f.goalTerms["error"] = 2.0
		f.goalTerms["warning"] = 1.5
		f.goalTerms["complete"] = 1.5

	case GoalModeTest:
		f.goalTerms["pass"] = 1.8
		f.goalTerms["passed"] = 1.8
		f.goalTerms["fail"] = 2.0
		f.goalTerms["failed"] = 2.0
		f.goalTerms["test"] = 1.3
		f.goalTerms["assert"] = 1.5
		f.goalTerms["expect"] = 1.5
	}
}

// Name returns the filter name
func (f *GoalDrivenFilter) Name() string {
	return "goal_driven"
}

// Apply applies goal-driven filtering
func (f *GoalDrivenFilter) Apply(input string, mode Mode) (string, int) {
	if mode == ModeNone {
		return input, 0
	}

	original := len(input)

	lines := strings.Split(input, "\n")

	// Score each line
	scores := f.scoreLines(lines)

	// Apply CRF-style sequential decision
	decisions := f.crfDecode(scores)

	// Build output
	var result []string
	for i, line := range lines {
		if decisions[i] {
			result = append(result, line)
		}
	}

	output := strings.Join(result, "\n")
	saved := (original - len(output)) / 4

	return output, saved
}

// scoreLines scores each line for goal relevance
func (f *GoalDrivenFilter) scoreLines(lines []string) []float64 {
	scores := make([]float64, len(lines))

	for i, line := range lines {
		scores[i] = f.scoreLine(line)
	}

	return scores
}

// scoreLine scores a single line for goal relevance
func (f *GoalDrivenFilter) scoreLine(line string) float64 {
	lineLower := strings.ToLower(line)
	score := 0.0

	// Check for goal terms
	words := tokenize(lineLower)
	for _, word := range words {
		if weight, exists := f.goalTerms[word]; exists {
			score += weight
		}
	}

	// Structural importance
	if isErrorLine(line) {
		score += 3.0
	}
	if isWarningLine(line) {
		score += 2.0
	}
	if isHeadingLine(line) {
		score += 1.5
	}
	if isCodeLine(line) {
		score += 0.5
	}

	// Length penalty (prefer shorter lines in debug mode)
	if f.mode == GoalModeDebug && len(line) > 200 {
		score *= 0.8
	}

	return score
}

// crfDecode applies CRF-style decoding for keep/prune decisions
func (f *GoalDrivenFilter) crfDecode(scores []float64) []bool {
	if len(scores) == 0 {
		return []bool{}
	}

	decisions := make([]bool, len(scores))

	// Normalize scores
	maxScore := 0.0
	for _, s := range scores {
		if s > maxScore {
			maxScore = s
		}
	}

	threshold := maxScore * 0.3 // Keep lines with 30%+ of max score

	// Viterbi-style decoding with transition weights
	for i := range scores {
		keepScore := scores[i]

		// Transition bonus
		if i > 0 && decisions[i-1] {
			keepScore += f.keepKeepWeight
		}

		decisions[i] = keepScore >= threshold
	}

	// Post-processing: ensure structural coherence
	f.ensureCoherence(decisions, scores)

	return decisions
}

// ensureCoherence ensures structural coherence of kept lines
func (f *GoalDrivenFilter) ensureCoherence(decisions []bool, scores []float64) {
	// Always keep first and last non-empty lines
	for i := 0; i < len(decisions); i++ {
		if scores[i] > 0 {
			decisions[i] = true
			break
		}
	}
	for i := len(decisions) - 1; i >= 0; i-- {
		if scores[i] > 0 {
			decisions[i] = true
			break
		}
	}

	// Don't allow more than 3 consecutive pruned lines
	prunedCount := 0
	for i := range decisions {
		if !decisions[i] {
			prunedCount++
			if prunedCount > 3 && i > 0 {
				// Keep this line to break the sequence
				decisions[i] = true
				prunedCount = 0
			}
		} else {
			prunedCount = 0
		}
	}
}

// isErrorLine checks if a line indicates an error
func isErrorLine(line string) bool {
	lineLower := strings.ToLower(line)
	return strings.Contains(lineLower, "error") ||
		strings.Contains(lineLower, "exception") ||
		strings.Contains(lineLower, "panic") ||
		strings.Contains(lineLower, "fatal") ||
		strings.Contains(lineLower, "failed:")
}

// isWarningLine checks if a line indicates a warning
func isWarningLine(line string) bool {
	lineLower := strings.ToLower(line)
	return strings.Contains(lineLower, "warning") ||
		strings.Contains(lineLower, "warn") ||
		strings.Contains(lineLower, "caution")
}

// isHeadingLine checks if a line is a heading/separator
func isHeadingLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Markdown headings
	if strings.HasPrefix(trimmed, "#") {
		return true
	}

	// ASCII separators
	if len(trimmed) > 0 {
		allDashes := true
		allEquals := true
		for _, c := range trimmed {
			if c != '-' {
				allDashes = false
			}
			if c != '=' {
				allEquals = false
			}
		}
		if (allDashes || allEquals) && len(trimmed) > 3 {
			return true
		}
	}

	return false
}

// isCodeLine checks if a line appears to be code
func isCodeLine(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Common code patterns
	if strings.HasPrefix(trimmed, "func ") ||
		strings.HasPrefix(trimmed, "def ") ||
		strings.HasPrefix(trimmed, "class ") ||
		strings.HasPrefix(trimmed, "if ") ||
		strings.HasPrefix(trimmed, "for ") ||
		strings.HasPrefix(trimmed, "while ") ||
		strings.HasPrefix(trimmed, "return ") ||
		strings.HasPrefix(trimmed, "import ") ||
		strings.HasPrefix(trimmed, "const ") ||
		strings.HasPrefix(trimmed, "let ") ||
		strings.HasPrefix(trimmed, "var ") {
		return true
	}

	// Contains typical code symbols
	return codeSymbolRe.MatchString(trimmed)
}
