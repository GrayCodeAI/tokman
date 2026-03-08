# TokMan 🌸

**Token-Aware CLI Proxy** — Reduce token usage in LLM interactions by filtering verbose command output.

A Go implementation inspired by [RTK (Rust Token Killer)](https://github.com/your-repo/rtk), designed to intercept CLI commands, filter their output, and track token savings.

## Features

- 🔧 **Git Command Wrappers** — Filtered `status`, `diff`, and `log` with smart formatting
- 📁 **LS Handler** — Hide noise directories (.git, node_modules, target, etc.)
- 🧪 **Test Aggregation** — Condense Go test output into summary format
- 🏗️ **Build Filtering** — Show only errors and warnings from builds
- 📊 **Token Tracking** — SQLite-based metrics on tokens saved
- 🔄 **Shell Integration** — Automatic command rewriting via shell hooks

## Installation

```bash
# Clone and build
git clone https://github.com/Patel230/tokman.git
cd tokman
go build -o tokman ./cmd/tokman

# Install to PATH (optional)
sudo mv tokman /usr/local/bin/
```

## Quick Start

```bash
# Initialize database and config
tokman init

# Check token savings
tokman status

# Use wrapped commands
tokman git status
tokman ls
tokman test ./...
```

## Commands

### Core Commands

| Command | Description |
|---------|-------------|
| `tokman init` | Initialize database and config |
| `tokman status` | Show token savings summary |
| `tokman report` | Detailed usage analytics |

### Git Wrappers

| Command | Filter Applied |
|---------|---------------|
| `tokman git status` | Porcelain parsing, emoji formatting |
| `tokman git diff` | Stats summary, 30-line hunk limit |
| `tokman git log` | Oneline format, 20-commit limit |

### Other Wrappers

| Command | Filter Applied |
|---------|---------------|
| `tokman ls [path]` | Hide noise dirs, group by type, human sizes |
| `tokman test [args]` | Aggregate test results, show failures only |
| `tokman build [args]` | Filter verbose output, show errors only |

### Rewriting

| Command | Description |
|---------|-------------|
| `tokman rewrite <cmd>` | Rewrite a command to use TokMan |
| `tokman rewrite list` | List all registered rewrites |

## Shell Integration

Add to your `.bashrc` or `.zshrc`:

```bash
source /path/to/tokman/hooks/tokman-rewrite.sh
```

This enables:
- `ts` — alias for `tokman status`
- `tr` — alias for `tokman rewrite`
- `tokman_install_hook` — install hook to shell config
- `tokman_status` — show integration status

## Configuration

Config file: `~/.config/tokman/config.toml`

```toml
[tracking]
enabled = true
database_path = ""  # Default: ~/.local/share/tokman/history.db

[filter]
mode = "minimal"  # "minimal" or "aggressive"
noise_dirs = [
    ".git", "node_modules", "target",
    "__pycache__", ".venv", "vendor"
]

[hooks]
excluded_commands = []
```

## Filter Modes

### Minimal (default)
- Strip ANSI escape codes
- Remove duplicate log lines
- Limit output size

### Aggressive
- Strip function bodies (brace-depth tracking)
- Condense imports
- Maximum token reduction

## Architecture

```
┌─────────────────────────────────────────────────┐
│                    TokMan CLI                   │
├─────────────────────────────────────────────────┤
│  Command Handlers (internal/commands/)          │
│  ├── git.go      - Git wrappers                │
│  ├── ls.go       - LS with noise filtering     │
│  ├── test.go     - Test aggregation            │
│  └── rewrite.go  - Command rewriting           │
├─────────────────────────────────────────────────┤
│  Core Engine (internal/)                        │
│  ├── filter/     - Output filtering            │
│  ├── tracking/   - SQLite token tracking       │
│  ├── config/     - TOML config loader          │
│  └── discover/   - Command registry            │
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
| Database | `~/.local/share/tokman/history.db` |
| Logs | `~/.local/share/tokman/tokman.log` |

Override with environment variables:
- `XDG_CONFIG_HOME`
- `XDG_DATA_HOME`
- `TOKMAN_DATABASE_PATH`

## Roadmap

- [ ] Windows support
- [ ] Custom filter plugins
- [ ] Web dashboard for analytics
- [ ] LLM API integration (direct token counting)
- [ ] Rust/Cargo test aggregation
- [ ] Python pytest aggregation

## License

MIT License - see [LICENSE](LICENSE) for details.

## Credits

Inspired by [RTK (Rust Token Killer)](https://github.com/your-repo/rtk).

---

🌸 **TokMan** — Save tokens, save money, save the context window.
