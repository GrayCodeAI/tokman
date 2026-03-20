# TokMan Architecture Documentation

> **TokMan (Token Manager)** - A CLI proxy that minimizes LLM token consumption through intelligent output filtering and a 14-layer compression pipeline.

This document provides an architectural overview of TokMan, including system design, data flows, module organization, and implementation patterns.

---

## Table of Contents

1. [System Overview](#system-overview)
2. [Command Lifecycle](#command-lifecycle)
3. [Module Organization](#module-organization)
4. [14-Layer Compression Pipeline](#14-layer-compression-pipeline)
5. [TOML Filter System](#toml-filter-system)
6. [Token Tracking System](#token-tracking-system)
7. [Hook System & Agent Integration](#hook-system--agent-integration)
8. [Configuration System](#configuration-system)
9. [Build & Deployment](#build--deployment)

---

## System Overview

### Proxy Pattern Architecture

TokMan is a Go CLI proxy that intercepts shell commands, runs them, and applies intelligent filtering/summarization to reduce LLM token consumption by 60-99%.

### Key Components

| Component | Location | Responsibility |
|-----------|----------|----------------|
| **CLI Parser** | `cmd/tokman/main.go` | Cobra-based argument parsing, global flags |
| **Command Router** | `internal/commands/` | 100+ command modules |
| **Filter Engine** | `internal/filter/` | 14-layer compression pipeline |
| **TOML Filters** | `internal/toml/` | 93+ built-in TOML filter definitions |
| **Tracking** | `internal/tracking/` | SQLite-based token metrics |
| **Rewrite Engine** | `internal/discover/` | Command classification and rewriting |
| **Config** | `internal/config/` | Hierarchical configuration |
| **Dashboard** | `internal/dashboard/` | Web-based savings visualization |
| **Hook Processors** | `internal/commands/hook.go` | Native Gemini/Copilot hook processors |

### Design Principles

1. **Single Binary**: Zero runtime dependencies
2. **Fail-Safe**: If filtering fails, fall back to original output
3. **Exit Code Preservation**: CI/CD reliability
4. **Agent-Agnostic**: Supports Claude Code, Cursor, Copilot, Gemini, Windsurf, Cline, OpenCode
5. **Research-Based**: 14-layer pipeline based on academic compression papers

---

## Command Lifecycle

### Six-Phase Execution Flow

1. **PARSE**: Cobra extracts command, args, flags
2. **ROUTE**: Match to command handler in `internal/commands/`
3. **EXECUTE**: Run underlying tool via `internal/core/runner.go`
4. **FILTER**: Apply 14-layer compression pipeline
5. **PRINT**: Output compressed result
6. **TRACK**: Record token savings in SQLite

### Verbosity Levels

| Flag | Behavior |
|------|----------|
| (none) | Compact output only |
| `-v` | + Debug messages |
| `-vv` | + Command being executed |
| `-vvv` | + Raw output before filtering |

---

## Module Organization

### Command Modules (`internal/commands/`)

| Category | Commands | Savings |
|----------|----------|---------|
| **Git** | status, diff, log, show, add, commit, push, branch, fetch, stash | 59-80% |
| **GitHub** | gh pr, issue, run, repo, api, release | 26-87% |
| **Cargo** | build, test, clippy, check | 80-90% |
| **Docker** | ps, images, logs, run, exec, build, compose | 85% |
| **Kubernetes** | get, logs, describe, apply | 85% |
| **JS/TS** | tsc, lint, prettier, next, vitest, jest, playwright, prisma, npm, pnpm, npx | 70-99% |
| **Python** | ruff, pytest, pip, mypy | 75-90% |
| **Go** | test, build, vet, golangci-lint | 75-90% |
| **.NET** | build, test, restore, format | 70-85% |
| **Files** | ls, tree, read, grep, find, diff, json | 60-80% |
| **Network** | curl, wget | 65-70% |
| **Infra** | aws, psql, kubectl | 70-85% |

### Internal Packages

| Package | Purpose |
|---------|---------|
| `internal/core/` | Command runner, token estimator, cost calculator |
| `internal/filter/` | 14-layer compression pipeline (75 files) |
| `internal/toml/` | TOML filter system (93+ built-in filters) |
| `internal/tracking/` | SQLite token tracking |
| `internal/discover/` | Command classification & rewriting |
| `internal/config/` | Hierarchical configuration |
| `internal/dashboard/` | Web dashboard |
| `internal/server/` | MCP server |
| `internal/economics/` | Spending vs savings analysis |
| `internal/integrity/` | Hook SHA-256 verification |
| `internal/telemetry/` | Optional usage telemetry |

---

## 14-Layer Compression Pipeline

TokMan's filter pipeline is based on 14 research papers:

| Layer | Module | Research Paper | Compression |
|-------|--------|---------------|-------------|
| 1 | `entropy.go` | Selective Context (Mila 2023) | 2-3x |
| 2 | `perplexity.go` | LLMLingua (Microsoft 2023) | 20x |
| 3 | `goal_driven.go` | SWE-Pruner (Shanghai Jiao Tong 2025) | 14.8x |
| 4 | `ast_preserve.go` | LongCodeZip (NUS 2025) | 4-8x |
| 5 | `contrastive.go` | LongLLMLingua (Microsoft 2024) | 4-10x |
| 6 | `ngram.go` | CompactPrompt (2025) | 2.5x |
| 7 | `evaluator_heads.go` | EHPC (Tsinghua/Huawei 2025) | 5-7x |
| 8 | `gist.go` | Stanford/Berkeley (2023) | 20x+ |
| 9 | `hierarchical_summary.go` | AutoCompressor (Princeton/MIT 2023) | Extreme |
| 10 | `budget.go` | Industry standard | Guaranteed |
| 11 | `compaction.go` | MemGPT (UC Berkeley 2023) | 98%+ |
| 12 | `attribution.go` | ProCut (LinkedIn 2025) | 78% |
| 13 | `h2o.go` | H2O / Heavy-Hitter Oracle (NeurIPS 2023) | 30x+ |
| 14 | `attention_sink.go` | StreamingLLM (2023) | Infinite stability |

See `docs/LAYERS.md` for detailed layer documentation.

---

## TOML Filter System

### 8-Stage Pipeline

1. `strip_ansi` — Remove ANSI escape codes
2. `replace` — Regex substitutions with backreferences
3. `match_output` — Short-circuit pattern matching with `unless`
4. `strip_lines_matching` / `keep_lines_matching` — Line filtering
5. `truncate_lines_at` — Truncate lines to N chars
6. `head_lines` / `tail_lines` — Keep first/last N lines
7. `max_lines` — Absolute line cap
8. `on_empty` — Message if result is empty

### Filter Locations

1. **Built-in**: `internal/toml/builtin/*.toml` (93 filters)
2. **User-global**: `~/.config/tokman/filters.toml`
3. **Project-local**: `.tokman/filters.toml` (trust-gated)

---

## Token Tracking System

- **Storage**: SQLite at `~/.local/share/tokman/tracking.db`
- **Retention**: 90-day automatic cleanup
- **Estimation**: Heuristic `ceil(len/4.0)` tokens
- **Metrics**: input/output tokens, savings %, execution time, command

---

## Hook System & Agent Integration

### Supported Platforms

| Platform | Init Flag | Hook Type |
|----------|-----------|-----------|
| Claude Code | `tokman init -g` | PreToolUse shell hook |
| Cursor | `tokman init --cursor` | PreToolUse shell hook |
| Copilot | `tokman init --copilot` | Native `tokman hook copilot` |
| Gemini | `tokman init --gemini` | Native `tokman hook gemini` |
| Windsurf | `tokman init --windsurf` | Rules file |
| Cline | `tokman init --cline` | Rules file |
| OpenCode | `tokman init --opencode` | TypeScript plugin |
| OpenClaw | Manual install | TypeScript plugin |
| Codex CLI | `tokman init --codex` | AGENTS.md + TOKMAN.md |

### Rewrite Engine

The rewrite engine in `internal/discover/registry.go` handles:
- Compound commands (`&&`, `||`, `;`, `|`, `&`)
- Environment variable prefixes (`NODE_ENV=test npm run build`)
- Git global options (`-C`, `-c`, `--git-dir`, etc.)
- Absolute path normalization (`/usr/bin/grep` → `grep`)
- Pipe-skip for `find`/`fd`/`locate`
- Heredoc and arithmetic expansion detection
- `head -N` / `tail -N` numeric rewriting

---

## Configuration System

### Config File: `~/.config/tokman/config.toml`

```toml
[tracking]
enabled = true
history_days = 90

[filter]
noise_dirs = [".git", "node_modules", "target"]
mode = "minimal"

[pipeline]
enable_entropy = true
enable_perplexity = true
budget_enforce = true

[hooks]
excluded_commands = ["curl", "playwright"]

[dashboard]
port = 8080
```

---

## Build & Deployment

### Build

```bash
go build -o tokman ./cmd/tokman/
```

### Package Formats

| Format | Location |
|--------|----------|
| Homebrew | `homebrew/tokman.rb`, `Formula/tokman.rb` |
| AUR | `aur/PKGBUILD` |
| Docker | `docker/Dockerfile`, `docker/docker-compose.yml` |
| Shell completions | `completions/` (bash, fish, zsh) |
