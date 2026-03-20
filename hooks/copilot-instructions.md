# TokMan — Token-Optimized CLI Proxy

This project uses [TokMan](https://github.com/GrayCodeAI/tokman) to reduce token consumption by 60-90% on shell command outputs.

## How to use

Prefix shell commands with `tokman`:

```bash
tokman git status        # Compact status (75% less tokens)
tokman git diff          # Compact diff (80% less tokens)
tokman git log -5        # Compact log (90% less tokens)
tokman ls                # Filtered directory listing
tokman cargo test        # Show failures only
tokman npm test          # Show failures only
tokman grep "pattern" .  # Grouped search results
```

## What TokMan does

TokMan intercepts command output and applies smart filtering before it reaches the LLM context:

- **git operations**: Compact status, stat-only diffs, one-line commits
- **test runners**: Show failures only (90% reduction)
- **build tools**: Show errors/warnings only
- **file listing**: Hide noise dirs (node_modules, .git, target, etc.)

## Meta commands

```bash
tokman gain              # Token savings dashboard
tokman gain --history    # Per-command history
tokman discover          # Find missed opportunities
tokman proxy <cmd>       # Run raw (no filtering)
```
