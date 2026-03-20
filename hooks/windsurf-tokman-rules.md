# TokMan — Windsurf Integration

**Usage**: Token-optimized CLI proxy for shell commands.

## Rule

Always prefix shell commands with `tokman` to minimize token consumption.

Examples:

```bash
tokman git status
tokman cargo test
tokman ls src/
tokman grep "pattern" src/
tokman find "*.rs" .
tokman docker ps
tokman gh pr list
```

## Meta Commands

```bash
tokman gain              # Show token savings
tokman gain --history    # Command history with savings
tokman discover          # Find missed TokMan opportunities
tokman proxy <cmd>       # Run raw (no filtering, for debugging)
```

## Why

TokMan filters and compresses command output before it reaches the LLM context, saving 60-90% tokens on common operations. Always use `tokman <cmd>` instead of raw commands.
