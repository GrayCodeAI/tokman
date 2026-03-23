# AGENTS.md -- `internal/filter/` Package Guide

> **Purpose:** This document catalogs every non-test `.go` file in the `internal/filter`
> package. The package implements a 20-layer compression pipeline that reduces LLM
> context while preserving semantic meaning. All files live in a single Go package
> (`package filter`) because they share core types (`PipelineState`, `Mode`,
> `ContentType`, etc.) and tightly-coupled interfaces.

> **Subdirectory convention:** The `layers/` subdirectory exists as a placeholder for
> future extraction of less-coupled compression layers. Currently it is empty.

---

## 1. Core Pipeline

These files orchestrate the entire compression system. Every other file in the package
plugs into this core.

| File | Description |
|------|-------------|
| `pipeline.go` | Main pipeline engine that chains 20 compression layers, manages execution order, and produces the final compressed output. |
| `manager.go` | High-level lifecycle manager -- loads/saves pipeline configurations, handles SHA-256 cache keys, and coordinates preset selection. |
| `router.go` | Content router that detects content type (JSON, code, logs, etc.) and selects the optimal compression strategy. |
| `pipeline_state.go` | Immutable snapshot of pipeline processing state (layer results, token counts, cache keys) used for thread-safe, cacheable, debuggable data flow. |
| `filter.go` | Defines the core `Mode` type (compress/reversible/aggressive), `Filter` interface, and the top-level `Apply()` entry point that wires everything together. |

---

## 2. Compression Layers (L1--L20)

Each file implements one or more layers of the 20-layer compression pipeline. Most are
backed by published research (cited in file comments).

| File | Layer(s) | Description |
|------|----------|-------------|
| `entropy.go` | L1 | Entropy-based token pruning using Shannon entropy with SIMD-accelerated scoring; drops low-information tokens. |
| `perplexity.go` | L2 | LLMLingua-style perplexity-based iterative pruning (Microsoft/Tsinghua, 2023); ranks tokens by perplexity and removes the least surprising. |
| `query_aware.go` | L3 | Detects query intent (debug, explain, summarize, etc.) and adjusts compression behavior to preserve query-relevant content. |
| `ast_preserve.go` | L4 | LongCodeZip-style AST-aware compression (NUS, 2025); parses code structure and preserves function signatures while compressing bodies. |
| `contrastive.go` | L5 | LongLLMLingua contrastive perplexity (Microsoft, 2024); question-aware compression that ranks tokens by query relevance using contrastive scoring. |
| `ngram.go` | L6 | N-gram abbreviation filter; compresses output by abbreviating common multi-word patterns and repeated phrases using SIMD-accelerated matching. |
| `evaluator_heads.go` | L7 | EHPC-style evaluator-heads compression (Tsinghua/Huawei, 2025); identifies important tokens by simulating attention head behavior. |
| `gist.go` | L8 | Gisting compression (Stanford/Berkeley, 2023); compresses prompts into virtual "gist tokens" representing condensed meaning. |
| `hierarchical.go` | L9 | Multi-level hierarchical summarization; creates a tree structure where each level provides progressively more detail for progressive decompression. |
| `budget.go` | L10 | Budget enforcer that scores output segments and keeps only the most important ones to hit a strict token limit. |
| `compaction.go` | L11 | Full compactor engine with SHA-256 chunk deduplication, cross-reference resolution, delta encoding, and merge logic. |
| `attribution.go` | L12 | Token attribution filter; scores individual tokens by their contribution to downstream predictions and drops low-attribution tokens. |
| `h2o.go` | L13 | Heavy-Hitter Oracle (H2O) compression; uses a heap-based approach to identify and preserve "heavy-hitter" tokens that are most frequently attended to. |
| `attention_sink.go` | L14 | StreamingLLM-style attention-sink preservation (Xiao et al., 2023); keeps attention-sink tokens at sequence boundaries for stable infinite-length generation. |
| `meta_token.go` | L15 | Meta-token compression; replaces repeated sub-sequences with short virtual tokens and a lookup table, achieving high compression on repetitive content. |
| `semantic_chunk.go` | L16 | ChunkKV-style semantic chunk compression (NeurIPS, 2025); groups tokens into semantic chunks and prunes entire low-relevance chunks. |
| `sketch_store.go` | L17 | Sketch-based storage; maintains lightweight hash-based sketches of content for fast similarity comparison and cross-reference resolution. |
| `lazy_pruner.go` | L18 | Budget-aware dynamic pruning (LazyLLM style); defers pruning decisions until budget is exhausted, allocating capacity to the most valuable segments. |
| `semantic_anchor.go` | L19 | Semantic-Anchor Compression (SAC, 2024); identifies anchor points in content (definitions, declarations) and compresses relative to them. |
| `agent_memory.go` | L20 | Agent memory mode (Focus-inspired); maintains a compressed working memory of previous interactions for multi-turn agent sessions. |
| `semantic.go` | -- | General-purpose semantic filter; prunes low-information segments using statistical analysis and unicode-level heuristics. |

