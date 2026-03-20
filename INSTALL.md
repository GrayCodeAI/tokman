# TokMan Installation Guide

## Quick Install (Linux/macOS)

### Via Go

```bash
go install github.com/GrayCodeAI/tokman/cmd/tokman@latest
```

### From Source

```bash
git clone https://github.com/GrayCodeAI/tokman.git
cd tokman
go build -o tokman ./cmd/tokman/
sudo mv tokman /usr/local/bin/
```

### Verify Installation

```bash
tokman --version
tokman gain    # Should show token savings stats
```

---

## Project Initialization

### Which mode to choose?

```
  Do you want TokMan active across ALL coding agent projects?
  │
  ├─ YES → tokman init -g              (recommended)
  │         Hook + TOKMAN.md
  │         Commands auto-rewritten transparently
  │
  └─ NO, single project → tokman init
            Local setup only
```

### Recommended: Global Hook-First Setup

```bash
tokman init -g
# → Installs hook to ~/.claude/hooks/tokman-rewrite.sh
# → Creates ~/.claude/TOKMAN.md
# → Adds @TOKMAN.md reference to ~/.claude/CLAUDE.md
# → Prompts: "Patch settings.json? [y/N]"

# Automated alternatives:
tokman init -g --auto-patch    # Patch without prompting
tokman init -g --no-patch      # Print manual instructions
```

### Editor-Specific Setup

```bash
# Cursor Agent
tokman init -g --cursor

# GitHub Copilot
tokman init -g --copilot

# Gemini CLI
tokman init -g --gemini

# Codex CLI
tokman init -g --codex

# Windsurf
tokman init -g --windsurf

# Cline
tokman init -g --cline

# OpenCode
tokman init -g --opencode
```

### Combine Multiple Editors

```bash
tokman init -g --cursor --copilot --gemini --auto-patch
```

---

## How It Works

```
  Claude Code          settings.json        tokman-rewrite.sh       TokMan binary
       │                    │                      │                    │
       │  "git status"      │                      │                    │
       │ ──────────────────►│                      │                    │
       │                    │  PreToolUse trigger   │                    │
       │                    │ ─────────────────────►│                    │
       │                    │                      │  rewrite command   │
       │                    │                      │  → tokman git      │
       │                    │◄─────────────────────│                    │
       │                    │  updated command      │                    │
       │                    │                                            │
       │  execute: tokman git status                                    │
       │ ─────────────────────────────────────────────────────────────►│
       │                                                               │  filter
       │  "3 modified, 1 untracked ✓"    (~10 tokens vs ~200 raw)    │
       │◄─────────────────────────────────────────────────────────────│
```

---

## Uninstalling

### Complete Removal

```bash
tokman init -g --uninstall

# What gets removed:
#   - Hook: ~/.claude/hooks/tokman-rewrite.sh
#   - Context: ~/.claude/TOKMAN.md
#   - Reference: @TOKMAN.md from ~/.claude/CLAUDE.md
#   - Hook entry from settings.json
#   - Gemini artifacts (if installed)
#   - Codex artifacts (if installed)

# Restart Claude Code after uninstall
```

### Binary Removal

```bash
# If installed via go install
rm $(which tokman)

# If built from source
sudo rm /usr/local/bin/tokman
```

### Restore from Backup

```bash
cp ~/.claude/settings.json.bak ~/.claude/settings.json
```

---

## Essential Commands

### Files
```bash
tokman ls .              # Compact tree view
tokman read file.rs      # Optimized reading
tokman grep "pattern" .  # Grouped search results
```

### Git
```bash
tokman git status        # Compact status
tokman git log -n 10     # Condensed logs
tokman git diff          # Optimized diff
tokman git add .         # → "ok ✓"
tokman git commit -m "msg"  # → "ok ✓ abc1234"
tokman git push          # → "ok ✓ main"
```

### Tests
```bash
tokman test cargo test   # Failures only (-90%)
tokman vitest run        # Filtered Vitest output (-99.6%)
```

### Statistics
```bash
tokman gain              # Token savings
tokman gain --graph      # With ASCII graph
tokman gain --history    # With command history
tokman discover          # Find missed savings
```

### Hook Processors
```bash
tokman hook gemini       # Gemini CLI hook processor
tokman hook copilot      # Copilot hook processor
```

---

## Troubleshooting

### TokMan command not found after installation
```bash
# Check Go bin path
echo $GOPATH/bin
# or
echo $GOBIN

# Add to PATH if needed (~/.bashrc or ~/.zshrc)
export PATH="$PATH:$(go env GOPATH)/bin"

# Reload shell
source ~/.bashrc
```

### Hook not working
```bash
# Verify hook is installed
ls -la ~/.claude/hooks/tokman-rewrite.sh

# Verify hook is executable
chmod +x ~/.claude/hooks/tokman-rewrite.sh

# Check settings.json has hook entry
cat ~/.claude/settings.json | jq '.hooks'

# Test rewrite directly
tokman rewrite "git status"
```

### Debug hook execution
```bash
# Enable verbose mode
tokman rewrite -v "git status"

# Check hook script directly
echo '{"tool_name":"Bash","tool_input":{"command":"git status"}}' | bash ~/.claude/hooks/tokman-rewrite.sh
```

---

## Validated Token Savings

| Operation | Standard | TokMan | Reduction |
|-----------|----------|--------|-----------|
| `vitest run` | 102,199 chars | 377 chars | **-99.6%** |
| `git status` | 529 chars | 217 chars | **-59%** |
| `pnpm list` | ~8,000 tokens | ~2,400 | **-70%** |
| `pnpm outdated` | ~12,000 tokens | ~1,200-2,400 | **-80-90%** |

### Typical Claude Code Session (30 min)
- **Without TokMan**: ~150,000 tokens
- **With TokMan**: ~45,000 tokens
- **Savings**: **70% reduction**

---

## Support

- **GitHub**: https://github.com/GrayCodeAI/tokman
- **Issues**: https://github.com/GrayCodeAI/tokman/issues
- **Docs**: See `docs/` directory
