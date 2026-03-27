# TokMan Practical Improvements Plan
**Date:** 2026-03-24
**Goal:** Make TokMan practical and fast while preserving advanced capabilities

## Overview

Reorganize 31 layers into **context-aware tiers** that run automatically based on:
- Content size/type
- Available resources (budget, LLM)
- Query context

## Layer Tier System

### Tier 1: FAST (Always Run)
*Cost: <1ms | Benefit: Always applicable*

| Layer | File | When to Skip |
|-------|------|--------------|
| TF-IDF Filter | `tfidf.go` | Never (3x faster pipeline) |
| Entropy | `entropy.go` | Content < 50 chars |
| Meta-Token | `meta_token.go` | Content < 500 chars |

**Auto-activation:** Always enabled, no configuration needed.

### Tier 2: BALANCED (Context-Aware)
*Cost: 1-10ms | Benefit: Moderate content*

| Layer | File | Activation Context |
|-------|------|-------------------|
| Perplexity | `perplexity.go` | 5+ lines |
| N-gram | `ngram.go` | 200+ chars, 20+ words |
| H2O | `h2o.go` | 50+ tokens |
| Attention Sink | `attention_sink.go` | 3+ lines |
| Semantic Chunk | `semantic_chunk.go` | 300+ chars |
| Compaction | `compaction.go` | Conversation markers detected |

**Auto-activation:** Based on content analysis, no user input needed.

### Tier 3: EXPENSIVE (Resource-Dependent)
*Cost: 10-100ms+ | Benefit: High compression*

| Layer | File | Activation Context |
|-------|------|-------------------|
| Goal-Driven | `goal_driven.go` | Query intent provided |
| Contrastive | `contrastive.go` | Query intent provided |
| AST Preserve | `ast_preserve.go` | Code content detected |
| Gist | `gist.go` | Large content (>1000 tokens) |
| Hierarchical | `hierarchical.go` | Very large content (>5000 tokens) |
| LLM-Aware | `llm_aware.go` | LLM enabled + provider available |

**Auto-activation:** Based on explicit flags or auto-detected context.

### Tier 4: SPECIALIZED (Use-Case Specific)
*Cost: Variable | Benefit: Domain-specific*

| Layer | File | Activation Context |
|-------|------|-------------------|
| Sketch Store | `sketch_store.go` | Budget tracking enabled |
| Lazy Pruner | `lazy_pruner.go` | Budget tracking enabled |
| Agent Memory | `agent_memory.go` | Agent session mode |
| Reasoning Trace | `reasoning_trace.go` | CoT/reasoning content detected |
| Symbolic Compress | `symbolic_compress.go` | Instruction-heavy content |
| Phrase Grouping | `phrase_grouping.go` | Natural language content |
| Numerical Quant | `numerical_quant.go` | Numeric data detected |

**Auto-activation:** Based on content patterns and session mode.

### Tier 5: REPAIR (Post-Processing)
*Cost: 1-5ms | Benefit: Quality assurance*

| Layer | File | Purpose |
|-------|------|---------|
| SmallKV Compensator | `smallkv.go` | Repair over-compression |
| Question-Aware | `question_aware.go` | Query relevance recovery |
| Density-Adaptive | `density_adaptive.go` | Balance compression density |

**Auto-activation:** Always run after other tiers complete.

---

## Implementation Plan

### Phase 1: Tier Infrastructure (PR 1)

1. **Create `tiers.go`** - Tier definitions and auto-detection
   ```go
   type Tier int
   const (
       TierFast Tier = iota
       TierBalanced
       TierExpensive
       TierSpecialized
       TierRepair
   )
   
   type TierConfig struct {
       ContentSize    int
       HasQuery       bool
       HasBudget      bool
       HasLLM         bool
       IsCode         bool
       IsConversation bool
       IsAgent        bool
   }
   ```

