# TokMan User Guide

**Complete guide for individual developers using TokMan**

---

## Quick Start

### Installation

```bash
# Clone and build
git clone https://github.com/GrayCodeAI/tokman.git
cd tokman
go build -o tokman ./cmd/tokman

# Initialize shell integration
./tokman init
source ~/.bashrc  # or ~/.zshrc
```

### Basic Usage

After installation, TokMan automatically filters output from supported commands:

```bash
# These are now automatically filtered:
git status          # → Compact status
npm test            # → Aggregated results
docker ps           # → Compact container list
cargo test          # → Summary with failures highlighted
```

---

## Output Modes

### Standard Mode (Default)
Human-readable with emoji icons:
```bash
tokman git status
# 📝 Modified: 3 | ➕ Staged: 2 | ❓ Untracked: 5
```

### Ultra-Compact Mode (`-u`)
Maximum token savings with ASCII-only output:
```bash
tokman -u git status
# M:3 S:2 U:5
# M src/main.go
```

### JSON Output (`--format json`)
For scripting and automation:
```bash
tokman status --format json
# {"commands": 1234, "saved": 89234, "percent": 71}
```

---

## Advanced Compression Features

### Query-Aware Compression

Tailor output to your current task:

```bash
# Focus on debugging
tokman --query debug cargo test

# Focus on code review
tokman --query review git diff

# Focus on deployment
tokman --query deploy docker ps

# Or use environment variable
TOKMAN_QUERY=debug tokman npm test
```

**Available intents:**
| Intent | Use Case | Prioritizes |
|--------|----------|-------------|
| `debug` | Finding bugs | Errors, stack traces, failures |
| `review` | Code review | Changes, diffs, file references |
| `deploy` | Deployments | Status, versions, health |
| `search` | Finding code | File names, definitions |
| `test` | Testing | Results, coverage, failures |
| `build` | Build status | Errors, warnings |

### Hierarchical Summarization

Automatic for very large outputs (500+ lines):

```bash
# Large test output automatically summarized
tokman cargo test --all

# Output structure:
[Hierarchical Summary: 1000 lines → 15 sections]

├─ [L1-200] Critical errors (preserved verbatim)
error[E0277]: trait bound not satisfied
  --> src/main.rs:10:5

├─ [L250-300] Warnings (one-line summary)
⚠ 5 warnings about unused variables

├─ [L400-800] Build progress (dropped)
...
```

### Local LLM Integration

For intelligent summarization when quality > speed:

```bash
# Requires Ollama or LM Studio running locally
tokman --llm cargo test

# Or via environment
TOKMAN_LLM=true tokman npm test

# Configure provider
TOKMAN_LLM_PROVIDER=ollama
TOKMAN_LLM_MODEL=llama3.2:3b
TOKMAN_LLM_BASE_URL=http://localhost:11434
```

**Performance:**
- Latency: 50-200ms (depends on hardware)
- Quality: 40-60% better semantic preservation
- Privacy: All processing is local

### Multi-File Context Optimization

Automatic when working with multiple files:

```bash
# Diff with multiple files gets deduplicated
tokman git diff main...feature

# Shared imports extracted, relationships preserved
=== Shared Imports ===
import "fmt"

=== File: main.go ===
...
```

---

## Token Savings Analysis

### View Your Savings

```bash
# Quick summary
tokman status
# 🌸 TokMan Status
# Commands: 1,234 | Tokens Saved: 89,234 (71%)

# Detailed report
tokman report --daily

# Full analysis with graphs
tokman gain --graph --all
```

### Cost Estimation

```bash
tokman economics
# 💰 Economics Analysis
# Estimated spent: $12.34
# Estimated saved:  $45.67
# Net benefit:      +$33.33
```

---

## Custom LLM Prompts

Create custom prompts for specific workflows:

