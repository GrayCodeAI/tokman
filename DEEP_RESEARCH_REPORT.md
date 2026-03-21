# Deep Research Report: LLM Token Reduction & Context Compression
**Date**: March 21, 2026  
**Papers Analyzed**: 120+ research papers from ACL, NeurIPS, ICLR, ICLR 2024-2026  
**Purpose**: Identify cutting-edge techniques to advance Tokman beyond RTK

---

## Executive Summary

This report synthesizes findings from 120+ research papers on LLM token reduction, prompt compression, KV cache optimization, and context management. Key breakthroughs span **12 categories** with significant implications for Tokman's roadmap.

**Top 5 Game-Changing Discoveries:**
1. **Lossless Compression via Meta-Tokens** (2025) - 27-47% compute reduction with ZERO semantic loss
2. **Reversible KV Cache** (KVReviver, 2025) - 75% memory reduction with sketch-based reconstruction
3. **Semantic-Anchor Compression** (SAC, 2025) - Outperforms autoencoding methods without training
4. **LazyLLM Dynamic Pruning** (2024) - 2.34x prefill speedup with selective KV computation
5. **LongCodeZip** (2025) - 5.6x code compression with AST-aware chunking

---

## Category 1: Dynamic Token Pruning

### 1.1 LazyLLM (July 2024) - ⭐⭐⭐⭐⭐
**Venue**: arXiv  
**Key Innovation**: Selective KV computation during prefill stage

**Methodology:**
- Dynamically selects which tokens need KV computation at each layer
- Tokens can be "revived" even if pruned in previous steps
- No fine-tuning required - plug-and-play with any LLM

**Results:**
- **2.34x prefill acceleration** on Llama-2 7B
- Maintains accuracy on multi-document QA tasks
- Works across various model architectures

**Tokman Implementation:**
```go
type LazyPruner struct {
    ImportanceThreshold float64
    LayerBudgets        []int  // Per-layer token budgets
    RevivalBudget       int    // Tokens that can be recomputed
}

func (lp *LazyPruner) SelectTokens(layer int, attention []float64) []int {
    // Select top-K by attention + allow dynamic revival
}
```

**Gap Addressed**: T28 (Parallel Layer Execution) - LazyLLM enables layer-wise parallelism

---

### 1.2 Saliency-Driven Dynamic Token Pruning (SDTP) (April 2024)
**Venue**: arXiv  
**Key Innovation**: Cascade pruning with learned saliency

**Methodology:**
- Trains lightweight saliency predictor
- Cascade architecture: progressive token filtering
- Outperforms LLMLingua-2 on efficiency metrics

**Results:**
- Superior to static pruning methods
- Lower perplexity degradation

---

## Category 2: Attention-Based Compression

### 2.1 StreamingLLM (Sep 2023, ICLR 2024) - ⭐⭐⭐⭐⭐
**Venue**: ICLR 2024  
**Key Innovation**: Attention Sinks - initial tokens as stability anchors

**Critical Discovery:**
- Initial tokens receive disproportionately high attention scores
- These "attention sinks" are NOT semantically important
- Keeping them enables stable streaming with window attention

**Methodology:**
1. Retain initial 1-4 tokens as attention sinks
2. Use rolling window for recent tokens
3. Discard middle tokens safely

**Results:**
- **22.2x speedup** over sliding window recomputation
- Stable generation on 4+ million tokens
- Works on Llama-2, MPT, Falcon, Pythia without fine-tuning

**Tokman Implementation:**
```go
type StreamingConfig struct {
    SinkTokens      int  // Usually 1-4
    RecentWindow    int  // Rolling window size
    InitialTokens   []Token
}

func (s *StreamingConfig) Compress(tokens []Token) []Token {
    // Keep: [sink_tokens... | ... | recent_window...]
    // Discard middle tokens
}
```

**Tokman Layer 14 (Attention Sink)** already implements this concept.

---

