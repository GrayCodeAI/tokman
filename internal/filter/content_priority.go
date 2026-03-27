package filter

import (
	"math"
	"strings"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// ContentPriority holds a scored content section for ranking.
type ContentPriority struct {
	Content    string
	Score      float64  // 0.0 – 1.0, higher = more important
	Tokens     int
	Reasons    []string // why this scored high
}

// PriorityScoringConfig configures the content priority scorer.
type PriorityScoringConfig struct {
	// ErrorWeight weights error-related content higher.
	ErrorWeight float64
	// RecencyWeight weights content appearing later in the input.
	RecencyWeight float64
	// DensityWeight weights information-dense content.
	DensityWeight float64
	// StructureWeight weights code/structure over prose.
	StructureWeight float64
	// MinSectionTokens is the minimum tokens for a section to be considered.
	MinSectionTokens int
}

// DefaultPriorityScoringConfig returns balanced defaults.
func DefaultPriorityScoringConfig() PriorityScoringConfig {
	return PriorityScoringConfig{
		ErrorWeight:      2.0,
		RecencyWeight:    1.5,
		DensityWeight:    1.2,
		StructureWeight:  1.3,
		MinSectionTokens: 5,
	}
}

// ContentPriorityScorer ranks content sections by importance.
type ContentPriorityScorer struct {
	config PriorityScoringConfig
}

// NewContentPriorityScorer creates a scorer with default config.
func NewContentPriorityScorer() *ContentPriorityScorer {
	return &ContentPriorityScorer{config: DefaultPriorityScoringConfig()}
}

// NewContentPriorityScorerWithConfig creates a scorer with custom config.
func NewContentPriorityScorerWithConfig(cfg PriorityScoringConfig) *ContentPriorityScorer {
	return &ContentPriorityScorer{config: cfg}
}

// Score scores all sections split from input by blank lines.
func (s *ContentPriorityScorer) Score(input string) []ContentPriority {
	sections := strings.Split(input, "\n\n")
	total := len(sections)
	if total == 0 {
		return nil
	}

	result := make([]ContentPriority, 0, total)
	for i, sec := range sections {
		sec = strings.TrimSpace(sec)
		if sec == "" {
			continue
		}
		toks := core.EstimateTokens(sec)
		if toks < s.config.MinSectionTokens {
			continue
		}

		score, reasons := s.scoreSection(sec, i, total)
		result = append(result, ContentPriority{
			Content: sec,
			Score:   score,
			Tokens:  toks,
			Reasons: reasons,
		})
	}
	return result
}

// scoreSection computes a priority score for a single section.
func (s *ContentPriorityScorer) scoreSection(sec string, idx, total int) (float64, []string) {
	var score float64
	var reasons []string

	lower := strings.ToLower(sec)
	words := strings.Fields(sec)

	// Error/warning content
	for _, kw := range []string{"error", "fail", "panic", "exception", "fatal", "critical", "warn"} {
		if strings.Contains(lower, kw) {
			score += s.config.ErrorWeight * 0.3
			reasons = append(reasons, "error-content")
			break
		}
	}

	// Recency bias (later sections score higher)
	recency := float64(idx+1) / float64(total)
	score += s.config.RecencyWeight * recency * 0.3
	if recency > 0.7 {
		reasons = append(reasons, "recent")
	}

	// Information density (unique words / total words)
	if len(words) > 0 {
		uniqueWords := make(map[string]struct{}, len(words))
		for _, w := range words {
			uniqueWords[strings.ToLower(w)] = struct{}{}
		}
		density := float64(len(uniqueWords)) / float64(len(words))
		score += s.config.DensityWeight * density * 0.25
		if density > 0.7 {
			reasons = append(reasons, "high-density")
		}
	}

	// Structure bonus (code, function signatures, etc.)
	structureMarkers := []string{
		"func ", "def ", "class ", "function ", "struct ",
		":=", "=>", "->", "```", "    ", "\t",
	}
	for _, m := range structureMarkers {
		if strings.Contains(sec, m) {
			score += s.config.StructureWeight * 0.1
			reasons = append(reasons, "structured")
			break
		}
	}

	// Cap at 1.0
	if score > 1.0 {
		score = 1.0
	}

	return score, reasons
}

// TopN returns the top-N highest priority sections.
func (s *ContentPriorityScorer) TopN(input string, n int) []ContentPriority {
	all := s.Score(input)
	// Sort descending by score
	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].Score > all[i].Score {
				all[i], all[j] = all[j], all[i]
			}
		}
	}
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

// FreshnessScorer scores content based on recency signals embedded in the text.
// Task #170: content freshness scoring.
type FreshnessScorer struct{}

// NewFreshnessScorer creates a freshness scorer.
func NewFreshnessScorer() *FreshnessScorer { return &FreshnessScorer{} }

// FreshnessResult holds freshness analysis.
type FreshnessResult struct {
	// Score: 0.0 (stale) to 1.0 (very fresh)
	Score       float64
	OldestDate  time.Time
	NewestDate  time.Time
	HasDates    bool
	AgeCategory string // "fresh" (<1h), "recent" (<1d), "old" (>7d), "unknown"
}

// Analyze estimates content freshness from embedded timestamps.
func (f *FreshnessScorer) Analyze(input string) FreshnessResult {
	result := FreshnessResult{Score: 0.5, AgeCategory: "unknown"}

	// Try to extract dates from timestamps
	timestamps := extractTimestamps(input)
	if len(timestamps) == 0 {
		return result
	}

	result.HasDates = true
	result.OldestDate = timestamps[0]
	result.NewestDate = timestamps[0]
	for _, ts := range timestamps[1:] {
		if ts.Before(result.OldestDate) {
			result.OldestDate = ts
		}
		if ts.After(result.NewestDate) {
			result.NewestDate = ts
		}
	}

	// Score based on newest timestamp age
	age := time.Since(result.NewestDate)
	switch {
	case age < time.Hour:
		result.Score = 1.0
		result.AgeCategory = "fresh"
	case age < 24*time.Hour:
		result.Score = 0.8
		result.AgeCategory = "recent"
	case age < 7*24*time.Hour:
		result.Score = 0.5
		result.AgeCategory = "week-old"
	case age < 30*24*time.Hour:
		result.Score = 0.3
		result.AgeCategory = "month-old"
	default:
		result.Score = 0.1
		result.AgeCategory = "old"
	}

	// Penalty if content spans a long time range
	span := result.NewestDate.Sub(result.OldestDate)
	if span > 30*24*time.Hour {
		result.Score *= 0.8
	}

	return result
}

// ScoreForCompression returns a compression priority based on freshness.
// Fresher content should be preserved; older content can be compressed more.
func (f *FreshnessScorer) ScoreForCompression(input string) float64 {
	res := f.Analyze(input)
	// Fresh content → low compression priority (preserve it)
	// Old content → high compression priority (compress aggressively)
	return 1.0 - res.Score
}

// extractTimestamps finds ISO-like timestamps in the text.
func extractTimestamps(input string) []time.Time {
	re := logTimestampFullRe // reuse from smart_log.go
	var times []time.Time
	for _, match := range re.FindAllString(input, 20) {
		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
		}
		for _, layout := range layouts {
			t, err := time.Parse(layout, match)
			if err == nil {
				times = append(times, t)
				break
			}
		}
	}
	return times
}

// clampPriority clamps a float to [0, 1].
func clampPriority(v float64) float64 {
	return math.Max(0, math.Min(1, v))
}
