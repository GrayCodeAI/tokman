# Tokman 200 Tasks — Perfect, Refactored, Optimized

> Goal: Make tokman the **best-in-class** token reduction CLI tool — clean architecture, maximum compression, top performance, zero tech debt.

---

## A. ARCHITECTURE & CODE ORGANIZATION (Tasks 1–30)

### Dependency Injection & Interfaces
- [ ] **T1** — Introduce a `CommandRunner` interface to decouple shell execution from command handlers (currently `exec.Command` called directly in 75+ files)
- [ ] **T2** — Create a `TokenEstimator` interface with `HeuristicEstimator` (current) and `TiktokenEstimator` implementations; inject everywhere
- [ ] **T3** — Extract a `FilterFactory` that constructs filters from config rather than hardcoding in `pipeline.go:100-180`
- [ ] **T4** — Introduce `Logger` interface (replace direct `log.Printf` / `fmt.Fprintf(os.Stderr, ...)` scattered across codebase)
- [ ] **T5** — Create `OutputWriter` interface for stdout/stderr separation — currently output and diagnostics mixed in `commands/fallback.go:180-200`
- [ ] **T6** — Add `Context` propagation to all filter `Apply()` methods for cancellation/timeout support
- [ ] **T7** — Extract `PipelineConfig` struct from hardcoded values in `pipeline.go:45-90` into a config-driven builder
- [ ] **T8** — Create `Tracker` interface (abstract SQLite) to allow in-memory/remote tracking backends
- [ ] **T9** — Define `Compressor` interface that wraps the entire pipeline; enables swapping implementations for benchmarks
- [ ] **T10** — Introduce `ConfigProvider` interface to unify TOML/YAML/env config loading

### Package Restructuring
- [ ] **T11** — Move `internal/filter/` 54 files into sub-packages: `internal/filter/layers/`, `internal/filter/core/`, `internal/filter/streaming/`
- [ ] **T12** — Split `internal/commands/git.go` (1404 lines) into `internal/commands/git/status.go`, `git/diff.go`, `git/log.go`, `git/stash.go`, `git/branch.go`
- [ ] **T13** — Extract `internal/commands/common.go` for shared command execution logic (currently duplicated in 10+ command files)
- [ ] **T14** — Create `internal/pipeline/` package separate from `internal/filter/` — pipeline orchestration ≠ individual filters
- [ ] **T15** — Move `internal/toml/builtin/` 69 TOML files to `filters/builtin/` at project root for easier community contributions
- [ ] **T16** — Create `internal/cache/` package — extract caching logic from `manager.go` into dedicated module
- [ ] **T17** — Move shell hook scripts from `hooks/` to `internal/hooks/scripts/` and embed them (like HTML templates in dashboard)
- [ ] **T18** — Rename `internal/discover/` to `internal/registry/` — clearer intent
- [ ] **T19** — Create `internal/formatter/` for output formatting (JSON, YAML, table, colored) — currently inline in commands
- [ ] **T20** — Move SDK shared types to `sdk/types/` to avoid duplication across Go/Python/TS SDKs

### Code Quality
- [ ] **T21** — Remove all `TODO`/`FIXME`/`HACK` comments — grep shows 12+ across codebase
- [ ] **T22** — Eliminate duplicate `EstimateTokens()` function — exists in `filter.go:209`, `tracker.go:160`, `tokenizer.go:185` — use single implementation
- [ ] **T23** — Standardize error types: create `tokman/errors.go` with typed errors (`ErrFilterFailed`, `ErrConfigInvalid`, `ErrCommandNotFound`)
- [ ] **T24** — Add `//go:generate` directives for embedding TOML filters and HTML templates
- [ ] **T25** — Enforce consistent naming: `Filter` vs `filter`, `Pipeline` vs `pipeline` — standardize to exported/unexported convention
- [ ] **T26** — Remove dead code — search for unused exported functions and unreachable branches
- [ ] **T27** — Add `golangci-lint` config (`.golangci.yml`) with strict linters: `gocritic`, `gosec`, `prealloc`, `unconvert`, `unparam`
- [ ] **T28** — Replace all `fmt.Errorf` with `errors.New` where no formatting needed — performance micro-optimization
- [ ] **T29** — Pre-allocate slices where size is known (e.g., `make([]string, 0, len(lines))` patterns)
- [ ] **T30** — Add `make lint` and `make typecheck` targets to Makefile with CI integration

---

## B. TOKEN REDUCTION ALGORITHM IMPROVEMENTS (Tasks 31–80)

