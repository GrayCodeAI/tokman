package filter

import (
	"strings"
)

// QueryIntent represents the type of agent query
type QueryIntent int

const (
	IntentUnknown QueryIntent = iota
	IntentDebug               // Finding errors, failures, crashes
	IntentReview              // Code review, diff analysis
	IntentDeploy              // Deployment status, version info
	IntentSearch              // Finding files, functions, definitions
	IntentTest                // Running/analyzing tests
	IntentBuild               // Build/compilation status
)

// QueryAwareFilter prioritizes output segments based on the agent's query intent.
// Based on "LongLLMLingua" (Jiang et al., 2024) and "ACON" (Zhang et al., 2024).
//
// Key insight: Different agent tasks need different output segments.
// A "debug" query needs errors/stack traces, not success messages.
// A "deploy" query needs status/version, not full logs.
type QueryAwareFilter struct {
	query  string
	intent QueryIntent
}

// NewQueryAwareFilter creates a new query-aware filter with an optional query.
func NewQueryAwareFilter(query ...string) *QueryAwareFilter {
	f := &QueryAwareFilter{
		query:  "",
		intent: IntentUnknown,
	}
	if len(query) > 0 && query[0] != "" {
		f.SetQuery(query[0])
	}
	return f
}

// SetQuery sets the query for context-aware filtering
func (f *QueryAwareFilter) SetQuery(query string) {
	f.query = query
	f.intent = f.classifyQuery(query)
}

// Name returns the filter name.
func (f *QueryAwareFilter) Name() string {
	return "query_aware"
}

// Apply filters output based on query relevance.
func (f *QueryAwareFilter) Apply(input string, mode Mode) (string, int) {
	// If no query set, pass through unchanged
	if f.query == "" || f.intent == IntentUnknown {
		return input, 0
	}

	// Don't process very short inputs
	if len(input) < 100 {
		return input, 0
	}

	segments := f.segmentOutput(input)
	if len(segments) < 2 {
		return input, 0
	}

	// Calculate relevance for each segment
	threshold := f.getRelevanceThreshold(mode)

	var kept []string
	for _, seg := range segments {
		relevance := f.calculateRelevance(seg, f.intent)

		if relevance >= threshold {
			kept = append(kept, seg)
		} else if relevance >= threshold*0.5 && len(kept) > 0 {
			// Keep partial - add a summary line
			summary := f.summarizeSegment(seg)
			if summary != "" {
				kept = append(kept, summary)
			}
		}
	}

	// Ensure we don't over-filter
	if len(kept) == 0 {
		// Fall back to keeping segments with highest relevance
		return f.keepTopSegments(input, segments, 5), 0
	}

	output := strings.Join(kept, "\n")
	bytesSaved := len(input) - len(output)
	tokensSaved := bytesSaved / 4

	return output, tokensSaved
}

// classifyQuery determines the intent from a natural language query
func (f *QueryAwareFilter) classifyQuery(query string) QueryIntent {
	lower := strings.ToLower(query)

	// Debug keywords (highest priority - errors are critical)
	debugKeywords := []string{"debug", "error", "fail", "crash", "bug", "fix", "broken", "exception", "panic"}
	for _, kw := range debugKeywords {
		if strings.Contains(lower, kw) {
			return IntentDebug
		}
	}

	// Test keywords (before review to catch "check test")
	testKeywords := []string{"test", "spec", "coverage", "assert", "expect"}
	for _, kw := range testKeywords {
		if strings.Contains(lower, kw) {
			return IntentTest
		}
	}

	// Build keywords (before review to catch "build")
	buildKeywords := []string{"build", "compile", "make", "cargo", "npm run", "pip install"}
	for _, kw := range buildKeywords {
		if strings.Contains(lower, kw) {
			return IntentBuild
		}
	}

	// Deploy keywords
	deployKeywords := []string{"deploy", "release", "ship", "publish", "version", "prod", "production"}
	for _, kw := range deployKeywords {
		if strings.Contains(lower, kw) {
			return IntentDeploy
		}
	}

	// Review keywords
	reviewKeywords := []string{"review", "diff", "change", "commit", "pr", "pull request", "analyze"}
	for _, kw := range reviewKeywords {
		if strings.Contains(lower, kw) {
			return IntentReview
		}
	}

	// Search keywords
	searchKeywords := []string{"find", "search", "where", "locate", "grep", "function", "class", "def"}
	for _, kw := range searchKeywords {
		if strings.Contains(lower, kw) {
			return IntentSearch
		}
	}

	return IntentUnknown
}

