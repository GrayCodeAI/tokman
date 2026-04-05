# TokMan

> Token-aware CLI proxy & AI gateway for coding assistants. 31-layer compression pipeline built on 30+ research papers, achieving 60–90% token savings.

[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Tests](https://img.shields.io/badge/Tests-163%2B%20tests-green)](#)
[![Packages](https://img.shields.io/badge/Packages-150%2B%20internal-blue)](#)
[![Discord](https://img.shields.io/discord/1470188214710046894?label=Discord&logo=discord)](https://discord.gg/HrVA7ePyV)

[Website](https://tokman.dev) · [Install](#installation) · [Quick Start](#quick-start) · [Discord](https://discord.gg/HrVA7ePyV) · [Architecture](#architecture) · [Contributing](#contributing)

---

TokMan intercepts CLI commands and applies a **31-layer compression pipeline** to reduce token usage for AI coding assistants. Built on research from 30+ papers, it achieves **60–90% token reduction** on common development operations — far beyond simple output filtering.

```
Input:  10,000 tokens  ──►  TokMan  ──►  Output:  1,500 tokens
                                        Savings:   85%
                                        Cost:      $0.085 → $0.013
```

## Token Savings (30-min Claude Code Session)

| Operation | Frequency | Standard | TokMan | Savings |
|-----------|-----------|----------|--------|---------|
| `ls` / `tree` | 10× | 2,000 | 400 | **−80%** |
| `cat` / `read` | 20× | 40,000 | 12,000 | **−70%** |
| `grep` / `rg` | 8× | 16,000 | 3,200 | **−80%** |
| `git status` | 10× | 3,000 | 600 | **−80%** |
| `git diff` | 5× | 10,000 | 2,500 | **−75%** |
| `git log` | 5× | 2,500 | 500 | **−80%** |
| `git add/commit/push` | 8× | 1,600 | 120 | **−92%** |
| `npm test` / `cargo test` | 5× | 25,000 | 2,500 | **−90%** |
| `ruff check` | 3× | 3,000 | 600 | **−80%** |
| `pytest` | 4× | 8,000 | 800 | **−90%** |
| `go test` | 3× | 6,000 | 600 | **−90%** |
| `docker ps` | 3× | 900 | 180 | **−80%** |
| **Total** | | **~118,000** | **~23,500** | **−80%** |

> Estimates based on medium-sized projects. Actual savings vary by project size and command complexity.

## Cost Reduction

| Session | Without TokMan | With TokMan | Saved |
|---------|---------------|-------------|-------|
| 30 min (Claude) | ~$0.50 | ~$0.08 | **84%** |
| 1 hour (GPT-4o) | ~$1.20 | ~$0.18 | **85%** |
| Daily (heavy) | ~$5.00 | ~$0.75 | **85%** |
| Monthly (team/5) | ~$750 | ~$112 | **$638** |

## Installation

### Go Install (recommended)

```bash
go install github.com/GrayCodeAI/tokman/cmd/tokman@latest
```

### Build from Source

```bash
git clone https://github.com/GrayCodeAI/tokman.git
cd tokman
make build
```

### SIMD-Optimized Build

```bash
make build-simd
```

### Verify Installation

```bash
tokman --version
tokman gain   # Show token savings stats
```

## Quick Start

```bash
# 1. Install for your AI tool
tokman init -g                     # Claude Code / Copilot (default)
tokman init -g --gemini            # Gemini CLI
tokman init -g --codex             # Codex (OpenAI)
tokman init --agent cursor         # Cursor
tokman init --agent windsurf       # Windsurf
tokman init --agent cline          # Cline / Roo Code

# 2. Restart your AI tool, then test
git status  # Automatically rewritten to tokman git status

# 3. Or use standalone
tokman summary --preset full < input.txt
tokman http-proxy start --listen :8080
tokman tui                         # Analytics dashboard
```

## How It Works

```
  Without TokMan:                              With TokMan:

  Claude  --git status-->  shell  -->  git    Claude  --git status-->  TokMan  -->  git
    ^                                   |        ^                      |           |
    |       ~2,000 tokens (raw)         |        |    ~200 tokens       | 31-layer  |
    +-----------------------------------+        +---- (filtered) ------+ pipeline  +
```

Unlike simple output filters (dedup + truncation), TokMan applies a **31-layer research-backed compression pipeline** with semantic understanding, AST preservation, goal-driven selection, and inter-layer feedback.

## Commands

### Files

```bash
tokman ls .                        # Token-optimized directory tree
tokman read file.go                # Smart file reading
tokman read file.go -l aggressive  # Signatures only (strips bodies)
tokman read file.go --mode graph   # Target file + related project files
tokman ctx read file.go --mode auto --max-tokens 300
tokman ctx delta file.go           # Show change since last snapshot
tokman ctx read file.go --mode graph --related-files 4
tokman smart file.go               # 2-line heuristic code summary
tokman find "*.go" .               # Compact find results
tokman grep "pattern" .            # Grouped search results
tokman diff file1 file2            # Condensed diff
```

Context workflow:

```bash
# Budget a single file for an agent
tokman ctx read internal/server/server.go --mode auto --max-tokens 350

# Show only what changed since the last snapshot
tokman ctx delta internal/server/server.go

# Pull related files into the same context bundle
tokman ctx read internal/server/server.go --mode graph --related-files 4
```

TokMan now tracks and visualizes smart context delivery separately from normal command filtering, including:
- direct file reads
- delta reads
- MCP-served reads
- top files and projects by context savings
- graph-aware MCP bundles for agent context delivery
- structured smart-read analytics by kind, mode, target, and bundle quality

### Git

```bash
tokman git status                  # Compact status
tokman git log -n 10               # One-line commits
tokman git diff                    # Condensed diff
tokman git add                     # → "ok"
tokman git commit -m "msg"         # → "ok abc1234"
tokman git push                    # → "ok main"
tokman git pull                    # → "ok 3 files +10 -2"
```

### GitHub CLI

```bash
tokman gh pr list                  # Compact PR listing
tokman gh pr view 42               # PR details + checks
tokman gh issue list               # Compact issue listing
tokman gh run list                 # Workflow run status
```

### Test Runners

```bash
tokman test cargo test             # Show failures only (-90%)
tokman err npm run build           # Errors/warnings only
tokman vitest run                  # Vitest compact (failures only)
tokman playwright test             # E2E results (failures only)
tokman pytest                      # Python tests (-90%)
tokman go test                     # Go tests (NDJSON, -90%)
tokman cargo test                  # Cargo tests (-90%)
tokman rake test                   # Ruby minitest (-90%)
tokman rspec                       # RSpec tests (JSON, -60%+)
```

### Build & Lint

```bash
tokman lint                        # ESLint grouped by rule/file
tokman lint biome                  # Supports other linters
tokman tsc                         # TypeScript errors grouped by file
tokman next build                  # Next.js build compact
tokman prettier --check .          # Files needing formatting
tokman cargo build                 # Cargo build (-80%)
tokman cargo clippy                # Cargo clippy (-80%)
tokman ruff check                  # Python linting (JSON, -80%)
tokman golangci-lint run           # Go linting (JSON, -85%)
tokman rubocop                     # Ruby linting (JSON, -60%+)
```

### Package Managers

```bash
tokman pnpm list                   # Compact dependency tree
tokman pip list                    # Python packages (auto-detect uv)
tokman pip outdated                # Outdated packages
tokman bundle install              # Ruby gems (strip Using lines)
tokman prisma generate             # Schema generation (no ASCII art)
```

### Containers

```bash
tokman docker ps                   # Compact container list
tokman docker images               # Compact image list
tokman docker logs <container>     # Deduplicated logs
tokman docker compose ps           # Compose services
tokman kubectl pods                # Compact pod list
tokman kubectl logs <pod>          # Deduplicated logs
tokman kubectl services            # Compact service list
```

### Data & Analytics

```bash
tokman json config.json            # Structure without values
tokman deps                        # Dependencies summary
tokman env -f AWS                  # Filtered env vars
tokman log app.log                 # Deduplicated logs
tokman curl <url>                  # Auto-detect JSON + schema
tokman wget <url>                  # Download, strip progress bars
tokman summary <long command>      # Heuristic summary
tokman proxy <command>             # Raw passthrough + tracking
```

### Token Savings Analytics

```bash
tokman gain                        # Summary stats
tokman gain --graph                # ASCII graph (last 30 days)
tokman gain --history              # Recent command history
tokman gain --daily                # Day-by-day breakdown
tokman gain --all --format json    # JSON export for dashboards

tokman discover                    # Find missed savings opportunities
tokman discover --all --since 7    # All projects, last 7 days

tokman session                     # Show TokMan adoption across recent sessions
```

## Global Flags

```bash
-u, --ultra-compact    # ASCII icons, inline format (extra token savings)
-v, --verbose          # Increase verbosity (-v, -vv, -vvv)
```

## Examples

**Directory listing:**

```
# ls -la (45 lines, ~800 tokens)         # tokman ls (12 lines, ~150 tokens)
drwxr-xr-x  15 user staff 480 ...        my-project/
-rw-r--r--   1 user staff 1234 ...        +-- src/ (8 files)
...                                       |   +-- main.go
                                          +-- go.mod
```

**Git operations:**

```
# git push (15 lines, ~200 tokens)        # tokman git push (1 line, ~10 tokens)
Enumerating objects: 5, done.              ok main
Counting objects: 100% (5/5), done.
Delta compression using up to 8 threads
...
```

**Test output:**

```
# cargo test (200+ lines on failure)      # tokman test cargo test (~20 lines)
running 15 tests                           FAILED: 2/15 tests
test utils::test_parse ... ok                test_edge_case: assertion failed
test utils::test_format ... ok               test_overflow: panic at utils.rs:18
...
```

## Compression Pipeline

TokMan runs a **31-layer compression pipeline** with stage gates, early-exit, and adaptive selection.

| # | Layer | Technique | Research Paper | Savings |
|---|-------|-----------|----------------|---------|
| L0 | TOML Filter | Declarative custom filters | — | — |
| L0 | TF-IDF Coarse | DSPC (Sep 2025) | — | — |
| L1 | Entropy Filtering | [Selective Context](https://arxiv.org/abs/2310.06201) — Li et al. | 2–3× |
| L2 | Perplexity Pruning | [LLMLingua](https://arxiv.org/abs/2310.05736) — Microsoft/Tsinghua | 20× |
| L3 | Goal-Driven Selection | [SWE-Pruner](https://arxiv.org/abs/2601.16746) — Shanghai Jiao Tong | 14.8× |
| L4 | AST Preservation | [LongCodeZip](https://arxiv.org/abs/2510.00446) — NUS | 4–8× |
| L5 | Contrastive Ranking | [LongLLMLingua](https://aclanthology.org/2024.acl-long.91/) — Microsoft | 4–10× |
| L6 | N-gram Abbreviation | CompactPrompt | — |
| L7 | Evaluator Heads | EHPC — Tsinghua/Huawei | — |
| L8 | Gist Compression | [Gist Tokens](https://arxiv.org/abs/2304.08467) — Stanford/Berkeley | — |
| L9 | Hierarchical Summary | [AutoCompressor](https://arxiv.org/abs/2305.14788) — Princeton/MIT | — |
| — | Neural Layer | LLM-aware summarization | — | — |
| L11 | Compaction | [MemGPT](https://arxiv.org/abs/2310.08560) — UC Berkeley | — |
| L12 | Attribution Filter | [ProCut](https://aclanthology.org/2025.emnlp-industry.20/) — LinkedIn | 78% |
| L13 | H2O Filter | [H₂O](https://arxiv.org/abs/2306.14048) — NeurIPS 2023 | 30×+ |
| L14 | Attention Sink | [StreamingLLM](https://arxiv.org/abs/2309.17453) — MIT/Meta | — |
| L15 | Meta-Token | [Meta-Tokens](https://arxiv.org/abs/2506.00307) | 27% |
| L16 | Semantic Chunk | [ChunkKV](https://arxiv.org/abs/2502.00299) — NeurIPS 2025 | — |
| L17 | Sketch Store | KVReviver | 90% memory |
| L18 | Lazy Pruner | LazyLLM | 2.34× speedup |
| L19 | Semantic Anchor | SAC | — |
| L20 | Agent Memory | Focus-inspired | — |
| L21 | Question-Aware | [LongLLMLingua](https://aclanthology.org/2024.acl-long.91/) — ACL 2024 | — |
| L22 | Density-Adaptive | DAST — Chen et al. | — |
| L23 | Symbolic Compress | MetaGlyph (Jan 2026) | — |
| L24 | Phrase Grouping | CompactPrompt (2025) | — |
| L25 | Numerical Quant | CompactPrompt (2025) | — |
| L26 | Dynamic Ratio | PruneSID (Mar 2026) | — |
| L27 | Hypernym Compress | Mercury | — |
| L28 | Semantic Cache | SemantiCache | — |
| L29 | Scope Filter | SCOPE (ACL 2025) | — |
| L30 | SmallKV Compensation | SmallKV (2025) | — |
| L31 | KVzip Reconstruction | KVzip (2025) | — |

### Pipeline Features

- **Stage gates** — skip layers when not applicable (zero cost)
- **Early exit** — stop pipeline when budget already met
- **Inter-layer feedback** — quality estimation between layers
- **Result caching** — O(1) LRU with SHA-256 fingerprinting
- **Parallel execution** — Auto-vectorized (compiler optimizations)

## HTTP Proxy Mode

Transparent proxy that intercepts LLM API calls — works with **any** AI agent without hooks.

```bash
tokman http-proxy start --listen :8080
# Then point your AI agent to http://localhost:8080
```

## AI Gateway

```bash
tokman gateway --action set-quota --model gpt-4 --quota 10000
tokman gateway --action alias --model gpt-4 --alias gpt-4o-mini
tokman gateway --action kill-switch --model gpt-4 --kill-switch
```

## Analytics & Monitoring

```bash
tokman tui                          # Interactive dashboard
tokman analytics --action anomaly   # Anomaly detection
tokman analytics --action forecast  # Spend forecasting
tokman analytics --action heatmap   # Token breakdown
```

## Enterprise Features

TokMan includes **40+ enterprise-grade packages** for production-ready token management, cost intelligence, testing, and AI agent orchestration.

### Performance Testing

| Package | Purpose | CLI |
|---------|---------|-----|
| `benchmarking` | Multi-type benchmarks (compression, pipeline, memory, concurrency) with JSON/CSV/Table export, trend tracking, regression detection, parallel execution, and custom DSL | `tokman benchmark run [suite]` |
| `stress` | Load, spike, soak, and breakdown stress testing with 15 predefined scenarios (API, DB, cache, mixed workload, etc.), distributed testing, and report generation | `tokman stress run [scenario]` |
| `chaos` | Chaos engineering with 9 fault types (latency, error, memory, CPU, network, disk, kill, time drift, dependency), safety mechanisms, auto-rollback, and game day support | Programmatic API |

### Cost Intelligence

| Package | Purpose | Description |
|---------|---------|-------------|
| `costforecast` | Cost forecasting engine with 4 ML models (linear regression, moving average, exponential smoothing, seasonal decomposition) and ensemble predictions | Predict future spending |
| `budgetalerts` | Multi-level budget alerts with 5 notification channels (email, Slack, webhook, PagerDuty, console), auto-resolution, and cooldown periods | Stay within budget |
| `teamcosts` | Team-based cost allocation with chargeback, budget tracking, variance reporting, and trend analysis | Allocate costs by team |
| `anomaly` | Cost anomaly detection with 4 algorithms (statistical, sudden change, trend change, pattern break) and severity classification | Detect spending anomalies |
| `costpolicy` | Cost policy enforcement with rules for budget limits, model restrictions, rate limits, approval requirements, and auto-shutdown | Enforce cost policies |
| `costcenter` | Cost center hierarchy management with parent-child relationships, budget tracking, and utilization reporting | Organize costs hierarchically |
| `dashboard` | Cost intelligence dashboard with 10 widgets (cost per request, spend forecast, token breakdown, model comparison, budget cap, anomaly detection, cost alerts, pricing editor, allocation tags, cost trend) | Visualize costs |
| `digest` | Weekly cost digest generation in HTML, Markdown, and JSON formats with budget status, top models, anomalies, and recommendations | Automated reporting |

### AI Agent Framework

| Package | Purpose | Description |
|---------|---------|-------------|
| `iteragent` | Iterative agent framework with ReAct-style loop, tool usage, memory management (short-term, long-term, working), reflection, and pause/resume | Multi-step reasoning |
| `llmproviders` | LLM provider integration supporting OpenAI (GPT-4), Anthropic (Claude 3), and Ollama (local models) with completions, streaming, and embeddings | Multi-provider LLM access |
| `mcphost` | MCP (Model Context Protocol) host management with multi-server connections, tool discovery, resource access, session management, and event-driven architecture | MCP server management |
| `intelligent` | Intelligent filter selection with content classification, adaptive compression, usage analytics, and smart defaults | Smart filter choices |
| `sentiment` | Sentiment analysis for command output classification (positive/negative/neutral) | Content sentiment |
| `autotune` | Auto-tuning for compression parameters with workload prediction and content-aware optimization | Self-optimizing |

### Deployment & Reliability

| Package | Purpose | Description |
|---------|---------|-------------|
| `canary` | Canary deployment system with linear, stepped, analysis, and shadow strategies, automatic traffic splitting, metric-based promotion, and auto-rollback | Safe rollouts |
| `abtest` | A/B testing framework with A/B, multivariate, bandit, and switchback experiment types, statistical significance calculation, and segment targeting | Experimentation |
| `audit` | Comprehensive audit logging with JSON/CSV export, filtering, and retention policies | Compliance logging |
| `rbac` | Role-based access control with 4 roles (admin, manager, developer, viewer) and 6 permission types across 6 resource types | Access control |
| `encryption` | AES-256-GCM encryption at rest with key management and map encryption | Data protection |
| `retention` | Data retention policy management with configurable periods, auto-deletion, and archival | Data lifecycle |

### CLI Enhancements

| Package | Purpose | Description |
|---------|---------|-------------|
| `cli` | Shell completion (bash/zsh/fish), command aliases, progress indicators, spinners, color themes, dry-run mode, command chaining, and batch operations | Better CLI UX |
| `undo` | Undo/redo functionality for all CLI operations with configurable history depth | Reversible operations |
| `commandhistory` | Command history management with search, tagging, and statistics | History tracking |
| `favorites` | Favorite commands management with usage tracking and tag-based organization | Quick access |
| `scheduling` | Command scheduling with once, interval, daily, weekly, and cron schedules | Automated execution |
| `workload` | Workload prediction with trend analysis and intelligent cache optimization | Predictive scaling |

### CI/CD Integration

TokMan includes GitHub Actions workflows for:
- **Benchmark testing** — automated performance regression detection
- **Stress testing** — load validation on every PR
- **Multi-platform builds** — Linux, macOS, Windows (amd64/arm64)
- **Security scanning** — dependency vulnerability checks
- **Automated releases** — tag-based release with changelog generation

## Security

```bash
tokman security --action scan < input.txt    # Scan for vulnerabilities
tokman security --action redact < input.txt  # Redact PII
```

## Supported AI Tools

TokMan supports 10+ AI coding tools. Each integration transparently rewrites shell commands to `tokman` equivalents for 60–90% token savings.

| Tool | Install | Method |
|------|---------|--------|
| **Claude Code** | `tokman init -g` | PreToolUse hook (bash) |
| **GitHub Copilot** | `tokman init -g --copilot` | PreToolUse hook |
| **Cursor** | `tokman init --agent cursor` | preToolUse hook (hooks.json) |
| **Gemini CLI** | `tokman init -g --gemini` | BeforeTool hook |
| **Codex** | `tokman init -g --codex` | AGENTS.md + instructions |
| **Windsurf** | `tokman init --agent windsurf` | .windsurfrules (project-scoped) |
| **Cline / Roo Code** | `tokman init --agent cline` | .clinerules (project-scoped) |
| **OpenCode** | `tokman init -g --opencode` | Plugin TS (tool.execute.before) |
| **Aider** | `tokman init --agent aider` | Instructions |
| **OpenClaw** | Plugin install | before_tool_call hook |

## Auto-Rewrite Hook

The most effective way to use TokMan. The hook transparently intercepts Bash commands and rewrites them to TokMan equivalents before execution.

**Result**: 100% TokMan adoption across all conversations and subagents, zero token overhead.

### Setup

```bash
tokman init -g                 # Install hook + instructions (recommended)
tokman init -g --auto-patch    # Non-interactive (CI/CD)
tokman init -g --hook-only     # Hook only, no instructions
tokman init --show             # Verify installation
tokman init -g --uninstall     # Remove
```

After install, **restart your AI tool**.

## Supported Commands

| Raw Command | Rewritten To |
|-------------|-------------|
| `git status/diff/log/add/commit/push/pull` | `tokman git ...` |
| `gh pr/issue/run` | `tokman gh ...` |
| `cargo test/build/clippy` | `tokman cargo ...` |
| `cat/head/tail <file>` | `tokman read <file>` |
| `rg/grep <pattern>` | `tokman grep <pattern>` |
| `ls` | `tokman ls` |
| `vitest/jest` | `tokman vitest run` |
| `tsc` | `tokman tsc` |
| `eslint/biome` | `tokman lint` |
| `prettier` | `tokman prettier` |
| `playwright` | `tokman playwright` |
| `prisma` | `tokman prisma` |
| `ruff check/format` | `tokman ruff ...` |
| `pytest` | `tokman pytest` |
| `pip list/install` | `tokman pip ...` |
| `go test/build/vet` | `tokman go ...` |
| `golangci-lint` | `tokman golangci-lint` |
| `rake test` / `rails test` | `tokman rake test` |
| `rspec` / `bundle exec rspec` | `tokman rspec` |
| `rubocop` / `bundle exec rubocop` | `tokman rubocop` |
| `bundle install/update` | `tokman bundle ...` |
| `docker ps/images/logs` | `tokman docker ...` |
| `kubectl get/logs` | `tokman kubectl ...` |
| `curl` | `tokman curl` |
| `pnpm list/outdated` | `tokman pnpm ...` |

Commands already using `tokman`, heredocs (`<<`), and unrecognized commands pass through unchanged.

## Configuration

Config file at `~/.config/tokman/config.toml`:

```toml
[tracking]
enabled = true
database_path = "~/.local/share/tokman/tokman.db"

[filter]
mode = "minimal"  # or "aggressive"

[pipeline]
max_context_tokens = 2000000
enable_entropy = true
enable_compaction = true

[hooks]
excluded_commands = ["curl", "playwright"]

[dashboard]
port = 8080
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `TOKMAN_BUDGET` | Token budget |
| `TOKMAN_MODE` | Filter mode (`minimal`, `aggressive`) |
| `TOKMAN_PRESET` | Pipeline preset (`fast`, `balanced`, `full`) |
| `TOKMAN_QUERY` | Query intent |
| `TOKMAN_LLM` | Enable LLM compression |
| `TOKMAN_COMPACTION` | Enable compaction |
| `TOKMAN_H2O` | Enable H2O filter |
| `TOKMAN_ATTENTION_SINK` | Enable attention sink |

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
| SIMD | Auto-vectorized (native SIMD planned) |
| Pipeline | 31 layers, stage gates, early exit |

## vs Competitors

| Feature | TokMan | rtk | tokf | lean-ctx | kompact | tamp |
|---------|--------|-----|------|----------|---------|------|
| Compression layers | **31** | 4 | TOML | 90+ rules | 8 | 10 |
| Research-backed | **30+ papers** | No | No | No | No | No |
| HTTP proxy | **Yes** | No | No | No | Yes | Yes |
| AI gateway | **Yes** | No | No | No | No | No |
| Security (PII) | **Yes** | No | No | No | No | No |
| Analytics TUI | **Yes** | `gain` only | No | Dashboard | Grafana | No |
| SIMD acceleration | Auto-vectorized | No | No | No | No | No |
| WASM plugins | Planned | No | Lua | No | No | No |
| Agent support | **10+** | 10 | 8 | 10+ | 6+ | 6+ |

## Documentation

- **[ARCHITECTURE.md](docs/ARCHITECTURE.md)** — Technical architecture
- **[LAYERS.md](docs/LAYERS.md)** — Detailed layer descriptions
- **[CONTRIBUTING.md](CONTRIBUTING.md)** — How to contribute
- **[SECURITY.md](SECURITY.md)** — Security policy

## Contributing

Contributions welcome! Please open an issue or PR on [GitHub](https://github.com/GrayCodeAI/tokman).

```bash
# Run tests
make test

# Run linter
make lint

# Run benchmarks
make benchmark
```

## License

MIT — see [LICENSE](LICENSE)
