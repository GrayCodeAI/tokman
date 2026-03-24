# TokMan Migration Guide

**For users coming from RTK or other token optimization tools**

## Coming from RTK

RTK is a fast Rust-based tool using heuristics. TokMan offers deeper compression with research-backed layers.

### Key Differences

| Feature | RTK | TokMan |
|---------|-----|--------|
| Language | Rust | Go |
| Approach | Heuristics | 31 research layers |
| Config | Manual rules | Auto-detection |
| Compression | 50-70% | 60-90% |
| Tiers | 1 level | 4 adaptive tiers |

### Migration Steps

1. **Uninstall RTK hooks** (if installed)
   ```bash
   # Remove RTK from your shell config
   # Check ~/.bashrc, ~/.zshrc, etc.
   ```

2. **Run TokMan quickstart**
   ```bash
   tokman quickstart
   ```

3. **Verify the switch**
   ```bash
   tokman doctor
   tokman gain  # See your savings
   ```

### Behavior Changes

- **No manual filter rules needed** - TokMan auto-detects content types
- **Preset vs Manual** - Use `--preset=auto` instead of manual configuration
- **More compression on large output** - Tier 3 runs 20 layers automatically

### Equivalent Commands

| RTK | TokMan |
|-----|--------|
| `rtk --filter=git` | `tokman --preset=balanced git` |
| `rtk --aggressive` | `tokman --preset=full` |
| Manual config | `tokman quickstart` |

## Coming from Other Tools

### From LLMLingua

If you've used Microsoft's LLMLingua:

- TokMan's perplexity layer is based on LLMLingua research
- No Python required - TokMan is a compiled binary
- Automatic tier selection replaces manual compression levels

### From vanilla CLI (no optimization)

Just run:
```bash
tokman quickstart
```

All your existing commands will be automatically compressed.

## Configuration Migration

### RTK Config → TokMan Config

RTK manual config:
```yaml
# RTK style
filters:
  - pattern: "git diff"
    strip: ["^diff --git"]
```

TokMan equivalent:
```toml
# TokMan config (mostly automatic)
[filter]
preset = "auto"

[tracking]
enabled = true
```

Most patterns are built-in. Add custom filters in `~/.config/tokman/filters/`:

```toml
# ~/.config/tokman/filters/custom.toml
[my_tool]
match = "^my-tool "
strip_lines_matching = ["^DEBUG:"]
```

## Performance Expectations

| Content Type | RTK Savings | TokMan Savings |
|--------------|-------------|----------------|
| `git status` | 40-50% | 50-60% |
| `git diff` | 50-60% | 60-75% |
| Build logs | 60-70% | 70-85% |
| Error traces | 40-50% | 60-80% |
| Large files | 50-60% | 70-90% |

## Troubleshooting Migration

### "TokMan seems slower"

TokMan uses more layers for better compression. For speed-critical operations:
```bash
tokman --preset=fast git status  # Only 3 layers, ~3x faster
```

### "Different output format"

TokMan preserves more semantic content. If you need raw output:
```bash
tokman --no-filter git log
```

### "Missing features"

TokMan focuses on AI assistant optimization. Some RTK features may not apply:
- RTK's manual filters → TokMan's auto-detection
- RTK's shell integration → TokMan's agent hooks

## Rollback

To revert to your previous setup:
```bash
# Remove TokMan hooks
tokman undo

# Reinstall your previous tool
# ...
```

## Getting Help

- `tokman discover` - Find optimization opportunities
- `tokman doctor` - Diagnose issues
- `tokman gain` - See your savings
