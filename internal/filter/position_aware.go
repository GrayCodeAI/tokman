package filter

import (
	"sort"
	"strings"
)

// PositionAwareFilter reorders output segments to optimize LLM recall.
// Based on "LongLLMLingua" (Jiang et al., 2024) - LLMs exhibit "lost in the middle"
// phenomenon where information in the middle of context is less likely to be recalled.
//
// Strategy: Place high-importance segments at beginning AND end of output.
type PositionAwareFilter struct{}

// NewPositionAwareFilter creates a new position-aware filter.
func NewPositionAwareFilter() *PositionAwareFilter {
	return &PositionAwareFilter{}
}

// Name returns the filter name.
func (f *PositionAwareFilter) Name() string {
	return "position_aware"
}

// Apply reorders segments to optimize for LLM recall.
// This filter doesn't save tokens - it improves context quality.
func (f *PositionAwareFilter) Apply(input string, mode Mode) (string, int) {
	// Don't process very short inputs
	if len(input) < 200 {
		return input, 0
	}

	segments := f.segmentOutput(input)
	if len(segments) < 4 {
		return input, 0 // Not enough segments to benefit from reordering
	}

	// Score each segment for importance
	scored := make([]scoredPositionSegment, len(segments))
	for i, seg := range segments {
		scored[i] = scoredPositionSegment{
			content:       seg,
			score:         f.importanceScore(seg),
			originalIndex: i,
		}
	}

	// Reorder segments: high importance at beginning and end
	reordered := f.reorderByImportance(scored)

	output := strings.Join(reordered, "\n\n")

	// No token savings - this is a quality improvement
	return output, 0
}

// scoredPositionSegment is a segment with importance score and original position
type scoredPositionSegment struct {
	content       string
	score         float64
	originalIndex int
}

// segmentOutput splits output into logical segments for reordering
func (f *PositionAwareFilter) segmentOutput(input string) []string {
	lines := strings.Split(input, "\n")
	var segments []string
	var currentSegment []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Empty line can be a segment boundary
		if trimmed == "" && len(currentSegment) > 0 {
			segments = append(segments, strings.Join(currentSegment, "\n"))
			currentSegment = nil
			continue
		}

		// Structural markers indicate boundaries
		if f.isStructuralBoundary(trimmed) && len(currentSegment) > 0 {
			segments = append(segments, strings.Join(currentSegment, "\n"))
			currentSegment = nil
		}

		currentSegment = append(currentSegment, line)
	}

	// Add final segment
	if len(currentSegment) > 0 {
		segments = append(segments, strings.Join(currentSegment, "\n"))
	}

	return segments
}

// isStructuralBoundary checks if a line marks a structural boundary
func (f *PositionAwareFilter) isStructuralBoundary(line string) bool {
	// Diff hunks
	if strings.HasPrefix(line, "@@") || strings.HasPrefix(line, "diff --git") {
		return true
	}

	// Test results
	if strings.HasPrefix(line, "test result:") || strings.HasPrefix(line, "PASS") || strings.HasPrefix(line, "FAIL") {
		return true
	}

	// Build phases
	if strings.HasPrefix(line, "Compiling") || strings.HasPrefix(line, "Building") ||
		strings.HasPrefix(line, "Finished") || strings.HasPrefix(line, "Running") {
		return true
	}

	// Error blocks often start with "error:" or "Error:"
	if strings.HasPrefix(line, "error:") || strings.HasPrefix(line, "Error:") ||
		strings.HasPrefix(line, "ERROR:") {
		return true
	}

	// Stack traces
	if strings.HasPrefix(line, "stack backtrace") || strings.HasPrefix(line, "Stack trace") ||
		strings.HasPrefix(line, "Traceback") {
		return true
	}

	return false
}

