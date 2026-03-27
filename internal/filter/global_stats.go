package filter

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// GlobalCompressionStats aggregates compression statistics across all sessions.
// Thread-safe. Can be used as a singleton across the lifetime of the process.
type GlobalCompressionStats struct {
	mu sync.RWMutex

	totalRuns      atomic.Int64
	totalInputToks atomic.Int64
	totalSavedToks atomic.Int64

	// Per-filter savings
	filterSavings map[string]*atomic.Int64

	// Per-content-type savings
	ctypeSavings map[string]*atomic.Int64

	// Best reduction ratios seen (rolling top-10)
	topRatios []ratioEntry

	// Session start time
	startTime time.Time
}

type ratioEntry struct {
	ratio     float64
	inputToks int
	timestamp time.Time
}

// globalStats is the package-level singleton.
var globalStats = newGlobalStats()

func newGlobalStats() *GlobalCompressionStats {
	return &GlobalCompressionStats{
		filterSavings: make(map[string]*atomic.Int64),
		ctypeSavings:  make(map[string]*atomic.Int64),
		topRatios:     make([]ratioEntry, 0, 10),
		startTime:     time.Now(),
	}
}

// GetGlobalStats returns the package-level stats singleton.
func GetGlobalStats() *GlobalCompressionStats {
	return globalStats
}

// Record records the results of one compression run.
func (s *GlobalCompressionStats) Record(inputToks, savedToks int, filtersUsed []string, contentType string) {
	s.totalRuns.Add(1)
	s.totalInputToks.Add(int64(inputToks))
	s.totalSavedToks.Add(int64(savedToks))

	// Per-filter savings
	s.mu.Lock()
	for _, f := range filtersUsed {
		if _, ok := s.filterSavings[f]; !ok {
			s.filterSavings[f] = &atomic.Int64{}
		}
	}
	if contentType != "" {
		if _, ok := s.ctypeSavings[contentType]; !ok {
			s.ctypeSavings[contentType] = &atomic.Int64{}
		}
	}
	s.mu.Unlock()

	for _, f := range filtersUsed {
		s.mu.RLock()
		counter := s.filterSavings[f]
		s.mu.RUnlock()
		counter.Add(int64(savedToks / maxInt(len(filtersUsed), 1)))
	}

	if contentType != "" {
		s.mu.RLock()
		counter := s.ctypeSavings[contentType]
		s.mu.RUnlock()
		counter.Add(int64(savedToks))
	}

	// Track top ratios
	if inputToks > 0 && savedToks > 0 {
		ratio := float64(savedToks) / float64(inputToks)
		s.mu.Lock()
		s.topRatios = append(s.topRatios, ratioEntry{
			ratio:     ratio,
			inputToks: inputToks,
			timestamp: time.Now(),
		})
		sort.Slice(s.topRatios, func(i, j int) bool {
			return s.topRatios[i].ratio > s.topRatios[j].ratio
		})
		if len(s.topRatios) > 10 {
			s.topRatios = s.topRatios[:10]
		}
		s.mu.Unlock()
	}
}

// Summary returns a formatted summary of all compression statistics.
func (s *GlobalCompressionStats) Summary() string {
	totalRuns := s.totalRuns.Load()
	totalIn := s.totalInputToks.Load()
	totalSaved := s.totalSavedToks.Load()

	overallReduction := 0.0
	if totalIn > 0 {
		overallReduction = float64(totalSaved) / float64(totalIn) * 100
	}

	uptime := time.Since(s.startTime).Round(time.Second)

	var sb strings.Builder
	sb.WriteString("=== Global Compression Statistics ===\n\n")
	sb.WriteString(fmt.Sprintf("Uptime: %s | Total runs: %d\n", uptime, totalRuns))
	sb.WriteString(fmt.Sprintf("Total input: %d tokens | Saved: %d tokens (%.1f%%)\n\n",
		totalIn, totalSaved, overallReduction))

	// Top filters by savings
	s.mu.RLock()
	type kv struct {
		name  string
		saved int64
	}
	var topFilters []kv
	for name, counter := range s.filterSavings {
		topFilters = append(topFilters, kv{name, counter.Load()})
	}
	sort.Slice(topFilters, func(i, j int) bool { return topFilters[i].saved > topFilters[j].saved })

	var topCTypes []kv
	for ct, counter := range s.ctypeSavings {
		topCTypes = append(topCTypes, kv{ct, counter.Load()})
	}
	sort.Slice(topCTypes, func(i, j int) bool { return topCTypes[i].saved > topCTypes[j].saved })

	topRatios := make([]ratioEntry, len(s.topRatios))
	copy(topRatios, s.topRatios)
	s.mu.RUnlock()

	if len(topFilters) > 0 {
		sb.WriteString("Top Filters by Tokens Saved:\n")
		for i := 0; i < 5 && i < len(topFilters); i++ {
			sb.WriteString(fmt.Sprintf("  %d. %-25s  %d tokens\n", i+1, topFilters[i].name, topFilters[i].saved))
		}
		sb.WriteString("\n")
	}

	if len(topCTypes) > 0 {
		sb.WriteString("Best Compression by Content Type:\n")
		for _, ct := range topCTypes {
			sb.WriteString(fmt.Sprintf("  %-20s  %d tokens saved\n", ct.name, ct.saved))
		}
		sb.WriteString("\n")
	}

	if len(topRatios) > 0 {
		sb.WriteString("Best Compression Ratios (top 5):\n")
		for i := 0; i < 5 && i < len(topRatios); i++ {
			sb.WriteString(fmt.Sprintf("  %d. %.1f%% reduction (%d tokens in)\n",
				i+1, topRatios[i].ratio*100, topRatios[i].inputToks))
		}
	}

	return sb.String()
}

// Reset clears all statistics (e.g., at test teardown).
func (s *GlobalCompressionStats) Reset() {
	s.totalRuns.Store(0)
	s.totalInputToks.Store(0)
	s.totalSavedToks.Store(0)
	s.mu.Lock()
	s.filterSavings = make(map[string]*atomic.Int64)
	s.ctypeSavings = make(map[string]*atomic.Int64)
	s.topRatios = s.topRatios[:0]
	s.mu.Unlock()
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
