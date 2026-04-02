// Package intelligent provides intelligent filter selection
package intelligent

import (
	"math"
	"strings"
)

// FilterSelector provides intelligent filter selection
type FilterSelector struct {
	history []FilterUsage
}

// FilterUsage represents filter usage data
type FilterUsage struct {
	FilterName       string
	ContentType      string
	InputSize        int
	OutputSize       int
	CompressionRatio float64
	Duration         float64
	Timestamp        string
}

// NewFilterSelector creates a new filter selector
func NewFilterSelector() *FilterSelector {
	return &FilterSelector{
		history: make([]FilterUsage, 0),
	}
}

// RecordUsage records filter usage
func (fs *FilterSelector) RecordUsage(usage FilterUsage) {
	fs.history = append(fs.history, usage)
}

// RecommendFilters recommends filters based on content
func (fs *FilterSelector) RecommendFilters(contentType string, inputSize int) []FilterRecommendation {
	recommendations := make([]FilterRecommendation, 0)

	// Analyze history for this content type
	history := fs.getHistoryByType(contentType)

	if len(history) == 0 {
		// Use default recommendations
		return fs.getDefaultRecommendations(contentType, inputSize)
	}

	// Calculate best filters based on history
	filterScores := make(map[string]float64)
	filterCounts := make(map[string]int)

	for _, h := range history {
		filterScores[h.FilterName] += h.CompressionRatio
		filterCounts[h.FilterName]++
	}

	// Calculate average scores
	for name, score := range filterScores {
		filterScores[name] = score / float64(filterCounts[name])
	}

	// Sort by score
	for name, score := range filterScores {
		recommendations = append(recommendations, FilterRecommendation{
			FilterName: name,
			Score:      score,
			Confidence: fs.calculateConfidence(filterCounts[name]),
		})
	}

	// Sort recommendations
	for i := 0; i < len(recommendations); i++ {
		for j := i + 1; j < len(recommendations); j++ {
			if recommendations[j].Score > recommendations[i].Score {
				recommendations[i], recommendations[j] = recommendations[j], recommendations[i]
			}
		}
	}

	return recommendations
}

// FilterRecommendation represents a filter recommendation
type FilterRecommendation struct {
	FilterName string
	Score      float64
	Confidence float64
	Reason     string
}

func (fs *FilterSelector) getHistoryByType(contentType string) []FilterUsage {
	result := make([]FilterUsage, 0)
	for _, h := range fs.history {
		if strings.Contains(h.ContentType, contentType) {
			result = append(result, h)
		}
	}
	return result
}

func (fs *FilterSelector) getDefaultRecommendations(contentType string, inputSize int) []FilterRecommendation {
	switch {
	case strings.Contains(contentType, "code"):
		return []FilterRecommendation{
			{FilterName: "entropy", Score: 0.85, Confidence: 0.7},
			{FilterName: "ast_preserve", Score: 0.90, Confidence: 0.8},
			{FilterName: "compaction", Score: 0.75, Confidence: 0.6},
		}
	case strings.Contains(contentType, "log"):
		return []FilterRecommendation{
			{FilterName: "logdedup", Score: 0.90, Confidence: 0.8},
			{FilterName: "compaction", Score: 0.80, Confidence: 0.7},
			{FilterName: "entropy", Score: 0.70, Confidence: 0.6},
		}
	default:
		return []FilterRecommendation{
			{FilterName: "entropy", Score: 0.75, Confidence: 0.6},
			{FilterName: "compaction", Score: 0.70, Confidence: 0.5},
		}
	}
}

func (fs *FilterSelector) calculateConfidence(count int) float64 {
	if count < 5 {
		return float64(count) / 5.0 * 0.5
	}
	return 0.5 + math.Min(float64(count-5)/20.0, 0.5)
}

// AdaptiveCompression provides adaptive compression
type AdaptiveCompression struct {
	selector *FilterSelector
}

// NewAdaptiveCompression creates adaptive compression
func NewAdaptiveCompression() *AdaptiveCompression {
	return &AdaptiveCompression{
		selector: NewFilterSelector(),
	}
}

// SelectFilters selects filters adaptively
func (ac *AdaptiveCompression) SelectFilters(contentType string, inputSize int) []string {
	recommendations := ac.selector.RecommendFilters(contentType, inputSize)

	filters := make([]string, 0)
	for _, rec := range recommendations {
		if rec.Score > 0.7 {
			filters = append(filters, rec.FilterName)
		}
	}

	if len(filters) == 0 {
		filters = append(filters, "entropy", "compaction")
	}

	return filters
}