2. **Create `tier_selector.go`** - Auto-select tiers based on context
   ```go
   func SelectTiers(content string, cfg PipelineConfig) []Tier
   func DetectContentType(content string) ContentType
   func EstimateContentMetrics(content string) ContentMetrics
   ```

3. **Update `pipeline.go`** - Process by tier instead of individual layers
   ```go
   func (p *PipelineCoordinator) Process(input string) (string, *PipelineStats) {
       tiers := SelectTiers(input, p.config)
       for _, tier := range tiers {
           output = p.processTier(tier, output, stats)
           if p.shouldEarlyExit(stats) {
               break
           }
       }
       return output, stats
   }
   ```

### Phase 2: Presets as Tier Profiles (PR 2)

1. **Update `presets.go`** - Map presets to tier configurations
   ```go
   var PresetTiers = map[string][]Tier{
       "fast":    {TierFast, TierRepair},
       "balanced": {TierFast, TierBalanced, TierRepair},
       "full":    {TierFast, TierBalanced, TierExpensive, TierSpecialized, TierRepair},
   }
   ```

2. **Add `--auto` preset** - Let TokMan decide based on content
   ```go
   // Auto-select best tier combination based on content analysis
   tokman --preset=auto
   ```

### Phase 3: Simplified CLI (PR 3)

1. **Quick Start Command**
   ```go
   // cmd/tokman/quickstart.go
   tokman quickstart
   // - Detect agents
   // - Install hooks
   // - Apply sensible defaults
   // - Run doctor to verify
   ```

2. **Discovery Command**
   ```go
   // cmd/tokman/discover.go
   tokman discover
   // - Scan recent command history
   // - Show missed savings opportunities
   // - Suggest filters
   ```

3. **Gain Command**
   ```go
   // cmd/tokman/gain.go
   tokman gain
   // - Show token savings summary
   // - Compare with/without TokMan
   // - ROI metrics
   ```

4. **Doctor Command** (already exists, enhance it)
   ```go
   tokman doctor
   // - Check hooks installed
   // - Verify configuration
   // - Test sample compression
   ```

### Phase 4: Documentation Update (PR 4)

1. **Quick Start Guide** - 5-minute setup
2. **Tier System Docs** - Explain auto-activation
3. **Migration Guide** - From other tools
4. **Examples** - Common use cases

---

## API Changes

### Before (Complex)
```go
cfg := filter.PipelineConfig{
    Mode: filter.ModeMinimal,
    EnableEntropy: true,
    EnablePerplexity: true,
    EnableGoalDriven: true,
    // ... 30+ individual flags
}
```

### After (Simplified)
```go
// Option 1: Preset
cfg := filter.DefaultConfig("balanced")

// Option 2: Auto-detect
cfg := filter.AutoConfig(input)

// Option 3: Custom tiers
cfg := filter.TierConfig{
    Tiers: []filter.Tier{filter.TierFast, filter.TierBalanced},
    Budget: 2000,
}
```

---

## Benefits

| Before | After |
|--------|-------|
| 31 layers exposed | 5 tiers auto-selected |
| Manual configuration | Auto-detection |
| Research-heavy docs | Practical quick start |
| Expert-focused | Developer-friendly |

---

## Success Metrics

- [x] `tokman quickstart` works in < 30 seconds
- [x] Default preset achieves 60%+ compression
- [x] Documentation readable in < 5 minutes
- [x] Zero-config installation possible
- [x] Discovery shows actionable recommendations

## Implementation Status

### Completed
- ✅ Phase 1: Tier Infrastructure (`adaptive_pipeline.go` with 4 tiers)
- ✅ Phase 2: Presets as Tier Profiles (`presets.go` with fast/balanced/full/auto)
- ✅ Phase 3 (Partial): `quickstart`, `gain`, `discover` commands
- ✅ Phase 4: Documentation (Quick Start + Migration guides)
- ✅ `doctor` enhanced with tier system verification

### Pending
- 🔄 Integrate Phase 2 research layers more deeply (Hypernym, SemanticCache, Scope)
- 🔄 Performance benchmarks comparing tiers

