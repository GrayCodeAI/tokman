# TokMan Codebase Guide 🌸

**Purpose**: Complete navigation guide for humans and LLMs to understand and work with the TokMan token reduction system.

**Last Updated**: 2026-03-22

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Quick Start](#quick-start)
3. [Directory Structure](#directory-structure)
4. [Architecture Overview](#architecture-overview)
5. [Core Modules](#core-modules)
6. [20-Layer Compression Pipeline](#20-layer-compression-pipeline)
7. [Command System](#command-system)
8. [Configuration](#configuration)
9. [Testing](#testing)
10. [Development Workflow](#development-workflow)
11. [Key File Reference](#key-file-reference)

---

## Project Overview

**TokMan** is a Go-based token reduction system that intercepts CLI commands and applies 20 research-backed compression layers to reduce output by 95-99%. It's designed to optimize LLM context windows while maintaining critical information.

### Key Features
- 20-layer compression pipeline (based on 120+ research papers)
- 50+ command wrappers (git, docker, kubectl, npm, pytest, etc.)
- SQLite-based token tracking
- Shell integration with automatic command rewriting
- Web dashboard for analytics
- SHA-256 hook integrity verification

### Tech Stack
- **Language**: Go 1.21+
- **Config**: TOML (Viper)
- **Database**: SQLite
- **Tokenizer**: tiktoken (OpenAI)
- **Architecture**: Modular CLI with plugin support

---

## Quick Start

```bash
# Build
go build -o tokman ./cmd/tokman

# Run tests
go test ./...

# Initialize (install shell hook)
./tokman init

# Check status
./tokman status
```

---

## Directory Structure

```
tokman/
├── 📁 cmd/                          # Application entry points
│   └── tokman/main.go               # Main CLI entry point
│
├── 📁 internal/                     # Private application code
│   ├── 📂 agents/                   # Agent integration logic
│   ├── 📂 alerts/                   # Alert/notification system
│   ├── 📂 cache/                    # Caching layer (fingerprinting)
│   ├── 📂 ccusage/                  # Claude Code usage tracking
│   ├── 📂 commands/                 # CLI command handlers (50+ commands)
│   │   ├── root.go                  # Root command router
│   │   ├── git.go                   # Git wrappers
│   │   ├── docker.go                # Docker wrappers
│   │   ├── kubectl.go               # Kubernetes wrappers
│   │   ├── go.go                    # Go build/test
│   │   ├── cargo.go                 # Rust/Cargo commands
│   │   ├── npm.go                   # npm/pnpm/npx
│   │   ├── pytest.go                # Python test runner
│   │   ├── gain.go                  # Savings analysis
│   │   ├── economics.go             # Cost analysis
│   │   └── ... (50+ more)
│   │
│   ├── 📂 config/                   # Configuration management
│   │   ├── config.go                # Config loader
│   │   ├── defaults.go              # Default values
│   │   └── hierarchical.go          # Hierarchical config loading
│   │
│   ├── 📂 core/                     # Core abstractions & interfaces
│   │   ├── interfaces.go            # CommandRunner interface
│   │   ├── runner.go                # Command execution
│   │   ├── estimator.go             # Token estimation
│   │   ├── cost.go                  # Cost calculation
│   │   └── writer.go                # Output writing
│   │
│   ├── 📂 dashboard/                # Web dashboard
│   │   ├── dashboard.go             # HTTP server
│   │   └── html.go                  # Dashboard HTML templates
│   │
│   ├── 📂 discover/                 # Command discovery system
│   ├── 📂 economics/                # Economics/cost analysis
│   ├── 📂 feedback/                 # Feedback optimization loop
│   │
│   ├── 📂 filter/                   # ⭐ 20-LAYER COMPRESSION PIPELINE
│   │   ├── pipeline.go              # Pipeline coordinator
│   │   ├── filter.go                # Base filter interface
│   │   ├── manager.go               # Filter manager
│   │   │
│   │   ├── 📂 Layer 1-14: Core Compression
│   │   │   ├── entropy.go           # L1: Entropy filtering
│   │   │   ├── perplexity.go        # L2: Perplexity pruning
│   │   │   ├── goal_driven.go       # L3: Goal-driven selection
│   │   │   ├── ast_preserve.go      # L4: AST preservation
│   │   │   ├── contrastive.go       # L5: Contrastive ranking
│   │   │   ├── ngram.go             # L6: N-gram abbreviation
│   │   │   ├── evaluator_heads.go   # L7: Evaluator heads
│   │   │   ├── gist.go              # L8: Gist compression
│   │   │   ├── hierarchical.go      # L9: Hierarchical summary
│   │   │   ├── budget.go            # L10: Budget enforcement
│   │   │   ├── compaction.go        # L11: Compaction layer
│   │   │   ├── attribution.go       # L12: Attribution filter
│   │   │   ├── h2o.go               # L13: H2O filter
│   │   │   └── attention_sink.go    # L14: Attention sink
│   │   │
│   │   ├── 📂 Layer 15-20: Advanced Compression
│   │   │   ├── meta_token.go        # L15: Meta-token compression
│   │   │   ├── semantic_chunk.go    # L16: Semantic chunking
│   │   │   ├── sketch_store.go      # L17: Sketch store
│   │   │   ├── lazy_pruner.go       # L18: Lazy pruner
│   │   │   ├── semantic_anchor.go   # L19: Semantic anchor
│   │   │   └── agent_memory.go      # L20: Agent memory
│   │   │
│   │   └── ... (supporting filters)
│   │
│   ├── 📂 hooks/                    # Shell hook versioning
│   ├── 📂 integrity/                # SHA-256 verification
│   ├── 📂 llm/                      # LLM integration
│   │   ├── prompts.go               # Prompt templates
│   │   └── summarizer.go            # LLM summarization
│   │
│   ├── 📂 parser/                   # Output parsers
│   │   └── binlog/                  # Binary log parsing
│   │
│   ├── 📂 plugin/                   # Plugin system
│   │   ├── manager.go               # Plugin manager
│   │   └── wasm.go                  # WASM plugin support
│   │
│   ├── 📂 server/                   # API server
│   │   ├── server.go                # HTTP server
│   │   └── metrics.go               # Metrics endpoint
│   │
│   ├── 📂 session/                  # Session management
│   ├── 📂 simd/                     # SIMD optimizations
│   ├── 📂 tee/                      # Tee on failure
│   ├── 📂 telemetry/                # Telemetry system
│   ├── 📂 tokenizer/                # Token counting (tiktoken)
│   │
│   ├── 📂 toml/                     # TOML filter definitions
│   │   ├── loader.go                # TOML loader
│   │   ├── parser.go                # Parser
│   │   ├── filter.go                # Filter definitions
│   │   └── builtin/                 # 80+ built-in filters
│   │       ├── git.toml             # Git filters
│   │       ├── docker.toml          # Docker filters
│   │       ├── kubectl.toml         # Kubernetes filters
│   │       ├── pytest.toml          # pytest filters
│   │       └── ... (80+ more)
│   │
│   ├── 📂 tracking/                 # Token tracking
│   │   ├── tracker.go               # SQLite tracker
│   │   ├── models.go                # Data models
│   │   └── migrations.go            # DB migrations
│   │
│   └── 📂 utils/                    # Utilities
│       ├── logger.go                # Logging
│       └── utils.go                 # Helper functions
│
├── 📁 hooks/                        # Shell integration scripts
│   ├── tokman-rewrite.sh            # Main shell hook
│   ├── cursor-tokman-rewrite.sh     # Cursor editor integration
│   ├── copilot-tokman-rewrite.sh    # Copilot integration
│   ├── copilot-hooks.json           # Copilot hook config
│   ├── copilot-instructions.md      # Copilot instructions
│   └── ... (editor-specific hooks)
│
├── 📁 filters/                      # User filter plugins
│   ├── biome.toml
│   ├── eslint.toml
│   ├── jest.toml
│   └── ... (custom filters)
│
├── 📁 docs/                         # Documentation
│   ├── README.md                    # Docs overview
│   ├── ARCHITECTURE.md              # Architecture details
│   ├── LAYERS.md                    # 20-layer documentation
│   ├── API.md                       # API reference
│   ├── FEATURES.md                  # Feature guide
│   ├── GUIDE.md                     # User guide
│   ├── TROUBLESHOOTING.md           # Troubleshooting
│   └── adr/                         # Architecture Decision Records
│
├── 📁 benchmarks/                   # Performance benchmarks
│   └── performance_test.go
│
├── 📁 tests/                        # Integration tests
│   └── integration/
│
├── 📁 completions/                  # Shell completions
├── 📁 config/                       # Default config
│   └── tokman.yaml
│
├── 📁 docker/                       # Docker configuration
│   └── docker-compose.yml
│
├── 📄 README.md                     # Main documentation
├── 📄 ARCHITECTURE.md               # Architecture overview
├── 📄 CHANGELOG.md                  # Version history
├── 📄 CONTRIBUTING.md               # Contribution guide
└── 📄 ROADMAP.md                    # Future plans
```

---

## Architecture Overview

### High-Level Architecture

```
┌────────────────────────────────────────────────────────────────┐
│                        CLI Entry Point                         │
│                     cmd/tokman/main.go                         │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                     Command Router                             │
│                  internal/commands/root.go                     │
│                                                                │
│  Routes to 50+ command handlers based on user input           │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                   Command Handlers                             │
│                internal/commands/*.go                          │
│                                                                │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐         │
│  │ git.go   │ │docker.go │ │ pytest.go│ │  go.go   │ ...     │
│  └──────────┘ └──────────┘ └──────────┘ └──────────┘         │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                20-Layer Compression Pipeline                   │
│                internal/filter/pipeline.go                     │
│                                                                │
│  L1 → L2 → L3 → ... → L20 (each layer applies filters)        │
│                                                                │
│  Input → [Entropy] → [Perplexity] → ... → [AgentMemory] → Out │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                   Core Services                                │
│                                                                │
│  ┌──────────────┐ ┌──────────────┐ ┌──────────────┐          │
│  │   Tracking   │ │    Config    │ │   Integrity  │          │
│  │  (SQLite)    │ │   (TOML)     │ │  (SHA-256)   │          │
│  └──────────────┘ └──────────────┘ └──────────────┘          │
└────────────────────────────────────────────────────────────────┘
```

### Data Flow

```
User Command → Shell Hook → TokMan CLI → Command Handler
                                              │
                                              ▼
                                    Execute Original Command
                                              │
                                              ▼
                                    Capture Raw Output
                                              │
                                              ▼
                                    Apply 20-Layer Pipeline
                                              │
                                              ▼
                                    Track Savings (SQLite)
                                              │
                                              ▼
                                    Return Compressed Output
```

---

## Core Modules

### 1. Command System (`internal/commands/`)

**Purpose**: Handle 50+ CLI commands with filtering support.

**Key Files**:
- `root.go` - Main command router and CLI setup
- `git.go`, `docker.go`, `kubectl.go` - Infrastructure wrappers
- `go.go`, `cargo.go`, `pytest.go` - Test runner aggregators
- `npm.go`, `pip.go`, `cargo.go` - Package manager wrappers
- `gain.go`, `economics.go` - Analytics commands

**Pattern**:
```go
// Each command follows this pattern:
func NewGitCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "git [command]",
        Short: "Git with token filtering",
        Run:   runGit,
    }
    return cmd
}
```

### 2. Compression Pipeline (`internal/filter/`)

**Purpose**: 20-layer research-based compression system.

**Architecture**:
```go
type PipelineCoordinator struct {
    // 20 filter layers
    entropyFilter      *EntropyFilter      // L1
    perplexityFilter   *PerplexityFilter   // L2
    // ... L3-L19
    agentMemoryFilter  *AgentMemoryFilter  // L20
}

func (p *PipelineCoordinator) Process(input string) (string, *PipelineStats)
```

**Layer Order is Critical** - Each layer builds on the previous.

### 3. Configuration (`internal/config/`)

**Purpose**: Hierarchical configuration loading.

**Hierarchy** (lowest to highest priority):
1. Built-in defaults (`defaults.go`)
2. System config (`/etc/tokman/config.toml`)
3. User config (`~/.config/tokman/config.toml`)
4. Project config (`./tokman.toml`)
5. Environment variables

**Key Files**:
- `config.go` - Main config structure
- `hierarchical.go` - Layered loading
- `defaults.go` - Default values

### 4. Token Tracking (`internal/tracking/`)

**Purpose**: Persist token savings to SQLite.

**Schema**:
```sql
CREATE TABLE savings (
    id INTEGER PRIMARY KEY,
    timestamp DATETIME,
    command TEXT,
    original_tokens INTEGER,
    compressed_tokens INTEGER,
    reduction_percent REAL
);
```

**Key Files**:
- `tracker.go` - SQLite operations
- `models.go` - Data models
- `migrations.go` - Schema migrations

### 5. Shell Integration (`hooks/`)

**Purpose**: Intercept and rewrite shell commands.

**Flow**:
```bash
# User types: git status
# Hook intercepts → tokman git status
# TokMan filters output → returns compressed version
```

**Key File**: `hooks/tokman-rewrite.sh`

---

## 20-Layer Compression Pipeline

Each layer applies a specific compression technique from research papers:

| Layer | Name | Research Source | File | Reduction |
|-------|------|-----------------|------|-----------|
| 1 | Entropy Filtering | Selective Context (Mila 2023) | `entropy.go` | 2-3x |
| 2 | Perplexity Pruning | LLMLingua (Microsoft 2023) | `perplexity.go` | 20x |
| 3 | Goal-Driven Selection | SWE-Pruner (2025) | `goal_driven.go` | 14.8x |
| 4 | AST Preservation | LongCodeZip (NUS 2025) | `ast_preserve.go` | 4-8x |
| 5 | Contrastive Ranking | LongLLMLingua (2024) | `contrastive.go` | 4-10x |
| 6 | N-gram Abbreviation | CompactPrompt (2025) | `ngram.go` | 2.5x |
| 7 | Evaluator Heads | EHPC (Tsinghua 2025) | `evaluator_heads.go` | 5-7x |
| 8 | Gist Compression | Stanford/Berkeley 2023 | `gist.go` | 20x+ |
| 9 | Hierarchical Summary | AutoCompressor (MIT 2023) | `hierarchical.go` | Extreme |
| 10 | Budget Enforcement | Industry standard | `budget.go` | Guaranteed |
| 11 | Compaction | MemGPT (Berkeley 2023) | `compaction.go` | 98%+ |
| 12 | Attribution Filter | ProCut (LinkedIn 2025) | `attribution.go` | 78% |
| 13 | H2O Filter | NeurIPS 2023 | `h2o.go` | 30x+ |
| 14 | Attention Sink | StreamingLLM (2023) | `attention_sink.go` | Stability |
| 15 | Meta-Token | arXiv 2025 | `meta_token.go` | 27% lossless |
| 16 | Semantic Chunk | Dynamic boundaries | `semantic_chunk.go` | Context-aware |
| 17 | Sketch Store | KVReviver (2025) | `sketch_store.go` | 90% memory |
| 18 | Lazy Pruner | LazyLLM (2024) | `lazy_pruner.go` | 2.34x speedup |
| 19 | Semantic Anchor | Gradient detection | `semantic_anchor.go` | Preservation |
| 20 | Agent Memory | Knowledge graphs | `agent_memory.go` | Agent-optimized |

### Pipeline Configuration

```go
type PipelineConfig struct {
    Mode          PipelineMode  // Minimal, Balanced, Aggressive
    Budget        int           // Target token budget
    EnableLayers  []int         // Which layers to enable
    CustomFilters []Filter      // User-defined filters
}
```

---

## Command System

### Command Categories

1. **Core Commands** (`init`, `status`, `config`, `verify`)
2. **Git Wrappers** (`git status`, `git diff`, `git log`, etc.)
3. **Infrastructure** (`docker`, `kubectl`, `aws`, `gh`)
4. **Test Runners** (`go test`, `pytest`, `jest`, `vitest`)
5. **Package Managers** (`npm`, `pnpm`, `pip`, `cargo`)
6. **Build Tools** (`go build`, `cargo build`, `next build`)
7. **Utilities** (`ls`, `tree`, `grep`, `find`, `diff`)
8. **Analytics** (`gain`, `economics`, `report`, `dashboard`)

### Adding a New Command

```go
// 1. Create internal/commands/mycommand.go
func NewMyCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "mycommand [args]",
        Short: "My command description",
        Run:   runMyCommand,
    }
    return cmd
}

func runMyCommand(cmd *cobra.Command, args []string) {
    // Execute original command
    output := executeCommand("mycommand", args)
    
    // Apply filtering
    filtered := filter.Apply(output, filterConfig)
    
    // Track savings
    tracker.Record(original, filtered)
    
    // Output result
    fmt.Println(filtered)
}

// 2. Register in root.go
rootCmd.AddCommand(NewMyCommand())

// 3. Add TOML filter (optional)
// internal/toml/builtin/mycommand.toml
```

---

## Configuration

### Config File Structure (`~/.config/tokman/config.toml`)

```toml
[tracking]
enabled = true
telemetry = false
database_path = ""  # Default: ~/.config/tokman/tracking.db

[filter]
mode = "minimal"  # "minimal", "balanced", "aggressive"
noise_dirs = [
    ".git", "node_modules", "target",
    "__pycache__", ".venv", "vendor"
]

[hooks]
excluded_commands = ["vim", "nano"]
verify_integrity = true

[dashboard]
enabled = true
port = 8080

[llm]
enabled = false
provider = "openai"  # "openai", "anthropic", "local"
api_key = ""
```

### Environment Variables

```bash
TOKMAN_CONFIG_PATH=/custom/path/config.toml
TOKMAN_DATABASE_PATH=/custom/path/tracking.db
TOKMAN_LOG_LEVEL=debug
XDG_CONFIG_HOME=/custom/config
```

---

## Testing

### Test Structure

```
internal/
├── filter/
│   ├── filter_test.go              # Unit tests
│   ├── pipeline_test.go            # Pipeline tests
│   ├── bench_test.go               # Benchmarks
│   └── *_test.go                   # Per-layer tests
│
├── commands/
│   ├── config_cmds_test.go
│   └── *_test.go                   # Command tests
│
└── tracking/
    └── tracker_test.go

tests/
└── integration/
    └── pipeline_integration_test.go
```

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/filter/...

# With coverage
go test -cover ./...

# Benchmarks
go test -bench=. ./internal/filter/

# Integration tests
go test ./tests/integration/...
```

---

## Development Workflow

### Building

```bash
# Development build
go build -o tokman ./cmd/tokman

# Production build
go build -ldflags="-s -w" -o tokman ./cmd/tokman

# Cross-compile
GOOS=darwin GOARCH=arm64 go build -o tokman-darwin ./cmd/tokman
GOOS=linux GOARCH=amd64 go build -o tokman-linux ./cmd/tokman
```

### Code Quality

```bash
# Lint
golangci-lint run

# Format
go fmt ./...

# Vet
go vet ./...
```

### Debugging

```bash
# Enable debug logging
TOKMAN_LOG_LEVEL=debug ./tokman git status

# Verbose output
./tokman -v git status

# Check configuration
./tokman config get

# Verify hooks
./tokman verify
```

---

## Key File Reference

### Entry Points
| File | Purpose |
|------|---------|
| `cmd/tokman/main.go` | CLI entry point |
| `internal/commands/root.go` | Command router |

### Core Systems
| File | Purpose |
|------|---------|
| `internal/filter/pipeline.go` | 20-layer compression coordinator |
| `internal/filter/filter.go` | Base filter interface |
| `internal/config/config.go` | Configuration management |
| `internal/tracking/tracker.go` | SQLite token tracking |
| `internal/core/interfaces.go` | Core abstractions |

### Important Filters
| File | Layer |
|------|-------|
| `internal/filter/entropy.go` | L1: Entropy filtering |
| `internal/filter/perplexity.go` | L2: Perplexity pruning |
| `internal/filter/h2o.go` | L13: Heavy-hitter oracle |
| `internal/filter/attention_sink.go` | L14: StreamingLLM |
| `internal/filter/agent_memory.go` | L20: Agent memory |

### Shell Integration
| File | Purpose |
|------|---------|
| `hooks/tokman-rewrite.sh` | Main shell hook |
| `internal/integrity/integrity.go` | SHA-256 verification |

### Documentation
| File | Purpose |
|------|---------|
| `README.md` | Main documentation |
| `docs/LAYERS.md` | 20-layer deep dive |
| `docs/ARCHITECTURE.md` | Architecture details |
| `docs/API.md` | API reference |

---

## Quick Reference Commands

```bash
# Initialize
tokman init

# Check savings
tokman status
tokman gain

# View config
tokman config get

# Verify hooks
tokman verify

# Dashboard
tokman dashboard

# Economics
tokman economics

# Count tokens
tokman count "text or file"

# Discover savings
tokman discover

# Run tests
go test ./...

# Build
go build -o tokman ./cmd/tokman
```

---

## For LLMs: Working with This Codebase

### When adding a new command:
1. Create `internal/commands/<name>.go`
2. Add command to router in `internal/commands/root.go`
3. Optionally add filter in `internal/toml/builtin/<name>.toml`
4. Add tests in `internal/commands/<name>_test.go`

### When modifying compression:
1. Identify target layer in `internal/filter/`
2. Modify layer-specific file (e.g., `entropy.go`)
3. Update pipeline in `pipeline.go` if needed
4. Add/update tests in `<layer>_test.go`
5. Run benchmarks: `go test -bench=. ./internal/filter/`

### When working with configuration:
1. Config structure: `internal/config/config.go`
2. Defaults: `internal/config/defaults.go`
3. Loading logic: `internal/config/hierarchical.go`

### When debugging token tracking:
1. Database: `~/.config/tokman/tracking.db`
2. Tracker code: `internal/tracking/tracker.go`
3. Models: `internal/tracking/models.go`

---

**Generated**: 2026-03-22 | **Version**: Based on main branch