### Entropy Filter (Layer 1)
- [ ] **T31** — Expand common-word frequency table from ~150 to 1000+ entries; include code tokens (`func`, `return`, `import`, `class`, `def`, `const`)
- [ ] **T32** — Add language-specific frequency tables (Go, Python, JS, Rust) loaded dynamically based on file extension detection
- [ ] **T33** — Implement Shannon entropy with variable-length n-grams (currently word-level only)
- [ ] **T34** — Add entropy threshold as a tunable parameter in config (currently hardcoded)
- [ ] **T35** — Cache entropy scores for repeated lines within same output

### Perplexity Filter (Layer 2)
- [ ] **T36** — Replace simple perplexity with trigram perplexity for better local context awareness
- [ ] **T37** — Add perplexity-based line ranking (keep top-N% most surprising lines) instead of binary keep/drop
- [ ] **T38** — Implement perplexity smoothing for short outputs (<10 lines) where stats are unreliable
- [ ] **T39** — Add configurable perplexity percentile threshold

### Goal-Driven Filter (Layer 3)
- [ ] **T40** — Improve CRF-style scoring by incorporating command context (e.g., `git diff` → prioritize changed lines)
- [ ] **T41** — Add keyword extraction from user query to guide goal scoring
- [ ] **T42** — Implement relevance feedback loop — track which lines users actually read

### AST Preservation (Layer 4)
- [ ] **T43** — Add Go AST parser support (use `go/ast`) for go-specific output compression
- [ ] **T44** — Add Python AST support for pytest/python output
- [ ] **T45** — Implement JSON AST awareness — detect and compress JSON output intelligently (preserve structure, remove verbose fields)
- [ ] **T46** — Add YAML AST awareness for docker-compose/kubectl output
- [ ] **T47** — Preserve function signatures during compression (currently may truncate mid-signature)

### Contrastive Filter (Layer 5)
- [ ] **T48** — Implement BM25 scoring for question-relevance ranking instead of simple TF-IDF
- [ ] **T49** — Add cross-encoder scoring simulation using token overlap heuristics
- [ ] **T50** — Support multi-query contrastive filtering (user provides multiple questions)

### N-gram Filter (Layer 6)
- [ ] **T51** — Add configurable n-gram sizes per content type (2-gram for code, 3-gram for prose)
- [ ] **T52** — Implement n-gram frequency caching across sessions
- [ ] **T53** — Add abbreviation dictionary for common programming patterns (`npm install` → `ni`, `git commit` → `gc`)

### Evaluator Heads (Layer 7)
- [ ] **T54** — Calibrate attention head weights using actual transformer attention patterns on code data
- [ ] **T55** — Add 8 head configurations instead of current 4 for better coverage
- [ ] **T56** — Implement head importance scoring based on output type detection

### Gist Compression (Layer 8)
- [ ] **T57** — Improve gist token generation — use extractive summarization (pick most important sentences) instead of truncation
- [ ] **T58** — Add gist quality metric — measure information preservation after gist compression
- [ ] **T59** — Implement progressive gist levels (minimal/standard/detailed)

### Hierarchical Summary (Layer 9)
- [ ] **T60** — Add block-level grouping (group related lines before summarizing) instead of line-level
- [ ] **T61** — Implement summary compression ratio as configurable parameter
- [ ] **T62** — Preserve error messages and stack traces as atomic units (never split them)

### Budget Enforcement (Layer 10)
- [ ] **T63** — Add per-command budget overrides in config (e.g., `git log` gets 2K tokens, `docker ps` gets 500)
- [ ] **T64** — Implement adaptive budget — learn optimal budget per command type from usage history
- [ ] **T65** — Add budget soft-limits with graceful degradation instead of hard truncation
- [ ] **T66** — Track budget utilization efficiency (tokens kept vs tokens useful)

### Compaction (Layer 11 — MemGPT-style)
- [ ] **T67** — Implement event-based compaction — detect when output shifts topics and compact previous section
- [ ] **T68** — Add section headers during compaction for navigability
- [ ] **T69** — Preserve first/last lines of each section (attention sink pattern)
- [ ] **T70** — Add configurable compaction aggressiveness levels

### Attribution Filter (Layer 12 — ProCut-style)
- [ ] **T71** — Implement gradient-free attribution using output position analysis
- [ ] **T72** — Add cross-layer attribution — track which lines survive multiple layers and protect them
- [ ] **T73** — Implement attribution visualization in dashboard

### H2O Filter (Layer 13 — Heavy-Hitter Oracle)
- [ ] **T74** — Optimize H2O for outputs >100K tokens — implement true streaming H2O
- [ ] **T75** — Add H2O cache warming — pre-compute heavy hitters from common command outputs
- [ ] **T76** — Implement dynamic H2O budget allocation based on output entropy

