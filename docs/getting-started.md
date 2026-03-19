# Getting Started with TokMan

This guide will help you get TokMan up and running in under 5 minutes.

## Prerequisites

- Go 1.21+ (for building from source)
- Docker (optional, for containerized deployment)
- 4GB+ RAM (for 2M token contexts)

## Installation

### Option 1: Download Binary

```bash
# Linux
curl -sL https://github.com/GrayCodeAI/tokman/releases/latest/download/tokman-linux-amd64 -o tokman
chmod +x tokman

# macOS
curl -sL https://github.com/GrayCodeAI/tokman/releases/latest/download/tokman-darwin-amd64 -o tokman
chmod +x tokman

# Windows
curl -sL https://github.com/GrayCodeAI/tokman/releases/latest/download/tokman-windows-amd64.exe -o tokman.exe
```

### Option 2: Docker

```bash
docker pull graycode/tokman:latest
docker run -p 8080:8080 graycode/tokman:latest
```

### Option 3: Build from Source

```bash
git clone https://github.com/GrayCodeAI/tokman.git
cd tokman
go build -o tokman ./cmd/server
```

## Starting the Server

```bash
# Default port 8080
./tokman

# Custom port
PORT=9090 ./tokman

# With debug logging
LOG_LEVEL=debug ./tokman
```

## Your First Compression

### Using cURL

```bash
curl -X POST http://localhost:8080/compress \
  -H "Content-Type: application/json" \
  -d '{
    "input": "The quick brown fox jumps over the lazy dog. The quick brown fox jumps over the lazy dog. The quick brown fox jumps over the lazy dog."
  }'
```

Response:
```json
{
  "output": "quick brown fox jumps over lazy dog",
  "original_tokens": 27,
  "final_tokens": 6,
  "tokens_saved": 21,
  "reduction_percent": 77.78,
  "processing_time_ms": 2
}
```

### Using the Go SDK

```go
package main

import (
    "fmt"
    "github.com/GrayCodeAI/tokman/sdk/go/tokman"
)

func main() {
    client := tokman.NewClient("http://localhost:8080")
    
    result, err := client.Compress("Your long text here...")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Original: %d tokens\n", result.OriginalTokens)
    fmt.Printf("Compressed: %d tokens\n", result.FinalTokens)
    fmt.Printf("Saved: %.1f%%\n", result.ReductionPercent)
}
```

### Using the Python SDK

```python
from tokman import TokMan

client = TokMan(base_url="http://localhost:8080")

result = client.compress("Your long text here...")
print(f"Original: {result.original_tokens} tokens")
print(f"Compressed: {result.final_tokens} tokens")
print(f"Saved: {result.reduction_percent:.1f}%")
```

### Using TypeScript

```typescript
import { TokMan } from '@tokman/sdk';

const client = new TokMan({ baseUrl: 'http://localhost:8080' });

const result = await client.compress('Your long text here...');
console.log(`Original: ${result.originalTokens} tokens`);
console.log(`Compressed: ${result.finalTokens} tokens`);
console.log(`Saved: ${result.reductionPercent.toFixed(1)}%`);
```

## Compression Modes

TokMan supports three compression modes:

| Mode | Use Case | Typical Reduction |
|------|----------|-------------------|
| `conservative` | Preserve all critical info | 10-25% |
| `balanced` | General purpose (default) | 25-45% |
| `aggressive` | Maximum compression | 45-70% |

```bash
# Balanced mode (default)
curl -X POST http://localhost:8080/compress \
  -H "Content-Type: application/json" \
  -d '{"input": "text", "mode": "balanced"}'

# Aggressive mode
curl -X POST http://localhost:8080/compress \
  -H "Content-Type: application/json" \
  -d '{"input": "text", "mode": "aggressive"}'
```

## Adaptive Compression

Let TokMan auto-detect content type and optimize:

```bash
curl -X POST http://localhost:8080/compress/adaptive \
  -H "Content-Type: application/json" \
  -d '{"input": "function main() { return 42; }"}'
```

## Health Check

```bash
curl http://localhost:8080/health
# {"status":"ok","version":"1.2.0"}
```

## Metrics

```bash
# JSON stats
curl http://localhost:8080/stats

# Prometheus format
curl http://localhost:8080/metrics
```

## Next Steps

- [API Reference](./api-reference.md) - Full API documentation
- [Architecture](./architecture.md) - Understanding the 14-layer pipeline
- [Examples](./examples/) - Integration with LangChain, LlamaIndex