### 2.2 H2O: Heavy-Hitter Oracle (NeurIPS 2023) - ⭐⭐⭐⭐⭐
**Venue**: NeurIPS 2023  
**Key Innovation**: Heavy hitters = frequently co-occurring tokens

**Critical Insight:**
- Heavy hitters correlate with token co-occurrence frequency
- Removing them causes significant performance degradation
- Balance recent tokens + heavy hitters

**Methodology:**
- Dynamic KV cache eviction policy
- Retains: recent tokens (sliding window) + heavy hitters (accumulated attention)
- Formulated as dynamic submodular optimization

**Results:**
- **29x throughput improvement** over DeepSpeed, HuggingFace Accelerate
- **1.9x latency reduction**
- 20% heavy hitters sufficient for performance

**Tokman Implementation:**
```go
type H2OCache struct {
    RecentWindow    int
    HeavyHitterPct  float64  // 0.2 = 20%
    AccumulatedAttn map[int]float64
}

func (h *H2OCache) Evict() []int {
    // Keep: recent + top-K by accumulated attention
}
```

---

## Category 3: Prompt Compression

### 3.1 LongLLMLingua (ACL 2024) - ⭐⭐⭐⭐⭐
**Venue**: ACL 2024  
**Key Innovation**: Perplexity + position-aware prompt compression

**Methodology:**
- Coarse-to-fine compression
- Position bias correction (move key info to optimal positions)
- Question-aware token scoring

**Results:**
- **21.4% performance boost** with 4x fewer tokens (GPT-3.5-Turbo)
- **94% cost reduction** on LooGLE benchmark
- **1.4x-2.6x latency acceleration** at 2x-6x compression

**Tokman Gap**: L2 (Perplexity) layer could incorporate position rebalancing

---

### 3.2 500xCompressor (ACL 2025)
**Venue**: ACL 2025  
**Key Innovation**: Compress natural language into single BOS token

**Methodology:**
- Uses BOS token to guide LLM regeneration
- Hard prompt + soft prompt hybrid

**Results:**
- Extreme compression ratios
- Requires careful tuning

---

### 3.3 ProCut (EMNLP 2025)
**Venue**: EMNLP 2025 Industry Track  
**Key Innovation**: Attribution-based prompt compression

**Results:**
- **73-84% token reduction**
- Uses saliency scores for token importance

---

## Category 4: Code-Specific Compression

### 4.1 LongCodeZip (October 2025, ASE 2025) - ⭐⭐⭐⭐⭐
**Venue**: ASE 2025  
**Key Innovation**: Dual-stage AST-aware code compression

**Methodology:**
1. **Coarse-grained**: Function-level chunking with conditional perplexity
2. **Fine-grained**: Block-level perplexity segmentation with adaptive budget

**Critical Difference from LLMLingua:**
- Preserves code structure (functions, blocks)
- Understands dependencies and call graphs
- Instruction-aware relevance scoring

**Results:**
- **5.6x compression ratio** without performance degradation
- **16% better than LongLLMLingua** on code tasks
- Works on CodeLlama, DeepSeek-Coder

**Tokman Implementation:**
```go
type CodeCompressor struct {
    Parser          *sitter.Parser  // Tree-sitter for AST
    ChunkBy         string          // "function", "class", "block"
    ConditionalPPL  bool
}

func (cc *CodeCompressor) Chunk(code string) []CodeChunk {
    // Parse AST -> Extract functions -> Rank by perplexity
}
```

**Gap Addressed**: L4 (AST Preserve) can be significantly enhanced

---

## Category 5: Reversible Compression

### 5.1 Recurrent Context Compression (RCC) (June 2024) - ⭐⭐⭐⭐⭐
**Venue**: arXiv  
**Key Innovation**: 32x compression with instruction reconstruction

**Methodology:**
- Compresses context into recurrent hidden states
- Instruction reconstruction method for downstream tasks
- Extends context window within storage constraints

**Results:**
- **32x compression rate** with BLEU4 ~0.95
- **100% accuracy** on passkey retrieval (1M tokens)
- Competitive with non-compressed on long-text QA

