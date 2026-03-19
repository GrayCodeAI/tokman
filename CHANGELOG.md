# Changelog

All notable changes to TokMan will be documented in this file.

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
