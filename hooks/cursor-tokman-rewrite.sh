#!/usr/bin/env bash
# tokman-hook-version: 1
# TokMan Cursor Agent hook — rewrites shell commands to use tokman for token savings.
# Works with both Cursor editor and cursor-cli (they share ~/.cursor/hooks.json).
# Cursor preToolUse hook format: receives JSON on stdin, returns JSON on stdout.
# Requires: tokman, jq

if ! command -v jq &>/dev/null; then
  echo "[tokman] WARNING: jq is not installed. Hook cannot rewrite commands." >&2
  exit 0
fi

if ! command -v tokman &>/dev/null; then
  echo "[tokman] WARNING: tokman is not installed or not in PATH." >&2
  exit 0
fi

INPUT=$(cat)
CMD=$(echo "$INPUT" | jq -r '.tool_input.command // empty')

if [ -z "$CMD" ]; then
  echo '{}'
  exit 0
fi

# Delegate rewrite logic to tokman rewrite.
REWRITTEN=$(tokman rewrite "$CMD" 2>/dev/null) || { echo '{}'; exit 0; }

if [ "$CMD" = "$REWRITTEN" ]; then
  echo '{}'
  exit 0
fi

jq -n --arg cmd "$REWRITTEN" '{
  "permission": "allow",
  "updated_input": { "command": $cmd }
}'