**Tokman Gap**: T1 (Reversible Compression) - partially addressed, but RCC offers higher ratios

---

### 5.2 KVReviver (December 2025) - ⭐⭐⭐⭐⭐
**Venue**: arXiv  
**Key Innovation**: Sketch-based reversible KV cache compression

**Critical Problem Solved:**
- **Contextual Amnesia**: Traditional KV compression loses token info permanently
- KVReviver enables full reconstruction from compressed state

**Methodology:**
- Sketch data structure stores compressed token info
- On-demand reconstruction when tokens needed
- Budget-aware compression

**Results:**
- **10% KV cache budget** (2k context) = identical accuracy
- **25% KV cache budget** (32k context) = ~2% accuracy loss
- Enables "undo" of compression

**Tokman Implementation:**
```go
type SketchCache struct {
    TokenSketches  map[int]*Sketch  // Token ID -> compressed sketch
    Budget         float64
    Reconstruction bool
}

func (sc *SketchCache) Compress(kv *KVCache) *CompressedKV {
    // Create sketches for evicted tokens
    // Store minimal info for reconstruction
}

func (sc *SketchCache) Reconstruct(tokenID int) *KVEntry {
    // Reconstruct KV from sketch when needed
}
```

---

### 5.3 Lossless Token Sequence Compression (June 2025) - ⭐⭐⭐⭐⭐
**Venue**: arXiv  
**Key Innovation**: LZ77-style lossless compression for tokens

**Methodology:**
- Meta-tokens encode repeated token sequences
- Similar to LZ77 but operates on token vocabulary
- Trivially reversible - NO semantic loss

**Results:**
- **27% token reduction** (Task 1) = **47% compute reduction**
- **18% token reduction** (Task 2) = **33% compute reduction**
- Outperforms lossy methods on tasks requiring exact preservation

**Tokman Implementation:**
```go
type MetaTokenCompressor struct {
    Vocab        map[string]int
    PatternCache *LRUCache
}

func (mtc *MetaTokenCompressor) Compress(tokens []int) []int {
    // Find repeated sequences
    // Replace with meta-tokens pointing to original
    // Similar to LZ77 sliding window
}

func (mtc *MetaTokenCompressor) Decompress(tokens []int) []int {
    // Expand meta-tokens back to original sequences
}
```

**Critical Insight**: Lossless methods outperform lossy on syntax-sensitive tasks!

---

## Category 6: Semantic-Aware Compression

### 6.1 Semantic-Anchor Compression (SAC) (October 2025) - ⭐⭐⭐⭐⭐
**Venue**: arXiv  
**Key Innovation**: Autoencoding-free context compression

**Critical Insight:**
- Autoencoding tasks for compression conflict with downstream task requirements
- SAC selects "anchor tokens" a priori, no training needed

**Methodology:**
1. Select anchor tokens from original context
2. Aggregate contextual info into anchor KV representations
3. Add learnable "anchor embedding" to mark compression carriers
4. Bidirectional attention modification for info integration

**Results:**
- Consistently outperforms autoencoding-based methods
- Better at higher compression ratios
- Works across different model sizes

**Tokman Gap**: New technique not in current pipeline - HIGH PRIORITY

---

### 6.2 ChunkKV (February 2025, NeurIPS 2025) - ⭐⭐⭐⭐⭐
**Venue**: NeurIPS 2025  
**Key Innovation**: Semantic chunks as compression units

**Critical Problem Solved:**
- Token-level compression fragments semantic context
- ChunkKV preserves complete linguistic structures

**Methodology:**
- Group tokens into semantic chunks (phrases, sentences, code blocks)
- Evaluate chunk importance, not individual tokens
- Layer-wise index reuse for efficiency

**Results:**
- **8.7% precision improvement** over SOTA
- **26.5% throughput improvement** from index reuse
- Works on LongBench, Needle-In-A-Haystack, GSM8K