// segmentOutput splits output into logical segments
func (f *QueryAwareFilter) segmentOutput(input string) []string {
	lines := strings.Split(input, "\n")
	var segments []string
	var currentSegment []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Empty lines as boundaries
		if trimmed == "" && len(currentSegment) > 0 {
			segments = append(segments, strings.Join(currentSegment, "\n"))
			currentSegment = nil
			continue
		}

		// Structural markers as boundaries
		if f.isStructuralBoundary(trimmed) && len(currentSegment) > 0 {
			segments = append(segments, strings.Join(currentSegment, "\n"))
			currentSegment = nil
		}

		currentSegment = append(currentSegment, line)
	}

	if len(currentSegment) > 0 {
		segments = append(segments, strings.Join(currentSegment, "\n"))
	}

	return segments
}

// isStructuralBoundary checks for segment boundaries
func (f *QueryAwareFilter) isStructuralBoundary(line string) bool {
	// Test results
	if strings.HasPrefix(line, "test result:") || strings.HasPrefix(line, "PASS") || strings.HasPrefix(line, "FAIL") {
		return true
	}

	// Build phases
	if strings.HasPrefix(line, "Compiling") || strings.HasPrefix(line, "Building") ||
		strings.HasPrefix(line, "Finished") || strings.HasPrefix(line, "Running") {
		return true
	}

	// Errors
	if strings.HasPrefix(line, "error:") || strings.HasPrefix(line, "Error:") || strings.HasPrefix(line, "ERROR:") {
		return true
	}

	// Diffs
	if strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "@@") {
		return true
	}

	// Stack traces
	if strings.HasPrefix(line, "stack backtrace") || strings.HasPrefix(line, "Stack trace") ||
		strings.HasPrefix(line, "Traceback") {
		return true
	}

	return false
}

// calculateRelevance scores a segment based on query intent
func (f *QueryAwareFilter) calculateRelevance(segment string, intent QueryIntent) float64 {
	if strings.TrimSpace(segment) == "" {
		return 0.0
	}

	lower := strings.ToLower(segment)

	switch intent {
	case IntentDebug:
		return f.debugRelevanceScore(segment, lower)
	case IntentReview:
		return f.reviewRelevanceScore(segment, lower)
	case IntentDeploy:
		return f.deployRelevanceScore(segment, lower)
	case IntentSearch:
		return f.searchRelevanceScore(segment, lower)
	case IntentTest:
		return f.testRelevanceScore(segment, lower)
	case IntentBuild:
		return f.buildRelevanceScore(segment, lower)
	default:
		return 0.5 // Unknown intent: keep everything moderately
	}
}

// debugRelevanceScore scores segments for debugging queries
func (f *QueryAwareFilter) debugRelevanceScore(segment, lower string) float64 {
	score := 0.0

	// Critical: errors, failures, panics
	if strings.Contains(lower, "error") || strings.Contains(lower, "failed") ||
		strings.Contains(lower, "failure") || strings.Contains(lower, "panic") ||
		strings.Contains(lower, "exception") || strings.Contains(lower, "fatal") {
		score += 0.8
	}

	// Important: stack traces, assertions
	if strings.Contains(lower, "stack trace") || strings.Contains(lower, "traceback") ||
		strings.Contains(lower, "assertion") || strings.Contains(lower, "assert") {
		score += 0.6
	}

	// Useful: file references
	if f.hasFileReference(segment) {
		score += 0.4
	}

	// Less relevant: success, ok
	if strings.Contains(lower, "success") || strings.Contains(lower, "ok") ||
		strings.Contains(lower, "passed") {
		score -= 0.3
	}

	// Clamp
	if score > 1.0 {
		score = 1.0
	}
	if score < 0 {
		score = 0
	}

	return score
}

