# TokMan Architecture

Deep dive into the 14-layer token compression pipeline.

## Overview

TokMan uses a multi-layer approach to token reduction, where each layer applies a specific optimization technique. The layers are ordered by computational cost and impact, with lightweight operations first.

```
Input Text
    │
    ▼
┌─────────────────────────────────────┐
│  Layer 1:  Whitespace Normalization │
│  Layer 2:  Stopword Removal         │
│  Layer 3:  Punctuation Normalization│
│  Layer 4:  Case Normalization       │
│  Layer 5:  Number Abbreviation      │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│  Layer 6:  Repetition Detection     │
│  Layer 7:  Template Matching        │
│  Layer 8:  Semantic Compression     │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│  Layer 9:  Structure Analysis       │
│  Layer 10: Context Awareness        │
│  Layer 11: Importance Scoring       │
└─────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────┐
│  Layer 12: Entropy Optimization     │
│  Layer 13: Huffman Encoding         │
│  Layer 14: Adaptive Tuning          │
└─────────────────────────────────────┘
    │
    ▼
Compressed Output
```

## Layer Details

### Layer 1: Whitespace Normalization

**Technique**: Remove redundant whitespace, normalize line breaks.

**Implementation**:
- Collapse multiple spaces to single space
- Remove trailing whitespace
- Normalize tabs to spaces
- Remove empty lines (configurable)

**Typical Reduction**: 5-10%

**Example**:
```
Input:  "Hello    world  \n\n\n  Test"
Output: "Hello world\nTest"
```

---

### Layer 2: Stopword Removal

**Technique**: Remove common words that add little semantic value.

**Implementation**:
- Configurable stopword list
- Content-aware removal (preserves stopwords in questions)
- Position-aware (keeps first/last sentence stopwords)

**Typical Reduction**: 10-20%

**Example**:
```
Input:  "The quick brown fox jumps over the lazy dog"
Output: "quick brown fox jumps lazy dog"
```

---

### Layer 3: Punctuation Normalization

**Technique**: Standardize and reduce punctuation.

**Implementation**:
- Remove duplicate punctuation
- Standardize quotes
- Remove unnecessary punctuation in lists

**Typical Reduction**: 2-5%

---

### Layer 4: Case Normalization

**Technique**: Normalize text casing.

**Implementation**:
- Lowercase conversion (preserves proper nouns)
- Title case detection
- Acronym preservation

**Typical Reduction**: 1-3%

---

### Layer 5: Number Abbreviation

**Technique**: Abbreviate large numbers and statistics.

**Implementation**:
- 1,000 → 1K
- 1,000,000 → 1M
- Percentage normalization

**Typical Reduction**: 3-8%

**Example**:
```
Input:  "The project cost 1,500,000 dollars"
Output: "Project cost 1.5M dollars"
```

---

### Layer 6: Repetition Detection

**Technique**: Identify and deduplicate repeated n-grams.

**Implementation**:
- N-gram frequency analysis
- Template extraction
- Duplicate sentence removal

**Typical Reduction**: 5-15%

**Example**:
```
Input:  "The fox jumped. The fox jumped. The fox jumped."
Output: "The fox jumped [x3]"
```

---

### Layer 7: Template Matching

**Technique**: Identify and compress common patterns.

**Implementation**:
- Regex-based pattern matching
- Log format detection
- Code structure recognition

**Typical Reduction**: 10-30%

**Example**:
```
Input:  "[2024-01-01 10:00:00] INFO: Server started"
Output: "[TIMESTAMP] INFO: Server started"
```

---

### Layer 8: Semantic Compression

**Technique**: Meaning-preserving word substitution.

**Implementation**:
- Synonym replacement
- Phrase contraction
- Verbose phrase detection

**Typical Reduction**: 15-25%

**Example**:
```
Input:  "In order to accomplish this task"
Output: "To do this"
```

---

### Layer 9: Structure Analysis

**Technique**: Analyze document structure for optimization.

**Implementation**:
- Header/section detection
- List compression
- Table summarization

**Typical Reduction**: 5-15%

---

### Layer 10: Context Awareness

**Technique**: Preserve context-critical information.

**Implementation**:
- Entity recognition
- Quote preservation
- Question/answer structure

**Typical Reduction**: 10-20%

---

### Layer 11: Importance Scoring

**Technique**: TF-IDF based sentence ranking.

**Implementation**:
- Term frequency analysis
- Inverse document frequency
- Sentence importance scoring

**Typical Reduction**: 15-25%

**Example**:
```
Input:  Long document with key points buried
Output: Key points extracted, less important sentences compressed
```

---

### Layer 12: Entropy Optimization

**Technique**: Information theory-based compression.

**Implementation**:
- Shannon entropy calculation
- Information density optimization
- Redundancy elimination

**Typical Reduction**: 10-20%

---

### Layer 13: Huffman Encoding

**Technique**: Variable-length encoding for common patterns.

**Implementation**:
- Frequency-based encoding
- Pattern dictionary
- Decode table preservation

**Typical Reduction**: 5-15%

---

### Layer 14: Adaptive Tuning

**Technique**: Dynamic layer configuration based on content.

**Implementation**:
- Content type detection
- Layer weight adjustment
- Budget-aware processing

**Typical Reduction**: Variable

## Pipeline Configuration

### Mode Presets

**Conservative**:
- Layers 1-5 fully enabled
- Layers 6-14 partial
- Preserves all critical information
- Best for: Code, technical docs

**Balanced** (default):
- All layers enabled
- Moderate thresholds
- Good balance of compression and preservation
- Best for: General text, conversations

**Aggressive**:
- All layers at maximum
- Higher thresholds
- Maximum compression
- Best for: Logs, repetitive content

### Custom Configuration

```go
config := filter.PipelineConfig{
    Mode:               filter.ModeBalanced,
    Budget:             2000,
    SessionTracking:    true,
    NgramEnabled:       true,
    EnableCompaction:   true,
    EnableH2O:          true,
    EnableAttentionSink: true,
}
```

## Performance Characteristics

| Metric | Value |
|--------|-------|
| Latency (P50) | < 5ms |
| Latency (P99) | < 50ms |
| Throughput | 100K+ tokens/sec |
| Memory (1M tokens) | ~500MB |
| Memory (2M tokens) | ~1GB |

## Memory Optimization

For large contexts (1M+ tokens), TokMan uses:

1. **Streaming**: Process in chunks, not full buffer
2. **Lazy evaluation**: Only compute what's needed
3. **Memory pooling**: Reuse buffers across requests
4. **Garbage collection hints**: Optimize for throughput
