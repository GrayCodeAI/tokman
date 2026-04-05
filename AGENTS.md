# TokMan Codebase Guide

## Project Overview

TokMan is a token-aware CLI proxy written in Go. It intercepts CLI commands and applies a 31-layer compression pipeline to reduce token usage for AI coding assistants. Built on research from 120+ papers, it achieves 60-90% token reduction on common development operations.

**Module:** `github.com/GrayCodeAI/tokman`
**Go Version:** 1.21+ (1.26+ for planned SIMD support)
**CLI Framework:** Cobra (`github.com/spf13/cobra`)
**Config:** Viper + TOML (`~/.config/tokman/config.toml`)
**Database:** SQLite (`modernc.org/sqlite`)

## Directory Structure

```
tokman/
├── cmd/tokman/main.go          # Entry point
├── internal/
│   ├── commands/               # CLI command definitions
│   ├── filter/                 # 31-layer compression pipeline
│   ├── config/                 # Configuration loading
│   ├── core/                   # Command runner, token estimation
│   ├── tracking/               # Command tracking & analytics
│   ├── toml/                   # TOML filter configuration
│   ├── integrity/              # Hook integrity verification
│   ├── utils/                  # Logging utilities
│   ├── telemetry/              # Telemetry collection
│   ├── tee/                    # Output tee/logging system
│   ├── dashboard/              # Dashboard web interface
│   ├── economics/              # Cost analysis
│   ├── feedback/               # Guideline optimization
│   ├── llm/                    # LLM-based summarization
│   ├── discover/               # Command discovery
│   ├── agents/                 # AI agent integration
│   ├── cache/                  # Fingerprint caching
│   ├── parser/                 # Binary log parser
│   ├── tokenizer/              # Token counting
│   ├── session/                # Session context
│   ├── simd/                   # SIMD optimizations
│   ├── alerts/                 # Alert system
│   ├── ccusage/                # Claude Code usage tracking
│   ├── hooks/                  # Hook version management
│   ├── plugin/                 # Plugin system (WASM planned)
│   └── server/                 # Server components
├── config/                     # Default config files
├── templates/                  # Init templates
├── tests/                      # Integration tests
├── benchmarks/                 # Performance benchmarks
├── docs/                       # Documentation
├── docker/                     # Docker configuration
└── .github/                    # CI/CD workflows
```

## Entry Point

`cmd/tokman/main.go` - Simple entry point that calls `commands.Execute()`.

## Command System (`internal/commands/`)

### Root Command (`root.go`)

The root command is defined in `internal/commands/root.go` and handles:
- Global flag definitions (verbose, dry-run, budget, preset, etc.)
- Unknown command fallback via TOML filter system
- Configuration initialization via Viper

### Command Registration (`registry/`)

Commands register via the `registry` package using Go's `init()` mechanism:

```go
import "github.com/GrayCodeAI/tokman/internal/commands/registry"

func init() {
    registry.Add(func() { registry.Register(myCmd) })
}
```

All commands are registered in `root.go` via blank imports for side effects.

### Shared State (`shared/`)

Global state (flags, config) lives in `internal/commands/shared/shared.go`:
```go
import "github.com/GrayCodeAI/tokman/internal/commands/shared"

if shared.IsVerbose() { ... }
if shared.UltraCompact { ... }
```

### Command Categories

