# TokMan Codebase Reorganization Plan

**Date**: 2026-03-22  
**Status**: Draft for Review

---

## Executive Summary

TokMan is well-structured but has two main areas needing reorganization:
1. **`internal/commands/`** — 149 files in a flat structure (critical issue)
2. **`internal/filter/`** — 50+ files could benefit from sub-grouping

This plan proposes a logical categorization that improves navigation for both humans and LLMs while maintaining Go package conventions.

---

## Current State Analysis

### Directory Summary
```
tokman/
├── cmd/tokman/          # CLI entry point (1 file) ✓
├── internal/
│   ├── commands/        # 149 files — TOO FLAT ❌
│   ├── filter/          # 50+ files — COULD BE GROUPED ⚠️
│   ├── config/          # 4 files ✓
│   ├── server/          # 3 files ✓
│   ├── tracking/        # 1 file ✓
│   ├── [23 other pkgs]  # Various sizes
├── docs/                # Well-organized ✓
├── hooks/               # Shell scripts ✓
├── benchmarks/          # Performance tests ✓
└── tests/integration/   # Integration tests ✓
```

### Problem Areas

#### 1. `internal/commands/` (Critical — 149 files)
Current flat structure makes it hard to:
- Find related commands (e.g., all git-related handlers)
- Understand command categories
- Navigate mentally without IDE support

Files include:
- VCS: git*.go, hg.go, svn.go
- Containers: docker*.go, kubectl*.go, helm.go
- Cloud: aws*.go, gcloud.go, terraform.go
- Package managers: cargo*.go, npm*.go, pip*.go, go_mod.go
- Build tools: make.go, bazel.go, gradle.go
- Languages: python.go, node.go, rust.go, go*.go
- Testing: pytest.go, jest.go, vitest.go, gotest.go
- Config: config*.go, env*.go
- Core: help.go, version.go, doctor.go, completion.go

#### 2. `internal/filter/` (Moderate — 50+ files)
Contains 20+ compression layers plus supporting code:
- Core pipeline: pipeline.go, manager.go, router.go
- Layers: h2o.go, entropy.go, perplexity.go, etc.
- Utilities: utils.go, presets.go, cache.go
- Tests: Many *_test.go files mixed in

---

## Proposed Reorganization

### Phase 1: Categorize Commands (High Impact)

Create subdirectories under `internal/commands/`:

```
internal/commands/
├── vcs/                    # Version Control
│   ├── git.go
│   ├── git_*.go            # All git-related files
│   ├── hg.go
│   └── svn.go
│
├── container/              # Container & Orchestration
│   ├── docker.go
│   ├── docker_*.go
│   ├── kubectl.go
│   ├── kubectl_*.go
│   ├── helm.go
│   └── compose.go
│
├── cloud/                  # Cloud Infrastructure
│   ├── aws.go
│   ├── aws_*.go
│   ├── gcloud.go
│   ├── terraform.go
│   └── pulumi.go
│
├── pkgmgr/                 # Package Managers
│   ├── cargo.go
│   ├── cargo_*.go
│   ├── npm.go
│   ├── pnpm.go
│   ├── yarn.go
│   ├── pip.go
│   ├── pip_*.go
│   └── go_mod.go
│
├── build/                  # Build Tools
│   ├── make.go
│   ├── cmake.go
│   ├── bazel.go
│   ├── gradle.go
│   └── maven.go
│
├── lang/                   # Language Runtimes
│   ├── python.go
│   ├── python_*.go
│   ├── node.go
│   ├── node_*.go
│   ├── rust.go
│   ├── go_run.go
│   └── dotnet.go
│
├── test/                   # Test Runners
│   ├── pytest.go
│   ├── jest.go
│   ├── vitest.go
│   ├── gotest.go
│   ├── playwright.go
│   └── test_runner.go
│
├── core/                   # Core CLI Commands
│   ├── help.go
│   ├── version.go
│   ├── doctor.go
│   ├── completion.go
│   ├── alias.go
│   └── enable.go
│
├── config/                 # Configuration Commands
│   ├── config.go
│   ├── config_*.go
│   ├── env.go
│   └── env_*.go
│
├── analysis/               # Analysis & Metrics
│   ├── count.go
│   ├── cost.go
│   ├── budget.go
│   ├── compare.go
│   ├── audit.go
│   ├── benchmark.go
│   └── economics.go
│
├── output/                 # Output Processing
│   ├── diff.go
│   ├── explain.go
│   ├── export.go
│   ├── context.go
│   └── fallback.go
│
└── registry.go             # Command registry (stays at root)
```

