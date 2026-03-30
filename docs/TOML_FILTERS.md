# TOML Filter System

TokMan supports declarative output filters defined in TOML format. This allows users to customize command output compression without writing Go code.

## Overview

TOML filters are loaded from:
- `~/.config/tokman/filters/*.toml` - User-defined filters
- Embedded built-in filters - Default filters shipped with TokMan

## Filter Structure

```toml
[filters.filter-name]
description = "Human-readable description"
match_command = "^command-prefix\\b"

# Optional filtering options
strip_ansi = true
strip_lines_matching = ["^DEBUG:", "^\\s*$"]
keep_lines_matching = ["^ERROR:", "^WARN:"]
max_lines = 50
tail_lines = 10
truncate_lines_at = 120
on_empty = "no output"

# Regex replacements
[[filters.filter-name.replace]]
pattern = "\\b\\d+\\b"
replacement = "N"

# Short-circuit output matching
[[filters.filter-name.match_output]]
pattern = "already up to date"
message = "ok: up-to-date"
```

## Filter Fields

### `match_command` (required)

Regex pattern to match against the full command string.

```toml
match_command = "^git\\s+status"  # Matches "git status"
match_command = "^cargo\\s+(build|test)"  # Matches "cargo build" or "cargo test"
```

### `strip_ansi`

Remove ANSI escape codes (colors, formatting) before processing.

```toml
strip_ansi = true
```

### `strip_lines_matching`

Drop lines matching any of these regex patterns.

```toml
strip_lines_matching = [
    "^\\s*$",           # Empty lines
    "^DEBUG:",          # Debug log lines
    "^make\\[\\d+\\]:"  # Make job prefixes
]
```

### `keep_lines_matching`

Keep only lines matching at least one pattern. If defined, all non-matching lines are dropped.

```toml
keep_lines_matching = ["^ERROR:", "^WARN:", "^FATAL:"]
```

### `max_lines`

Keep only the first N lines after other filtering.

```toml
max_lines = 50
```

### `tail_lines`

Keep only the last N lines (applied after max_lines).

```toml
tail_lines = 20  # Show last 20 lines
```

### `truncate_lines_at`

Truncate lines longer than N characters.

```toml
truncate_lines_at = 120
```

### `on_empty`

Fallback message when filtered output is empty.

```toml
on_empty = "build succeeded"
```

### `replace`

Apply regex substitutions to the output.

```toml
[[filters.gcc.replace]]
pattern = "\\b\\d+(\\.\\d+)?\\b"
replacement = "N"

[[filters.gcc.replace]]
pattern = "error:"
replacement = "ERROR:"
```

### `match_output`

Short-circuit rules: if output matches a pattern, return the message immediately.

```toml
[[filters.git-pull.match_output]]
pattern = "(?i)already up to date"
message = "ok: up-to-date"

[[filters.git-status.match_output]]
pattern = "nothing to commit"
message = "ok: clean"
```

## Inline Tests

Filters can include test cases for validation:

```toml
[filters.mycmd]
match_command = "^mycmd"
strip_lines_matching = ["^DEBUG:"]

[[tests.mycmd]]
name = "strips debug lines"
input = "DEBUG: starting\nERROR: failed"
expected = "ERROR: failed"

[[tests.mycmd]]
name = "preserves other lines"
input = "INFO: processing\nWARNING: check this"
expected = "INFO: processing\nWARNING: check this"
```

Run tests with:
```bash
tokman filter test ~/.config/tokman/filters/mycmd.toml
```

## Example Filters

### Git Status

```toml
[filters.git-status]
description = "Compact git status output"
match_command = "^git\\s+status"
strip_lines_matching = ["^\\s*$"]
max_lines = 30

[[filters.git-status.match_output]]
pattern = "nothing to commit"
message = "ok: clean working tree"
```

### Cargo Test

```toml
[filters.cargo-test]
description = "Compact cargo test output"
match_command = "^cargo\\s+test"
strip_lines_matching = ["^running \\d+ tests", "^test .*\\.\\.\\. ok$"]
strip_ansi = true
max_lines = 50
on_empty = "all tests passed"
```

### NPM Install

```toml
[filters.npm-install]
description = "Compact npm install output"
match_command = "^npm\\s+install"
strip_lines_matching = ["^npm WARN", "^\\s*$"]
max_lines = 30

[[filters.npm-install.match_output]]
pattern = "up to date"
message = "ok: dependencies current"
```

### Docker Build

```toml
[filters.docker-build]
description = "Compact docker build output"
match_command = "^docker\\s+build"
strip_lines_matching = ["^\\s*$"]
truncate_lines_at = 100
tail_lines = 20
```

## Built-in Filters

TokMan includes built-in filters for common commands:

| Filter | Command Pattern | Description |
|--------|----------------|-------------|
| `git-status` | `^git status` | Compact status |
| `git-log` | `^git log` | Limited log lines |
| `cargo-test` | `^cargo test` | Test summary |
| `npm-test` | `^npm test` | Test output |
| `pytest` | `^pytest` | Python test output |
| `go-test` | `^go test` | Go test output |

## Debugging Filters

View active filters:
```bash
tokman filter list
```

Test a filter manually:
```bash
tokman filter test --command "git status" --input "On branch main\nnothing to commit"
```

Validate filter syntax:
```bash
tokman filter validate ~/.config/tokman/filters/myfilter.toml
```

## Performance

Filters are compiled once at load time and cached for subsequent use. The regex engine uses Go's RE2 syntax for safe, linear-time matching.
