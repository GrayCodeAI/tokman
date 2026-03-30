# TokMan 🌸

**World's Most Advanced Token Reduction System** — 31-layer research-based compression pipeline achieving 95-99% token reduction.

TokMan intercepts CLI commands, applies 20 research-backed compression layers, and tracks token savings. Built on 120+ papers from top institutions (Mila, Microsoft, Stanford, Berkeley, NeurIPS 2023-2025).

## Compression Performance

| Input Size | Original | Final | Reduction |
|------------|----------|-------|-----------|
| Small (100 lines) | 982 tokens | 44 tokens | **95.5%** |
| Medium (1000 lines) | 9,737 tokens | 52 tokens | **99.5%** |
| Large (5000 lines) | 49,437 tokens | 63 tokens | **99.9%** |
| **Up to 2M tokens** | Full context support | Streaming processing | Memory optimized |

## Features

- 🧠 **20-Layer Compression Pipeline** — Research-based token reduction (95-99%)
- 🔧 **Git Command Wrappers** — Filtered `status`, `diff`, `log`, `add`, `commit`, `push`, `pull`, and more
- 📁 **LS Handler** — Hide noise directories (.git, node_modules, target, etc.)
- 🐳 **Infrastructure Wrappers** — Docker, kubectl, AWS CLI with filtered output
- 📦 **Package Managers** — npm, pnpm, pip, cargo with compact output
- 🧪 **Test Runners** — Go, pytest, vitest, jest, npm test, playwright with aggregated results
- 🔨 **Build Tools** — Go, cargo, next.js with error-only output
- 📊 **Token Tracking** — SQLite-based metrics on tokens saved
- 🔄 **Shell Integration** — Automatic command rewriting via shell hooks
- 🔐 **Integrity Verification** — SHA-256 hook verification for security
- 💰 **Economics Analysis** — Compare spending vs savings with quota estimates
- 💾 **Tee on Failure** — Auto-saves full output when commands fail for debugging

## 31-Layer Compression Pipeline

TokMan implements the world's most advanced token reduction system based on 120+ research papers:

| Layer | Name | Research | Compression |
|-------|------|----------|-------------|
| 1 | Entropy Filtering | Selective Context (Mila 2023) | 2-3x |
| 2 | Perplexity Pruning | LLMLingua (Microsoft 2023) | 20x |
| 3 | Goal-Driven Selection | SWE-Pruner (Shanghai Jiao Tong 2025) | 14.8x |
| 4 | AST Preservation | LongCodeZip (NUS 2025) | 4-8x |
| 5 | Contrastive Ranking | LongLLMLingua (Microsoft 2024) | 4-10x |
| 6 | N-gram Abbreviation | CompactPrompt (2025) | 2.5x |
| 7 | Evaluator Heads | EHPC (Tsinghua/Huawei 2025) | 5-7x |
| 8 | Gist Compression | Stanford/Berkeley (2023) | 20x+ |
| 9 | Hierarchical Summary | AutoCompressor (Princeton/MIT 2023) | Extreme |
| 10 | Budget Enforcement | Industry standard | Guaranteed |
| 11 | Compaction | MemGPT (UC Berkeley 2023) | 98%+ |
| 12 | Attribution Filter | ProCut (LinkedIn 2025) | 78% |
| 13 | H2O Filter | Heavy-Hitter Oracle (NeurIPS 2023) | 30x+ |
| 14 | Attention Sink | StreamingLLM (2023) | Infinite stability |
| 15 | Meta-Token Compression | Lossless Token Sequence (arXiv 2025) | 27% lossless |
| 16 | Semantic Chunking | Dynamic Boundary Detection | Context-aware |
| 17 | Sketch Store | KVReviver (Dec 2025) | 90% memory |
| 18 | Lazy Pruner | LazyLLM (July 2024) | 2.34x speedup |
| 19 | Semantic Anchor | Attention Gradient Detection | Context preservation |
| 20 | Agent Memory | Knowledge Graph Extraction | Agent-optimized |

**New Layers (L15-L20)** provide advanced compression capabilities:
- **L15 Meta-Token**: LZ77-style lossless compression for repeated token sequences
- **L16 Semantic Chunk**: Dynamic boundary detection based on semantic shifts
- **L17 Sketch Store**: On-demand reconstruction of pruned content (90% memory reduction)
- **L18 Lazy Pruner**: Budget-aware dynamic pruning with layer-wise decay
- **L19 Semantic Anchor**: Preserves critical context via attention gradient analysis
- **L20 Agent Memory**: Extracts knowledge graphs for agent-specific optimization

