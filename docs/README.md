# TokMan Documentation

> **TokMan** - The World's Best Token Reduction System

A research-backed, production-ready token compression system with 14 layers of optimization.

## Quick Links

- [Getting Started](./getting-started.md) - Install and run TokMan in 5 minutes
- [API Reference](./api-reference.md) - Complete REST API documentation
- [SDK Reference](./sdk-reference.md) - Go, Python, TypeScript SDKs
- [Architecture](./architecture.md) - Deep dive into the 14-layer pipeline
- [Benchmarks](./BENCHMARKS.md) - Performance comparisons
- [Examples](./examples/) - Integration examples

## Features

- 🚀 **14 Research-Backed Layers** - Each layer implements proven compression techniques
- 🎯 **Adaptive Compression** - Auto-detects content type and optimizes accordingly
- 📊 **2M+ Token Support** - Handle massive contexts with streaming
- ⚡ **<10ms Latency** - Optimized for real-time applications
- 🔧 **Multiple SDKs** - Go, Python, TypeScript/Node.js
- 🌐 **REST API** - Language-agnostic HTTP interface
- 📈 **Prometheus Metrics** - Built-in observability

## Installation

### Binary

```bash
# Download from releases
curl -sL https://github.com/GrayCodeAI/tokman/releases/latest/download/tokman-linux-amd64 -o tokman
chmod +x tokman
./tokman
```

### Docker

```bash
docker run -p 8080:8080 graycode/tokman:latest
```

### From Source

```bash
git clone https://github.com/GrayCodeAI/tokman.git
cd tokman
go build -o tokman ./cmd/server
./tokman
```

## Quick Start

```bash
# Start the server
./tokman

# Compress text
curl -X POST http://localhost:8080/compress \
  -H "Content-Type: application/json" \
  -d '{"input": "Your long text here..."}'
```

## SDK Usage

### Go

```go
import "github.com/GrayCodeAI/tokman/sdk/go/tokman"

client := tokman.NewClient("http://localhost:8080")
result, _ := client.Compress("long text...")
fmt.Printf("Reduced from %d to %d tokens\n", result.OriginalTokens, result.FinalTokens)
```

### Python

```python
from tokman import TokMan

client = TokMan(base_url="http://localhost:8080")
result = client.compress("long text...")
print(f"Reduced from {result.original_tokens} to {result.final_tokens} tokens")
```

### TypeScript

```typescript
import { TokMan } from '@tokman/sdk';

const client = new TokMan();
const result = await client.compress('long text...');
console.log(`Reduced from ${result.originalTokens} to ${result.finalTokens} tokens`);
```

## Architecture

TokMan uses a 14-layer compression pipeline:

| Layer | Method | Typical Reduction |
|-------|--------|-------------------|
| 1. Whitespace | Remove redundant spaces | 5-10% |
| 2. Stopwords | Remove common words | 10-20% |
| 3. Punctuation | Normalize punctuation | 2-5% |
| 4. Case | Normalize casing | 1-3% |
| 5. Numbers | Abbreviate numbers | 3-8% |
| 6. Repetition | Deduplicate n-grams | 5-15% |
| 7. Template | Pattern matching | 10-30% |
| 8. Semantic | Meaning-preserving | 15-25% |
| 9. Structure | Structural analysis | 5-15% |
| 10. Context | Context-aware | 10-20% |
| 11. Importance | TF-IDF scoring | 15-25% |
| 12. Entropy | Information theory | 10-20% |
| 13. Huffman | Encoding optimization | 5-15% |
| 14. Adaptive | Dynamic tuning | Variable |

## Benchmarks

See [BENCHMARKS.md](./BENCHMARKS.md) for detailed comparisons with:
- LLMLingua
- MemGPT
- Native LLM compression

## License

MIT © GrayCodeAI
