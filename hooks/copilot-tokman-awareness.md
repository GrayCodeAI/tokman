# TokMan — Copilot Integration (VS Code Copilot Chat + Copilot CLI)

**Usage**: Token-optimized CLI proxy (60-90% savings on dev operations)

## What's automatic

The `.github/copilot-instructions.md` file is loaded at session start by both Copilot CLI and VS Code Copilot Chat.
It instructs Copilot to prefix commands with `tokman` automatically.

The `.github/hooks/tokman-rewrite.json` hook adds a `PreToolUse` safety net via `tokman hook` —
a cross-platform binary that intercepts raw bash tool calls and rewrites them.

## Meta commands (always use directly)

```bash
tokman gain              # Token savings dashboard for this session
tokman gain --history    # Per-command history with savings %
tokman discover          # Scan session history for missed tokman opportunities
tokman proxy <cmd>       # Run raw (no filtering) but still track it
```

## Installation verification

```bash
tokman --version   # Should print: tokman X.Y.Z
tokman gain        # Should show a dashboard
which tokman       # Verify correct binary path
```

## How the hook works

`tokman hook` reads `PreToolUse` JSON from stdin, detects the agent format, and responds appropriately:

**VS Code Copilot Chat** (supports `updatedInput` — transparent rewrite, no denial):
1. Agent runs `git status` → `tokman hook` intercepts via `PreToolUse`
2. `tokman hook` detects VS Code format (`tool_name`/`tool_input` keys)
3. Returns `hookSpecificOutput.updatedInput.command = "tokman git status"`
4. Agent runs the rewritten command silently — no denial, no retry

**GitHub Copilot CLI** (deny-with-suggestion):
1. Agent runs `git status` → `tokman hook` intercepts via `PreToolUse`
2. `tokman hook` detects Copilot CLI format (`toolName`/`toolArgs` keys)
3. Returns `permissionDecision: deny` with reason: `"Token savings: use 'tokman git status' instead"`
4. Copilot reads the reason and re-runs `tokman git status`

## Integration comparison

| Tool                  | Mechanism                               | Hook output              | File                               |
|-----------------------|-----------------------------------------|--------------------------|------------------------------------|
| Claude Code           | `PreToolUse` hook with `updatedInput`   | Transparent rewrite      | `hooks/tokman-rewrite.sh`          |
| VS Code Copilot Chat  | `PreToolUse` hook with `updatedInput`   | Transparent rewrite      | `.github/hooks/tokman-rewrite.json`|
| GitHub Copilot CLI    | `PreToolUse` deny-with-suggestion       | Denial + retry           | `.github/hooks/tokman-rewrite.json`|
| OpenCode              | Plugin `tool.execute.before`            | Transparent rewrite      | `hooks/opencode-tokman.ts`         |
| Cursor                | `PreToolUse` hook with `updatedInput`   | Transparent rewrite      | `hooks/cursor-tokman-rewrite.sh`   |
| Windsurf              | Rules file                              | Prompt-level guidance    | `.windsurfrules`                   |
| Cline                 | Rules file                              | Prompt-level guidance    | `.clinerules`                      |