| Package | Commands | Purpose |
|---------|----------|---------|
| `vcs/` | git, gh, gt | Version control (status, log, diff, ops) |
| `container/` | docker, kubectl, psql | Container operations |
| `cloud/` | aws | Cloud commands |
| `pkgmgr/` | cargo, npm, npx, pip, pnpm | Package managers |
| `lang/` | go, dotnet | Language runtimes |
| `ruby/` | rake, rspec, rubocop, bundle, rails | Ruby ecosystem |
| `build/` | golangci, next, prisma, tsc | Build tools |
| `infra/` | terraform, helm, ansible | Infrastructure tools |
| `buildtools/` | gradle, mvn, make, mix | Build systems |
| `test/` | jest, pytest, vitest, playwright | Test runners |
| `core/` | doctor, completion, alias, enable, status, plugin, mcp, marketplace, trust, verify, undo, changelog | Core CLI |
| `analysis/` | audit, benchmark, compare, cost, budget, economics, count, freq, gain, stats, top, smart, suggest, report, learn | Analytics |
| `output/` | diff, explain, export, context, format, json, summary, err, rewrite | Output processing |
| `init/` | init (agents, codex) | AI agent setup |
| `hooks/` | hook, hook-audit | Hook management |
| `sessioncmd/` | sessions, snapshot, restore, history | Session management |
| `filtercmd/` | filter (pipeline, layers, create, validate, test) | Filter configuration |
| `web/` | proxy, api_proxy, wget, curl | HTTP operations |
| `linter/` | lint, ruff, prettier, mypy | Linter commands |
| `system/` | ls, grep, find, tree, wc, watch, read, search, log, deps, clean, ccusage, discover, tee, profile | System utilities |
| `agents/` | agents | AI agent management |
| `configcmd/` | config | Configuration |

## Filter Pipeline (`internal/filter/`)

The 31-layer compression pipeline is the core of TokMan. The `PipelineCoordinator` in `pipeline.go` orchestrates all layers with early-exit support and stage gates.

### Layer Architecture

Each layer implements `Apply(input string, mode Mode) (string, int)`:
- `input` - text to compress
- `mode` - `ModeNone`, `ModeMinimal`, or `ModeAggressive`
- Returns filtered text and tokens saved

### Layer Descriptions

| # | Layer | File | Research Paper | Mechanism |
|---|-------|------|----------------|-----------|
| 1 | Entropy Filtering | `entropy.go` | Selective Context (Mila 2023) | Remove low-information tokens |
| 2 | Perplexity Pruning | `perplexity.go` | LLMLingua (Microsoft/Tsinghua 2023) | Iterative token removal |
| 3 | Goal-Driven Selection | `goal_driven.go` | SWE-Pruner (Shanghai Jiao Tong 2025) | CRF-style line scoring |
| 4 | AST Preservation | `ast_preserve.go` | LongCodeZip (NUS 2025) | Syntax-aware compression |
| 5 | Contrastive Ranking | `contrastive.go` | LongLLMLingua (Microsoft 2024) | Question-relevance scoring |
| 6 | N-gram Abbreviation | `ngram.go` | CompactPrompt (2025) | Lossless pattern compression |
| 7 | Evaluator Heads | `evaluator_heads.go` | EHPC (Tsinghua/Huawei 2025) | Early-layer attention sim |
| 8 | Gist Compression | `gist.go` | Stanford/Berkeley (2023) | Virtual token embedding |
| 9 | Hierarchical Summary | `hierarchical.go` | AutoCompressor (Princeton/MIT 2023) | Recursive summarization |
| 10 | Budget Enforcement | `budget.go` | Industry standard | Strict token limits |
| 11 | Compaction | `compaction.go` | MemGPT (UC Berkeley 2023) | Semantic compression |
| 12 | Attribution Filter | `attribution.go` | ProCut (LinkedIn 2025) | 78% pruning |
| 13 | H2O Filter | `h2o.go` | Heavy-Hitter Oracle (NeurIPS 2023) | 30x+ compression |
| 14 | Attention Sink | `attention_sink.go` | StreamingLLM (2023) | Infinite context stability |
| 15 | Meta-Token | `meta_token.go` | arXiv:2506.00307 (2025) | 27% lossless |
| 16 | Semantic Chunk | `semantic_chunk.go` | ChunkKV-style | Context-aware boundaries |
| 17 | Sketch Store | `sketch_store.go` | KVReviver (Dec 2025) | 90% memory reduction |
| 18 | Lazy Pruner | `lazy_pruner.go` | LazyLLM (July 2024) | 2.34x speedup |
| 19 | Semantic Anchor | `semantic_anchor.go` | Attention Gradient Detection | Context preservation |
| 20 | Agent Memory | `agent_memory.go` | Focus-inspired | Knowledge graph extraction |

### Additional Layers

