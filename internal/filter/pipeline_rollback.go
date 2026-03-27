package filter

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/GrayCodeAI/tokman/internal/core"
)

// RollbackPolicy defines when the pipeline should roll back to a prior output.
// Task #195: Pipeline rollback on quality regression.
type RollbackPolicy struct {
	// MaxReductionPct is the upper bound on token reduction.
	// If compression reduces tokens by MORE than this, the output is probably
	// over-compressed (lossy). Default: 90 (90%).
	MaxReductionPct float64

	// MinQualityScore is the minimum acceptable quality score [0,1].
	// Requires QualityScorer to be set. Default: 0 (disabled).
	MinQualityScore float64

	// MaxBytesGrowth allows rollback if the output is larger than the input
	// by more than this fraction. Default: 0.05 (5% growth triggers rollback).
	MaxBytesGrowth float64

	// RequireContentPreservation checks that key content tokens survive compression.
	RequireContentPreservation bool

	// MinPreservationRatio: what fraction of "important" words must survive.
	// Default: 0.5 (50%).
	MinPreservationRatio float64
}

// DefaultRollbackPolicy returns a conservative rollback policy.
func DefaultRollbackPolicy() RollbackPolicy {
	return RollbackPolicy{
		MaxReductionPct:            90,
		MaxBytesGrowth:             0.05,
		RequireContentPreservation: true,
		MinPreservationRatio:       0.5,
	}
}

// RollbackResult records the outcome of a rollback check.
type RollbackResult struct {
	// RolledBack is true if the pipeline output was rejected.
	RolledBack bool
	// Reason explains why the rollback happened.
	Reason string
	// Final is the text that was ultimately returned.
	Final string
	// FinalTokens is the token count of Final.
	FinalTokens int
}

// RollbackGuard wraps a pipeline run and rolls back if quality regresses.
type RollbackGuard struct {
	mu     sync.Mutex
	policy RollbackPolicy

	// history records the last N (input, output) checkpoints for recovery.
	history []rollbackEntry
	maxHist int
}

type rollbackEntry struct {
	at      time.Time
	input   string
	output  string
	tokens  int
	reason  string
}

// NewRollbackGuard creates a guard with the given policy.
func NewRollbackGuard(policy RollbackPolicy) *RollbackGuard {
	return &RollbackGuard{policy: policy, maxHist: 10}
}

// Check evaluates whether compressed is an acceptable replacement for original.
// Returns the final text to use (may be original if rolling back) and a RollbackResult.
func (g *RollbackGuard) Check(original, compressed string) (string, RollbackResult) {
	origTokens := core.EstimateTokens(original)
	compTokens := core.EstimateTokens(compressed)

	// Growth check: compressed is larger — rollback
	if len(compressed) > int(float64(len(original))*(1+g.policy.MaxBytesGrowth)) {
		return g.rollback(original, compressed, origTokens,
			fmt.Sprintf("output grew by more than %.0f%%", g.policy.MaxBytesGrowth*100))
	}

	// Over-compression check
	if origTokens > 0 {
		reductionPct := float64(origTokens-compTokens) / float64(origTokens) * 100
		if reductionPct > g.policy.MaxReductionPct {
			return g.rollback(original, compressed, origTokens,
				fmt.Sprintf("%.1f%% reduction exceeds max %.1f%%", reductionPct, g.policy.MaxReductionPct))
		}
	}

	// Content preservation check
	if g.policy.RequireContentPreservation {
		ratio := contentPreservationRatio(original, compressed)
		if ratio < g.policy.MinPreservationRatio {
			return g.rollback(original, compressed, origTokens,
				fmt.Sprintf("content preservation %.1f%% below minimum %.1f%%",
					ratio*100, g.policy.MinPreservationRatio*100))
		}
	}

	// All checks passed
	g.record(original, compressed, compTokens, "")
	return compressed, RollbackResult{
		RolledBack:  false,
		Final:       compressed,
		FinalTokens: compTokens,
	}
}

func (g *RollbackGuard) rollback(original, rejected string, origTokens int, reason string) (string, RollbackResult) {
	g.record(original, rejected, origTokens, reason)
	return original, RollbackResult{
		RolledBack:  true,
		Reason:      reason,
		Final:       original,
		FinalTokens: origTokens,
	}
}

func (g *RollbackGuard) record(input, output string, tokens int, reason string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	entry := rollbackEntry{
		at:     time.Now(),
		input:  input,
		output: output,
		tokens: tokens,
		reason: reason,
	}
	g.history = append(g.history, entry)
	if len(g.history) > g.maxHist {
		g.history = g.history[len(g.history)-g.maxHist:]
	}
}

// RollbackHistory returns a summary of recent rollback decisions.
func (g *RollbackGuard) RollbackHistory() []string {
	g.mu.Lock()
	defer g.mu.Unlock()
	var out []string
	for _, e := range g.history {
		if e.reason != "" {
			out = append(out, fmt.Sprintf("%s ROLLBACK: %s", e.at.Format(time.RFC3339), e.reason))
		} else {
			out = append(out, fmt.Sprintf("%s OK: %d tokens", e.at.Format(time.RFC3339), e.tokens))
		}
	}
	return out
}

// contentPreservationRatio checks what fraction of "important" words in the
// original appear in the compressed output.
// Important = words longer than 4 chars (skips function words like "the", "and").
func contentPreservationRatio(original, compressed string) float64 {
	origWords := importantWords(original)
	if len(origWords) == 0 {
		return 1.0
	}
	compLower := strings.ToLower(compressed)
	var preserved int
	for w := range origWords {
		if strings.Contains(compLower, w) {
			preserved++
		}
	}
	return float64(preserved) / float64(len(origWords))
}

// importantWords returns a set of lowercase words longer than 4 chars.
func importantWords(text string) map[string]struct{} {
	words := make(map[string]struct{})
	lower := strings.ToLower(text)
	start := -1
	for i, r := range lower {
		isAlpha := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if isAlpha {
			if start < 0 {
				start = i
			}
		} else {
			if start >= 0 {
				w := lower[start:i]
				if len(w) > 4 {
					words[w] = struct{}{}
				}
				start = -1
			}
		}
	}
	if start >= 0 {
		w := lower[start:]
		if len(w) > 4 {
			words[w] = struct{}{}
		}
	}
	return words
}