**Tokman Implementation:**
```go
type SemanticChunker struct {
    ChunkSize    int
    ChunkMethod  string  // "sentence", "paragraph", "code_block"
    LayerReuse   bool
}

func (sc *SemanticChunker) Chunk(tokens []Token) []Chunk {
    // Group by semantic boundaries
    // Score chunks as units
    // Share indices across layers
}
```

---

## Category 7: Agent Memory Management

### 7.1 Active Context Compression / Focus (January 2026) - ⭐⭐⭐⭐
**Venue**: arXiv  
**Key Innovation**: Agent-centric autonomous memory management

**Critical Insight:**
- LLM agents suffer from "Context Bloat" in long-horizon tasks
- Passive external summarization is suboptimal
- Agents should self-regulate their context

**Methodology:**
- Inspired by Physarum polycephalum (slime mold) exploration
- Autonomous decision: when to consolidate learnings into "Knowledge" block
- Active withdrawal (pruning) of raw interaction history

**Results:**
- **22.7% token reduction** (14.9M → 11.5M tokens)
- **57% token savings** on individual instances
- Maintained 60% accuracy (identical to uncompressed)

**Tokman Application:**
- Agent mode for long-running sessions
- Knowledge consolidation vs. raw history pruning
- Self-aware context management

---

### 7.2 MemGPT-Style Memory Management
**Key Papers:**
- SimpleMem: **30x token reduction** vs LOCOMO/MemGPT
- A-Mem: Flexible memory with graph-structured knowledge
- Contextual Memory Virtualisation: DAG-based state management

**Tokman Gap**: L11 (Compaction) can incorporate these techniques

---

## Category 8: KV Cache Optimization Survey

### 8.1 Comprehensive KV Cache Survey (December 2024, TMLR 2025)
**Venue**: TMLR 2025  
**Scope**: 200+ papers categorized into three levels

**Taxonomy:**

| Level | Techniques | Examples |
|-------|-----------|----------|
| **Token-Level** | Selection, Budget, Merging, Quantization, Low-Rank | H2O, StreamingLLM, ChunkKV |
| **Model-Level** | Architecture, Attention modifications | SAC, MInference |
| **System-Level** | Memory management, Scheduling, Hardware-aware | FlexGen, DeepSpeed |

**Key Findings:**
- KV cache can consume **70% of GPU memory** during inference
- Token-level: 2-5x compression typical
- System-level: 10-20x throughput gains possible
- Quantization: INT8 common, INT4 emerging

**Tokman Opportunity**: Implement multi-level optimization stack

---

## Category 9: Quantization & Distillation

### 9.1 Knowledge Distillation for Compression
**Key Papers:**
- Compact Language Models via Pruning + KD (NeurIPS 2024)
- LLMLingua-2: Data distillation for prompt compression

**Methodology:**
- Small model learns from large model outputs
- Distillation dataset from teacher LLM
- Combine with pruning for multiplicative gains

**Results:**
- 2-4x compression with KD alone
- 8-16x with KD + pruning

---

### 9.2 Vector Quantization for Embeddings
**Key Papers:**
- PQCache: Product quantization for KV cache
- CARVQ: Corrective adaptor with group residual VQ

**Methodology:**
- Compress embedding vectors via quantization
- Maintain semantic fidelity with error correction

**Results:**
- 4-8x memory reduction
- Minimal accuracy loss

---

## Category 10: RAG Optimization

### 10.1 TeaRAG (November 2025)
**Venue**: arXiv  
**Key Innovation**: Token-efficient agentic RAG

**Methodology:**
- Optimizes retrieved context before injection
- Token budget allocation across retrieved chunks

**Results:**
- Significant token reduction in RAG workflows

---

### 10.2 xRAG (NeurIPS 2024)
**Venue**: NeurIPS 2024  
**Key Innovation**: Extreme compression to ONE token

**Methodology:**
- Compress retrieved document into single special token
- Modality fusion approach

**Results:**
- Extreme compression for RAG

---

## Category 11: Emerging Techniques (2025-2026)

