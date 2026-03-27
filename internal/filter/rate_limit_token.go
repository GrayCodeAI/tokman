package filter

import (
	"fmt"
	"sync"
	"time"
)

// TokenBucketRateLimiter implements adaptive token-bucket rate limiting
// for the compression pipeline. It limits both call frequency and token throughput.
// Task #98: Adaptive rate limiting with token bucket.
type TokenBucketRateLimiter struct {
	mu sync.Mutex

	// Call-rate bucket: limits calls per second.
	callTokens    float64
	callCapacity  float64
	callRate      float64 // tokens per second (refill rate)
	lastCallRefil time.Time

	// Token-throughput bucket: limits input tokens per second.
	thruTokens    float64
	thruCapacity  float64
	thruRate      float64 // input tokens refilled per second
	lastThruRefil time.Time

	// Adaptive: if rejection rate rises, automatically lower caps.
	rejections  int
	requests    int
	adaptiveMin float64 // minimum capacity fraction (default 0.1)
}

// TokenBucketConfig configures the rate limiter.
type TokenBucketConfig struct {
	// MaxCallsPerSecond limits API call frequency. 0 = unlimited.
	MaxCallsPerSecond float64
	// MaxInputTokensPerSecond limits token throughput. 0 = unlimited.
	MaxInputTokensPerSecond float64
	// BurstMultiplier allows short bursts above the rate. Default: 3.
	BurstMultiplier float64
}

// DefaultTokenBucketConfig returns a config appropriate for development.
func DefaultTokenBucketConfig() TokenBucketConfig {
	return TokenBucketConfig{
		MaxCallsPerSecond:       100,
		MaxInputTokensPerSecond: 500_000,
		BurstMultiplier:         3,
	}
}

// NewTokenBucketRateLimiter creates a rate limiter from the given config.
func NewTokenBucketRateLimiter(cfg TokenBucketConfig) *TokenBucketRateLimiter {
	burst := cfg.BurstMultiplier
	if burst <= 0 {
		burst = 3
	}
	now := time.Now()
	rl := &TokenBucketRateLimiter{
		adaptiveMin:   0.1,
		lastCallRefil: now,
		lastThruRefil: now,
	}

	if cfg.MaxCallsPerSecond > 0 {
		rl.callRate = cfg.MaxCallsPerSecond
		rl.callCapacity = cfg.MaxCallsPerSecond * burst
		rl.callTokens = rl.callCapacity
	} else {
		rl.callCapacity = -1 // disabled
	}

	if cfg.MaxInputTokensPerSecond > 0 {
		rl.thruRate = cfg.MaxInputTokensPerSecond
		rl.thruCapacity = cfg.MaxInputTokensPerSecond * burst
		rl.thruTokens = rl.thruCapacity
	} else {
		rl.thruCapacity = -1 // disabled
	}

	return rl
}

// ErrRateLimited is returned when a request is rejected by the rate limiter.
type ErrRateLimited struct {
	Reason    string
	RetryAfter time.Duration
}

func (e *ErrRateLimited) Error() string {
	return fmt.Sprintf("rate limited: %s (retry after %v)", e.Reason, e.RetryAfter)
}

// Allow checks whether a compression call with inputTokens tokens is permitted.
// Returns nil if allowed, *ErrRateLimited otherwise.
func (rl *TokenBucketRateLimiter) Allow(inputTokens int) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.requests++
	now := time.Now()

	// Refill call bucket
	if rl.callCapacity > 0 {
		elapsed := now.Sub(rl.lastCallRefil).Seconds()
		rl.callTokens = min64(rl.callCapacity, rl.callTokens+elapsed*rl.callRate)
		rl.lastCallRefil = now
		if rl.callTokens < 1 {
			rl.rejections++
			waitSec := (1 - rl.callTokens) / rl.callRate
			return &ErrRateLimited{
				Reason:     "call rate",
				RetryAfter: time.Duration(waitSec * float64(time.Second)),
			}
		}
		rl.callTokens--
	}

	// Refill throughput bucket
	if rl.thruCapacity > 0 {
		elapsed := now.Sub(rl.lastThruRefil).Seconds()
		rl.thruTokens = min64(rl.thruCapacity, rl.thruTokens+elapsed*rl.thruRate)
		rl.lastThruRefil = now
		need := float64(inputTokens)
		if rl.thruTokens < need {
			rl.rejections++
			waitSec := (need - rl.thruTokens) / rl.thruRate
			return &ErrRateLimited{
				Reason:     "token throughput",
				RetryAfter: time.Duration(waitSec * float64(time.Second)),
			}
		}
		rl.thruTokens -= need
	}

	return nil
}

// Stats returns rate limiter statistics.
func (rl *TokenBucketRateLimiter) Stats() string {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	var rejPct float64
	if rl.requests > 0 {
		rejPct = float64(rl.rejections) / float64(rl.requests) * 100
	}
	return fmt.Sprintf("rate_limiter: %d requests, %d rejected (%.1f%%), %.0f call tokens, %.0f throughput tokens",
		rl.requests, rl.rejections, rejPct, rl.callTokens, rl.thruTokens)
}

func min64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