```bash
# Create template directory
mkdir -p ~/.local/share/tokman/prompts

# Create custom template
cat > ~/.local/share/tokman/prompts/security.json << 'EOF'
{
  "name": "security_review",
  "description": "Security-focused code review",
  "system_prompt": "You are a security auditor.",
  "user_prompt": "Focus on: vulnerabilities, auth issues, data exposure\n\n{{content}}",
  "intent": "review",
  "max_tokens": 400,
  "temperature": 0.2
}
EOF

# Use with --llm
tokman --llm git diff
```

**Built-in templates:**
- `debug` - Error and stack trace focus
- `review` - Code change analysis
- `test` - Test results summary
- `build` - Build status
- `deploy` - Deployment health
- `search` - File/definition lookup
- `concise` - Brief summary
- `detailed` - Full technical details

---

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `TOKMAN_DISABLED=1` | Disable command rewriting | Enabled |
| `TOKMAN_QUERY` | Query intent (debug/review/...) | None |
| `TOKMAN_LLM=true` | Enable LLM summarization | Disabled |
| `TOKMAN_LLM_PROVIDER` | LLM provider (ollama/lmstudio) | auto-detect |
| `TOKMAN_LLM_MODEL` | Model name | llama3.2:3b |
| `TOKMAN_LLM_BASE_URL` | LLM API endpoint | auto-detect |
| `TOKMAN_DATABASE_PATH` | Custom database location | ~/.local/share/tokman |

---

## Performance Tips

### 1. Use Ultra-Compact Mode for LLM Context
```bash
tokman -u git status  # Maximum token savings
```

### 2. Match Query Intent to Task
```bash
TOKMAN_QUERY=debug tokman cargo test  # Focus on failures
```

### 3. Use Hierarchical for Large Outputs
```bash
# Automatic - no flag needed
tokman go test ./...  # Large output summarized
```

### 4. LLM for Critical Summaries
```bash
# When quality matters more than speed
tokman --llm npm test
```

---

## Common Workflows

### Agent Context Delivery

Use `ctx` when you want TokMan to shape context for an AI tool directly.

```bash
# Single-file context
tokman ctx read cmd/tokman/main.go --mode auto --max-tokens 300

# Incremental re-read after editing
tokman ctx delta cmd/tokman/main.go

# Project-aware context bundle
tokman ctx read internal/server/server.go --mode graph --related-files 4
```

Recommended usage:
- `--mode auto` for a single file with a tight token budget
- `--mode delta` after iterative edits
- `--mode graph` when the agent needs surrounding project context

Dashboard support:
- `/api/context-reads` shows recent smart reads
- `/api/context-read-summary` breaks them down into `read`, `delta`, and `mcp`
- `/api/context-read-trend` shows smart-read savings over time
- `/api/context-read-top-files` and `/api/context-read-projects` show where context delivery is paying off
- the dashboard UI can filter smart-read activity by type

### Debugging Failed Tests

```bash
# Focus on errors and failures
TOKMAN_QUERY=debug tokman cargo test

# Save full output for later
tokman cargo test  # Auto-saved to ~/.local/share/tokman/tee/ on failure
```

### Code Review

```bash
# Focus on changes
tokman --query review git diff main...feature

# Ultra-compact for LLM context
tokman -u --query review git diff
```

### CI/CD Integration

```bash
# In CI scripts - use JSON output
tokman status --format json >> metrics.json

# Fail-fast with filtered output
tokman go test ./... || exit 1
```

### Finding Code

```bash
# Search-focused output
tokman --query search grep -r "function" src/
```

---

## Troubleshooting

### Command Not Being Filtered

Check if the command is registered:
```bash
tokman rewrite list
```

### LLM Not Working

Verify Ollama/LM Studio is running:
```bash
curl http://localhost:11434/api/tags  # Ollama
curl http://localhost:1234/v1/models   # LM Studio
```

### Reset Installation

```bash
tokman init --force
source ~/.bashrc
```

---

## See Also

- [FEATURES.md](FEATURES.md) - Complete feature reference
- [GUIDE.md](GUIDE.md) - Getting started guide
- [ADVANCED_COMPRESSION.md](ADVANCED_COMPRESSION.md) - Deep dive into compression algorithms