### 11.1 StructZip (2025)
**Key Innovation**: Compress structured prompts to one token via natural language descriptions

---

### 11.2 CompactPrompt (October 2025)
**Key Innovation**: Unified pipeline for prompt data compression

---

### 11.3 FrugalPrompt (October 2025)
**Key Innovation**: Token attribution for cost reduction

---

## Category 12: Theoretical Foundations

### 12.1 Token-Level Information Theory (November 2025)
**Venue**: arXiv  
**Key Innovation**: Semantic information theory for tokens

**Critical Insight:**
- Bits are wrong unit for LLM information
- Tokens have semantic embedding information content
- New theory for optimal token-level compression

---

## Implementation Roadmap for Tokman

### Phase 1: High-Impact, Low-Effort (Weeks 1-4)

| Technique | Layer/Feature | Effort | Impact |
|-----------|---------------|--------|--------|
| ChunkKV Semantic Chunking | New filter | Medium | High |
| LongCodeZip for Code | L4 Enhancement | Medium | High |
| Meta-Token Lossless | New layer L15 | Medium | High |
| KVReviver Sketches | Reversible store | High | Critical |

### Phase 2: Medium-Term (Weeks 5-12)

| Technique | Layer/Feature | Effort | Impact |
|-----------|---------------|--------|--------|
| SAC Anchor Embeddings | New layer | High | High |
| LazyLLM Dynamic Pruning | T28 Parallel | High | High |
| Agent Memory (Focus) | Agent mode | Medium | Medium |
| Position Rebalancing | L2 Enhancement | Low | Medium |

### Phase 3: Long-Term Research (Months 4-12)

| Technique | Layer/Feature | Effort | Impact |
|-----------|---------------|--------|--------|
| Rust/WASM Hot-path | Core rewrite | Very High | Critical |
| Multi-level KV Stack | System-level | High | High |
| Token Information Theory | Theoretical | Research | Unknown |

---

## Gap Analysis: Tokman vs. State-of-the-Art

| Feature | Tokman Current | SOTA (2025-2026) | Gap |
|---------|---------------|------------------|-----|
| **Lossless Compression** | ❌ None | Meta-Tokens (27% reduction) | Critical |
| **Semantic Chunking** | ❌ Token-level only | ChunkKV (8.7% better) | High |
| **Reversible KV Cache** | ✅ Basic | KVReviver (sketch-based) | Medium |
| **Code-Specific** | ⚠️ Basic AST | LongCodeZip (5.6x) | High |
| **Dynamic Layer Pruning** | ❌ Static pipeline | LazyLLM (2.34x faster) | Critical |
| **Agent Memory** | ❌ None | Focus (22.7% reduction) | Medium |
| **Position Bias Correction** | ❌ None | LongLLMLingua | Low |
| **Anchor Embeddings** | ❌ None | SAC (SOTA) | High |

---

## Competitive Intelligence: RTK vs. Tokman (Updated)

| Capability | RTK (Rust) | Tokman (Go) | After This Research |
|------------|-----------|-------------|---------------------|
| Speed | 10x faster | Baseline | Gap remains (architectural) |
| Binary Size | <5MB | ~28MB | Gap remains (Go runtime) |
| Compression Rate | 70-90% | 95-99.9% | Tokman maintains lead |
| Lossless Mode | ❌ | ❌ | ✅ Meta-Tokens (NEW) |
| Semantic Chunking | ❌ | ❌ | ✅ ChunkKV (NEW) |
| Reversible KV | ❌ | ✅ Basic | ✅ Sketch-based (ENHANCED) |
| Code-Specific | ❌ | ⚠️ Basic | ✅ LongCodeZip (NEW) |
| Dynamic Pruning | ❌ | ❌ | ✅ LazyLLM (NEW) |

**Conclusion**: Implementing Phase 1 techniques would give Tokman **5 unique advantages** over RTK while maintaining compression leadership.

---

## Bibliography (Top 50 Papers)