- **T12 Question-Aware:** `question_aware.go` - LongLLMLingua-style relevance
- **T17 Density-Adaptive:** `density_adaptive.go` - DAST-style allocation

### Stage Gates

Each layer has a `shouldSkip*()` method that checks if the layer would provide value:
- `shouldSkipEntropy()` - Skip if content < 50 chars
- `shouldSkipPerplexity()` - Skip if < 5 lines
- `shouldSkipQueryDependent()` - Skip if no query intent (used by goal-driven and contrastive layers)
- `shouldSkipBudgetDependent()` - Skip if no budget (used by sketch store and lazy pruner layers)
- `shouldSkipH2O()` - Skip if < 50 tokens
- `shouldSkipAttentionSink()` - Skip if < 3 lines
- Early exit if budget already met

### Pipeline Configuration

```go
cfg := filter.PipelineConfig{
    Mode:           filter.ModeMinimal,
    QueryIntent:    "debug",
    Budget:         2000,
    LLMEnabled:     true,
    SessionTracking: true,
    // Layer enables...
}
pipeline := filter.NewPipelineCoordinator(cfg)
output, stats := pipeline.Process(input)
```

### Presets (`presets.go`)

Three presets available via `--preset`:
- `fast` - Fewer layers, faster processing
- `balanced` - Default mix
- `full` - All layers enabled

## Configuration (`internal/config/`)

### Config Structure

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
# ... 20+ layer configurations

[hooks]
excluded_commands = []