### Attention Sink (Layer 14 — StreamingLLM)
- [ ] **T77** — Tune sink token count based on output length (currently fixed)
- [ ] **T78** — Add semantic sink tokens (always keep error messages, exit codes, key metrics)
- [ ] **T79** — Implement adaptive window size based on content density
- [ ] **T80** — Add attention sink position learning — track which positions users actually read

---

## C. PIPELINE & PERFORMANCE OPTIMIZATION (Tasks 81–110)

### Pipeline Execution
- [ ] **T81** — Implement early-exit: skip remaining layers if budget already met after Layer N
- [ ] **T82** — Add layer-level timing metrics to identify bottlenecks (currently no per-layer profiling)
- [ ] **T83** — Parallelize independent filters within same layer group (Layers 1-3 are independent)
- [ ] **T84** — Implement pipeline warmup — pre-initialize filter state on first run
- [ ] **T85** — Add `--layers` flag to run subset of layers for debugging/benchmarking
- [ ] **T86** — Implement conditional layer execution — skip layers that don't apply to current content type
- [ ] **T87** — Add pipeline result caching by (command + content_hash) — avoid re-processing identical output
- [ ] **T88** — Reduce memory allocations in hot path — profile and optimize `strings.Split`, `strings.Join` patterns
- [ ] **T89** — Implement zero-copy string operations where possible (use `[]byte` instead of `string` in pipeline)
- [ ] **T90** — Add pipeline mode presets: `fast` (layers 1-3-10 only), `balanced` (1-6-10-14), `full` (all 14)

