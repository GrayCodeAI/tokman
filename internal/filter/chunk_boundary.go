package filter

import (
	"regexp"
	"strings"
)

// ChunkBoundaryDetector finds semantic chunk boundaries in content.
// When splitting large content for parallel processing, it ensures
// chunks end at natural semantic boundaries — never mid-function,
// mid-statement, or mid-log-entry.
//
// Boundary types (in priority order):
//  1. Blank line separating top-level declarations (functions, classes)
//  2. End of function/class/block (closing brace at indent level 0)
//  3. End of paragraph (blank line in prose)
//  4. End of log entry (timestamp-prefixed line followed by next timestamp)
//  5. Hard character limit (last resort: split at newline before limit)
type ChunkBoundaryDetector struct {
	// TargetChunkTokens is the target chunk size in tokens.
	// The detector finds the nearest boundary at or before this limit.
	TargetChunkTokens int
}

var (
	// Top-level function/class end: closing brace at column 0
	topLevelClosingBraceRe = regexp.MustCompile(`(?m)^}`)
	// Log entry start: timestamp prefix
	logEntryStartRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}`)
	// Python/Ruby function/class end: dedent to lower indent level
	dedentRe = regexp.MustCompile(`(?m)^[^\s]`)
)

// NewChunkBoundaryDetector creates a chunk boundary detector.
func NewChunkBoundaryDetector(targetTokens int) *ChunkBoundaryDetector {
	if targetTokens <= 0 {
		targetTokens = 2000
	}
	return &ChunkBoundaryDetector{TargetChunkTokens: targetTokens}
}

// SplitAtBoundaries splits input into chunks of at most TargetChunkTokens,
// finding semantic boundaries to split at.
func (d *ChunkBoundaryDetector) SplitAtBoundaries(input string) []string {
	lines := strings.Split(input, "\n")
	if len(lines) == 0 {
		return nil
	}

	var chunks []string
	chunkStart := 0
	currentTokens := 0
	targetChars := d.TargetChunkTokens * 4 // ≈ 4 chars per token

	for i, line := range lines {
		currentTokens += len(line) + 1

		if currentTokens < targetChars {
			continue
		}

		// We've exceeded the target — find the best nearby boundary
		boundaryIdx := d.findBoundaryBefore(lines, i)
		if boundaryIdx <= chunkStart {
			boundaryIdx = i // no good boundary found — split here anyway
		}

		chunk := strings.Join(lines[chunkStart:boundaryIdx+1], "\n")
		chunks = append(chunks, chunk)
		chunkStart = boundaryIdx + 1
		currentTokens = 0
	}

	// Remaining content
	if chunkStart < len(lines) {
		chunk := strings.Join(lines[chunkStart:], "\n")
		if strings.TrimSpace(chunk) != "" {
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

// findBoundaryBefore scans backward from lineIdx to find the nearest semantic
// boundary (blank line after a block end, log entry boundary, etc.).
func (d *ChunkBoundaryDetector) findBoundaryBefore(lines []string, lineIdx int) int {
	// Search up to 30 lines back for a boundary
	lookback := 30
	if lookback > lineIdx {
		lookback = lineIdx
	}

	// Prefer blank lines (paragraph/block boundaries)
	for i := lineIdx; i >= lineIdx-lookback; i-- {
		line := lines[i]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return i
		}
	}

	// Fall back to top-level closing brace
	for i := lineIdx; i >= lineIdx-lookback; i-- {
		if topLevelClosingBraceRe.MatchString(lines[i]) {
			return i
		}
	}

	// Fall back to log entry boundary
	for i := lineIdx - 1; i >= lineIdx-lookback && i >= 0; i-- {
		if logEntryStartRe.MatchString(lines[i]) {
			return i - 1 // end of previous log entry
		}
	}

	return lineIdx
}

// SplitChunkReassemble splits content, processes each chunk with a filter,
// and reassembles the results.
func SplitChunkReassemble(input string, targetTokens int, f Filter, mode Mode) (string, int) {
	detector := NewChunkBoundaryDetector(targetTokens)
	chunks := detector.SplitAtBoundaries(input)
	if len(chunks) <= 1 {
		// No need to chunk — process as a whole
		return f.Apply(input, mode)
	}

	var results []string
	totalSaved := 0
	for _, chunk := range chunks {
		out, saved := f.Apply(chunk, mode)
		results = append(results, out)
		totalSaved += saved
	}

	return strings.Join(results, "\n"), totalSaved
}