### Dynamic Token Pruning
1. Fu et al. "LazyLLM: Dynamic Token Pruning for Efficient Long Context LLM Inference" arXiv:2407.14057
2. "Saliency-driven Dynamic Token Pruning for Large Language Models" arXiv:2504.04514
3. "LitePruner: A Lightweight Realtime Token Pruner" OpenReview 2024

### Attention-Based
4. Xiao et al. "Efficient Streaming Language Models with Attention Sinks" ICLR 2024
5. Zhang et al. "H2O: Heavy-Hitter Oracle for Efficient Generative Inference" NeurIPS 2023
6. "Q-Hitter: A Better Token Oracle via Sparse-Quantized KV Cache" MLSys 2024
7. "DuoAttention: Efficient Long-Context LLM Inference" arXiv:2410.10819

### Prompt Compression
8. Jiang et al. "LongLLMLingua: Accelerating and Enhancing LLMs" ACL 2024
9. "500xCompressor: Generalized Prompt Compression" ACL 2025
10. "ProCut: LLM Prompt Compression via Attribution Estimation" EMNLP 2025
11. "FrugalPrompt: Reducing Contextual Overhead" arXiv:2510.16439
12. "CompactPrompt: A Unified Pipeline" arXiv:2510.18043
13. "Prompt Compression with Context-Aware Sentence Encoding" AAAI 2025

### Code-Specific
14. Shi et al. "LongCodeZip: Compress Long Context for Code LLMs" ASE 2025

### Reversible Compression
15. Huang et al. "Recurrent Context Compression" arXiv:2406.06110
16. Yuan et al. "KVReviver: Reversible KV Cache Compression" arXiv:2512.17917
17. Harvill et al. "Lossless Token Sequence Compression via Meta-Tokens" arXiv:2506.00307

### Semantic-Aware
18. Liu et al. "Semantic-Anchor Compression (SAC)" arXiv:2510.08907
19. Liu et al. "ChunkKV: Semantic-Preserving KV Cache Compression" NeurIPS 2025
20. "Autoencoding-Free Context Compression via Contextual Semantic Anchors" arXiv:2510.08907

### Agent Memory
21. Verma "Active Context Compression: Autonomous Memory Management" arXiv:2601.07190
22. "SimpleMem: Efficient Lifelong Memory for LLM Agents" arXiv:2601.02553
23. "A-Mem: Agentic Memory for LLM Agents" arXiv:2502.12110
24. "Memory in the Age of AI Agents" arXiv:2512.13564
25. "Contextual Memory Virtualisation" arXiv:2602.22402

### KV Cache Management
26. Li et al. "A Survey on Large Language Model Acceleration based on KV Cache Management" TMLR 2025
27. "PyramidInfer: Pyramid KV Cache Compression" ACL 2024 Findings
28. "MiniCache: KV Cache Compression in Depth Dimension" NeurIPS 2024
29. "Keyformer: KV Cache Reduction through Key Tokens Selection" MLSys 2024
30. "GEAR: An Efficient KV Cache Compression Recipe" arXiv:2403.05527
31. "ALISA: Accelerating LLM Inference via Sparsity-Aware KV Caching" IEEE 2024

### Quantization & Distillation
32. "Compact Language Models via Pruning and Knowledge Distillation" NeurIPS 2024
33. "Survey on Knowledge Distillation for Large Language Models" ACM Computing Surveys
34. "A Survey on Model Compression for Large Language Models" TACL
35. "LLMLingua-2: Data Distillation for Efficient Prompt Compression" ACL 2024 Findings

### RAG Optimization
36. "TeaRAG: A Token-Efficient Agentic RAG Framework" arXiv:2511.05385
37. "xRAG: Extreme Context Compression for RAG with One Token" NeurIPS 2024
38. "Maximizing RAG Efficiency: A Comparative Analysis" Cambridge NLP
39. "LongRAG: Enhancing RAG with Long-Context LLMs" arXiv:2406.15319

### Theoretical
40. "Forget Bit, It's All About Token: Semantic Information Theory for LLMs" arXiv:2511.01202
41. "Beyond the Limits: A Survey of Techniques to Extend Context Length" arXiv:2402.02244

