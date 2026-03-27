# Changelog

All notable changes to TokMan will be documented in this file.

## [2.2.0] - 2026-03-25

### Added

#### Bundle Install TOML Filter
- **`bundle-install.toml`** - New TOML filter for `bundle install` and `bundle update`
- Strips `Using`, `Fetching gem metadata`, `Resolving dependencies` lines
- Compact summary on completion: `ok bundle: complete` / `ok bundle: updated`
- Inline tests included for validation

#### Unit Tests for Ruby Toolchain
- **`bundle_test.go`** - 10 tests covering install, update, list, outdated, errors
- **`rspec_test.go`** - 12 tests covering JSON/text parsing, noise stripping, failures
- **`rake_test.go`** - 14 tests covering minitest output, runner selection, ANSI stripping
- **`rubocop_test.go`** - 9 tests covering JSON/text parsing, severity sorting, overflow

#### Hook Test Script
- **`scripts/test-hook-rewrite.sh`** - CI validation for hook rewrite behavior
- Tests git, gh, docker, kubectl, npm, cargo, pip, test runners, linters, edge cases
- 30+ test cases with pass/fail reporting

### Fixed
- **`numerical_quant.go`** - Removed unreachable code after premature `return` in `compressTimestamps`
- **`presets.go`** - Fixed `EnableGoalDriven` and `EnableContrastive` flags for fast/balanced presets
- **Stage gates** - Entropy density check was too coarse; replaced 4-bit grouping with 256-bit lookup table

### Performance
- **Pipeline hot path optimizations**:
  - O(1) running total for early exit checks (was O(n) per check)
  - Conditional `time.Now()` syscalls (only when session tracking enabled)
  - Zero-allocation stage gates: `shouldSkipPerplexity`, `shouldSkipNgram`, `shouldSkipEntropy`
  - 256-bit lookup table for character diversity (replaces map allocation)
- **Benchmark results** (31-layer pipeline):
  - Full pipeline: 47.52ms avg (9% faster)
  - Adaptive pipeline: 1.33x speedup over full
  - Simple inputs: <0.5ms (3 layers)


## [2.1.0] - 2026-03-21

### Added - Ecosystem Integration

#### MCP Server Mode (T8)
- **HTTP API Server** - `tokman mcp --port 8080` for Claude/ChatGPT integration
- **Endpoints**: `/compress`, `/explain`, `/restore`, `/health`
- **JSON API** - Full token stats, quality scores, and layer breakdowns
- **Query-aware compression** - Pass `query` parameter for intent-based filtering

#### Proxy Mode (T9)
- **API Proxy** - `tokman api-proxy --port 7878 --upstream https://api.anthropic.com`
- **Automatic compression** - Intercepts LLM API responses, compresses large strings
- **Transparent** - Zero code changes required in client applications

#### Question-Aware Recovery (T12)
- **Layer 21: QuestionAwareFilter** - Preserves query-relevant context during compression
- **Keyword extraction** - Identifies important terms from user queries
- **Context protection** - Prevents removal of answer-relevant passages

#### Density-Adaptive Allocation (T17)
- **Layer 22: DensityAdaptiveFilter** - Non-uniform compression ratios per section
- **Density analysis** - Measures information density across content
- **Selective compression** - Preserves dense regions, aggressively compresses sparse areas

#### Result Fingerprinting (T35)
- **Content hashing** - SHA-256 fingerprints for compression results
- **Cache optimization** - Enables content-based deduplication
- **Layer 11 integration** - Compaction layer reuses fingerprinted content

### Fixed
- **Audit command** - Removed conflicting `-q` shorthand that caused panic

### Performance
- **Go 1.26.1** - Upgraded to latest Go with native SIMD support
- **Benchmark results**:
  - Small input: 6.6µs/op (167K ops/sec)
  - Log compression: 47K tokens saved in 5.4ms
  - 82% compression on test output (balanced mode)

### Pipeline Summary
- **22 active layers** (L1-L14 original + L15-L22 research-backed)
- **Stage gates** for zero-cost layer skipping
- **BM25 scoring** for superior relevance ranking
- **Adaptive attention sinks** for streaming contexts

## [2.0.0] - 2026-03-21

### Added - Go 1.26 SIMD Optimizations

#### Native SIMD Support
- **Go 1.26 Upgrade** - Project upgraded to Go 1.26.0 with `GOEXPERIMENT=simd`
- **SIMD Package** - New `internal/simd` package with vectorized operations
- **ANSI Stripping** - SIMD-accelerated ANSI escape sequence removal (3-5x faster than regex)
- **Byte Operations** - Vectorized word boundary detection and byte matching

#### Build System
- `make build-simd` - Build with SIMD optimizations enabled
- `make build-all` - Build all platform binaries with SIMD
- Binary size: 28MB (stripped, includes SIMD tables)

### Added - 20-Layer Compression Pipeline

#### New Research-Backed Layers (L15-L20)
- **Layer 15: Adaptive Attention Sinks** - StreamingLLM-style dynamic sink count based on output length
- **Layer 16: Tiered Memory** - MemGPT-style hot/warm/cold memory tiers for compaction
- **Layer 17: Density-Adaptive Allocation** - DAST-style non-uniform compression ratios
- **Layer 18: Information Bottleneck** - QUITO-X-style optimal compression theory
- **Layer 19: Explicit Information Transmission** - ComprExIT-style explicit transmission modeling
- **Layer 20: BM25 Relevance** - BM25 scoring for better relevance ranking than TF-IDF

