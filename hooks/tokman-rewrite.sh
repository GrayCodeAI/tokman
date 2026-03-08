#!/bin/bash
# TokMan Command Rewriter for Shell Integration
# Source this file in your .bashrc or .zshrc:
#   source /path/to/tokman/hooks/tokman-rewrite.sh
#
# This hook intercepts commands and rewrites them to use TokMan wrappers
# when appropriate, reducing token usage in LLM interactions.

# Path to tokman binary (auto-detect)
_TOKMAN_BIN="${TOKMAN_BIN:-tokman}"

# Check if tokman is available
if ! command -v "$_TOKMAN_BIN" &> /dev/null; then
    # TokMan not found, skip rewriting
    return 0 2>/dev/null || exit 0
fi

# Function to rewrite commands before execution
# This is called via DEBUG trap or PROMPT_COMMAND
_tokman_rewrite() {
    local last_cmd
    
    # Get the last command from history
    # This works in bash; zsh uses different mechanism
    last_cmd=$(history 1 | sed 's/^[ ]*[0-9]*[ ]*//')
    
    # Skip if empty or tokman command itself
    if [[ -z "$last_cmd" ]] || [[ "$last_cmd" == tokman* ]]; then
        return 0
    fi
    
    # Ask tokman for rewrite
    local rewritten
    rewritten=$("$_TOKMAN_BIN" rewrite "$last_cmd" 2>/dev/null)
    
    # If rewrite differs, show a hint (optional, for debugging)
    if [[ -n "$TOKMAN_DEBUG" ]] && [[ "$rewritten" != "$last_cmd" ]]; then
        echo "[tokman] Rewrote: $last_cmd → $rewritten" >&2
    fi
}

# Function for JSON-based rewriting (for Claude Code integration)
# Reads JSON from stdin, extracts command, rewrites if needed
_tokman_rewrite_json() {
    local json_input
    local command
    local rewritten
    
    # Read JSON from stdin
    json_input=$(cat)
    
    # Extract command field using basic parsing (no jq dependency)
    # Matches: "command": "value"
    command=$(echo "$json_input" | grep -o '"command"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/"command"[[:space:]]*:[[:space:]]*"\([^"]*\)"/\1/')
    
    if [[ -z "$command" ]]; then
        # No command field, return as-is
        echo "$json_input"
        return 0
    fi
    
    # Ask tokman for rewrite
    rewritten=$("$_TOKMAN_BIN" rewrite "$command" 2>/dev/null)
    
    if [[ -n "$rewritten" ]] && [[ "$rewritten" != "$command" ]]; then
        # Replace command in JSON
        echo "$json_input" | sed "s/\"command\"[[:space:]]*:[[:space:]]*\"[^\"]*\"/\"command\": \"$rewritten\"/"
    else
        # No rewrite, return original
        echo "$json_input"
    fi
}

# Installation function - adds hook to shell config
tokman_install_hook() {
    local shell_rc
    local hook_source
    
    # Detect shell config file
    if [[ -n "$ZSH_VERSION" ]]; then
        shell_rc="$HOME/.zshrc"
    elif [[ -n "$BASH_VERSION" ]]; then
        shell_rc="$HOME/.bashrc"
    else
        echo "Unsupported shell" >&2
        return 1
    fi
    
    # Get absolute path to this script
    hook_source="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/tokman-rewrite.sh"
    
    # Check if already installed
    if grep -q "source.*tokman-rewrite.sh" "$shell_rc" 2>/dev/null; then
        echo "✓ TokMan hook already installed in $shell_rc"
        return 0
    fi
    
    # Add to shell config
    echo "" >> "$shell_rc"
    echo "# TokMan shell integration" >> "$shell_rc"
    echo "source \"$hook_source\"" >> "$shell_rc"
    
    echo "✓ TokMan hook installed to $shell_rc"
    echo "  Run 'source $shell_rc' or restart your shell to activate"
}

# Show current status
tokman_status() {
    echo "🌸 TokMan Shell Integration"
    echo "─────────────────────────────"
    echo "Binary: $_TOKMAN_BIN"
    
    if command -v "$_TOKMAN_BIN" &> /dev/null; then
        echo "Status: ✓ Installed"
        "$_TOKMAN_BIN" rewrite list 2>/dev/null
    else
        echo "Status: ✗ Not found"
        echo "  Set TOKMAN_BIN environment variable to specify path"
    fi
}

# Alias common commands to use tokman
# These are lightweight and only active when tokman is available
if command -v "$_TOKMAN_BIN" &> /dev/null; then
    alias ts='tokman status'
    alias tr='tokman rewrite'
fi

# Export functions for subshells
export -f _tokman_rewrite _tokman_rewrite_json tokman_install_hook tokman_status 2>/dev/null || true

# Print status on source (optional, comment out if undesired)
if [[ -n "$TOKMAN_VERBOSE" ]]; then
    tokman_status
fi