// importanceScore calculates the importance of a segment
// Higher score = more critical for agent tasks
func (f *PositionAwareFilter) importanceScore(segment string) float64 {
	if strings.TrimSpace(segment) == "" {
		return 0.0
	}

	lower := strings.ToLower(segment)
	score := 0.0

	// Critical importance (0.4-1.0): Errors, failures, crashes
	if strings.Contains(lower, "error") ||
		strings.Contains(lower, "failed") ||
		strings.Contains(lower, "failure") ||
		strings.Contains(lower, "fatal") ||
		strings.Contains(lower, "panic") ||
		strings.Contains(lower, "exception") {
		score += 0.5
	}

	// High importance (0.3-0.6): Test failures, stack traces
	if strings.Contains(lower, "stack trace") ||
		strings.Contains(lower, "traceback") ||
		strings.Contains(lower, "assertion") ||
		strings.Contains(lower, "test result:") && strings.Contains(lower, "fail") {
		score += 0.4
	}

	// File references (0.2-0.4): Actionable locations
	if f.hasFileReference(segment) {
		score += 0.2
	}

	// Medium importance (0.1-0.3): Warnings, changes
	if strings.Contains(lower, "warning") ||
		strings.Contains(lower, "warn") ||
		strings.Contains(lower, "deprecated") {
		score += 0.1
	}

	// Diff hunks (0.2-0.4): Critical for code review
	if strings.Contains(segment, "@@") ||
		strings.Contains(segment, "diff --git") ||
		strings.Contains(segment, "---") ||
		strings.Contains(segment, "+++") {
		score += 0.2
	}

	// Low importance (negative): Success messages, noise
	if strings.Contains(lower, "success") ||
		strings.Contains(lower, "ok") ||
		strings.Contains(lower, "passed") && !strings.Contains(lower, "failed") {
		score -= 0.1
	}

	// Normalize to [0, 1]
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}

	return score
}

// hasFileReference checks if segment contains file:line references
func (f *PositionAwareFilter) hasFileReference(segment string) bool {
	// Common patterns: file.go:42, file.rs:10:5, file.py:100
	extensions := []string{".go:", ".rs:", ".py:", ".js:", ".ts:", ".java:", ".c:", ".cpp:"}
	for _, ext := range extensions {
		if strings.Contains(segment, ext) {
			return true
		}
	}
	return false
}

// reorderByImportance places high-importance segments at beginning and end
func (f *PositionAwareFilter) reorderByImportance(segments []scoredPositionSegment) []string {
	if len(segments) == 0 {
		return nil
	}

	// Sort by score (descending)
	sorted := make([]scoredPositionSegment, len(segments))
	copy(sorted, segments)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].score > sorted[j].score
	})

	n := len(sorted)
	var beginning, middle, end []string

	// Top 20% → beginning AND end (duplicated for emphasis)
	topCount := max(1, n/5)
	for i := 0; i < topCount && i < n; i++ {
		beginning = append(beginning, sorted[i].content)
	}

	// Middle 60% → middle (lower importance)
	middleStart := topCount
	middleEnd := n - topCount
	if middleEnd <= middleStart {
		middleEnd = n
	}

	// Preserve original order for middle segments (less disorienting)
	middleMap := make(map[int]string)
	for _, seg := range segments {
		middleMap[seg.originalIndex] = seg.content
	}

	// Collect middle segments (those not in top 20%)
	topSet := make(map[int]bool)
	for i := 0; i < topCount; i++ {
		// Find original index of this sorted segment
		for j, seg := range segments {
			if seg.content == sorted[i].content {
				topSet[j] = true
				break
			}
		}
	}

	for i := 0; i < n; i++ {
		if !topSet[i] {
			middle = append(middle, segments[i].content)
		}
	}

	// Top 20% → end (repeat most important)
	for i := 0; i < topCount && i < n; i++ {
		end = append(end, sorted[i].content)
	}

	// Combine: beginning + middle + end
	var result []string
	result = append(result, beginning...)
	result = append(result, middle...)
	result = append(result, end...)

	return result
}