#### Algorithm Improvements
- **T11: Dynamic Frequency Estimation** - Zipf's law-based per-document frequency (+15-20% accuracy)
- **T13: Attention Score Simulation** - TF-IDF and local attention patterns in H2O filter
- **T1: Reversible Compression** - Full compression reversibility with metadata preservation

### Added - Filter Marketplace (T40)

#### Community TOML Filters
15 production-ready community filters in `/filters/`:
- `jest.toml` - Jest test framework output
- `eslint.toml` - ESLint linting output
- `prettier.toml` - Prettier formatting output
- `webpack.toml` - Webpack build output
- `vite.toml` - Vite dev server output
- `terraform.toml` - Terraform plan/apply output
- `docker.toml` - Docker build/run output
- `kubectl.toml` - Kubernetes CLI output
- `ansible.toml` - Ansible playbook output
- `gradle.toml` - Gradle build output
- `maven.toml` - Maven build output
- `cargo.toml` - Cargo/Rust build output
- `pip.toml` - Pip install output
- `npm.toml` - NPM install output
- `go-mod.toml` - Go mod download output

### Performance

- SIMD ANSI stripping: 3-5x faster than regex-based approach
- N-gram abbreviation: 2-3x faster with SIMD byte operations
- Pipeline throughput: 15-20% improvement with SIMD optimizations
- Memory efficiency: Reduced allocations in hot paths

### Changed

- Minimum Go version: 1.26.0 (for native SIMD support)
- Build requires `GOEXPERIMENT=simd` environment variable
- `internal/simd.IsWordChar` (byte) separate from `internal/filter.isWordChar` (rune) for Unicode

## [1.2.0] - 2025-03-19

### Added - 14-Layer Compression Pipeline

#### New Compression Layers
- **Layer 11: Compaction** - MemGPT-style semantic compression for chat/conversation content (98%+ reduction)
- **Layer 12: Attribution Filter** - ProCut-style pruning based on LinkedIn Research 2025 (78% reduction)
- **Layer 13: H2O Filter** - Heavy-Hitter Oracle from NeurIPS 2023 (30x+ compression)
- **Layer 14: Attention Sink** - StreamingLLM-style infinite context stability

#### Performance Benchmarks
- Comprehensive benchmark suite for all 14 layers
- Context capacity validation up to 2M tokens
- Compression ratio tests (95-99% reduction verified)
- Layer-by-layer performance isolation

### Performance

- Small output (100 lines): 95.5% reduction (982 → 44 tokens)
- Medium output (1000 lines): 99.5% reduction (9,737 → 52 tokens)
- Large output (5000 lines): 99.9% reduction (49,437 → 63 tokens)
- 2M token context: 207 seconds processing time

### Optimized

- H2O filter memory usage for large contexts (>50K tokens)
- Line-based processing path reduces allocations by 10-20x
- Compaction layer returns original on empty output

### Documentation

- Updated LAYERS.md with Layer 11-14 documentation
- Updated README.md with 14-layer pipeline table
- Added research references for all layers

## [1.1.0] - 2025-03-08

### Added - Beyond Parity Features

#### Custom Filter Plugins
- JSON-based user-defined filter rules
- `tokman plugin list` - List all loaded plugins
- `tokman plugin create <name>` - Create new plugin template
- `tokman plugin enable/disable <name>` - Toggle plugins
- `tokman plugin examples` - Generate example plugins
- Pattern matching with hide/replace modes
- Auto-loading from `~/.config/tokman/plugins/*.json`

#### Web Dashboard
- `tokman dashboard` - Launch interactive web dashboard
- Real-time token savings visualization with Chart.js
- Daily/weekly/monthly breakdowns
- Command-level analytics
- Cost tracking with Claude API rates
- RESTful API endpoints for custom integrations

#### Shell Completions
- Bash completion script (`completions/tokman.bash`)
- Zsh completion script (`completions/_tokman`)
- Fish completion script (`completions/tokman.fish`)

#### CI/CD Integration
- GitHub Actions workflow template (`templates/github-actions.yml`)
- GitLab CI template (`templates/gitlab-ci.yml`)
- Token savings reporting in CI pipelines

### Added - Testing & Quality

- Unit tests for `internal/ccusage` package
- Unit tests for `internal/commands` (smart command)
- Unit tests for `internal/config` (Windows path validation)
- Unit tests for `internal/economics` package
- Unit tests for `internal/utils` package
- Performance benchmarks for filter engine
- Sub-millisecond filtering performance

### Performance

- Filter short input: 8.9 µs
- Filter git status: 26 µs
- Filter large output (2000 lines): 1.07 ms
- Zero-allocation utility functions (EstimateTokens, IsCode)

## [1.0.0] - 2025-03-08

### Added - Core Features

- Token-aware CLI proxy for LLM interactions
- Smart command rewriting (git, npm, go, pytest, etc.)
- Intelligent output filtering to reduce token usage
- Token savings tracking with SQLite database
- Economics reporting with Claude API pricing
- Runtime integrity verification
- Cross-platform support (Linux, macOS, Windows)

### Commands

- `tokman git` - Git command filtering
- `tokman smart <file>` - 2-line code summary
- `tokman status` - Project token savings
- `tokman report` - Detailed savings report
- `tokman economics` - Cost analysis
- `tokman init` - Initialize TokMan for project
- `tokman verify` - Verify installation integrity

### Configuration

- XDG Base Directory support
- Windows path support (%APPDATA%, %LOCALAPPDATA%)
- Environment variable overrides (TOKMAN_*)
- TOML configuration file support
- TokMan environment variable compatibility

### TokMan Feature Parity

- 100% feature parity with TokMan (Go)
- All filtering patterns implemented
- All command rewrites supported
- All configuration options available
- Environment variable compatibility maintained