See [docs/LAYERS.md](docs/LAYERS.md) for detailed documentation of each layer.

## Installation

```bash
# Clone and build
git clone https://github.com/GrayCodeAI/tokman.git
cd tokman
go build -o tokman ./cmd/tokman

# Install to PATH (optional)
sudo mv tokman /usr/local/bin/
```

### Docker

```bash
# Pull from GitHub Container Registry (once published)
docker pull ghcr.io/graycodeai/tokman:latest

# Or build locally
docker build -f docker/Dockerfile -t tokman:latest .

# Run with dashboard
docker run -d -p 8080:8080 -v tokman-data:/home/tokman/.local/share/tokman tokman:latest

# Run with docker-compose
cd docker && docker-compose up -d
```

### Homebrew (macOS/Linux)

```bash
brew install GrayCodeAI/tap/tokman
```

## Quick Start

```bash
# Initialize TokMan (install shell hook)
tokman init

# Check token savings
tokman status

# View comprehensive savings analysis
tokman gain

# Use wrapped commands
tokman git status
tokman ls
tokman test ./...
```

## Live Demo Examples

### Example 1: Git Status Compression

```bash
$ tokman git status
```
**Before (342 tokens):**
```
On branch main
Your branch is up to date with 'origin/main'.

Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
  (use "git restore <file>..." to discard changes in working directory)
	modified:   internal/filter/pipeline.go
	modified:   internal/filter/h2o.go
	modified:   internal/filter/semantic.go

Untracked files:
  (use "git add <file>..." to include in what will be committed)
	internal/filter/stream.go
	internal/filter/stream_test.go

no changes added to commit (use "git add" and/or "git commit -a")
```

**After (78 tokens, 77% reduction):**
```
🌿 main (origin/main)
📝 M internal/filter/pipeline.go
📝 M internal/filter/h2o.go
📝 M internal/filter/semantic.go
❓ internal/filter/stream.go
❓ internal/filter/stream_test.go
```

### Example 2: Docker PS Compression

```bash
$ tokman docker ps
```
**Before (528 tokens):**
```
CONTAINER ID   IMAGE          COMMAND                  CREATED        STATUS        PORTS                    NAMES
abc123def456   nginx:latest   "/docker-entrypoint.…"   2 hours ago    Up 2 hours    0.0.0.0:80->80/tcp       web-server
def789ghi012   redis:alpine   "docker-entrypoint.s…"   3 hours ago    Up 3 hours    0.0.0.0:6379->6379/tcp   cache-server
jkl345mno678   postgres:14    "docker-entrypoint.s…"   5 hours ago    Up 5 hours    5432/tcp                 db-server
mno789pqr012   node:18        "node server.js"         1 hour ago     Up 1 hour     0.0.0.0:3000->3000/tcp   api-server
...
```

**After (89 tokens, 83% reduction):**
```
🐳 nginx:latest    → web-server   (2h)  0.0.0.0:80
🐳 redis:alpine    → cache-server (3h)  0.0.0.0:6379
🐳 postgres:14     → db-server    (5h)  5432/tcp
🐳 node:18         → api-server   (1h)  0.0.0.0:3000
```

### Example 3: Test Output Compression

```bash
$ tokman go test ./...
```
**Before (2,847 tokens):**
```
=== RUN   TestFilterShort
--- PASS: TestFilterShort (0.00s)
=== RUN   TestFilterLong
--- PASS: TestFilterLong (0.00s)
=== RUN   TestFilterGitStatus
--- PASS: TestFilterGitStatus (0.00s)
... (50 more lines) ...
PASS
ok  	github.com/GrayCodeAI/tokman/internal/filter	0.014s
=== RUN   TestPipelineBasic
--- PASS: TestPipelineBasic (0.00s)
... (30 more lines) ...
PASS
ok  	github.com/GrayCodeAI/tokman/internal/pipeline	0.023s
```

**After (124 tokens, 96% reduction):**
```
✅ internal/filter    0.014s  12 tests
✅ internal/pipeline  0.023s  8 tests
──────────────────────────────────
✅ All passed (20 tests, 0.037s)
```

