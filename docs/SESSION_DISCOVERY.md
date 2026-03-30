# Session Discovery

TokMan can analyze your Claude Code session history to show adoption metrics and identify missed optimization opportunities.

## Overview

The session discovery system scans `~/.claude/projects/` for session JSONL files and:
- Extracts Bash commands executed in each session
- Calculates TokMan adoption percentage
- Identifies commands that could benefit from TokMan wrapping
- Tracks output token savings over time

## Commands

### `tokman session`

Show adoption metrics across recent sessions:

```bash
tokman session
```

Output:
```
TokMan Session Overview (last 10)
───────────────────────────────────────────────────────────────────────
Session      Date          Cmds TokMan Adoption         Output
───────────────────────────────────────────────────────────────────────
a1b2c3d4     Today          142    89 ████████▏   125.4K
e5f6g7h8     Yesterday       87    65 ██████▏      67.2K
i9j0k1l2     2d ago         203    72 ███████▏    234.1K
───────────────────────────────────────────────────────────────────────
Average adoption: 75%
Tip: Run `tokman discover` to find missed TokMan opportunities
```

### `tokman discover`

Analyze session history to find optimization opportunities:

```bash
tokman discover
```

Output:
```
TokMan Discovery Report
───────────────────────────────────────────────────────────────────────
Missed Opportunities (by frequency)

Command                              Count    Est. Savings
───────────────────────────────────────────────────────────────────────
git log --oneline -20                   47         23.5K
npm test                                32         48.0K
cargo build                             28         33.6K
pytest tests/                           21         25.2K
───────────────────────────────────────────────────────────────────────

Total missed savings: ~130K tokens
Run `tokman init` to enable more wrappers
```

## How It Works

### Session File Format

Claude Code stores sessions as JSONL files in:
```
~/.claude/projects/<project-hash>/<session-id>.jsonl
```

Each line is a JSON object with `type` field:
- `assistant` - Contains tool use blocks with Bash commands
- `user` - Contains tool results with output content

### Command Extraction

The discovery system:
1. Scans all `.jsonl` files modified in the last 30 days
2. Extracts `Bash` tool use commands from assistant messages
3. Matches tool results to capture output lengths
4. Handles command chains (`&&`, `||`, `;`, `|`)
5. Classifies each command against TokMan's registry

### Classification

Commands are classified as:
- **Supported**: Already wrapped by TokMan
- **Unsupported**: Could be wrapped (opportunity)
- **Excluded**: Should not be wrapped (interactive, editors, etc.)

## Configuration

### Exclude Commands

Add to `~/.config/tokman/config.toml`:

```toml
[hooks]
excluded_commands = [
    "vim",
    "nano",
    "less",
    "man",
    "ssh *",
    "mysql -u*"
]
```

### Time Range

Filter sessions by time:

```bash
tokman session --since 7d   # Last 7 days
tokman discover --since 30d # Last 30 days
```

## Output Formats

Both commands support multiple output formats:

```bash
# Table format (default)
tokman session --format table

# JSON format for scripting
tokman session --format json

# CSV for spreadsheets
tokman discover --format csv
```

### JSON Output Example

```bash
tokman session --format json
```

```json
{
  "sessions": [
    {
      "id": "a1b2c3d4",
      "date": "2026-03-30",
      "total_commands": 142,
      "tokman_commands": 126,
      "adoption_pct": 88.7,
      "output_tokens": 125400
    }
  ],
  "summary": {
    "total_sessions": 10,
    "average_adoption": 75.2,
    "total_commands": 892,
    "total_tokman_commands": 671
  }
}
```

## Use Cases

### 1. Track Adoption Progress

```bash
# Weekly adoption check
tokman session --since 7d
```

### 2. Identify Optimization Targets

```bash
# Find commands to wrap
tokman discover --since 30d | head -20
```

### 3. Export for Reporting

```bash
# Generate CSV report
tokman session --format csv > adoption_report.csv
```

### 4. CI Integration

```bash
# Check adoption threshold
ADOPTION=$(tokman session --format json | jq '.summary.average_adoption')
if (( $(echo "$ADOPTION < 70" | bc) )); then
  echo "Warning: Low TokMan adoption: $ADOPTION%"
fi
```

## Privacy

Session discovery only reads:
- Command strings from Bash tool invocations
- Output lengths (character counts, not content)

No actual command output content is stored or transmitted.

## Troubleshooting

### No Sessions Found

```
No Claude Code sessions found in the last 30 days.
```

Solutions:
- Ensure Claude Code has been used at least once
- Check `~/.claude/projects/` exists
- Verify session files have `.jsonl` extension

### Subagent Sessions

Sessions in `subagents/` directories are automatically excluded to avoid double-counting.

### Large Session Files

For sessions with many commands, extraction may take a few seconds. Use `--since` to limit the date range.
