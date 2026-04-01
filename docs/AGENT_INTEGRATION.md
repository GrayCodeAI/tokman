# AI Coding Agent CLI Integration Guide

## Top 10 AI Coding Agent CLIs (2025)

| Rank | Agent | Platform | Hook Support | Shell Integration | Config Location |
|------|-------|----------|--------------|-------------------|-----------------|
| 1 | **Claude Code** | CLI/Terminal | PreToolUse, PostToolUse, Notification | Child process via shell | `~/.claude/settings.json` |
| 2 | **Cursor** | VSCode/IDE | PreToolUse, PostToolUse | VSCode Terminal API | `~/.cursor/settings.json` |
| 3 | **Aider** | CLI/Terminal | Git hooks only | Direct shell exec | `~/.aider.conf.yml` |
| 4 | **Cline** | VSCode Extension | Hooks (v3.36+) | VSCode Terminal API | `~/.vscode/extensions/...` |
| 5 | **OpenCode** | CLI/Terminal (Go) | Custom tools/middleware | Go exec.Command | `~/.config/opencode/config.toml` |
| 6 | **Kiro** | CLI/Terminal | Lifecycle hooks | Shell subprocess | `~/.kilorc` |
| 7 | **Kilo Code** | CLI/Terminal | Beta lifecycle hooks | Shell subprocess | `~/.kilorc` |
| 8 | **AdaL** | CLI/Terminal | MCP tools | Shell exec | `~/.adal/config` |
| 9 | **Continue** | VSCode/JetBrains | Limited | Terminal API | `~/.continue/config.json` |
| 10 | **AutoHand** | CLI/Terminal | Unknown | Shell exec | Project-based |

---

## TokMan Integration Architecture

### Hook-Based Integration (Claude Code / Cursor)

TokMan uses a **thin delegator** pattern with Claude Code's `PreToolUse` hook:

```
┌─────────────────────────────────────────────────────────────┐
│                    Claude Code Flow                          │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  User: "Check git status"                                    │
│            │                                                 │
│            ▼                                                 │
│  Claude decides: bash("git status")                         │
│            │                                                 │
│            ▼                                                 │
│  ┌─────────────────────────────────────────────────────┐    │
│  │ PreToolUse Hook (tokman-rewrite.sh)                 │    │
│  │                                                      │    │
│  │  INPUT: {"tool_input": {"command": "git status"}}   │    │
│  │            │                                         │    │
│  │            ▼                                         │    │
│  │  REWRITTEN=$(tokman rewrite "$CMD") || exit 0       │    │
│  │            │                                         │    │
│  │            ▼                                         │    │
│  │  OUTPUT: {"updatedInput": {"command": "tokman git status"}}│ │
│  └─────────────────────────────────────────────────────┘    │
│            │                                                 │
│            ▼                                                 │
│  Shell executes: tokman git status                          │
│            │                                                 │
│            ▼                                                 │
│  Filtered output (~200 tokens instead of ~2000)              │
│            │                                                 │
│            ▼                                                 │
│  Claude receives compressed context                          │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### TokMan Hook Installation (`tokman init -g`)

1. **Creates hook file**: `~/.claude/hooks/tokman-rewrite.sh` (45 lines)
2. **Patches settings.json**: Adds PreToolUse hook entry
3. **Creates TOKMAN.md**: Instructions for Claude to understand tokman commands
4. **Patches CLAUDE.md**: Adds `@TOKMAN.md` reference

### settings.json Structure

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [
          {
            "type": "command",
            "command": "~/.claude/hooks/tokman-rewrite.sh"
          }
        ]
      }
    ]
  }
}
```

---

## TokMan Architecture

| Aspect | TokMan (Go) |
|--------|-------------|
| **Binary Size** | ~12MB (dynamic) |
| **Startup Time** | ~15ms |
| **Registry** | 31 patterns (regex) |
| **Hook Size** | 45 lines |
| **Rewrite Logic** | In binary (registry.go) |
| **Compound Commands** | &&, ||, ;, \|, & |
| **Env Prefixes** | sudo, VAR=val |
| **Exclusion** | config.toml |
| **Status** | TokmanStatus int |

### Command Coverage

| Category | TokMan Commands |
|----------|-----------------|
| Git | git status/diff/log/add/commit/push/pull |
| GitHub | gh pr/issue/run/repo/api/release |
| Cargo | cargo build/test/clippy/check/fmt |
| Files | cat/head/tail, rg/grep, ls, find, tree, diff |
| Build | tsc, eslint, biome, prettier, next |
| Tests | vitest, jest, playwright, pytest |
| Go | go test/build/vet, golangci-lint |
| Python | ruff, pytest, pip, mypy |
| Containers | docker, kubectl |
| Cloud | aws, psql |
| Network | curl, wget |

---

## Integration Strategies by Agent

### 1. Claude Code (Full Support)

**Method**: PreToolUse hook
**Integration**: Automatic with `tokman init -g`

```bash
tokman init -g
# Restarts Claude Code
# All bash commands auto-rewritten
```

### 2. Cursor (Full Support)