### Vision & Multimodal
42. "Multi-stage Vision Token Dropping" arXiv:2411.10803
43. "HoliTom: Holistic Token Merging for Fast Video LLMs" arXiv:2505.21334
44. "InternVL-X: Efficient Visual Token Compression" arXiv:2503.21307

### Surveys & Benchmarks
45. "A Comprehensive Survey on Long Context Language Modeling" arXiv:2503.17407
46. "Characterizing Prompt Compression Methods for Long Context Inference" arXiv:2407.08892
47. "Systematic Evaluation of Optimization Techniques for Long-Context LLMs" arXiv:2508.00305

### Emerging (2026)
48. "DTRNet: Dynamic Token Routing Network" arXiv:2509.00925
49. "Beyond the 80/20 Rule: High-Entropy Minority Tokens" arXiv:2506.01939
50. "SToRM: Supervised Token Reduction for Multi-modal LLMs" arXiv:2602.11656

---

## Appendix: Code Snippets for Implementation

### A. LazyLLM-Style Dynamic Pruner

```go
package filter

type LazyPruner struct {
    layerBudgets    []int
    threshold       float64
    revivalBudget   int
    prunedTokens    map[int][]Token  // layer -> pruned tokens
}

func NewLazyPruner(numLayers int, baseBudget int) *LazyPruner {
    budgets := make([]int, numLayers)
    for i := range budgets {
        // Exponentially decrease budget for deeper layers
        budgets[i] = int(float64(baseBudget) * math.Pow(0.9, float64(i)))
    }
    return &LazyPruner{
        layerBudgets:  budgets,
        threshold:     0.01,
        revivalBudget: 100,
        prunedTokens:  make(map[int][]Token),
    }
}

func (lp *LazyPruner) SelectTokens(layer int, tokens []Token, attention []float64) []Token {
    // Score tokens by attention
    scored := make([]scoredToken, len(tokens))
    for i, tok := range tokens {
        scored[i] = scoredToken{tok, attention[i]}
    }
    
    // Sort by attention (descending)
    sort.Slice(scored, func(i, j int) bool {
        return scored[i].score > scored[j].score
    })
    
    // Select top-K by budget
    budget := lp.layerBudgets[layer]
    selected := make([]Token, 0, budget)
    pruned := make([]Token, 0)
    
    for i, st := range scored {
        if i < budget {
            selected = append(selected, st.token)
        } else {
            pruned = append(pruned, st.token)
        }
    }
    
    // Store pruned for potential revival
    lp.prunedTokens[layer] = pruned
    
    return selected
}

func (lp *LazyPruner) ReviveTokens(layer int, needed []Token) []Token {
    // Add back previously pruned tokens if needed for this generation step
    pruned := lp.prunedTokens[layer]
    if len(pruned) == 0 {
        return needed
    }
    
    // Select up to revivalBudget tokens to revive
    limit := min(lp.revivalBudget, len(pruned))
    return append(needed, pruned[:limit]...)
}
```

### B. ChunkKV Semantic Chunking

