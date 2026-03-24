# TokMan Quick Start Guide

**Get started in under 30 seconds**

## One-Command Setup

```bash
tokman quickstart
```

This automatically:
- Detects your AI agents (Claude Code, Cursor, Windsurf, etc.)
- Installs compression hooks
- Creates default configuration
- Verifies everything works

## What Happens Automatically

TokMan uses a **4-tier adaptive system** that selects the right compression level based on content:

| Tier | Layers | When | Speed |
|------|--------|------|-------|
| 0 (Trivial) | 0 | Empty content | Instant |
| 1 (Simple) | 3 | <50 tokens, simple commands | <0.5ms |
| 2 (Medium) | 8 | Git diffs, tests, builds | <2ms |
| 3 (Full) | 20 | Large output, code, logs | <15ms |

**You don't need to configure anything** - TokMan auto-detects content size and type.

## Presets (Optional)

If you want explicit control:

```bash
# Fast: 3 layers, maximum speed
tokman --preset=fast git status

# Balanced: 8 layers, good compression (default)
tokman --preset=balanced git diff

# Full: 20 layers, maximum compression
tokman --preset=full cat large-file.log

# Auto: Let TokMan decide (recommended)
tokman --preset=auto make build
```

## Common Commands

```bash
# Check everything is working
tokman doctor

# See your token savings
tokman gain

# Find missed optimization opportunities
tokman discover

# View current status
tokman status
```

## How It Works

When you run commands through TokMan:

```
git diff → tokman → compressed output (60-90% fewer tokens)
```

The 20 compression layers are based on research from 120+ papers:
- **Entropy filtering** - Remove low-information tokens
- **Perplexity pruning** - Iterative token removal (Microsoft LLMLingua)
- **AST preservation** - Keep code structure intact
- **H2O filter** - Heavy-hitter detection (NeurIPS 2023)
- **And 16 more...**

## Manual Setup (if quickstart doesn't detect your agent)

```bash
# Claude Code
tokman init --claude

# Cursor
tokman init --cursor

# Windsurf
tokman init --windsurf

# All detected agents
tokman init --all
```

## Verification

```bash
tokman doctor
```

Expected output:
```
tokman doctor — diagnosing setup
================================
  ✓ Binary: /usr/local/bin/tokman
  ✓ Config Dir: /home/user/.config/tokman
  ✓ Database: /home/user/.local/share/tokman/tracking.db
  ✓ Shell Hook: /home/user/.claude/hooks/tokman.sh
  ✓ PATH: /usr/local/bin/tokman
  ✓ Platform: linux/amd64 Go go1.26.0
  ✓ Tokenizer: tiktoken-go (embedded)
  ✓ TOML Filters: 15 built-in filters
  ✓ Disk Space: database is 0.1MB
  ✓ Go: available (for development)
  ✓ Tier System: 4 tiers (0-3) with auto-detection

All checks passed!
```

## Need Help?

- `tokman --help` - Command reference
- `tokman <command> --help` - Command-specific help
- [Documentation](./docs/) - Full docs
- [GitHub Issues](https://github.com/GrayCodeAI/tokman/issues) - Bug reports
