// Package sentiment provides sentiment analysis for content
package sentiment

import (
	"strings"
)

// Analyzer analyzes sentiment of text
type Analyzer struct {
	positiveWords map[string]bool
	negativeWords map[string]bool
}

// NewAnalyzer creates a new sentiment analyzer
func NewAnalyzer() *Analyzer {
	return &Analyzer{
		positiveWords: map[string]bool{
			"good": true, "great": true, "excellent": true, "amazing": true,
			"awesome": true, "perfect": true, "wonderful": true, "fantastic": true,
			"success": true, "passed": true, "complete": true, "done": true,
			"fast": true, "efficient": true, "optimized": true, "improved": true,
		},
		negativeWords: map[string]bool{
			"bad": true, "terrible": true, "awful": true, "horrible": true,
			"fail": true, "failed": true, "error": true, "broken": true,
			"slow": true, "inefficient": true, "regression": true, "bug": true,
			"crash": true, "crashed": true, "exception": true, "fatal": true,
		},
	}
}

// Result represents sentiment analysis result
type Result struct {
	Score         float64
	Sentiment     string
	PositiveCount int
	NegativeCount int
	Confidence    float64
}

// Analyze analyzes text sentiment
func (a *Analyzer) Analyze(text string) *Result {
	text = strings.ToLower(text)
	words := strings.Fields(text)

	positiveCount := 0
	negativeCount := 0

	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if a.positiveWords[word] {
			positiveCount++
		}
		if a.negativeWords[word] {
			negativeCount++
		}
	}

	total := positiveCount + negativeCount
	if total == 0 {
		return &Result{
			Score:      0,
			Sentiment:  "neutral",
			Confidence: 0,
		}
	}

	score := float64(positiveCount-negativeCount) / float64(total)
	sentiment := "neutral"
	if score > 0.2 {
		sentiment = "positive"
	} else if score < -0.2 {
		sentiment = "negative"
	}

	return &Result{
		Score:         score,
		Sentiment:     sentiment,
		PositiveCount: positiveCount,
		NegativeCount: negativeCount,
		Confidence:    float64(total) / float64(len(words)),
	}
}
