# RTK to TokMan Migration Plan

> 100 detailed tasks to implement RTK features in TokMan
> Created: 2026-03-30
> Status: In Progress

## Overview

This plan migrates high-value features from RTK (Rust Token Killer) to TokMan, focusing on:
1. TOML-based declarative filters
2. Session discovery and analytics
3. Expanded language support
4. Additional AI integrations
5. Enhanced command coverage

---

## Phase 1: TOML Filter System (Tasks 1-20)

### Core Infrastructure

- [ ] **Task 1**: Create `internal/tomlfilter/` package structure
- [ ] **Task 2**: Define `TomlFilter` struct with all filter fields
- [ ] **Task 3**: Implement `LoadFilters(directory string)` to load .toml files
- [ ] **Task 4**: Implement `MatchFilter(command string)` to find matching filter
- [ ] **Task 5**: Implement `ApplyFilter(input string, filter TomlFilter)` 
- [ ] **Task 6**: Add `strip_ansi` preprocessing support
- [ ] **Task 7**: Add `strip_lines_matching` regex filtering
- [ ] **Task 8**: Add `keep_lines_matching` regex filtering
- [ ] **Task 9**: Add `replace` regex substitution support
- [ ] **Task 10**: Add `max_lines` truncation support
- [ ] **Task 11**: Add `tail_lines` support
- [ ] **Task 12**: Add `truncate_lines_at` support
- [ ] **Task 13**: Add `on_empty` fallback message support
- [ ] **Task 14**: Add `match_output` short-circuit rules

### Testing & Validation

- [ ] **Task 15**: Implement inline test parser for TOML filters
- [ ] **Task 16**: Create `tomlfilter test` command to run filter tests
- [ ] **Task 17**: Add TOML syntax validation
- [ ] **Task 18**: Add filter benchmarking support
- [ ] **Task 19**: Create sample TOML filter files (5 templates)
- [ ] **Task 20**: Integrate TOML filters with pipeline coordinator

---

## Phase 2: Session Discovery (Tasks 21-35)

### Provider System

- [ ] **Task 21**: Create `internal/discover/` package structure
- [ ] **Task 22**: Define `SessionProvider` interface
- [ ] **Task 23**: Implement `ClaudeProvider` for Claude Code sessions
- [ ] **Task 24**: Add JSONL file discovery in `~/.claude/projects/`
- [ ] **Task 25**: Implement JSONL streaming parser
- [ ] **Task 26**: Extract Bash commands from session files
- [ ] **Task 27**: Extract output content and lengths
- [ ] **Task 28**: Handle subagent session files
- [ ] **Task 29**: Implement project filtering by path
- [ ] **Task 30**: Add time-based filtering (since N days)

### Command Classification

- [ ] **Task 31**: Create command classification registry
- [ ] **Task 32**: Implement `classify_command()` function
- [ ] **Task 33**: Handle chained commands (&&, ;, ||)
- [ ] **Task 34**: Track RTK/TokMan adoption metrics
- [ ] **Task 35**: Create session summary data structures

---

## Phase 3: Discover Command (Tasks 36-45)

- [ ] **Task 36**: Create `internal/commands/analysis/discover.go`
- [ ] **Task 37**: Implement command history analysis
- [ ] **Task 38**: Calculate missed savings opportunities
- [ ] **Task 39**: Group opportunities by command type
- [ ] **Task 40**: Add project-level aggregation
- [ ] **Task 41**: Add time-range filtering (--since flag)
- [ ] **Task 42**: Add output format options (--format json/table)
- [ ] **Task 43**: Create recommendation engine
- [ ] **Task 44**: Add actionable suggestions output
- [ ] **Task 45**: Integrate with `tokman gain` command

---

## Phase 4: Session Command (Tasks 46-55)

- [ ] **Task 46**: Create `internal/commands/sessioncmd/session.go`
- [ ] **Task 47**: Implement session listing functionality
- [ ] **Task 48**: Add session adoption percentage calculation
- [ ] **Task 49**: Implement progress bar visualization
- [ ] **Task 50**: Add session detail view
- [ ] **Task 51**: Add per-session command breakdown
- [ ] **Task 52**: Add output token tracking per session
- [ ] **Task 53**: Implement session comparison view
- [ ] **Task 54**: Add export functionality (--export csv/json)
- [ ] **Task 55**: Create session history trend analysis

---

## Phase 5: Ruby Language Support (Tasks 56-65) ✅

