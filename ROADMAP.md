# TokMan: World's Best Token Reduction System - Enhancement Roadmap

**Date:** 2026-03-20  
**POC:** L Patel  
**TL;DR:** TokMan already has a world-class 14-layer compression pipeline. This roadmap focuses on: (1) universal agent support, (2) enhanced intelligence, and (3) developer experience improvements.

---

## Executive Summary

TokMan is positioned as the leading token reduction system with:
- ✅ **14 research-based compression layers** (95-99% reduction)
- ✅ **146+ CLI command wrappers**
- ✅ **6 agent integrations** (Claude Code, Cursor, Gemini CLI, Codex CLI, OpenCode, Cline)

To become the undisputed world's best, we need:
- 🎯 **Universal agent support** (all 15+ major coding agents)
- 🎯 **AI-powered smart compression** (LLM-aware optimization)
- 🎯 **Real-time analytics dashboard**
- 🎯 **Plugin ecosystem**

---

## Current State Analysis

### Supported Agents (6/15+)

| Agent | Status | Integration Quality |
|-------|--------|---------------------|
| Claude Code | ✅ Full | PreToolUse hook, 100% coverage |
| Cursor | ✅ Full | Same hook system, 100% coverage |
| Gemini CLI | ✅ Full | Hook + config, 100% coverage |
| Codex CLI | ✅ Full | Config-based, 100% coverage |
| OpenCode | ✅ Partial | Plugin needed, 60% coverage |
| Cline | ✅ Partial | Rules file only, 50% coverage |
| **Aider** | ⚠️ Shell aliases | Needs native integration |
| **Continue** | ⚠️ Limited | Environment var only |
| **Windsurf** | ⚠️ Rules only | No hook support |
| **Copilot** | ⚠️ Hook only | No full integration |
| **AdaL** | ❌ Missing | High priority |
| **Kiro** | ❌ Missing | Medium priority |
| **Kilo Code** | ❌ Missing | Medium priority |
| **AutoHand** | ❌ Missing | Research needed |
| **Replit Agent** | ❌ Missing | Research needed |

### Compression Pipeline (World-Class)

The 14-layer pipeline is already state-of-the-art:
1. Entropy Filtering (2-3x)
2. Perplexity Pruning (20x)
3. Goal-Driven Selection (14.8x)
4. AST Preservation (4-8x)
5. Contrastive Ranking (4-10x)
6. N-gram Abbreviation (2.5x)
7. Evaluator Heads (5-7x)
8. Gist Compression (20x+)
9. Hierarchical Summary (Extreme)
10. Budget Enforcement (Guaranteed)
11. Compaction Layer (Auto)
12. Attribution Filter (78%)
13. H2O Filter (30x+)
14. Attention Sink (Infinite stability)

---

## Phase 1: Universal Agent Support (Priority: P0)

### 1.1 AdaL Integration (High Priority)

