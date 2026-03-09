# TokMan 🌸

**Token-Aware CLI Proxy** — Reduce token usage in LLM interactions by filtering verbose command output.

TokMan intercepts CLI commands, filters their output, and tracks token savings in a SQLite database.

## Features

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
| `tokman learn` | Generate CLI correction rules from errors |
| `tokman err <cmd>` | Run command, show only errors/warnings |
| `tokman proxy <cmd>` | Run without filtering (still tracked) |
| `tokman hook-audit` | Show hook rewrite metrics |

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
┌─────────────────────────────────────────────────┐
│                    TokMan CLI                   │
├─────────────────────────────────────────────────┤
│  Command Handlers (internal/commands/)          │
│  ├── git.go       - Git wrappers               │
│  ├── docker.go    - Docker filtering           │
│  ├── go.go        - Go build/test              │
│  ├── cargo.go     - Rust commands              │
│  ├── npm.go       - npm/pnpm/npx               │
│  ├── pytest.go    - Python tests               │
│  ├── gain.go      - Savings analysis           │
│  ├── economics.go - Cost analysis              │
│  └── ...          - 40+ command handlers       │
├─────────────────────────────────────────────────┤
│  Core Engine (internal/)                        │
│  ├── filter/      - Output filtering            │
│  ├── tracking/    - SQLite token tracking       │
│  ├── config/      - TOML config loader          │
│  ├── integrity/   - SHA-256 hook verification   │
│  └── economics/   - Cost analysis engine        │
├─────────────────────────────────────────────────┤
│  Shell Integration (hooks/)                     │
│  └── tokman-rewrite.sh - Bash/Zsh hook         │
└─────────────────────────────────────────────────┘
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

## License

MIT License - see [LICENSE](LICENSE) for details.

---

🌸 **TokMan** — Save tokens, save money, save the context window.
