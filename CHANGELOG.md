# Changelog

All notable changes to TokMan will be documented in this file.

## [1.3.0](https://github.com/GrayCodeAI/tokman/compare/v1.2.0...v1.3.0) (2026-03-21)


### Features

* add 14 more test files, 4 TOML filters ([9c4e332](https://github.com/GrayCodeAI/tokman/commit/9c4e332d15a30a2a9da051a158033fe94d32ee6e))
* add adaptive layer selection based on content type ([3527f8e](https://github.com/GrayCodeAI/tokman/commit/3527f8e33304bd04311a8bc63cd0dd5e4a48a8dd))
* add community TOML filter marketplace (T40) ([deaa28a](https://github.com/GrayCodeAI/tokman/commit/deaa28a01cac0aa78c15c613df41c96f818096f4))
* add comprehensive benchmark suite for Tokman performance comparison ([357a872](https://github.com/GrayCodeAI/tokman/commit/357a87232f8fcf3788ad38af9f939a2aba9d1842))
* add dynamic frequency estimation (T11) for +15-20% entropy accuracy ([0beeb6a](https://github.com/GrayCodeAI/tokman/commit/0beeb6a906759c55a77c569e9cc34b36cbec2652))
* add Go SDK for native token compression ([72d216d](https://github.com/GrayCodeAI/tokman/commit/72d216d0f726dc6e19d7e0e1fd066c91dc796ad8))
* add LangChain and LlamaIndex integration examples ([2328d0f](https://github.com/GrayCodeAI/tokman/commit/2328d0f7f6d5aeec2cbc0f8ae80ced732c1d0812))
* add performance optimizations and new features ([e997bfa](https://github.com/GrayCodeAI/tokman/commit/e997bfa22f0c7565337fab0224b014b085efcf3a))
* add Prometheus metrics and structured logging ([c30c026](https://github.com/GrayCodeAI/tokman/commit/c30c026149e4a5f8ed11001fcd8ca5d851e775bd))
* add Python SDK for token compression ([cef5caf](https://github.com/GrayCodeAI/tokman/commit/cef5caf3e716f5dc55bd851dbcad298b441f39e6))
* add reversible compression and restore command ([714a837](https://github.com/GrayCodeAI/tokman/commit/714a8370cac43f0101dd82d0eaa468787b0e3df3))
* add SIMD-optimized operations for compression hot paths ([5018549](https://github.com/GrayCodeAI/tokman/commit/50185499b8f45ba8227dc477f0a0d17743cb38e0))
* add stage gates and adaptive attention sinks (T7, T14) ([2a44e69](https://github.com/GrayCodeAI/tokman/commit/2a44e69d2244b79f49334114c235b27802eaf93c))
* add streaming API for real-time compression ([e24302e](https://github.com/GrayCodeAI/tokman/commit/e24302e181ffd7f2d184c1dee0ada34ba0b17a7b))
* add T12, T17, T30 - question-aware, density-adaptive, and binary optimization ([f87dfb8](https://github.com/GrayCodeAI/tokman/commit/f87dfb85e9dd4c12634c499be258c84acfa9bfda))
* add tests, multi-language READMEs, expand TOML filters ([6f76a1a](https://github.com/GrayCodeAI/tokman/commit/6f76a1a4ac44c1e272db4b0d9ff1c2ae81a9609b))
* add TypeScript/Node.js SDK ([224d690](https://github.com/GrayCodeAI/tokman/commit/224d69098ca91298ce584e5183ea4c0f76bb734c))
* add WASM build for browser-based compression ([2bff3fc](https://github.com/GrayCodeAI/tokman/commit/2bff3fc1456da0942e7d390120bbeaba9cece43a))
* implement 7 research-backed compression layers (Layers 15-20) ([cfe87ba](https://github.com/GrayCodeAI/tokman/commit/cfe87bab387fba456a02ff601571c4b77c9f1a93))
* improve attention score simulation (T13) for heavy-hitter identification ([5d0b1f9](https://github.com/GrayCodeAI/tokman/commit/5d0b1f9b39b8bcf9f23b2080cb2d09a07edb554c))
* port all tokf features to tokman ([7e7ce4a](https://github.com/GrayCodeAI/tokman/commit/7e7ce4a4a7f43b0bfb2f95d1119a9b153ee239bb))


### Bug Fixes

* correct unit tests for new compression layers (L15-L19) ([497de4c](https://github.com/GrayCodeAI/tokman/commit/497de4cf284fa4ef6b2f21d3c02e91b8008f8bd3))
* handle short strings in analyze test preview ([6801134](https://github.com/GrayCodeAI/tokman/commit/68011349622e3f15a4e121ee3ee6258383573cf1))
* prevent over-pruning on small inputs in compression layers ([ea25908](https://github.com/GrayCodeAI/tokman/commit/ea259085b8122489e1c5d89ac1146555cb906526))
* remove conflicting -q shorthand from audit command ([e819535](https://github.com/GrayCodeAI/tokman/commit/e8195357ffdd1c882ac36709b23a8ea13f6812ac))
* resolve 17 code quality issues ([f2bc16a](https://github.com/GrayCodeAI/tokman/commit/f2bc16a121938248f6a9b62112d2a2bf428b6146))


### Performance Improvements

* streaming memory optimization for large contexts ([0e216ae](https://github.com/GrayCodeAI/tokman/commit/0e216ae3c793e5f2caa6ecd43efe80fcb81aabb0))

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
