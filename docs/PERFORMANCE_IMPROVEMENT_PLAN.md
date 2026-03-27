# TokMan Performance Improvement Plan

**Generated:** 2026-03-27  
**Test:** 1000-Agent Stress Test  
**Overall Status:** ⚠️ **Optimization Required**

---

## Executive Summary

The 1000-agent stress test identified significant performance bottlenecks that impact TokMan's scalability:

| Metric | Current | Target | Status |
|--------|---------|--------|--------|
| P99 Latency | 1.14s | <100ms | 🔴 Critical |
| P95 Latency | 434ms | <50ms | 🔴 Critical |
| Bottleneck Rate | 42.3% | <5% | 🔴 Critical |
| Throughput | 383 agents/s | 1000+ agents/s | 🟡 Needs Improvement |
| Compression Ratio | 98.35% | 95%+ | 🟢 Excellent |

**Key Insight:** While token compression is excellent, the pipeline processing overhead is too high for production scale.

---

## Critical Bottlenecks Identified

### 1. PerplexityFilter (Layer 2) - HIGHEST IMPACT

**Location:** `internal/filter/perplexity.go` (L46-61)

**Problem:**
- Iterative pruning with default 3 steps
- Each step tokenizes entire input: **O(N × iterations)**
- For 100KB logs: ~300KB of tokenization work

**Current Code:**
```go
for i := 0; i < f.iterationSteps; i++ {
    output = f.pruneIteration(output, mode)  // Full tokenization each time
}
```

**Solution:**
- Add early-exit when perplexity score stabilizes
- Cache tokenization results between iterations
- Reduce default iterations from 3 to 2 for aggressive mode
- Add adaptive iteration count based on input size

**Expected Impact:** 40-60% latency reduction

---

### 2. EntropyFilter (Layer 1) - HIGH IMPACT

**Location:** `internal/filter/entropy.go` (L309-337)

**Problem:**
- Rebuilds frequency table on every `Apply` call
- Dynamic frequency estimation (T11) has no cross-call caching
- Processes input twice (frequency building + pruning)

**Current Code:**
```go
if f.useDynamicEst {
    f.buildDynamicFrequencies(input)  // Rebuilt every time
}
```

**Solution:**
- Add LRU cache for frequency tables (key = input hash)
- Implement fingerprint-based caching (similar to `internal/cache/`)
- Process in single pass using streaming

**Expected Impact:** 20-30% latency reduction for repeated content

---

### 3. SymbolicCompressFilter - MEDIUM IMPACT

**Location:** `internal/filter/symbolic_compress.go` (L98-135)

**Problem:**
- 30+ sequential `regexp.ReplaceAllString` calls
- Regex compilation inside processing path (should be pre-compiled)
- No parallelization of independent replacements

**Current Code:**
```go
output = regexp.MustCompile(`...`).ReplaceAllString(output, "...")
output = regexp.MustCompile(`...`).ReplaceAllString(output, "...")
// ... 30+ times
```

**Solution:**
- Pre-compile all regexes at initialization
- Group independent replacements for parallel execution
- Use `strings.Builder` for memory efficiency

**Expected Impact:** 15-25% latency reduction

---

### 4. Sequential Pipeline Architecture - HIGH IMPACT

**Location:** `internal/filter/pipeline.go` (L650-910)

**Problem:**
- All 26 layers execute strictly sequentially
- No parallelization even for independent operations
- Stage gates help but don't solve fundamental issue

**Current Architecture:**
```
Input → L1 → L2 → L3 → ... → L26 → Output
```

**Proposed Architecture:**
```
Input
  ├─→ L1 (Entropy) ─┐
  ├─→ L2 (Perplexity) ─┤
  ├─→ L3 (Goal-Driven) ─├─→ Merge → L10+ → Output
  └─→ L4-L9 (Parallel) ─┘
```

**Solution:**
- Implement parallel layer groups using `sync.WaitGroup`
- Merge results before dependency-requiring layers
- Add concurrent pipeline variant in `parallel_pipeline.go`

**Expected Impact:** 2-3x throughput improvement

---

### 5. No Result Caching - MEDIUM IMPACT

**Location:** Entire pipeline

**Problem:**
- Same inputs processed multiple times without memoization
- No fingerprint-based caching for pipeline results
- Cache infrastructure exists (`internal/cache/`) but not used

**Solution:**
- Integrate `internal/cache/fingerprint_cache.go` into pipeline
- Cache results for repeated inputs (e.g., git diff headers)
- Implement cache invalidation on config changes

**Expected Impact:** 50-90% latency reduction for repeated content

---

## Optimization Roadmap

### Phase 1: Quick Wins (1-2 days)

| Task | File | Effort | Impact |
|------|------|--------|--------|
| Pre-compile regexes | `symbolic_compress.go` | Low | High |
| Reduce perplexity iterations | `perplexity.go` | Low | High |
| Add early-exit conditions | Multiple layers | Low | Medium |

**Expected Improvement:** 30-40% latency reduction

---

### Phase 2: Caching Infrastructure (3-5 days)

| Task | File | Effort | Impact |
|------|------|--------|--------|
| Entropy LRU cache | `entropy.go` | Medium | High |
| Pipeline result cache | `pipeline.go` | Medium | High |
| Fingerprint integration | `pipeline.go` | Medium | Medium |

