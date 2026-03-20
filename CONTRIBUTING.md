# Contributing to TokMan

**Welcome!** We appreciate your interest in contributing to TokMan.

## Quick Links

- [Report an Issue](../../issues/new)
- [Open Pull Requests](../../pulls)

---

## What is TokMan?

**TokMan (Token Manager)** is a CLI proxy that cuts noise from command outputs. It filters and compresses CLI output before it reaches your LLM context, saving 60-99% of tokens on common operations. TokMan is written in Go and features a research-based 14-layer compression pipeline.

---

## Ways to Contribute

| Type | Examples |
|------|----------|
| **Report** | File a clear issue with steps to reproduce, expected vs actual behavior |
| **Fix** | Bug fixes, broken filter repairs |
| **Build** | New filters, new command support, performance improvements |
| **Review** | Review open PRs, test changes locally, leave constructive feedback |
| **Document** | Improve docs, add usage examples, clarify existing docs |

---

## Branch Naming Convention

Every branch **must** follow one of these prefixes:

| Prefix | Semver Impact | When to Use |
|--------|---------------|-------------|
| `fix/...` | Patch | Bug fixes, corrections, minor adjustments |
| `feat/...` | Minor | New features, new filters, new command support |
| `chore/...` | Major | Breaking changes, API changes, removed functionality |

Examples:
```
fix/git-log-drops-merge-commits
feat/kubectl-add-pod-list-filter
chore/remove-deprecated-flags
```

---

## Pull Request Process

### Scope Rules

**Each PR must focus on a single feature, fix, or change.** Out-of-scope changes must go in a separate PR.

**For large features**, prefer multi-part PRs:
- Part 1: Add data model and tests
- Part 2: Add CLI command and integration
- Part 3: Update documentation and CHANGELOG

### 1. Create Your Branch

```bash
git checkout main
git pull origin main
git checkout -b "feat/your-clear-description"
```

### 2. Make Your Changes

**Respect the existing folder structure.** Place new files where similar files already live.

**Keep functions short and focused.** Each function should do one thing.

**No obvious comments.** Comments should explain *why*, never *what*.

### 3. Add Tests

Every change **must** include tests.

### 4. Add Documentation

Every change **must** include documentation updates.

### 5. Merge

Once your work is ready, open a Pull Request targeting the `main` branch.

---

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/filter/
go test ./internal/commands/

# Run with verbose output
go test -v ./...

# Run specific test
go test -run TestRewriteCommand ./internal/discover/
```

### Test Types

| Type | Location | Run With |
|------|----------|----------|
| **Unit tests** | `*_test.go` files alongside source | `go test ./...` |
| **Hook tests** | `hooks/test-tokman-rewrite.sh` | `bash hooks/test-tokman-rewrite.sh` |
| **Copilot tests** | `hooks/test-copilot-tokman-rewrite.sh` | `bash hooks/test-copilot-tokman-rewrite.sh` |

### Pre-Commit Gate (mandatory)

All must pass before any PR:

```bash
go build ./... && go test ./...
```

If you have golangci-lint installed:
```bash
golangci-lint run ./...
```

### PR Testing Checklist

- [ ] Unit tests added/updated for changed code
- [ ] Token savings >=60% verified
- [ ] Edge cases covered
- [ ] `go build ./... && go test ./...` passes
- [ ] Manual test: run `tokman <cmd>` and inspect output

---

## Project Structure

```
tokman/
├── cmd/tokman/          # Entry point
├── internal/
│   ├── commands/        # CLI command implementations
│   ├── core/            # Runner, estimator, interfaces
│   ├── filter/          # 14-layer compression pipeline
│   ├── toml/            # TOML filter system
│   │   └── builtin/     # 93+ built-in filter definitions
│   ├── tracking/        # SQLite token tracking
│   ├── discover/        # Command classification & rewriting
│   ├── config/          # Configuration management
│   ├── dashboard/       # Web dashboard
│   ├── server/          # MCP server
│   ├── economics/       # Cost analysis
│   ├── integrity/       # Hook verification
│   └── telemetry/       # Usage telemetry
├── hooks/               # Shell hook scripts
├── openclaw/            # OpenClaw plugin
├── docs/                # Documentation
├── completions/         # Shell completions
├── docker/              # Docker files
├── Formula/             # Homebrew formula
└── aur/                 # AUR package
```

---

## Adding a New Command

1. Create `internal/commands/yourcommand.go`
2. Register it in `init()` with `rootCmd.AddCommand(yourCmd)`
3. Add test file `internal/commands/yourcommand_test.go`
4. If it needs a TOML filter, add `internal/toml/builtin/yourtool.toml`
5. Update README.md with usage examples
6. Update CHANGELOG.md

---

## Adding a TOML Filter

1. Create `internal/toml/builtin/yourtool.toml`:
```toml
schema_version = 1
match_command = "yourtool"

[[filters]]
name = "yourtool-filter"
description = "Compact output for yourtool"

  [filters.match_output]
  pattern = "some verbose pattern"
  message = "compact replacement"

  [filters.strip_lines_matching]
  patterns = ["^noise.*"]

  [filters.max_lines]
  value = 50
```

2. Add inline tests (optional):
```toml
[[tests.yourtool-filter]]
input = "verbose output"
expected = "compact output"
```

---

## Documentation

Every change **must** include documentation updates:

| What you changed | Update |
|------------------|--------|
| New command or filter | README.md + CHANGELOG.md |
| Architecture or internal design | ARCHITECTURE.md |
| Installation or setup | README.md |
| Bug fix or breaking change | CHANGELOG.md |

---

## Questions?

- **Bug reports & features**: [Issues](../../issues)
- **Pull Requests**: [Pull Requests](../../pulls)

---

**Thank you for contributing to TokMan!**