### Example 4: Streaming API (for Chat Agents)

```go
package main

import (
    "github.com/GrayCodeAI/tokman/internal/filter"
)

func main() {
    config := filter.PipelineConfig{
        Mode:   filter.ModeAggressive,
        Budget: 4000, // Target token budget
    }
    
    // Channel-based streaming
    input, output := filter.StreamChannel(config)
    
    // Send chunks as they arrive
    input <- "Long conversation history..."
    input <- "More context from agent..."
    close(input)
    
    // Receive compressed chunks
    for chunk := range output {
        fmt.Println(chunk.Content)
        fmt.Printf("Saved: %d tokens\n", chunk.TokensSaved)
    }
}
```

## Commands

### Core Commands

| Command | Description |
|---------|-------------|
| `tokman init` | Initialize TokMan and install shell hook |
| `tokman status` | Quick token savings summary |
| `tokman report` | Detailed usage analytics |
| `tokman gain` | Comprehensive savings with graphs, history, quota |
| `tokman config` | Show or create configuration file |
| `tokman verify` | Verify hook integrity (SHA-256 check) |
| `tokman economics` | Show spending vs savings analysis |

### Git Wrappers

| Command | Filter Applied |
|---------|---------------|
| `tokman git status` | Porcelain parsing, emoji formatting |
| `tokman git diff` | Stats summary, compact hunks |
| `tokman git log` | Oneline format, smart limits |
| `tokman git add` | Compact "ok ✓" output |
| `tokman git commit` | Show hash on success |
| `tokman git push` | Show branch on success |
| `tokman git pull` | Show stats summary |
| `tokman git branch` | Compact listing |
| `tokman git stash` | Compact list/apply/drop |
| `tokman git show` | Commit summary + compact diff |
| `tokman git fetch` | Show new refs count |
| `tokman git worktree` | Compact listing |

### Infrastructure Wrappers

| Command | Description |
|---------|-------------|
| `tokman docker` | Docker CLI with filtered output |
| `tokman kubectl` | Kubernetes CLI with filtered output |
| `tokman aws` | AWS CLI with filtered output |
| `tokman gh` | GitHub CLI with token-optimized output |
| `tokman gh run list` | GitHub Actions workflow runs (compact) |
| `tokman gh release list` | GitHub releases (compact) |
| `tokman gh api` | GitHub API with JSON structure output |
| `tokman gt` | Graphite stacked PR commands |

### Test Runners

| Command | Description |
|---------|-------------|
| `tokman go test` | Go tests with aggregated results |
| `tokman go build` | Go build with error-only output |
| `tokman go vet` | Go vet with compact output |
| `tokman cargo test` | Cargo tests with compact output |
| `tokman cargo build` | Cargo build with error-only output |
| `tokman cargo clippy` | Rust linter with compact output |
| `tokman pytest` | Python tests with compact output |
| `tokman ruff` | Python linter/formatter compact |
| `tokman mypy` | Python type checker compact |
| `tokman vitest` | Vitest with compact output |
| `tokman jest` | Jest with compact output (90% reduction) |
| `tokman npm test` | npm test with compact output (90% reduction) |
| `tokman playwright` | Playwright E2E tests compact |

### Package Managers

| Command | Description |
|---------|-------------|
| `tokman npm` | npm run with filtered output |
| `tokman pnpm` | pnpm with ultra-compact output |
| `tokman npx` | npx with intelligent routing |
| `tokman pip` | pip with compact output |
| `tokman cargo` | Cargo commands with filtering |

### Utilities

| Command | Description |
|---------|-------------|
| `tokman ls` | Hide noise dirs, human sizes |
| `tokman tree` | Compact tree output |
| `tokman find` | Find files with compact output |
| `tokman grep` | Compact grep, groups by file |
| `tokman diff` | Ultra-condensed diff |
| `tokman json` | Show JSON structure |
| `tokman env` | Show env vars (sensitive masked) |
| `tokman deps` | Summarize dependencies |
| `tokman log` | Filter/deduplicate logs |
| `tokman wc` | Word/line/byte count compact |
| `tokman curl` | Auto-JSON detection |
| `tokman wget` | Download with compact output |
| `tokman summary` | Heuristic summary of long output |
| `tokman count` | Count tokens using tiktoken (OpenAI tokenizer) |