---

## 3. Adaptive Selection

These files dynamically adjust which compression layers run and how aggressively they
compress, based on content characteristics.

| File | Description |
|------|-------------|
| `adaptive.go` | Adaptive layer selector; uses heuristic content-type analysis to dynamically enable/disable layers and tune thresholds per input. |
| `adaptive_attention.go` | ADSC-style attention-driven self-compression (Berkeley/Clemson, 2025); mimics LLM attention patterns to score and filter tokens. |
| `density_adaptive.go` | DAST-style density-adaptive token allocation (Chen et al., 2025); allocates compression budget proportionally to regional information density. |

---

## 4. Utilities

Shared helpers, data structures, and low-level primitives used across the package.

| File | Description |
|------|-------------|
| `utils.go` | Core tokenizer and string normalization helpers (`cleanWord`, `tokenizeRe`); used by nearly every filter. |
| `presets.go` | Defines `PipelinePreset` constants (fast/balanced/full) and the layer sets each preset activates. |
| `lru_cache.go` | Thread-safe LRU cache with TTL and optional persistence for caching compression results (T101--T105). |
| `fingerprint.go` | Content-hash (SHA-256) based cache key generation; enables fast lookups without storing full content (R13). |
| `bytes.go` | `ByteSlicePool` -- reusable byte-slice pool to reduce GC pressure in the hot path (T89). |
| `ansi.go` | ANSI escape-sequence stripper; uses SIMD-optimized byte scanning for 10--40x speedup over regex. |
| `noise.go` | Progress-bar and noise detector; identifies and removes transient CLI output (progress bars, download stats) that has no informational value. |
| `detector.go` | `ContentAnalyzer` -- content-type detection wrapper that delegates to `AdaptiveLayerSelector` for conditional layer execution (T86). |
| `dedup.go` | Line-level deduplication filter; removes duplicate lines common in logs and test output (R60). |
| `equivalence.go` | Semantic equivalence checker; verifies that compressed output preserves critical information (R16). |
| `bm25.go` | Okapi BM25 scorer; provides relevance ranking superior to TF-IDF for segment prioritization (R8). |

---

## 5. Code-Aware Processing

Filters that understand programming-language structure (imports, braces, comments).

| File | Description |
|------|-------------|
| `ast_preserve.go` | *(Also listed in Compression Layers L4.)* AST-aware filter that parses function signatures, class declarations, and control-flow boundaries. |
| `brace_depth.go` | `BodyFilter` that strips function bodies based on brace depth; preserves signatures while removing implementation detail in aggressive mode. |
| `comment_patterns.go` | Language-specific comment pattern registry; defines line-comment, block-comment, and doc-comment syntax for ~20 languages. |
| `import.go` | Import statement condenser; collapses verbose import blocks into compact representations per language. |

---

## 6. Quality / Analysis

Metrics and analysis tools that measure or preserve compression quality.

| File | Description |
|------|-------------|
| `quality.go` | `QualityMetrics` -- measures information preservation ratio, key-term retention, and structural integrity after compression (T185). |
| `density.go` | `DensityAdaptiveAllocator` -- allocates compression budget based on per-section information density (R9, DAST). |
| `attribution.go` | *(Also listed in Compression Layers L12.)* Per-token attribution scoring that measures each token's contribution to downstream predictions. |
| `error_trace.go` | Error-trace compressor; reduces stack traces to error type + first user file + line number + message (~4x compression). |
| `stacktrace.go` | Stack-trace preserver; treats stack traces as atomic units that must never be split during compression (R61). |

---

## 7. Multi-File Handling

Filters that operate across multiple files or outputs as a unit.

