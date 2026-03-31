# TokMan

> Token-aware CLI proxy for AI coding assistants. 31-layer compression pipeline built on research from 120+ papers.

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-42%20packages-green)](#)
[![Features](https://img.shields.io/badge/Features-200%2B%20from%2010%20repos-orange)](#)

## What is TokMan?

TokMan intercepts CLI commands and applies a **31-layer compression pipeline** to reduce token usage for AI coding assistants. It achieves **60–90% token reduction** on common development operations.

```
Input:  10,000 tokens  ──►  TokMan  ──►  Output:  1,500 tokens
                                        Savings:   85%
                                        Cost:      $0.085 → $0.013
```

## Quick Start

```bash
# Install
go install github.com/GrayCodeAI/tokman/cmd/tokman@latest

# Compress command output
tokman summary --preset full < input.txt

# Start HTTP proxy (works with any AI agent)
tokman http-proxy start --listen :8080

# Analytics dashboard
tokman tui
```

## Core Features

### Compression Pipeline
| Layer | Technique | Research | Savings |
|-------|-----------|----------|---------|
| L1 | Entropy Filtering | Selective Context (Mila 2023) | 2–3× |
| L2 | Perplexity Pruning | LLMLingua (Microsoft 2023) | 20× |
| L3 | Goal-Driven Selection | SWE-Pruner (2025) | 14.8× |
| L4 | AST Preservation | LongCodeZip (NUS 2025) | 4–8× |
| L5 | Contrastive Ranking | LongLLMLingua (2024) | 4–10× |
| L6–L20 | 15 additional layers | Various | — |
| L21–L31 | Phase 2 layers | Mercury, SemantiCache, SCOPE | — |

### HTTP Proxy Mode
Transparent proxy that intercepts LLM API calls — works with **any** AI agent without hooks.

```bash
tokman http-proxy start --listen :8080
# Then point your AI agent to http://localhost:8080
```

### Analytics & Monitoring
```bash
tokman tui              # Interactive dashboard
tokman analytics --action anomaly   # Anomaly detection
tokman analytics --action forecast  # Spend forecasting
tokman analytics --action heatmap   # Token breakdown
```

### AI Gateway
```bash
tokman gateway --action set-quota --model gpt-4 --quota 10000
tokman gateway --action alias --model gpt-4 --alias gpt-4o-mini
tokman gateway --action kill-switch --model gpt-4 --kill-switch
```

### Security
```bash
tokman security --action scan < input.txt    # Scan for vulnerabilities
tokman security --action redact < input.txt  # Redact PII
```

## Architecture

```
┌─────────────┐    ┌──────────────────┐    ┌──────────────┐
│ CLI Command │───►│ 31-Layer Pipeline│───►│ Compressed   │
│ (git, npm,  │    │ (filter/)        │    │ Output       │
│  docker...) │    │                  │    │              │
└─────────────┘    └──────────────────┘    └──────────────┘
                          │
                    ┌─────┴─────┐
                    │ HTTP Proxy│
                    │ (proxy/)  │
                    └───────────┘
```

## Performance

| Metric | Value |
|--------|-------|
| Compression | 60–90% on common dev operations |
| Tokenizer | BPE (tiktoken cl100k_base) |
| Cache | O(1) LRU with fingerprinting |
| SIMD | Go 1.26+ vectorized |
| Pipeline | 31 layers, parallel execution |

## Supported AI Agents

Claude Code · Cursor · Copilot · Gemini CLI · Windsurf · Cline · Codex · OpenCode · Aider · and 10+ more

## License

MIT — see [LICENSE](LICENSE)