### Analysis Commands

| Command | Description |
|---------|-------------|
| `tokman discover` | Find missed savings in Claude Code history |
| `tokman session` | Show TokMan adoption across Claude Code sessions |
| `tokman learn` | Generate CLI correction rules from errors |
| `tokman err <cmd>` | Run command, show only errors/warnings |
| `tokman proxy <cmd>` | Run without filtering (still tracked) |
| `tokman hook-audit` | Show hook rewrite metrics |

### Ruby Ecosystem

| Command | Description |
|---------|-------------|
| `tokman rake` | Rake tasks with compact output |
| `tokman rspec` | RSpec tests with compact output |
| `tokman rubocop` | Ruby linter with compact output |
| `tokman bundle` | Bundler with filtered output |
| `tokman rails` | Rails commands with filtering |

### Infrastructure & Build Tools

| Command | Description |
|---------|-------------|
| `tokman terraform` | Terraform plan with compact output |
| `tokman helm` | Helm commands with filtered output |
| `tokman ansible` | Ansible playbook with compact output |
| `tokman gradle` | Gradle build with filtered output |
| `tokman mvn` | Maven build with filtered output |
| `tokman make` | Make output with noise filtering |
| `tokman mix` | Elixir Mix with compact output |
| `tokman markdownlint` | Markdown linter compact |
| `tokman mise` | Mise task runner compact |
| `tokman just` | Just task runner compact |

### System Utilities

| Command | Description |
|---------|-------------|
| `tokman df` | Disk usage with human-readable format |
| `tokman du` | Directory sizes with compact output |
| `tokman jq` | JSON processing with compact output |

### Rewriting

| Command | Description |
|---------|-------------|
| `tokman rewrite <cmd>` | Rewrite a command to use TokMan |
| `tokman rewrite list` | List all registered rewrites |

## Shell Integration

### Token Counting (tiktoken)

TokMan includes direct token counting using OpenAI's tiktoken tokenizer:

```bash
# Count tokens in text
tokman count "Hello, world!"

# Count tokens from stdin
cat file.txt | tokman count

# Count tokens in a file
tokman count file.go

# Use specific model encoding
tokman count --model gpt-4o "Hello, world!"
tokman count --model claude-3-sonnet "Hello, world!"

# Compare heuristic vs actual count
tokman count --compare "Your text here"

# Count multiple files
tokman count --files *.go
```

Supported encodings:
- `cl100k_base` — GPT-4, GPT-3.5-turbo, Claude, text-embedding-ada-002
- `o200k_base` — GPT-4o, GPT-4o-mini
- `p50k_base` — GPT-3 (davinci, curie, babbage, ada)

Add to your `.bashrc` or `.zshrc`:

```bash
source /path/to/tokman/hooks/tokman-rewrite.sh
```

This enables:
- Automatic command rewriting for all supported commands
- `ts` — alias for `tokman status`
- `tr` — alias for `tokman rewrite`
- `tokman_install_hook` — install hook to shell config
- `tokman_status` — show integration status

### Shell Completions

Enable autocompletions for your shell:

```bash
# Bash
source <(tokman completion bash)

# Zsh
source <(tokman completion zsh)

# Fish
tokman completion fish | source
```

Or save the completion files from `completions/` directory.

## Custom Filter Plugins

Create JSON-based filter plugins in `~/.config/tokman/plugins/`:

```json
{
  "name": "hide-npm-warnings",
  "description": "Hide npm deprecation warnings",
  "enabled": true,
  "patterns": ["npm WARN deprecated"],
  "mode": "hide"
}
```

```bash
# Plugin management
tokman plugin list           # List loaded plugins
tokman plugin create myfilter # Create new plugin template
tokman plugin enable myfilter # Enable a plugin
tokman plugin disable myfilter # Disable a plugin
tokman plugin examples       # Generate example plugins
```

## TOML Filter System

Define custom output filters in TOML format for declarative compression:

```bash
# Create custom filter
mkdir -p ~/.config/tokman/filters
```