| File | Description |
|------|-------------|
| `multifile.go` | Multi-file output optimizer; identifies cross-file relationships, deduplicates shared content, and produces unified summaries for LLM context. |
| `multi_file.go` | Multi-file filter with cross-file relationship detection; sorts, deduplicates, and creates consolidated views of related file outputs. |

---

## 8. Hierarchical Summarization

Tree-based progressive summarization layers.

| File | Description |
|------|-------------|
| `hierarchical.go` | *(Also listed in Compression Layers L9.)* Multi-level summarization tree for progressive detail recovery. |
| `hierarchical_summary.go` | AutoCompressor-style recursive summarization (Princeton/MIT, 2023); compresses context into hierarchical summary vectors. |

---

## 9. Streaming

Real-time compression for content arriving incrementally.

| File | Description |
|------|-------------|
| `stream.go` | `StreamingProcessor` for real-time compression of streaming content; designed for chat agents and long-running sessions with bounded memory. |

---

## 10. Reversible Compression

Lossless compression that can reconstruct the original output.

| File | Description |
|------|-------------|
| `reversible.go` | Reversible compression with on-disk full-content storage and SHA-256 integrity verification; allows lossless decompression. |

---

## 11. Goal-Driven Compression

Compression guided by a specific task or goal context.

| File | Description |
|------|-------------|
| `goal_driven.go` | SWE-Pruner style goal-driven compression (Shanghai Jiao Tong, 2025); uses task context (e.g., bug-fix goal) to prioritize relevant code constructs. |

---

## 12. Position-Aware Compression

Reordering strategies that exploit LLM recall patterns.

| File | Description |
|------|-------------|
| `position_aware.go` | Reorders output segments to counteract the "lost in the middle" phenomenon (LongLLMLingua, Jiang et al., 2024); puts important content at sequence ends. |

---

## 13. Question-Aware Compression

Preserves query-relevant subsequences during compression.

| File | Description |
|------|-------------|
| `question_aware.go` | LongLLMLingua-style question-aware recovery (Jiang et al., ACL 2024); preserves query-relevant subsequences by scoring token--question similarity. |

---

## 14. LLM-Aware Compression

Filters that leverage an LLM (local or remote) as part of the compression process.

| File | Description |
|------|-------------|
| `llm_aware.go` | LLM-aware filter using a local LLM for high-quality summarization when heuristic compression is insufficient. |
| `llm_compress.go` | LLM-driven compression via external process invocation; shells out to an LLM binary for context-aware summarization with JSON I/O. |

---

## 15. Plugin System

Extensibility layer for user-defined compression rules.

| File | Description |
|------|-------------|
| `plugin.go` | Plugin loader and runner; reads JSON/regex-based compression rule definitions from disk and executes them as custom filter layers. |

---

## 16. Session Management

Persistence and state management for multi-turn compression sessions.

| File | Description |
|------|-------------|
| `session.go` | Session manager with SHA-256-keyed state persistence; stores compressed context snapshots on disk for continuity across agent turns. |

---

## 17. Log Aggregation

Dedicated log-output compression.

| File | Description |
|------|-------------|
| `aggregator.go` | `LogAggregator` that deduplicates and compresses log output; groups repeated log lines into count-annotated summaries. |

---

## File Count Summary

| Category | Count |
|----------|-------|
| Core Pipeline | 5 |
| Compression Layers (L1--L20) | 21 |
| Adaptive Selection | 3 |
| Utilities | 11 |
| Code-Aware Processing | 4 |
| Quality / Analysis | 5 |
| Multi-File Handling | 2 |
| Hierarchical Summarization | 2 |
| Streaming | 1 |
| Reversible | 1 |
| Goal-Driven | 1 |
| Position-Aware | 1 |
| Question-Aware | 1 |
| LLM-Aware | 2 |
| Plugin System | 1 |
| Session Management | 1 |
| Log Aggregation | 1 |
| **Total non-test `.go` files** | **62** |

> Note: `ast_preserve.go`, `attribution.go`, `density.go`, and `hierarchical.go` appear
> in multiple categories because they serve dual roles (compression layer + quality
> analysis / code-aware). The unique file count is 60.

---

## Subdirectory: `layers/`

The `layers/` subdirectory exists as an empty placeholder. It is reserved for future
extraction of less-coupled compression layers out of the root package while keeping
shared types and interfaces at the root level.