[dashboard]
port = 8080
```

### Environment Variables

- `TOKMAN_BUDGET` - Token budget
- `TOKMAN_MODE` - Filter mode
- `TOKMAN_PRESET` - Pipeline preset
- `TOKMAN_QUERY` - Query intent
- `TOKMAN_LLM` - Enable LLM compression
- `TOKMAN_COMPACTION` - Enable compaction
- `TOKMAN_H2O` - Enable H2O filter
- `TOKMAN_ATTENTION_SINK` - Enable attention sink

## Core Packages

### Command Runner (`internal/core/`)

- `runner.go` - `OSCommandRunner` executes shell commands via `os/exec`
- `estimator.go` - Unified token estimation (`len(text) / 4`)
- `interfaces.go` - `CommandRunner` interface for testability
- `cost.go` - Cost calculation utilities

### Tracking (`internal/tracking/`)

SQLite-based command tracking:
- `tracker.go` - `Tracker` struct with `Record()` method
- `models.go` - `CommandRecord` struct
- `migrations.go` - Database schema migrations

### TOML Filters (`internal/toml/`)

Custom filter definitions in TOML format:
- `loader.go` - `Loader` discovers and loads `.toml` filter files
- `filter.go` - `FilterRegistry` matches commands to filters
- `parser.go` - TOML filter parsing
- `builtin/` - Built-in TOML filters

### Integrity (`internal/integrity/`)

Hook integrity verification:
- `integrity.go` - `RuntimeCheck()`, `StoreHash()`, `RemoveHash()`
- Ensures hooks haven't been tampered with

## Key Patterns

### Adding a New Command

1. Create a file in the appropriate `internal/commands/<category>/` directory
2. Define the Cobra command
3. Register via `init()`:
```go
func init() {
    registry.Add(func() { registry.Register(myCmd) })
}
```

4. Add blank import to `root.go` if new category:
```go
_ "github.com/GrayCodeAI/tokman/internal/commands/mycategory"
```

### Adding a New Filter Layer

1. Create `internal/filter/my_layer.go`
2. Define struct with `Apply(input string, mode Mode) (string, int)` method
3. Add to `PipelineConfig` in `pipeline.go`
4. Add to `PipelineCoordinator` struct
5. Initialize in `NewPipelineCoordinator()`
6. Add `processLayer*()` method with timing
7. Add `shouldSkip*()` stage gate
8. Add to `Process()` pipeline execution

### TOML Filter Pattern

Create a `.toml` file in `~/.config/tokman/filters/`:
```toml
[my_command]
match = "^my-tool (build|test)"
output_patterns = ["^Building...", "^Testing..."]
strip_lines_matching = ["^INFO:"]
```

## Build & Test

### Commands

```bash
make build          # Standard build
make build-small    # Optimized small binary
make build-simd     # SIMD-optimized build
make build-all      # Multi-platform
make test           # Run tests with race detector
make test-cover     # Tests with coverage
make lint           # golangci-lint
make typecheck      # go vet
make benchmark      # Performance benchmarks
make check          # fmt + vet + typecheck + lint + test
```

### Testing

- Unit tests: `*_test.go` files alongside source
- Integration tests: `tests/` directory
- Benchmarks: `benchmarks/` directory
- Fuzz tests: `filter/fuzz_test.go`

## Key Files

| File | Purpose |
|------|---------|
| `cmd/tokman/main.go` | Entry point |
| `internal/commands/root.go` | Root command, flags, fallback handler |
| `internal/commands/shared/shared.go` | Global state, helper functions |
| `internal/filter/pipeline.go` | 31-layer pipeline coordinator |
| `internal/filter/filter.go` | Filter types and modes |
| `internal/config/config.go` | Configuration loading |
| `internal/core/runner.go` | Command execution |
| `internal/core/estimator.go` | Token estimation |
| `internal/tracking/tracker.go` | SQLite command tracking |
| `internal/toml/loader.go` | TOML filter discovery |

## Environment Variables

All environment variables use the `TOKMAN_` prefix:

| Variable | Purpose | Equivalent Flag |
|----------|---------|----------------|
| `TOKMAN_BUDGET` | Maximum tokens allowed in output | `--budget` |
| `TOKMAN_MODE` | Filter mode: `minimal`, `aggressive` | `--mode` |
| `TOKMAN_PRESET` | Pipeline preset: `fast`, `balanced`, `full` | `--preset` |
| `TOKMAN_QUERY` | Query intent for goal-driven filtering | `--query` |
| `TOKMAN_LLM` | Enable LLM compression (0/1) | `--llm` |
| `TOKMAN_COMPACTION` | Enable semantic compaction (0/1) | `--compaction` |
| `TOKMAN_H2O` | Enable H2O heavy-hitter filter (0/1) | `--h2o` |
| `TOKMAN_ATTENTION_SINK` | Enable attention sink stability (0/1) | `--attention-sink` |
| `TOKMAN_VERBOSE` | Enable verbose output | `-v, --verbose` |
| `TOKMAN_DRY_RUN` | Show what would happen without executing | `--dry-run` |
| `TOKMAN_JSON` | Output in JSON format | `--json` |
| `TOKMAN_SILENT` | Suppress all non-essential output | `--silent` |
| `TOKMAN_OUTPUT` | Show full unfiltered output | `--output` |
| `TOKMAN_ULTRA_COMPACT` | Aggressively minimize output | `--ultra-compact` |

## Adding AI Agent Integration

TokMan supports multiple AI agents via `tokman init`:
- Claude Code: `tokman init -g`
- Cursor: `tokman init --cursor`
- Copilot: `tokman init --copilot`
- Windsurf: `tokman init --windsurf`
- Cline: `tokman init --cline`
- Gemini: `tokman init -g --gemini`
- Codex: `tokman init --codex`
- All detected: `tokman init --all`

Hook scripts are installed to agent-specific directories and patch configuration files to route commands through TokMan.

## Performance Considerations

- **SIMD:** Auto-vectorized by Go compiler (native SIMD planned for Go 1.26+)
- **Streaming:** Large inputs (>500K tokens) use streaming processing
- **Caching:** Fingerprint-based result caching in `internal/cache/`
- **Stage Gates:** Skip layers when not applicable (zero cost)
- **Early Exit:** Stop pipeline when budget already met
- **Binary Size:** `make build-tiny` produces minimal binaries (~5MB)

## Dependencies

- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - Configuration
- `github.com/BurntSushi/toml` - TOML parsing
- `modernc.org/sqlite` - Pure Go SQLite
- `github.com/fatih/color` - Terminal colors
- `github.com/tiktoken-go/tokenizer` - Tiktoken tokenizer
- `github.com/tetratelabs/wazero` - WASM runtime for plugins (planned)