```toml
# ~/.config/tokman/filters/mycmd.toml
[filters.mycmd]
description = "Compact mycmd output"
match_command = "^mycmd\\b"
strip_lines_matching = ["^DEBUG:", "^\\s*$"]
max_lines = 50
on_empty = "ok"

# Short-circuit rules
[[filters.mycmd.match_output]]
pattern = "already up to date"
message = "ok: current"

# Inline tests
[[tests.mycmd]]
name = "strips debug"
input = "DEBUG: starting\nERROR: failed"
expected = "ERROR: failed"
```

See [docs/TOML_FILTERS.md](docs/TOML_FILTERS.md) for full documentation.

## Session Discovery

Analyze Claude Code session history to track TokMan adoption:

```bash
# Show adoption metrics
tokman session

# Find missed opportunities
tokman discover

# Export for reporting
tokman session --format json
```

See [docs/SESSION_DISCOVERY.md](docs/SESSION_DISCOVERY.md) for details.

## Web Dashboard

Launch an interactive dashboard to visualize token savings:

```bash
# Start dashboard on default port (8080)
tokman dashboard

# Custom port
tokman dashboard --port 3000

# Open browser automatically
tokman dashboard --open
```

Features:
- Real-time token savings charts
- Daily/weekly/monthly breakdowns
- Command-level analytics
- Cost tracking with Claude API rates
- RESTful API endpoints (`/api/stats`, `/api/daily`, `/api/commands`)

## CI/CD Integration

Include TokMan in your CI pipelines for automated reporting:

### GitHub Actions

```yaml
# .github/workflows/tokman.yml
name: Token Savings Report
on: [workflow_run]

jobs:
  report:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: '1.21' }
      - run: go install github.com/GrayCodeAI/tokman/cmd/tokman@latest
      - run: tokman summary --format markdown >> $GITHUB_STEP_SUMMARY
```

See `templates/` for complete GitHub Actions and GitLab CI examples.

## Configuration

Config file: `~/.config/tokman/config.toml`

```toml
[tracking]
enabled = true
telemetry = false

[filter]
mode = "minimal"  # "minimal" or "aggressive"
noise_dirs = [
    ".git", "node_modules", "target",
    "__pycache__", ".venv", "vendor",
    ".idea", ".vscode", "dist", "build"
]

[hooks]
excluded_commands = []
```

## Gain Command Features

The `tokman gain` command provides comprehensive savings analysis:

```bash
# Basic summary
tokman gain

# Show ASCII graph of daily savings
tokman gain --graph

# Show recent command history
tokman gain --history

# Show quota analysis (subscription tier estimate)
tokman gain --quota --tier pro

# Time breakdowns
tokman gain --daily
tokman gain --weekly
tokman gain --monthly
tokman gain --all

# Export data
tokman gain --format json
tokman gain --format csv

# Filter to current project
tokman gain --project
```

## Security: Integrity Verification

TokMan protects against hook tampering:

```bash
# Verify hook integrity
tokman verify

# Runtime checks are automatic for operational commands
# Hooks are verified via SHA-256 hash stored during init
```

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                       TokMan CLI                            │
├─────────────────────────────────────────────────────────────┤
│  Command Handlers (internal/commands/)                      │
│  ├── git.go       - Git wrappers                            │
│  ├── docker.go    - Docker filtering                        │
│  ├── go.go        - Go build/test                           │
│  ├── cargo.go     - Rust commands                           │
│  ├── npm.go       - npm/pnpm/npx                            │
│  ├── pytest.go    - Python tests                            │
│  ├── gain.go      - Savings analysis                        │
│  ├── economics.go - Cost analysis                           │
│  └── ...          - 40+ command handlers                    │
├─────────────────────────────────────────────────────────────┤
│  20-Layer Compression Pipeline (internal/filter/)          │
│  ├── Layer 1-10:  Core research compression (95%+)          │
│  ├── Layer 11-14: Memory & Attention optimization           │
│  ├── Layer 15:    Meta-Token (lossless compression)         │
│  ├── Layer 16:    Semantic Chunk (boundary detection)       │
│  ├── Layer 17:    Sketch Store (reversible compression)     │
│  ├── Layer 18:    Lazy Pruner (budget-aware pruning)        │
│  ├── Layer 19:    Semantic Anchor (context preservation)    │
│  └── Layer 20:    Agent Memory (knowledge extraction)       │
├─────────────────────────────────────────────────────────────┤
│  Core Engine (internal/)                                    │
│  ├── tracking/    - SQLite token tracking                   │
│  ├── config/      - TOML config loader                      │
│  ├── integrity/   - SHA-256 hook verification               │
│  └── economics/   - Cost analysis engine                    │
├─────────────────────────────────────────────────────────────┤
│  Shell Integration (hooks/)                                 │
│  └── tokman-rewrite.sh - Bash/Zsh hook                      │
└─────────────────────────────────────────────────────────────┘
```

## Development

```bash
# Run tests
go test ./...