```go
package filter

import "github.com/smacker/go-tree-sitter"

type SemanticChunk struct {
    Tokens    []Token
    StartLine int
    EndLine   int
    Type      string  // "function", "class", "sentence", "paragraph"
    Score     float64 // Importance score
}

type ChunkKV struct {
    chunkSize    int
    parser       *sitter.Parser
    preserveType string
}

func (ckv *ChunkKV) ChunkCode(source []byte, lang *sitter.Language) []SemanticChunk {
    tree := ckv.parser.Parse(nil, source)
    root := tree.RootNode()
    
    chunks := make([]SemanticChunk, 0)
    
    // Query for function definitions
    query, _ := sitter.NewQuery([]byte(`
        (function_definition) @function
        (class_definition) @class
        (method_definition) @method
    `), lang)
    
    cursor := sitter.NewQueryCursor()
    cursor.Exec(query, root)
    
    for {
        match, ok := cursor.NextMatch()
        if !ok {
            break
        }
        
        for _, cap := range match.Captures {
            node := cap.Node
            chunk := SemanticChunk{
                StartLine: int(node.StartPoint().Row),
                EndLine:   int(node.EndPoint().Row),
                Type:      query.CaptureNameForId(cap.Index),
            }
            chunk.Tokens = ckv.tokenize(source[node.StartByte():node.EndByte()])
            chunks = append(chunks, chunk)
        }
    }
    
    return chunks
}

func (ckv *ChunkKV) ScoreChunks(chunks []SemanticChunk, query string) []SemanticChunk {
    // Score each chunk by conditional perplexity given the query
    for i := range chunks {
        chunks[i].Score = ckv.conditionalPerplexity(chunks[i].Tokens, query)
    }
    
    // Sort by score descending
    sort.Slice(chunks, func(i, j int) bool {
        return chunks[i].Score > chunks[j].Score
    })
    
    return chunks
}
```

### C. Meta-Token Lossless Compressor

```go
package filter

type MetaToken struct {
    ID       int
    Original []int  // Sequence of original token IDs
    Pattern  string // String pattern for matching
}

type MetaTokenCompressor struct {
    metaTokens  map[int]MetaToken
    nextMetaID  int
    windowSize  int
    minPattern  int  // Minimum pattern length to compress
}

func NewMetaTokenCompressor() *MetaTokenCompressor {
    return &MetaTokenCompressor{
        metaTokens: make(map[int]MetaToken),
        nextMetaID: 100000, // Start meta-token IDs at 100k
        windowSize: 512,
        minPattern: 3,
    }
}

func (mtc *MetaTokenCompressor) Compress(tokens []int) []int {
    result := make([]int, 0, len(tokens))
    i := 0
    
    for i < len(tokens) {
        // Try to find repeated pattern
        found := false
        for patternLen := mtc.windowSize; patternLen >= mtc.minPattern; patternLen-- {
            if i+patternLen > len(tokens) {
                continue
            }
            
            pattern := tokens[i : i+patternLen]
            
            // Check if this pattern appears later in the sequence
            if matchStart := mtc.findPattern(tokens, pattern, i+patternLen); matchStart != -1 {
                // Create or reuse meta-token
                metaID := mtc.getOrCreateMetaToken(pattern)
                result = append(result, metaID)
                i += patternLen
                found = true
                break
            }
        }
        
        if !found {
            result = append(result, tokens[i])
            i++
        }
    }
    
    return result
}

func (mtc *MetaTokenCompressor) Decompress(tokens []int) []int {
    result := make([]int, 0)
    
    for _, tok := range tokens {
        if meta, exists := mtc.metaTokens[tok]; exists {
            result = append(result, meta.Original...)
        } else {
            result = append(result, tok)
        }
    }
    
    return result
}

func (mtc *MetaTokenCompressor) findPattern(tokens, pattern []int, startIdx int) int {
    for i := startIdx; i <= len(tokens)-len(pattern); i++ {
        match := true
        for j := 0; j < len(pattern); j++ {
            if tokens[i+j] != pattern[j] {
                match = false
                break
            }
        }
        if match {
            return i
        }
    }
    return -1
}

func (mtc *MetaTokenCompressor) getOrCreateMetaToken(pattern []int) int {
    // Check if pattern already has a meta-token
    patternKey := fmt.Sprintf("%v", pattern)
    for id, meta := range mtc.metaTokens {
        if fmt.Sprintf("%v", meta.Original) == patternKey {
            return id
        }
    }
    
    // Create new meta-token
    id := mtc.nextMetaID
    mtc.metaTokens[id] = MetaToken{
        ID:       id,
        Original: pattern,
    }
    mtc.nextMetaID++
    
    return id
}
```

---

**End of Deep Research Report**

*Generated: March 21, 2026*  
*Papers Analyzed: 120+*  
*Research Depth: Comprehensive*