### Streaming
- [ ] **T91** — Make all 14 layers streaming-compatible (currently only `stream.go` provides streaming)
- [ ] **T92** — Implement adaptive chunk sizing — larger chunks for low-entropy content, smaller for high-entropy
- [ ] **T93** — Add backpressure handling in streaming mode
- [ ] **T94** — Implement incremental token counting in streaming mode (don't wait for full chunk)
- [ ] **T95** — Add streaming progress indicator with real-time savings display

### Large Context Handling
- [ ] **T96** — Optimize `PipelineManager` for >1M token inputs — implement true file-level chunking
- [ ] **T97** — Add memory-mapped file processing for very large outputs (>100MB)
- [ ] **T98** — Implement out-of-core processing — don't load entire output into memory
- [ ] **T99** — Add parallel chunk processing in `PipelineManager` (currently sequential)
- [ ] **T100** — Implement chunk boundary awareness — don't split mid-line or mid-JSON-object

### Caching
- [ ] **T101** — Implement LRU cache for filtered outputs (size configurable, default 100 entries)
- [ ] **T102** — Add command fingerprinting for cache keys (command + args + cwd hash)
- [ ] **T103** — Implement cache invalidation on file system changes (watch for git state changes)
- [ ] **T104** — Add persistent disk cache for expensive compressions
- [ ] **T105** — Implement cache warming on tokman startup for frequently used commands

### Build & Binary
- [ ] **T106** — Optimize binary size — strip debug symbols, use `-ldflags="-s -w"`, consider UPX compression
- [ ] **T107** — Implement lazy loading of filter layers (don't init unused layers)
- [ ] **T108** — Reduce dependency tree — evaluate if all 5 direct deps are necessary
- [ ] **T109** — Add `tinygo` build target for WASM (smaller binary than standard Go WASM)
- [ ] **T110** — Implement compile-time filter registration (avoid runtime reflection)

---

## D. TESTING & QUALITY ASSURANCE (Tasks 111–140)

### Unit Tests
- [ ] **T111** — Achieve 90%+ test coverage on `internal/filter/` (currently ~60%)
- [ ] **T112** — Add table-driven tests for all 14 filter layers with edge cases (empty input, single line, unicode, binary)
- [ ] **T113** — Test all TOML filter builtin files parse correctly (69 files)
- [ ] **T114** — Add property-based tests for pipeline: "output is never longer than input"
- [ ] **T115** — Test pipeline with adversarial inputs (extremely long lines, no newlines, null bytes)
- [ ] **T116** — Add race condition tests for concurrent pipeline execution
- [ ] **T117** — Test error paths — what happens when each layer fails individually
- [ ] **T118** — Add regression tests for token estimation accuracy
- [ ] **T119** — Test SDK clients against mock server (Go, Python, TypeScript)
- [ ] **T120** — Add config parsing edge case tests (missing fields, invalid types, nested structures)

### Integration Tests
- [ ] **T121** — Add end-to-end test: real command → pipeline → verify output correctness
- [ ] **T122** — Test shell hook integration with bash/zsh/fish
- [ ] **T123** — Add Docker integration tests — run tokman in container, verify behavior
- [ ] **T124** — Test dashboard HTTP endpoints with real-ish data
- [ ] **T125** — Add cross-platform tests (linux, macos, windows) in CI

### Benchmarks
- [ ] **T126** — Add benchmarks for each individual filter layer
- [ ] **T127** — Add pipeline throughput benchmark (tokens/second processed)
- [ ] **T128** — Benchmark memory usage for large inputs (1K, 10K, 100K, 1M tokens)
- [ ] **T129** — Add comparative benchmarks vs competitors (ccusage, llm-compress, context-compression)
- [ ] **T130** — Create benchmark dashboard — auto-generate performance report on each PR

### Fuzzing & Security
- [ ] **T131** — Add Go fuzz tests for TOML parser (fuzz on malformed TOML)
- [ ] **T132** — Fuzz test all filter layers with random input
- [ ] **T133** — Add fuzz testing for command rewriting engine
- [ ] **T134** — Test for path traversal in TOML filter loading (project-local filters)
- [ ] **T135** — Audit shell command injection vectors in `exec.Command` calls

### Test Infrastructure
- [ ] **T136** — Add test fixtures directory with sample outputs for each command type
- [ ] **T137** — Implement golden file testing for filter outputs
- [ ] **T138** — Add test coverage reporting to CI (Codecov/Coveralls)
- [ ] **T139** — Create test helper for asserting token savings (expected vs actual)
- [ ] **T140** — Add mutation testing (go-mutesting) to verify test quality

---

## E. CLI & USER EXPERIENCE (Tasks 141–160)

### Commands
- [ ] **T141** — Add `tokman doctor` command — diagnose setup issues (shell hook, config, tokenizer)
- [ ] **T142** — Add `tokman benchmark <command>` — run command through pipeline and show savings comparison
- [ ] **T143** — Add `tokman diff` — show before/after compression side-by-side
- [ ] **T144** — Add `tokman explain` — explain which layers removed what and why
- [ ] **T145** — Add `tokman undo` — revert last compression, show original output
- [ ] **T146** — Add `tokman config validate` — check config files for errors
- [ ] **T147** — Add `tokman plugins list/install/remove` — manage community filter plugins
- [ ] **T148** — Add `tokman profile` — show per-layer performance breakdown

### Output & Formatting
- [ ] **T149** — Add `--json` flag to all commands for machine-readable output
- [ ] **T150** — Add `--quiet` mode — suppress all non-essential output
- [ ] **T151** — Implement colored diff output showing what was removed
- [ ] **T152** — Add `--verbose` flag with layer-by-layer compression stats
- [ ] **T153** — Support output to file (`-o/--output` flag)

### Shell Integration
- [ ] **T154** — Add fish shell support to `init` command (currently only bash/zsh)
- [ ] **T155** — Implement PowerShell support for Windows
- [ ] **T156** — Add shell completion for filter names and config options
- [ ] **T157** — Implement `tokman alias` — create shorthand for common command+filter combos
- [ ] **T158** — Add tmux integration — show savings in tmux status bar

### Error Handling
- [ ] **T159** — Add actionable error messages with fix suggestions (e.g., "Config not found. Run `tokman init` to create one")
- [ ] **T160** — Implement graceful degradation — if one layer fails, continue with remaining layers instead of crashing

---

## F. CONFIGURATION & EXTENSIBILITY (Tasks 161–180)

### Config System
- [ ] **T161** — Support hierarchical config: system → user → project → command-specific overrides
- [ ] **T162** — Add config schema validation with detailed error messages
- [ ] **T163** — Support environment variable overrides for all config options (`TOKMAN_BUDGET`, `TOKMAN_LAYERS`)
- [ ] **T164** — Add `tokman config migrate` — auto-upgrade config between versions
- [ ] **T165** — Implement config diff — show what differs from defaults

### Filter Extensibility
- [ ] **T166** — Add Lua/wasm plugin support for custom filters
- [ ] **T167** — Implement filter priority system — allow users to reorder layers
- [ ] **T168** — Add conditional filter rules (e.g., "if output contains 'error', skip compression")
- [ ] **T169** — Support regex capture groups in TOML filter replace rules
- [ ] **T170** — Add filter chaining — output of one filter feeds into next within TOML rules

### TOML Filters
- [ ] **T171** — Add TOML filters for: `rg` (ripgrep), `fd`, `jq`, `curl`, `ssh`, `terraform`, `ansible`, `helm`
- [ ] **T172** — Create TOML filter for `git log --oneline` with smart truncation
- [ ] **T173** — Add TOML filter for `npm ls` tree deduplication
- [ ] **T174** — Implement TOML filter for compiler error output (gcc, rustc, tsc) — keep error lines, strip context
- [ ] **T175** — Add TOML filter for `docker logs` — deduplicate repeated log lines

### Community & Ecosystem
- [ ] **T176** — Create `tokman-filters` community repository with contribution guidelines
- [ ] **T177** — Add `tokman search <tool>` — search available filters for a command
- [ ] **T178** — Implement filter versioning — support multiple versions of same filter
- [ ] **T179** — Add `tokman filter test <file>` — test a TOML filter against sample input
- [ ] **T180** — Create filter marketplace/registry with ratings and usage stats

---

## G. TRACKING, ANALYTICS & DASHBOARD (Tasks 181–195)

### Tracking
- [ ] **T181** — Add composite SQLite indexes on `(project_path, timestamp)` and `(command, timestamp)` for query performance
- [ ] **T182** — Replace GLOB queries with LIKE or proper path-prefix indexing
- [ ] **T183** — Implement data retention policy — auto-archive records older than N days
- [ ] **T184** — Add per-layer savings tracking (which layer saves the most tokens)
- [ ] **T185** — Track compression quality — measure if compressed output still answers user's question

### Analytics
- [ ] **T186** — Add cost-per-token tracking using configurable model pricing
- [ ] **T187** — Implement ROI dashboard — tokens saved vs time spent configuring
- [ ] **T188** — Add command frequency analysis — show which commands are compressed most
- [ ] **T189** — Create weekly/monthly savings reports
- [ ] **T190** — Add export to CSV/JSON for external analysis

### Dashboard
- [ ] **T191** — Add real-time WebSocket updates to dashboard (currently requires refresh)
- [ ] **T192** — Implement dark mode for web dashboard
- [ ] **T193** — Add filter performance charts — which layers are slowest/most effective
- [ ] **T194** — Create comparison view — tokman vs no-tokman token usage over time
- [ ] **T195** — Add mobile-responsive layout for dashboard

---

## H. CI/CD, DOCS & RELEASE (Tasks 196–200)

- [ ] **T196** — Add SBOM generation to release pipeline (Software Bill of Materials)
- [ ] **T197** — Implement automated changelog generation from conventional commits
- [ ] **T198** — Add performance regression detection in CI — fail if benchmarks degrade >10%
- [ ] **T199** — Create architecture decision records (ADRs) for all major design choices
- [ ] **T200** — Add `tokman changelog` command — show changelog for current version with upgrade instructions

---

## PRIORITY MATRIX

| Priority | Category | Tasks | Impact |
|----------|----------|-------|--------|
| **P0 Critical** | Architecture (A1-A10) | T1-T10 | Foundation for all improvements |
| **P0 Critical** | Pipeline Perf (C81-C90) | T81-T90 | Direct user-facing performance |
| **P1 High** | Token Reduction (B31-B50) | T31-T50 | Core product value |
| **P1 High** | Testing (D111-D120) | T111-T120 | Quality assurance |
| **P1 High** | CLI UX (E141-E148) | T141-T148 | User adoption |
| **P2 Medium** | Config (F161-F170) | T161-T170 | Extensibility |
| **P2 Medium** | Tracking (G181-G190) | T181-T190 | Analytics value |
| **P3 Low** | Dashboard (G191-G195) | T191-T195 | Nice-to-have |
| **P3 Low** | Community (F176-F180) | T176-T180 | Ecosystem growth |

---

## COMPETITIVE ADVANTAGES TO BUILD

| Feature | tokman (current) | tokman (after tasks) | ccusage | llm-compress |
|---------|------------------|----------------------|---------|--------------|
| Compression Layers | 14 | 14 (optimized) | 3 | 5 |
| Streaming | Partial | Full | No | No |
| Plugin System | TOML only | TOML + Lua/WASM | None | None |
| Token Estimation | Heuristic | Hybrid (tiktoken) | Exact | Heuristic |
| Dashboard | Basic | Real-time + Mobile | None | Basic |
| Cost Tracking | Yes | Advanced ROI | Basic | No |
| Multi-language SDK | 3 | 3 (improved) | 0 | 1 |
| Pipeline Modes | Full only | Fast/Balanced/Full | N/A | N/A |

---

*Generated for tokman v1.2.0 — 200 tasks across 8 categories*