// ContentClassifier classifies content type
type ContentClassifier struct{}

// NewContentClassifier creates a content classifier
func NewContentClassifier() *ContentClassifier {
	return &ContentClassifier{}
}

// Classify classifies content based on samples
func (cc *ContentClassifier) Classify(content string) string {
	content = strings.ToLower(content)

	indicators := map[string][]string{
		"code":     {"func ", "class ", "import ", "package ", "def ", "const ", "var ", "let "},
		"log":      {"INFO", "ERROR", "WARN", "DEBUG", "TRACE", "FATAL"},
		"json":     {"{", "}", "[", "]", "\""},
		"markdown": {"#", "*", "_", "`", "["},
		"html":     {"<html", "<div", "<span", "<p>", "<h1"},
	}

	scores := make(map[string]int)
	for contentType, keywords := range indicators {
		for _, keyword := range keywords {
			if strings.Contains(content, keyword) {
				scores[contentType]++
			}
		}
	}

	bestType := "text"
	bestScore := 0
	for contentType, score := range scores {
		if score > bestScore {
			bestType = contentType
			bestScore = score
		}
	}

	return bestType
}

// UsageAnalytics provides usage analytics
type UsageAnalytics struct {
	data []FilterUsage
}

// NewUsageAnalytics creates usage analytics
func NewUsageAnalytics() *UsageAnalytics {
	return &UsageAnalytics{
		data: make([]FilterUsage, 0),
	}
}

// AddData adds usage data
func (ua *UsageAnalytics) AddData(data FilterUsage) {
	ua.data = append(ua.data, data)
}

// GetTopFilters returns top performing filters
func (ua *UsageAnalytics) GetTopFilters(count int) []FilterStats {
	filterStats := make(map[string]*FilterStats)

	for _, d := range ua.data {
		if _, ok := filterStats[d.FilterName]; !ok {
			filterStats[d.FilterName] = &FilterStats{
				Name: d.FilterName,
			}
		}

		stats := filterStats[d.FilterName]
		stats.TotalUsage++
		stats.TotalCompression += d.CompressionRatio
		stats.TotalDuration += d.Duration
	}

	results := make([]FilterStats, 0, len(filterStats))
	for _, stats := range filterStats {
		if stats.TotalUsage > 0 {
			stats.AvgCompression = stats.TotalCompression / float64(stats.TotalUsage)
			stats.AvgDuration = stats.TotalDuration / float64(stats.TotalUsage)
		}
		results = append(results, *stats)
	}

	// Sort by avg compression
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].AvgCompression > results[i].AvgCompression {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if count > len(results) {
		count = len(results)
	}

	return results[:count]
}

// FilterStats represents filter statistics
type FilterStats struct {
	Name             string
	TotalUsage       int
	AvgCompression   float64
	AvgDuration      float64
	TotalCompression float64
	TotalDuration    float64
}

// PredictCompression predicts compression ratio for content
func (ua *UsageAnalytics) PredictCompression(contentType string, inputSize int) float64 {
	// Simple prediction based on content type
	switch contentType {
	case "code":
		return 0.65
	case "log":
		return 0.75
	case "json":
		return 0.70
	default:
		return 0.60
	}
}

// SmartDefaults provides smart default configurations
type SmartDefaults struct{}

// GetDefaults returns smart defaults based on context
func (sd *SmartDefaults) GetDefaults(context map[string]interface{}) map[string]interface{} {
	defaults := map[string]interface{}{
		"compression_mode":   "balanced",
		"max_tokens":         2000,
		"preserve_structure": true,
		"remove_comments":    false,
		"minify":             false,
	}

	// Adjust based on context
	if mode, ok := context["mode"].(string); ok {
		switch mode {
		case "fast":
			defaults["compression_mode"] = "fast"
			defaults["max_tokens"] = 1000
		case "aggressive":
			defaults["compression_mode"] = "aggressive"
			defaults["max_tokens"] = 500
		}
	}

	if contentType, ok := context["content_type"].(string); ok {
		if contentType == "code" {
			defaults["preserve_structure"] = true
			defaults["remove_comments"] = true
		} else if contentType == "log" {
			defaults["minify"] = true
		}
	}

	return defaults
}