**Method**: Same PreToolUse hook as Claude Code
**Integration**: Configure in `~/.cursor/settings.json`

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "Bash",
        "hooks": [{ "type": "command", "command": "~/.claude/hooks/tokman-rewrite.sh" }]
      }
    ]
  }
}
```

### 3. Aider (Shell Wrapper)

**Method**: Shell aliases since Aider has no hook system
**Integration**: Source tokman wrapper

```bash
# In ~/.bashrc or ~/.zshrc
alias git='tokman git'
alias ls='tokman ls'
alias cat='tokman read'
alias rg='tokman grep'
```

### 4. Cline (VSCode Terminal)

**Method**: Configure default shell with tokman
**Integration**: VSCode settings

```json
{
  "terminal.integrated.defaultProfile.linux": "bash",
  "terminal.integrated.profiles.linux": {
    "bash": {
      "path": "/bin/bash",
      "args": ["-c", "source ~/.local/share/tokman/hooks/tokman-rewrite.sh && exec bash"]
    }
  }
}
```

### 5. OpenCode (Custom Tool)

**Method**: Custom tool wrapper in config
**Integration**: `~/.config/opencode/config.toml`

```toml
[tools.shell]
command = "tokman proxy"
```

### 6. Kiro (Lifecycle Hooks)

**Method**: Kiro's hook system
**Integration**: `~/.kilorc`

```yaml
hooks:
  preToolUse:
    - matcher: "Bash"
      command: "tokman rewrite"
```

### 7. Continue (Limited)

**Method**: Terminal environment
**Integration**: Set environment variable

```bash
export TOKMAN_AUTO_REWRITE=1
```

---

## Recommended Integration Priority

| Priority | Agent | Method | Effort | Coverage |
|----------|-------|--------|--------|----------|
| **P0** | Claude Code | PreToolUse hook | Low | 100% |
| **P0** | Cursor | PreToolUse hook | Low | 100% |
| **P1** | Aider | Shell aliases | Medium | 80% |
| **P1** | Cline | Terminal profile | Medium | 70% |
| **P2** | OpenCode | Custom tool | Medium | 60% |
| **P2** | Kiro | Lifecycle hooks | Medium | 60% |
| **P3** | Continue | Environment | Low | 30% |
| **P3** | Others | Shell wrapper | Medium | Varies |

---

## TokMan Integration TODO

1. **tokman init -g** command enhancement:
   - Auto-patch `~/.claude/settings.json`
   - Create `~/.claude/hooks/tokman-rewrite.sh`
   - Create `~/.claude/TOKMAN.md` instructions
   - Add `@TOKMAN.md` to `~/.claude/CLAUDE.md`

2. **Multi-agent support**:
   - Detect which agent is running
   - Apply appropriate integration method
   - Support `--agent=cursor`, `--agent=aider` flags

3. **Dashboard integration**:
   - Track per-agent token savings
   - Show integration status per agent

4. **Uninstall command**:
   - `tokman init -g --uninstall`
   - Remove all artifacts cleanly

---

## MCP Context Examples

TokMan can also act as a context service instead of only a shell rewriter.

### Start the MCP server

```bash
tokman mcp --port 8080
```

### Read one file under a token budget

```bash
curl -X POST http://localhost:8080/read \
  -H "Content-Type: application/json" \
  -d '{
    "path": "internal/server/server.go",
    "mode": "auto",
    "max_tokens": 350,
    "save_snapshot": true
  }'
```

### Request a graph-aware bundle

```bash
curl -X POST http://localhost:8080/bundle \
  -H "Content-Type: application/json" \
  -d '{
    "path": "internal/server/server.go",
    "mode": "graph",
    "related_files": 4,
    "max_tokens": 500
  }'
```

### Recommended agent usage

- Claude Code / Cursor:
  - use shell hooks for normal command rewriting
  - use `POST /read` or `POST /bundle` when the agent needs curated file context
- Codex / OpenCode:
  - keep shell wrapping for command noise reduction
  - use `POST /bundle` for target-file + related-file context delivery
- Any MCP-capable tool:
  - use `POST /read` for single-file refreshes
  - use `POST /bundle` for multi-file graph context

### Direct integration snippets

Claude Code / Cursor style bundle request:

```json
{
  "tool": "tokman.read_bundle",
  "server": "http://localhost:8080",
  "method": "POST",
  "path": "/bundle",
  "body": {
    "path": "internal/server/server.go",
    "mode": "graph",
    "related_files": 4,
    "max_tokens": 500,
    "save_snapshot": true
  }
}
```

Codex / OpenCode style single-file refresh:

```json
{
  "tool": "tokman.read_file",
  "server": "http://localhost:8080",
  "method": "POST",
  "path": "/read",
  "body": {
    "path": "internal/contextread/read.go",
    "mode": "auto",
    "max_tokens": 320,
    "save_snapshot": true
  }
}
```

Recommended pattern:
- use `/bundle` first for target file + neighbors
- switch to `/read` for focused refreshes
- use `mode=delta` after edits when the agent already saw the file earlier