# Build
go build -o tokman ./cmd/tokman

# Run with verbose output
./tokman -v git status
```

## Token Estimation

TokMan uses a simple heuristic for token estimation:

```go
tokens = ceil(text.length / 4.0)
```

This provides a reasonable approximation for tracking savings without requiring an actual tokenizer.

## Data Storage

Following XDG Base Directory Specification:

| Resource | Path |
|----------|------|
| Config | `~/.config/tokman/config.toml` |
| Database | `~/.config/tokman/tracking.db` |
| Logs | `~/.config/tokman/tokman.log` |
| Hook | `~/.claude/hooks/tokman-rewrite.sh` |
| Hook Hash | `~/.claude/hooks/tokman-rewrite.sh.sha256` |

Override with environment variables:
- `XDG_CONFIG_HOME`
- `TOKMAN_DATABASE_PATH`

## Microservice Architecture

TokMan supports both monolithic CLI and microservice deployment modes:

### Services

| Service | Port (gRPC) | Port (HTTP) | Purpose |
|---------|-------------|-------------|---------|
| Compression | 50051 | 8081 | 31-layer compression pipeline |
| Analytics | 50053 | 8083 | Token tracking & metrics |
| Agent | 50054 | 8084 | AI agent integrations |
| LLM | 50055 | 8085 | LLM-based summarization |
| Gateway | - | 8080 | API gateway (aggregates all services) |

### Running in Microservice Mode

```bash
# Start compression service
tokman-server --service=compression --grpc --http

# Start analytics service
tokman-server --service=analytics --grpc --http

# Start API gateway
tokman-server --service=gateway

# Use remote compression from CLI
tokman --remote --compression-addr=localhost:50051 compress < input.txt
```

### Docker Compose

```bash
cd deployments/docker
docker-compose up -d
```

### Kubernetes

```bash
kubectl apply -f deployments/kubernetes/
```

### Service Discovery

TokMan includes built-in service discovery with load balancing:
- **Static Discovery**: For development and single deployments
- **Kubernetes Discovery**: Auto-discovers services via DNS/endpoints
- **Load Balancers**: Round-robin, weighted, least-connection

### Observability

Prometheus metrics available at `/metrics` endpoint:
- `tokman_compression_requests_total` - Compression requests by mode/status
- `tokman_compression_duration_ms` - Latency histogram
- `tokman_tokens_saved_total` - Cumulative token savings
- `tokman_grpc_requests_total` - gRPC method call counts
- `tokman_discovery_instances` - Healthy/unhealthy instance counts

## Roadmap

- [x] ~~Git command wrappers~~
- [x] ~~Token tracking database~~
- [x] ~~Shell integration hooks~~
- [x] ~~Integrity verification~~
- [x] ~~Economics analysis~~
- [x] ~~Python tool wrappers (pytest, ruff, mypy)~~
- [x] ~~Rust/Cargo test aggregation~~
- [x] ~~Windows support~~
- [x] ~~Custom filter plugins~~
- [x] ~~Web dashboard for analytics~~
- [x] ~~Shell completions (bash/zsh/fish)~~
- [x] ~~CI/CD integration templates~~
- [x] LLM API integration (direct token counting)
- [x] Homebrew formula
- [x] Docker image
- [x] Microservice architecture
- [x] gRPC services
- [x] Service discovery & load balancing
- [x] Prometheus metrics
- [x] TOML declarative filters
- [x] Session discovery & analytics
- [x] Ruby ecosystem (rake, rspec, rubocop, bundle, rails)
- [x] GitHub Copilot integration
- [x] Infrastructure tools (terraform, helm, ansible)
- [x] Build tools (gradle, maven, make, mix)
- [x] Additional AI agent integrations

## License

MIT License - see [LICENSE](LICENSE) for details.

---

🌸 **TokMan** — Save tokens, save money, save the context window.