// reviewRelevanceScore scores segments for code review queries
func (f *QueryAwareFilter) reviewRelevanceScore(segment, lower string) float64 {
	score := 0.0

	// Critical: diffs, changes
	if strings.Contains(segment, "@@") || strings.Contains(segment, "diff --git") ||
		strings.Contains(segment, "---") || strings.Contains(segment, "+++") {
		score += 0.8
	}

	// Important: modified files, additions, deletions
	if strings.Contains(lower, "modified") || strings.Contains(lower, "added") ||
		strings.Contains(lower, "deleted") || strings.Contains(lower, "changed") {
		score += 0.6
	}

	// Useful: file references
	if f.hasFileReference(segment) {
		score += 0.4
	}

	// Warnings matter for review
	if strings.Contains(lower, "warning") || strings.Contains(lower, "warn") {
		score += 0.3
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < 0 {
		score = 0
	}

	return score
}

// deployRelevanceScore scores segments for deployment queries
func (f *QueryAwareFilter) deployRelevanceScore(segment, lower string) float64 {
	score := 0.0

	// Critical: success/failure status
	if strings.Contains(lower, "success") || strings.Contains(lower, "failed") ||
		strings.Contains(lower, "deployed") || strings.Contains(lower, "released") {
		score += 0.7
	}

	// Important: version, tag, build info
	if strings.Contains(lower, "version") || strings.Contains(lower, "tag") ||
		strings.Contains(lower, "build") || strings.Contains(lower, "release") {
		score += 0.5
	}

	// Useful: timestamps
	if strings.Contains(lower, "at ") || strings.Contains(lower, "time") {
		score += 0.3
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < 0 {
		score = 0
	}

	return score
}

// searchRelevanceScore scores segments for search queries
func (f *QueryAwareFilter) searchRelevanceScore(segment, lower string) float64 {
	score := 0.0

	// Critical: file paths
	if f.hasFileReference(segment) {
		score += 0.8
	}

	// Important: definitions
	if strings.Contains(lower, "function") || strings.Contains(lower, "class") ||
		strings.Contains(lower, "def ") || strings.Contains(lower, "fn ") ||
		strings.Contains(lower, "struct ") || strings.Contains(lower, "interface ") {
		score += 0.6
	}

	// Useful: imports
	if strings.Contains(lower, "import") || strings.Contains(lower, "require") ||
		strings.Contains(lower, "use ") {
		score += 0.4
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < 0 {
		score = 0
	}

	return score
}

// testRelevanceScore scores segments for test queries
func (f *QueryAwareFilter) testRelevanceScore(segment, lower string) float64 {
	score := 0.0

	// Critical: test results
	if strings.Contains(lower, "test result:") || strings.Contains(lower, "passed") ||
		strings.Contains(lower, "failed") {
		score += 0.8
	}

	// Important: test names, assertions
	if strings.Contains(lower, "test ") || strings.Contains(lower, "spec ") ||
		strings.Contains(lower, "assert") {
		score += 0.5
	}

	// Useful: coverage
	if strings.Contains(lower, "coverage") {
		score += 0.4
	}

	// Stack traces from test failures
	if strings.Contains(lower, "stack trace") || strings.Contains(lower, "traceback") {
		score += 0.6
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < 0 {
		score = 0
	}

	return score
}

// buildRelevanceScore scores segments for build queries
func (f *QueryAwareFilter) buildRelevanceScore(segment, lower string) float64 {
	score := 0.0

	// Critical: build status
	if strings.Contains(lower, "finished") || strings.Contains(lower, "complete") ||
		strings.Contains(lower, "failed") || strings.Contains(lower, "error") {
		score += 0.7
	}

	// Important: compiling phases
	if strings.Contains(lower, "compiling") || strings.Contains(lower, "building") ||
		strings.Contains(lower, "generating") {
		score += 0.4
	}

	// Warnings matter
	if strings.Contains(lower, "warning") {
		score += 0.3
	}

	if score > 1.0 {
		score = 1.0
	}
	if score < 0 {
		score = 0
	}

	return score
}

// hasFileReference checks for file:line patterns
func (f *QueryAwareFilter) hasFileReference(segment string) bool {
	extensions := []string{".go:", ".rs:", ".py:", ".js:", ".ts:", ".java:", ".c:", ".cpp:"}
	for _, ext := range extensions {
		if strings.Contains(segment, ext) {
			return true
		}
	}
	return false
}

// getRelevanceThreshold returns the threshold for keeping segments
func (f *QueryAwareFilter) getRelevanceThreshold(mode Mode) float64 {
	switch mode {
	case ModeAggressive:
		return 0.5 // Only highly relevant
	case ModeMinimal:
		return 0.3 // Moderately relevant
	default:
		return 0.2 // Keep most content
	}
}

// summarizeSegment creates a one-line summary of a segment
func (f *QueryAwareFilter) summarizeSegment(segment string) string {
	lines := strings.Split(segment, "\n")
	if len(lines) == 0 {
		return ""
	}

	// Return first non-empty line as summary
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && len(trimmed) > 10 {
			if len(trimmed) > 80 {
				return trimmed[:77] + "..."
			}
			return trimmed
		}
	}

	return ""
}

// keepTopSegments keeps the most relevant segments when filtering would remove everything
func (f *QueryAwareFilter) keepTopSegments(input string, segments []string, count int) string {
	// Score all segments
	type scoredSeg struct {
		content string
		score   float64
	}

	scored := make([]scoredSeg, len(segments))
	for i, seg := range segments {
		scored[i] = scoredSeg{content: seg, score: f.calculateRelevance(seg, f.intent)}
	}

	// Simple selection of top segments (not full sort for efficiency)
	var top []string
	for i := 0; i < count && i < len(scored); i++ {
		maxIdx := i
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[maxIdx].score {
				maxIdx = j
			}
		}
		if scored[maxIdx].score > 0 {
			top = append(top, scored[maxIdx].content)
		}
		// Swap to avoid re-selecting
		scored[i], scored[maxIdx] = scored[maxIdx], scored[i]
	}

	if len(top) == 0 {
		return input
	}

	return strings.Join(top, "\n")
}