**Note**: Each subdirectory becomes a separate Go package. This requires:
- Updating import paths across the codebase
- Adding package declarations
- Updating `cmd/tokman/main.go` imports

### Phase 2: Organize Filter Layers (Moderate Impact)

Create subdirectories under `internal/filter/`:

```
internal/filter/
├── pipeline.go             # Core orchestrator (stays at root)
├── manager.go              # Layer manager (stays at root)
├── router.go               # Request router (stays at root)
├── presets.go              # User-facing presets (stays at root)
├── utils.go                # Shared utilities (stays at root)
│
├── layers/                 # All compression layers
│   ├── entropy.go          # L1
│   ├── perplexity.go       # L2
│   ├── goal_aware.go       # L3 (rename from query_aware)
│   ├── ast_preserve.go     # L4
│   ├── contrastive.go      # L5
│   ├── ngram.go            # L6
│   ├── evaluator_heads.go  # L7
│   ├── gist.go             # L8
│   ├── hierarchical.go     # L9
│   ├── budget.go           # L10
│   ├── compaction.go       # L11
│   ├── attribution.go      # L12 (rename)
│   ├── h2o.go              # L13
│   ├── attention_sink.go   # L14
│   ├── meta_token.go       # L15
│   ├── semantic_chunk.go   # L16
│   ├── sketch_store.go     # L17
│   ├── lazy_pruner.go      # L18
│   ├── semantic_anchor.go  # L19
│   └── agent_memory.go     # L20
│
├── adaptive/               # Adaptive/dynamic logic
│   ├── adaptive.go
│   ├── adaptive_attention.go
│   └── density_adaptive.go
│
├── cache/                  # Caching subsystem
│   ├── lru_cache.go
│   └── fingerprint.go
│
└── [tests stay at current locations]
```

### Phase 3: Add Architecture Documentation

Create `docs/ARCHITECTURE.md`:

```markdown
# TokMan Architecture

## Overview
TokMan is a token reduction system implementing a 20-layer compression pipeline...

## Directory Structure
[Detailed explanation of each package]

## Package Dependencies
[Dependency graph]

## Data Flow
[How requests flow through the system]

## Adding New Commands
[Guide for contributors]

## Adding New Filter Layers
[Guide for contributors]
```

---

## Implementation Order

1. **Phase 1** (Commands): Highest impact, most files affected
2. **Phase 3** (Docs): Document the new structure immediately
3. **Phase 2** (Filters): Lower priority, optional improvement

---

## Risk Assessment

### Low Risk
- Creating subdirectories
- Moving files (with proper import updates)
- Adding documentation

### Medium Risk
- Import path changes across the codebase
- Test file locations
- Build system updates

### Mitigation
1. Run tests after each file move
2. Use `gofmt` and `goimports` after reorganization
3. Update `go.mod` if needed
4. Git rename tracking: use `git mv` to preserve history

---

## Expected Outcomes

### For Humans
- ✅ Logical file groupings by domain
- ✅ Easier navigation without IDE
- ✅ Clear separation of concerns
- ✅ Better onboarding for new contributors

### For LLMs
- ✅ Clearer context when reading directory structure
- ✅ Package names communicate purpose
- ✅ Reduced cognitive load for codebase exploration
- ✅ Better import path semantics

---

## Alternatives Considered

### Option A: Keep Flat Structure
- Pro: No import changes needed
- Con: 149 files in one directory is unmaintainable

### Option B: Use Internal Subpackages Only
- Keep `internal/commands` flat but add `internal/vcs`, `internal/container`, etc.
- Con: Breaks the semantic grouping (commands belong together)

### Option C: Functional Grouping (Chosen)
- Group commands by function (vcs, container, cloud, etc.)
- Best balance of organization and discoverability

---

## Questions for Review

1. **Package naming**: Are `vcs`, `pkgmgr`, `lang` clear enough, or prefer longer names like `version_control`, `package_managers`?

2. **Filter layers**: Should layers be in their own `layers/` subdirectory, or stay flat for simpler imports?

3. **Test files**: Should tests move with their source files, or stay in a central `tests/` directory?

4. **Registry file**: Should `registry.go` stay at `internal/commands/` root, or move to each subdirectory?

---

## Approval

- [ ] Phase 1 approved: Categorize commands into subdirectories
- [ ] Phase 2 approved: Organize filter layers  
- [ ] Phase 3 approved: Add architecture documentation

**Please confirm or suggest modifications before implementation.**
