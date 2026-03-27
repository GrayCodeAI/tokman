package filter

import (
	"fmt"
	"strings"
)

// HeatmapEntry represents a single line's importance in the heatmap.
// Task #194: Content importance heatmap visualization.
type HeatmapEntry struct {
	Line     int
	Content  string
	Score    float64
	Category string // "high", "medium", "low", "removed"
}

// Heatmap holds all entries for an original vs. compressed comparison.
type Heatmap struct {
	Entries []HeatmapEntry
}

// ComputeHeatmap compares original and compressed content line-by-line,
// assigning scores and categories to each line of the original.
//
// Lines that survive into compressed:
//   - First quartile  → "high"   (score ≈ 0.75–1.0, boosted by length)
//   - Last quartile   → "medium" (score ≈ 0.50–0.75)
//   - Middle lines    → "medium" (score ≈ 0.40–0.65)
//
// Lines removed from original: category = "removed", score = 0.
func ComputeHeatmap(original, compressed string) *Heatmap {
	origLines := splitLines(original)
	compLines := splitLines(compressed)

	// Build a set of compressed lines for fast lookup (trimmed content).
	compSet := make(map[string]bool, len(compLines))
	for _, l := range compLines {
		compSet[strings.TrimSpace(l)] = true
	}

	total := len(compLines)
	if total == 0 {
		total = 1 // avoid division by zero
	}

	// Map compressed line → its index (first occurrence) for position scoring.
	compIndex := make(map[string]int, len(compLines))
	for i, l := range compLines {
		trimmed := strings.TrimSpace(l)
		if _, exists := compIndex[trimmed]; !exists {
			compIndex[trimmed] = i
		}
	}

	entries := make([]HeatmapEntry, 0, len(origLines))

	for i, line := range origLines {
		trimmed := strings.TrimSpace(line)
		entry := HeatmapEntry{
			Line:    i + 1,
			Content: line,
		}

		if trimmed == "" || !compSet[trimmed] {
			// Line was removed during compression.
			entry.Score = 0
			entry.Category = "removed"
		} else {
			// Line survived — score by its position in compressed output.
			pos := compIndex[trimmed]
			posRatio := float64(pos) / float64(total)

			// Base score from position.
			var baseScore float64
			switch {
			case posRatio <= 0.25:
				// First quartile → high importance.
				baseScore = 0.75 + (0.25 * (1.0 - posRatio/0.25))
			case posRatio >= 0.75:
				// Last quartile → medium.
				baseScore = 0.50 + 0.15*(1.0-(posRatio-0.75)/0.25)
			default:
				// Middle → medium.
				baseScore = 0.40 + 0.25*(1.0-((posRatio-0.25)/0.50))
			}

			// Length bonus: longer lines carry more information.
			lineLen := len(trimmed)
			lengthBonus := 0.0
			if lineLen > 80 {
				lengthBonus = 0.10
			} else if lineLen > 40 {
				lengthBonus = 0.05
			}

			score := baseScore + lengthBonus
			if score > 1.0 {
				score = 1.0
			}

			entry.Score = score
			switch {
			case score >= 0.75:
				entry.Category = "high"
			case score >= 0.45:
				entry.Category = "medium"
			default:
				entry.Category = "low"
			}
		}

		entries = append(entries, entry)
	}

	return &Heatmap{Entries: entries}
}

// splitLines splits text into lines, preserving empty lines.
func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// ANSI escape codes for terminal coloring.
const (
	ansiReset         = "\033[0m"
	ansiGreen         = "\033[32m"
	ansiYellow        = "\033[33m"
	ansiDim           = "\033[2m"
	ansiStrikethrough = "\033[9m"
)

// FormatTerminal returns an ANSI color-coded representation of the heatmap.
//   - high     → green
//   - medium   → yellow
//   - low      → dim
//   - removed  → strikethrough (dim)
func (h *Heatmap) FormatTerminal() string {
	var sb strings.Builder
	for _, e := range h.Entries {
		var prefix, suffix string
		switch e.Category {
		case "high":
			prefix = ansiGreen
			suffix = ansiReset
		case "medium":
			prefix = ansiYellow
			suffix = ansiReset
		case "low":
			prefix = ansiDim
			suffix = ansiReset
		case "removed":
			prefix = ansiDim + ansiStrikethrough
			suffix = ansiReset
		}
		sb.WriteString(fmt.Sprintf("%s%4d: %s%s\n", prefix, e.Line, e.Content, suffix))
	}
	return sb.String()
}

// FormatCSV returns a CSV representation of the heatmap.
// Columns: line,score,category
func (h *Heatmap) FormatCSV() string {
	var sb strings.Builder
	sb.WriteString("line,score,category\n")
	for _, e := range h.Entries {
		sb.WriteString(fmt.Sprintf("%d,%.4f,%s\n", e.Line, e.Score, e.Category))
	}
	return sb.String()
}