AdaL (SylphAI's agent) is missing but mentioned in docs. Need:

```go
// internal/agents/adal.go
type AdaLAgent struct {
    Name        string
    ConfigPath  string // ~/.adal/config
    HookSupport bool   // MCP tools available
}

func (a *AdaLAgent) Detect() bool {
    // Check for ~/.adal/config
    // Check for ADAL_SESSION_ID env var
}

func (a *AdaLAgent) Setup() error {
    // Configure MCP tool integration
    // Add tokman as MCP server
}
```

**Files to create:**
- `internal/agents/adal.go` - Agent detector and setup
- `internal/commands/init_adal.go` - Init command integration
- `hooks/adal-mcp.json` - MCP server configuration

### 1.2 Aider Native Integration

Currently uses shell aliases. Improve with:

```yaml
# ~/.aider.conf.yml integration
map-run:
  "git *": "tokman git"
  "ls *": "tokman ls"
  "cat *": "tokman read"
```

**Files to create:**
- `internal/agents/aider.go` - Native config generation
- Update `init.go` with `--aider` flag

### 1.3 Continue Dev Integration

```json
// ~/.continue/config.json
{
  "experimental": {
    "tokenOptimization": {
      "enabled": true,
      "wrapper": "tokman"
    }
  }
}
```

### 1.4 Kiro/Kilo Code Integration

Research hook system and create lifecycle hooks.

### 1.5 Replit Agent Integration

Cloud-based agent - may need API-level integration.

---

## Phase 2: Intelligence Enhancements (Priority: P1)

### 2.1 LLM-Aware Compression Mode

Add intelligent mode that uses LLM for context-aware compression:

```go
// internal/filter/llm_compress.go
type LLMCompressor struct {
    Provider string // openai, anthropic, local
    Model    string // gpt-4, claude-3, llama
    Mode     string // summarize, extract, prioritize
}

func (l *LLMCompressor) Compress(input string, queryIntent string) string {
    // Use LLM to identify critical information
    // Generate dense summary preserving key facts
}
```

### 2.2 Multi-File Optimization

When reading multiple files, deduplicate shared context:

```go
type MultiFileOptimizer struct {
    SharedImports  map[string]bool
    SharedTypes    map[string]bool
    SharedPatterns []string
}

func (m *MultiFileOptimizer) Optimize(files []string) map[string]string {
    // Extract shared context once
    // Apply to all files
    // Track what was seen per session
}
```

### 2.3 Query-Intent Detection

Automatically detect what the user wants:

```go
type QueryIntent string

const (
    IntentDebug      QueryIntent = "debug"      // Focus on errors
    IntentUnderstand QueryIntent = "understand" // Focus on structure
    IntentModify     QueryIntent = "modify"     // Focus on specific areas
    IntentSearch     QueryIntent = "search"     // Focus on matches
)
```

---

## Phase 3: Developer Experience (Priority: P1)

### 3.1 Real-Time Dashboard

Web-based dashboard for token analytics:

```bash
tokman dashboard --port 8080
```

Features:
- Live token savings counter
- Per-agent breakdown
- Cost savings calculator
- Command frequency analysis
- Compression ratio trends

**Files to create:**
- `internal/dashboard/server.go` - HTTP server
- `internal/dashboard/ws.go` - WebSocket for live updates
- `internal/dashboard/templates/` - HTML templates

### 3.2 Unified Init Command

Single command to set up all agents:

```bash
# Detect and setup all installed agents
tokman init --all

# Setup specific agents
tokman init --agents claude,cursor,aider,adal
```

### 3.3 Smart Suggestions

```bash
tokman suggest
```

Analyzes recent commands and suggests optimizations:
```
💡 Suggestions based on your workflow:

  1. You run 'git status' frequently. 
     TokMan can save ~1,500 tokens/run.
     Run: tokman init -g

  2. Your 'npm test' outputs are large (~5K tokens).
     Use: tokman test npm test
     Estimated savings: 4,800 tokens/run

  3. Consider enabling aggressive mode:
     tokman config set mode aggressive
```

---

## Phase 4: Plugin Ecosystem (Priority: P2)

### 4.1 Plugin System

Allow community extensions:

```go
// internal/plugin/manager.go
type Plugin interface {
    Name() string
    Version() string
    Commands() []Command
    Filters() []Filter
}

func LoadPlugin(path string) (Plugin, error) {
    // Load .so (Go) or .wasm (universal) plugins
}
```

### 4.2 Plugin Registry

```bash
tokman plugin search pytest-advanced
tokman plugin install tokman-plugin-pytest
tokman plugin list
```

---

## Phase 5: Advanced Features (Priority: P2)

### 5.1 Session Context Sharing

Share context between agents:

```go
type SessionContext struct {
    ID           string
    FilesSeen    map[string]FileSummary
    TypesDefined map[string]TypeDefinition
    ErrorsSeen   []Error
}

func (s *SessionContext) Sync() error {
    // Sync to ~/.local/share/tokman/sessions/
    // Available to all agents
}
```

### 5.2 MCP Server Mode

Expose TokMan as MCP server for deeper integration:

```json
// mcp-server-config.json
{
  "name": "tokman",
  "command": "tokman",
  "args": ["mcp", "serve"],
  "tools": [
    "compress",
    "status", 
    "suggest"
  ]
}
```

### 5.3 Compression Profiles

Agent-specific optimization profiles:

```yaml
# ~/.config/tokman/profiles.yaml
profiles:
  claude-code:
    mode: aggressive
    budget: 2000
    layers: [1,2,3,4,5,6,7,8,9,10,11,12,13,14]
  
  cursor:
    mode: minimal
    budget: 4000
    preserve_comments: true
  
  aider:
    mode: aggressive
    focus: [git, files, tests]
```

---

## Implementation Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| **Phase 1** | 2 weeks | AdaL, Aider, Continue, Kiro integrations |
| **Phase 2** | 2 weeks | LLM compression, multi-file optimization |
| **Phase 3** | 1 week | Dashboard, unified init, suggestions |
| **Phase 4** | 2 weeks | Plugin system, registry |
| **Phase 5** | 2 weeks | Session sharing, MCP server, profiles |

**Total: ~9 weeks to world's best status**

---

## Success Metrics

1. **Agent Coverage:** 15+ agents supported (currently 6)
2. **Token Savings:** Maintain 95-99% reduction rate
3. **User Adoption:** 10,000+ GitHub stars
4. **Performance:** <20ms overhead per command
5. **Ecosystem:** 50+ community plugins

---

## Next Steps

1. ✅ Create this roadmap
2. ✅ Prioritize Phase 1 agents (AdaL first)
3. ✅ Implement agent detection improvements
4. ✅ Build LLM-aware compression prototype
5. ✅ Design dashboard UI
6. ✅ Implement plugin system (Go .so + WASM)
7. ✅ Session context sharing

---

## Implementation Status (Updated 2026-03-20)

### ✅ Completed

**Phase 1: Universal Agent Support**
- AdaL integration (`internal/commands/init_adal.go`)
- Kiro, Kilo Code, Windsurf, Replit, Gemini CLI, OpenCode detection
- Multi-agent init flags (`--all`, `--adal`, `--kiro`, etc.)

**Phase 2: Intelligence Enhancements**
- LLM Compressor (`internal/filter/llm_compress.go`)
- Multi-file optimization (`internal/filter/multifile.go`, `internal/filter/multi_file.go`)

**Phase 3: Developer Experience**
- Enhanced `tokman suggest` command (workflow analysis mode)
- Dashboard already comprehensive with real-time updates

**Phase 4: Plugin System**
- Go .so native plugin support (`internal/plugin/manager.go`)
- WASM plugin support with wazero runtime (`internal/plugin/wasm.go`)
- Plugin interface with Filters, Commands, lifecycle hooks

**Phase 5: Advanced Features**
- Session context sharing (`internal/session/context.go`)
- MCP server mode (existing `internal/commands/mcp.go`)
- File tracking, type definitions, error tracking per session

### 🔄 Remaining Enhancements
- Compression profiles per agent (`~/.config/tokman/profiles.yaml`)
- Plugin registry for community plugins
- WASI filesystem support for WASM plugins

---

## References

- [AGENT_INTEGRATION.md](docs/AGENT_INTEGRATION.md) - Current integration docs
- [LAYERS.md](docs/LAYERS.md) - Compression layer documentation
- Research papers in `/docs/research/`
