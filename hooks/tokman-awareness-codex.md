<!-- tokman-codex-instructions v1 -->
# TokMan (Token Manager) - Token-Optimized Commands

## Golden Rule

**Always prefix commands with `tokman`** for 60-90% token savings.

## Key Commands

```bash
tokman git status / log / diff / show   # Git (59-80% savings)
tokman cargo build / test / clippy      # Rust (80-90%)
tokman tsc / lint / prettier            # JS/TS (70-87%)
tokman vitest / playwright              # Tests (90-99%)
tokman docker / kubectl                 # Infra (85%)
tokman ls / grep / find / tree          # Files (60-75%)
tokman gain                             # View savings stats
tokman discover                         # Find missed savings
tokman proxy <cmd>                      # Run without filtering
```

## Categories & Savings

| Category | Commands | Typical Savings |
|----------|----------|-----------------|
| Tests | vitest, playwright, cargo test | 90-99% |
| Build | next, tsc, lint, prettier | 70-87% |
| Git | status, log, diff, add, commit | 59-80% |
| GitHub | gh pr, gh run, gh issue | 26-87% |
| Infra | docker, kubectl | 85% |
| Files | ls, grep, find, tree | 60-75% |

<!-- /tokman-codex-instructions -->