- [x] **Task 56**: Create `internal/commands/lang/ruby.go` package
- [x] **Task 57**: Implement `rake test` command wrapper
- [x] **Task 58**: Implement `rspec` command wrapper with JSON output
- [x] **Task 59**: Implement `rubocop` command wrapper with JSON output
- [x] **Task 60**: Implement `bundle install` filtering
- [x] **Task 61**: Add Ruby test output compression
- [x] **Task 62**: Add Ruby lint output compression
- [x] **Task 63**: Create Ruby ecosystem tests
- [x] **Task 64**: Add Ruby command registry entries
- [x] **Task 65**: Document Ruby command support

---

## Phase 6: Additional AI Integrations (Tasks 66-72) ✅

- [x] **Task 66**: Implement GitHub Copilot hook integration
- [x] **Task 67**: Create `.github/hooks/tokman-rewrite.json` template
- [x] **Task 68**: Add `.github/copilot-instructions.md` generation
- [x] **Task 69**: Implement OpenCode plugin integration
- [x] **Task 70**: Create OpenCode plugin TypeScript template
- [x] **Task 71**: Add Mistral Vibe placeholder (track upstream)
- [x] **Task 72**: Update `tokman init` with new agent options

---

## Phase 7: Extended Command Coverage (Tasks 73-85) ✅

### Infrastructure Tools

- [x] **Task 73**: Add `terraform plan` command wrapper
- [x] **Task 74**: Add `helm` command wrapper
- [x] **Task 75**: Add `ansible-playbook` command wrapper

### Build Tools

- [x] **Task 76**: Add `gradle` command wrapper
- [x] **Task 77**: Add `mvn` / `maven` command wrapper
- [x] **Task 78**: Add `make` output filtering

### Language Tools

- [x] **Task 79**: Add `mix compile` (Elixir) command wrapper
- [x] **Task 80**: Add `markdownlint` command wrapper
- [x] **Task 81**: Add `mise` command wrapper
- [x] **Task 82**: Add `just` command wrapper

### System Tools

- [x] **Task 83**: Add `df` command wrapper
- [x] **Task 84**: Add `du` command wrapper
- [x] **Task 85**: Add `jq` command wrapper

---

## Phase 8: Configuration & UX (Tasks 86-92) ✅

- [x] **Task 86**: Add hook exclusion configuration (`hooks.exclude_commands`)
- [x] **Task 87**: Enhance tee configuration with modes (`failures`, `always`, `never`)
- [x] **Task 88**: Add `--auto-patch` flag for non-interactive init
- [x] **Task 89**: Implement progress bar utility function
- [x] **Task 90**: Add color-coded output for analytics
- [x] **Task 91**: Improve verbose logging structure
- [x] **Task 92**: Add configuration validation command

---

## Phase 9: Testing & Documentation (Tasks 93-100)

- [x] **Task 93**: Add unit tests for TOML filter system (coverage >80%)
- [x] **Task 94**: Add unit tests for session discovery (coverage >80%)
- [ ] **Task 95**: Add integration tests for new commands
- [ ] **Task 96**: Update README.md with new features
- [ ] **Task 97**: Update AGENTS.md with Ruby support
- [ ] **Task 98**: Create TOML_FILTERS.md documentation
- [ ] **Task 99**: Create SESSION_DISCOVERY.md documentation
- [ ] **Task 100**: Final review and cleanup

---

## Progress Tracking

| Phase | Tasks | Completed | Status |
|-------|-------|-----------|--------|
| 1. TOML Filter System | 1-20 | 0 | Pending |
| 2. Session Discovery | 21-35 | 0 | Pending |
| 3. Discover Command | 36-45 | 0 | Pending |
| 4. Session Command | 46-55 | 0 | Pending |
| 5. Ruby Support | 56-65 | 10 | ✅ Complete |
| 6. AI Integrations | 66-72 | 7 | ✅ Complete |
| 7. Extended Commands | 73-85 | 13 | ✅ Complete |
| 8. Configuration & UX | 86-92 | 7 | ✅ Complete |
| 9. Testing & Docs | 93-100 | 2 | In Progress |
| **Total** | **100** | **39** | **39%** |

---

## Execution Order

Tasks are designed to be executed sequentially within phases. Dependencies:
- Phase 2 depends on Phase 1 (TOML filters used in discovery)
- Phase 3-4 depend on Phase 2 (discovery infrastructure)
- Phase 5-7 can run in parallel after Phase 1
- Phase 8 can start after Phase 1
- Phase 9 runs throughout and finalizes at end
