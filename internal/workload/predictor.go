// Package workload provides workload prediction
package workload

import (
	"math"
	"time"
)

// Predictor predicts workload patterns
type Predictor struct {
	history []WorkloadData
}

// WorkloadData represents historical workload data
type WorkloadData struct {
	Timestamp time.Time
	Requests  int
	Tokens    int
	Cost      float64
	Duration  time.Duration
}

// NewPredictor creates a new workload predictor
func NewPredictor() *Predictor {
	return &Predictor{
		history: make([]WorkloadData, 0),
	}
}

// AddData adds workload data
func (p *Predictor) AddData(data WorkloadData) {
	p.history = append(p.history, data)
}

// Predict predicts future workload
func (p *Predictor) Predict(hoursAhead int) WorkloadPrediction {
	if len(p.history) < 2 {
		return WorkloadPrediction{}
	}

	// Calculate trend
	totalRequests := 0
	totalTokens := 0
	totalCost := 0.0

	for _, d := range p.history {
		totalRequests += d.Requests
		totalTokens += d.Tokens
		totalCost += d.Cost
	}

	avgRequests := float64(totalRequests) / float64(len(p.history))
	avgTokens := float64(totalTokens) / float64(len(p.history))
	avgCost := totalCost / float64(len(p.history))

	// Calculate growth rate
	if len(p.history) >= 2 {
		first := p.history[0]
		last := p.history[len(p.history)-1]

		if first.Requests > 0 {
			growthRate := float64(last.Requests-first.Requests) / float64(first.Requests)
			avgRequests *= (1 + growthRate*float64(hoursAhead)/24)
		}
		if first.Tokens > 0 {
			growthRate := float64(last.Tokens-first.Tokens) / float64(first.Tokens)
			avgTokens *= (1 + growthRate*float64(hoursAhead)/24)
		}
		if first.Cost > 0 {
			growthRate := (last.Cost - first.Cost) / first.Cost
			avgCost *= (1 + growthRate*float64(hoursAhead)/24)
		}
	}

	return WorkloadPrediction{
		PredictedRequests: int(math.Ceil(avgRequests)),
		PredictedTokens:   int(math.Ceil(avgTokens)),
		PredictedCost:     avgCost,
		Confidence:        p.calculateConfidence(),
		TimeRange:         time.Duration(hoursAhead) * time.Hour,
	}
}

// WorkloadPrediction represents a workload prediction
type WorkloadPrediction struct {
	PredictedRequests int
	PredictedTokens   int
	PredictedCost     float64
	Confidence        float64
	TimeRange         time.Duration
}

func (p *Predictor) calculateConfidence() float64 {
	if len(p.history) < 5 {
		return float64(len(p.history)) / 5.0 * 0.5
	}
	return 0.5 + math.Min(float64(len(p.history)-5)/50.0, 0.5)
}

// CacheOptimizer provides intelligent caching
type CacheOptimizer struct {
	hitRate  float64
	missRate float64
	totalOps int
}

// NewCacheOptimizer creates a cache optimizer
func NewCacheOptimizer() *CacheOptimizer {
	return &CacheOptimizer{}
}

// RecordHit records a cache hit
func (co *CacheOptimizer) RecordHit() {
	co.totalOps++
	co.hitRate = float64(co.totalOps-int(co.missRate*float64(co.totalOps))) / float64(co.totalOps)
}

// RecordMiss records a cache miss
func (co *CacheOptimizer) RecordMiss() {
	co.totalOps++
	co.missRate = float64(co.totalOps-int(co.hitRate*float64(co.totalOps))) / float64(co.totalOps)
}

// GetHitRate returns the cache hit rate
func (co *CacheOptimizer) GetHitRate() float64 {
	return co.hitRate
}

// ShouldCache determines if content should be cached
func (co *CacheOptimizer) ShouldCache(contentSize int, accessFrequency float64) bool {
	// Cache if content is large enough and accessed frequently
	return contentSize > 1024 && accessFrequency > 0.5
}

// OptimizeCacheSize recommends optimal cache size
func (co *CacheOptimizer) OptimizeCacheSize(currentSize int, hitRate float64) int {
	if hitRate < 0.5 {
		return currentSize * 2 // Double cache size
	} else if hitRate > 0.9 {
		return currentSize / 2 // Halve cache size
	}
	return currentSize
}