**Expected Improvement:** Additional 50% latency reduction for repeated inputs

---

### Phase 3: Parallel Pipeline (1-2 weeks)

| Task | File | Effort | Impact |
|------|------|--------|--------|
| Parallel layer groups | `parallel_pipeline.go` | High | Very High |
| Concurrent merging | `pipeline.go` | High | High |
| Adaptive parallelism | `adaptive.go` | Medium | Medium |

**Expected Improvement:** 2-3x throughput increase

---

### Phase 4: Advanced Optimizations (2-3 weeks)

| Task | File | Effort | Impact |
|------|------|--------|--------|
| SIMD-accelerated layers | `simd/` | High | Medium |
| Streaming for large inputs | Multiple | High | High |
| Benchmark harness improvements | `benchmark_harness.go` | Medium | Low |

**Expected Improvement:** 10-20% additional gains

---

## Specific Code Changes

### 1. Fix PerplexityFilter

```go
// BEFORE
func (f *PerplexityFilter) Apply(input string, mode Mode) (string, int) {
    output := input
    for i := 0; i < f.iterationSteps; i++ {
        output = f.pruneIteration(output, mode)
    }
    return output, saved
}

// AFTER
func (f *PerplexityFilter) Apply(input string, mode Mode) (string, int) {
    output := input
    prevScore := f.computePerplexity(output)
    
    for i := 0; i < f.iterationSteps; i++ {
        newOutput := f.pruneIteration(output, mode)
        newScore := f.computePerplexity(newOutput)
        
        // Early exit if convergence
        if math.Abs(newScore-prevScore) < f.threshold {
            break
        }
        
        output = newOutput
        prevScore = newScore
    }
    return output, saved
}
```

### 2. Add Entropy Caching

```go
type EntropyFilter struct {
    cache    *lru.Cache[uint64, frequencyTable]
    cacheMu  sync.RWMutex
}

func (f *EntropyFilter) Apply(input string, mode Mode) (string, int) {
    hash := fingerprint(input)
    
    // Check cache
    f.cacheMu.RLock()
    if cached, ok := f.cache.Get(hash); ok {
        f.cacheMu.RUnlock()
        return f.applyWithFreq(input, cached, mode)
    }
    f.cacheMu.RUnlock()
    
    // Build and cache
    freq := f.buildDynamicFrequencies(input)
    f.cacheMu.Lock()
    f.cache.Add(hash, freq)
    f.cacheMu.Unlock()
    
    return f.applyWithFreq(input, freq, mode)
}
```

### 3. Parallel Layer Groups

```go
func (pc *PipelineCoordinator) Process(input string) (string, Stats) {
    // Group 1: Independent layers (can run in parallel)
    var wg sync.WaitGroup
    results := make(chan layerResult, 4)
    
    wg.Add(4)
    go pc.runLayerAsync(&wg, results, "entropy", input)
    go pc.runLayerAsync(&wg, results, "perplexity", input)
    go pc.runLayerAsync(&wg, results, "goal_driven", input)
    go pc.runLayerAsync(&wg, results, "ast", input)
    
    go func() {
        wg.Wait()
        close(results)
    }()
    
    // Merge results
    merged := pc.mergeResults(results)
    
    // Group 2: Sequential layers (depend on Group 1)
    output := pc.processSequentialLayers(merged)
    
    return output, stats
}
```

---

## Testing Strategy

### Benchmark Suite Enhancement

1. **Add layer-level benchmarks** to measure individual layer performance
2. **Implement memory profiling** to identify allocation hotspots
3. **Add concurrent benchmarks** to simulate real-world load
4. **Create regression tests** to prevent performance degradation

### Performance Gates

```bash
# Required performance thresholds
P99_LATENCY_MS=100
P95_LATENCY_MS=50
THROUGHPUT_AGENTS_SEC=1000
COMPRESSION_RATIO=0.95
```

---

## Monitoring & Observability

### Add Metrics

1. Per-layer latency histograms
2. Cache hit/miss rates
3. Parallel vs sequential execution ratio
4. Memory allocation tracking

### Dashboard Integration

Leverage existing `internal/dashboard/` for real-time monitoring.

---

## Expected Final Results

After all optimizations:

| Metric | Current | Expected | Improvement |
|--------|---------|----------|-------------|
| P99 Latency | 1.14s | <100ms | **11x faster** |
| P95 Latency | 434ms | <50ms | **8x faster** |
| Throughput | 383 agents/s | 1000+ agents/s | **2.6x faster** |
| Bottleneck Rate | 42.3% | <5% | **8x reduction** |
| Memory Efficiency | Unknown | Optimized | TBD |

---

## Conclusion

TokMan's compression algorithm is excellent (98.35% token reduction), but the processing pipeline needs optimization for production scale. The primary issues are:

1. **Sequential processing** - layers execute one after another
2. **No caching** - repeated work for identical inputs
3. **Inefficient algorithms** - perplexity and entropy filters

By implementing the phased roadmap, we expect to achieve:
- **10x latency improvement** for P99 requests
- **2-3x throughput increase** for concurrent agents
- **Better resource utilization** through parallelism

**Priority:** Start with Phase 1 (quick wins) for immediate impact, then proceed to caching and parallelization.
