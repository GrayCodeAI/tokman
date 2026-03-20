# Architecture Decision Records

This directory contains Architecture Decision Records (ADRs) for TokMan.

## ADR Index

| ADR | Title | Status |
|-----|-------|--------|
| 001 | 14-Layer Compression Pipeline | Accepted |
| 002 | TOML Filter Configuration | Accepted |
| 003 | SQLite for Tracking | Accepted |
| 004 | CommandRunner Interface | Accepted |
| 005 | Pipeline Presets | Accepted |

---

## ADR-001: 14-Layer Compression Pipeline

**Date:** 2025-01-01
**Status:** Accepted

### Context
CLI output contains varying levels of redundancy. Different content types (code, logs, prose) benefit from different compression techniques.

### Decision
Use a 14-layer sequential pipeline where each layer applies a specific compression technique. Layers are ordered from least destructive to most destructive.

### Consequences
- **Positive:** Maximum compression for all content types; modular design allows disabling specific layers
- **Negative:** Sequential execution adds latency; some layers are redundant for certain content types
- **Mitigation:** Early-exit (T81) and content-aware skipping (T86) reduce unnecessary processing

---

## ADR-002: TOML Filter Configuration

**Date:** 2025-01-01
**Status:** Accepted

### Context
Users need a simple way to define custom compression rules for CLI tools without writing Go code.

### Decision
Use TOML files for filter configuration with match patterns, replace rules, and line filtering.

### Consequences
- **Positive:** Simple syntax; easy to contribute new filters; no compilation needed
- **Negative:** Limited expressiveness compared to code; TOML parsing overhead

---

## ADR-003: SQLite for Tracking

**Date:** 2025-01-01
**Status:** Accepted

### Context
Need persistent storage for command tracking data with zero external dependencies.

### Decision
Use SQLite via modernc.org/sqlite (pure Go, no CGO) with WAL mode.

### Consequences
- **Positive:** Zero dependencies; embedded; fast reads; WAL mode for concurrent access
- **Negative:** Single-file database; no remote access; migration complexity

---

## ADR-004: CommandRunner Interface

**Date:** 2025-03-20
**Status:** Accepted

### Context
Command execution was tightly coupled to os/exec, making testing difficult and preventing alternative implementations.

### Decision
Introduce CommandRunner interface with OSCommandRunner (production) and MockCommandRunner (testing).

### Consequences
- **Positive:** Testable; swappable implementations; dependency injection
- **Negative:** Additional abstraction layer; interface overhead negligible

---

## ADR-005: Pipeline Presets

**Date:** 2025-03-20
**Status:** Accepted

### Context
Running all 14 layers is overkill for many use cases. Users need a simple way to choose between speed and compression.

### Decision
Provide three presets: fast (3 layers), balanced (7 layers), full (14 layers).

### Consequences
- **Positive:** Simple UX; significant speedup for fast mode; backward compatible
- **Negative:** Preset choices may not match all use cases
- **Mitigation:** Users can still configure individual layers via config file
