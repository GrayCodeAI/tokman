# TokMan Implementation Plan
**Date:** March 8, 2026  
**Project:** TokMan - Token-Aware CLI Proxy  
**Language:** Go 1.21+  
**Based on:** RTK (Rust Token Killer)

---

## TL;DR

TokMan is a CLI token-reduction proxy using the **Proxy Pattern**: intercept commands → execute → filter output → track savings in SQLite. It provides shell hooks for automatic command rewriting and aggregated reporting.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                          TokMan CLI                             │
│                       (Cobra Framework)                         │
├─────────────────────────────────────────────────────────────────┤
│  Command Router (cmd/tokman/main.go)                            │
│  ├── Recognized commands → Execute with filtering               │
│  └── Unrecognized commands → Fallback execution + log failure   │
├─────────────────────────────────────────────────────────────────┤
│  Command Handlers (internal/commands/)                          │
│  ├── root.go         - Entry point, global flags                │
│  ├── init.go         - Initialize database & config             │
│  ├── status.go       - Token savings summary                    │
│  ├── report.go       - Detailed usage analytics                 │
│  ├── git.go          - Git command wrappers                     │
│  ├── ls.go           - ls wrapper with noise filtering          │
│  ├── cargo.go        - Cargo/test aggregation                   │
│  ├── rewrite.go      - Command rewriting for hooks              │
│  └── hook.go         - Shell hook management                    │
├─────────────────────────────────────────────────────────────────┤
│  Core Engine (internal/)                                        │
│  ├── filter/                                                     │
│  │   ├── filter.go       - Minimal & Aggressive modes           │
│  │   ├── brace_depth.go  - Brace tracking for body stripping    │
│  │   ├── patterns.go     - Regex patterns (ANSI, imports, etc) │
│  │   └── aggregator.go   - Log deduplication & compression      │
│  ├── tracking/                                                   │
│  │   ├── tracker.go      - Token estimation & persistence       │
│  │   ├── migrations.go   - SQLite schema                        │
│  │   ├── gain.go         - Savings calculation                  │
│  │   └── models.go       - Data structures                      │
│  ├── config/                                                     │
│  │   ├── config.go       - TOML/YAML config loader              │
│  │   └── defaults.go     - Default values                       │
│  ├── discover/                                                   │
│  │   ├── registry.go     - Command rewrite registry             │
│  │   └── rewriter.go     - Command transformation logic         │
│  └── utils/                                                      │
│      ├── utils.go        - Helpers (canonical path, etc)        │
│      └── logger.go       - Structured logging (slog)            │
├─────────────────────────────────────────────────────────────────┤
│  Shell Integration (hooks/)                                      │
│  └── tokman-rewrite.sh   - Bash hook for Claude Code            │
├─────────────────────────────────────────────────────────────────┤
│  Data Layer                                                      │
│  ├── SQLite (~/.local/share/tokman/history.db)                  │
│  ├── Config (~/.config/tokman/config.toml)                      │
│  └── Logs (~/.local/share/tokman/tokman.log)                    │
└─────────────────────────────────────────────────────────────────┘
```

---

## Directory Structure

```
tokman/
├── cmd/
│   └── tokman/
│       └── main.go              # Entry point & router
├── internal/
│   ├── commands/
│   │   ├── root.go              # Root command definition
│   │   ├── init.go              # `tokman init`
│   │   ├── status.go            # `tokman status`
│   │   ├── report.go            # `tokman report`
│   │   ├── git.go               # `tokman git` subcommands
│   │   ├── ls.go                # `tokman ls`
│   │   ├── cargo.go             # `tokman cargo` wrappers
│   │   ├── rewrite.go           # `tokman rewrite`
│   │   └── hook.go              # `tokman hook`
│   ├── filter/
│   │   ├── filter.go            # Core filtering engine
│   │   ├── brace_depth.go       # Brace depth tracker
│   │   ├── patterns.go          # Regex definitions
│   │   ├── aggregator.go        # Output aggregation
│   │   └── ansi.go              # ANSI stripping
│   ├── tracking/
│   │   ├── tracker.go           # Database operations
│   │   ├── migrations.go        # Schema migrations
│   │   ├── gain.go              # Token savings calculator
│   │   └── models.go            # Structs & interfaces
│   ├── config/
│   │   ├── config.go            # Config loading
│   │   └── defaults.go          # Default configuration
│   ├── discover/
│   │   ├── registry.go          # Command rewrite mappings
│   │   └── rewriter.go          # Rewrite logic
│   └── utils/
│       ├── utils.go             # Shared utilities
│       ├── canonical.go         # Canonical path resolution
│       └── logger.go            # Logging setup
├── hooks/
│   └── tokman-rewrite.sh        # Shell hook script
├── configs/
│   └── default.toml             # Default config template
├── go.mod
├── go.sum
├── Makefile
├── LICENSE
└── README.md
```

---

## Phase 1: Core Infrastructure

### 1.1 Project Initialization

**Tasks:**
- [ ] Initialize Go module: `go mod init github.com/yourusername/tokman`
- [ ] Add dependencies:
```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get modernc.org/sqlite@latest
go get github.com/fatih/color@latest
go get github.com/BurntSushi/toml@latest
```

### 1.2 Configuration System

**File:** `internal/config/config.go`

```go
package config

type Config struct {
    Tracking TrackingConfig `mapstructure:"tracking"`
    Filter   FilterConfig   `mapstructure:"filter"`
    Hooks    HooksConfig    `mapstructure:"hooks"`
}

type TrackingConfig struct {
    Enabled     bool   `mapstructure:"enabled"`
    DatabasePath string `mapstructure:"database_path"`
    Telemetry   bool   `mapstructure:"telemetry"`
}

type FilterConfig struct {
    NoiseDirs []string `mapstructure:"noise_dirs"`
    Mode      string   `mapstructure:"mode"` // "minimal" or "aggressive"
}

type HooksConfig struct {
    ExcludedCommands []string `mapstructure:"excluded_commands"`
}

// Default configuration
func Defaults() *Config {
    return &Config{
        Tracking: TrackingConfig{
            Enabled:      true,
            DatabasePath: "", // Will be set to ~/.local/share/tokman/history.db
            Telemetry:    false,
        },
        Filter: FilterConfig{
            NoiseDirs: []string{
                ".git", "node_modules", "target", 
                "__pycache__", ".venv", "vendor",
            },
            Mode: "minimal",
        },
        Hooks: HooksConfig{
            ExcludedCommands: []string{},
        },
    }
}
```

**File:** `internal/config/defaults.go`

```go
package config

import (
    "os"
    "path/filepath"
)

// Config paths (XDG Base Directory Specification)
func ConfigPath() string {
    if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
        return filepath.Join(xdg, "tokman", "config.toml")
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".config", "tokman", "config.toml")
}

func DataPath() string {
    if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
        return filepath.Join(xdg, "tokman")
    }
    home, _ := os.UserHomeDir()
    return filepath.Join(home, ".local", "share", "tokman")
}

func DatabasePath() string {
    return filepath.Join(DataPath(), "history.db")
}

func LogPath() string {
    return filepath.Join(DataPath(), "tokman.log")
}
```

### 1.3 SQLite Database & Migrations

**File:** `internal/tracking/migrations.go`

```go
package tracking

const SchemaVersion = 1

const CreateCommandsTable = `
CREATE TABLE IF NOT EXISTS commands (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    command TEXT NOT NULL,
    original_output TEXT,
    filtered_output TEXT,
    original_tokens INTEGER NOT NULL,
    filtered_tokens INTEGER NOT NULL,
    saved_tokens INTEGER NOT NULL,
    project_path TEXT NOT NULL,
    session_id TEXT,
    exec_time_ms INTEGER,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
    parse_success BOOLEAN DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_commands_timestamp ON commands(timestamp);
CREATE INDEX IF NOT EXISTS idx_commands_project ON commands(project_path);
CREATE INDEX IF NOT EXISTS idx_commands_session ON commands(session_id);
`

// Migration history
var Migrations = []string{
    CreateCommandsTable,
}
```

**File:** `internal/tracking/tracker.go`

```go
package tracking

import (
    "database/sql"
    "time"
    _ "modernc.org/sqlite"
)

type Tracker struct {
    db *sql.DB
}

type CommandRecord struct {
    ID              int64
    Command         string
    OriginalOutput  string
    FilteredOutput  string
    OriginalTokens  int
    FilteredTokens  int
    SavedTokens     int
    ProjectPath     string
    SessionID       string
    ExecTimeMs      int64
    Timestamp       time.Time
    ParseSuccess    bool
}

func NewTracker(dbPath string) (*Tracker, error) {
    db, err := sql.Open("sqlite", dbPath)
    if err != nil {
        return nil, err
    }
    
    // Run migrations
    for _, migration := range Migrations {
        if _, err := db.Exec(migration); err != nil {
            return nil, err
        }
    }
    
    return &Tracker{db: db}, nil
}

// Token estimation heuristic (from RTK)
// ceil(text.length / 4.0)
func EstimateTokens(text string) int {
    return (len(text) + 3) / 4
}

func (t *Tracker) Record(record *CommandRecord) error {
    query := `
        INSERT INTO commands (
            command, original_output, filtered_output,
            original_tokens, filtered_tokens, saved_tokens,
            project_path, session_id, exec_time_ms, parse_success
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
    `
    
    _, err := t.db.Exec(query,
        record.Command,
        record.OriginalOutput,
        record.FilteredOutput,
        record.OriginalTokens,
        record.FilteredTokens,
        record.SavedTokens,
        record.ProjectPath,
        record.SessionID,
        record.ExecTimeMs,
        record.ParseSuccess,
    )
    
    return err
}

// Query savings for project path (uses GLOB for path matching)
func (t *Tracker) GetSavings(projectPath string) (*SavingsSummary, error) {
    query := `
        SELECT 
            COUNT(*) as total_commands,
            SUM(saved_tokens) as total_saved,
            SUM(original_tokens) as total_original
        FROM commands
        WHERE project_path GLOB ? OR project_path = ?
    `
    
    pattern := projectPath + "*"
    var summary SavingsSummary
    
    err := t.db.QueryRow(query, pattern, projectPath).Scan(
        &summary.TotalCommands,
        &summary.TotalSaved,
        &summary.TotalOriginal,
    )
    
    return &summary, err
}
```

### 1.4 Logging System

**File:** `internal/utils/logger.go`

```go
package utils

import (
    "log/slog"
    "os"
    "path/filepath"
)

var Logger *slog.Logger

func InitLogger(logPath string, verbose bool) error {
    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
        return err
    }
    
    file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
    if err != nil {
        return err
    }
    
    level := slog.LevelInfo
    if verbose {
        level = slog.LevelDebug
    }
    
    Logger = slog.New(slog.NewJSONHandler(file, &slog.HandlerOptions{
        Level: level,
    }))
    
    return nil
}
```

---

## Phase 2: Filtering Engine

### 2.1 Core Filter Logic

**File:** `internal/filter/filter.go`

```go
package filter

type FilterMode string

const (
    ModeMinimal    FilterMode = "minimal"
    ModeAggressive FilterMode = "aggressive"
)

type Filter interface {
    Name() string
    Apply(input string, mode FilterMode) (output string, tokensSaved int)
}

type Engine struct {
    filters []Filter
    mode    FilterMode
}

func NewEngine(mode FilterMode) *Engine {
    return &Engine{
        filters: []Filter{
            &ANSIFilter{},
            &CommentFilter{},
            &ImportFilter{},
            &BodyFilter{},      // aggressive mode only
            &LogAggregator{},
        },
        mode: mode,
    }
}

func (e *Engine) Process(input string) (string, int) {
    output := input
    totalSaved := 0
    
    for _, filter := range e.filters {
        if e.mode == ModeAggressive || filter.Name() != "body" {
            filtered, saved := filter.Apply(output, e.mode)
            output = filtered
            totalSaved += saved
        }
    }
    
    return output, totalSaved
}
```

### 2.2 ANSI Stripping

**File:** `internal/filter/ansi.go`

```go
package filter

import "regexp"

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

type ANSIFilter struct{}

func (f *ANSIFilter) Name() string {
    return "ansi"
}

func (f *ANSIFilter) Apply(input string, mode FilterMode) (string, int) {
    original := len(input)
    output := ansiPattern.ReplaceAllString(input, "")
    return output, (original - len(output)) / 4 // token estimation
}
```

### 2.3 Brace Depth Tracker (Aggressive Mode)

**File:** `internal/filter/brace_depth.go`

```go
package filter

import (
    "strings"
)

// BraceDepthTracker strips function bodies while preserving signatures
// Used for aggressive filtering mode
type BraceDepthTracker struct{}

// Preserves function signatures, strips bodies
// Pattern: fn name(args) { body } → fn name(args) { ... }
func StripFunctionBody(code string) string {
    var result strings.Builder
    var inBody bool
    depth := 0
    sigStart := 0
    
    // Regex for function signatures
    // Matches: fn, def, function, func, class, struct, enum, trait, interface, type
    sigPattern := regexp.MustCompile(`^(pub\s+)?(async\s+)?(fn|def|function|func|class|struct|enum|trait|interface|type)\s+\w+`)
    
    lines := strings.Split(code, "\n")
    
    for i, line := range lines {
        // Check if this line starts a function
        if sigPattern.MatchString(strings.TrimSpace(line)) {
            sigStart = i
            inBody = true
            depth = 0
        }
        
        // Track brace depth
        for _, ch := range line {
            if ch == '{' {
                depth++
            } else if ch == '}' {
                depth--
                if depth == 0 && inBody {
                    inBody = false
                    // Output signature + placeholder
                    result.WriteString(line + " // body stripped\n")
                    continue
                }
            }
        }
        
        if !inBody || depth == 0 {
            result.WriteString(line + "\n")
        }
    }
    
    return result.String()
}
```

### 2.4 Pattern Definitions

**File:** `internal/filter/patterns.go`

```go
package filter

import "regexp"

var (
    // Language-specific comment patterns
    CommentPatterns = map[string]*regexp.Regexp{
        "go":     regexp.MustCompile(`//.*$|/\*[\s\S]*?\*/`),
        "rust":   regexp.MustCompile(`//.*$|/\*[\s\S]*?\*/`),
        "python": regexp.MustCompile(`#.*$|"""[\s\S]*?"""|'''[\s\S]*?'''`),
        "js":     regexp.MustCompile(`//.*$|/\*[\s\S]*?\*/`),
        "sh":     regexp.MustCompile(`#.*$`),
    }
    
    // Import patterns for various languages
    ImportPatterns = []*regexp.Regexp{
        regexp.MustCompile(`^use\s+`),           // Rust
        regexp.MustCompile(`^import\s+`),        // Python/JS
        regexp.MustCompile(`^from\s+`),          // Python
        regexp.MustCompile(`^require\(`),        // JS
        regexp.MustCompile(`#include\s*<`),      // C/C++
    }
    
    // Function/class signature patterns
    SignaturePatterns = []*regexp.Regexp{
        regexp.MustCompile(`^(pub\s+)?(async\s+)?fn\s+\w+`),           // Rust
        regexp.MustCompile(`^(pub\s+)?(async\s+)?fn\s+\w+`),           // Rust
        regexp.MustCompile(`^def\s+\w+`),                              // Python
        regexp.MustCompile(`^function\s+\w+`),                         // JS
        regexp.MustCompile(`^func\s+\w+`),                             // Go
        regexp.MustCompile(`^(pub\s+)?class\s+\w+`),                   // Various
        regexp.MustCompile(`^(pub\s+)?struct\s+\w+`),                  // Rust
        regexp.MustCompile(`^(pub\s+)?enum\s+\w+`),                    // Rust
        regexp.MustCompile(`^(pub\s+)?trait\s+\w+`),                   // Rust
        regexp.MustCompile(`^interface\s+\w+`),                        // TypeScript
        regexp.MustCompile(`^type\s+\w+`),                             // Go/TypeScript
    }
)
```

### 2.5 Log Aggregator

**File:** `internal/filter/aggregator.go`

```go
package filter

import (
    "strings"
)

// Deduplicate identical lines and append count
// Example: 10 lines of "error: connection refused" 
//          → "error: connection refused (x10)"
type LogAggregator struct{}

func (a *LogAggregator) Aggregate(output string) string {
    lines := strings.Split(output, "\n")
    if len(lines) == 0 {
        return output
    }
    
    type lineCount struct {
        line  string
        count int
    }
    
    // Track consecutive duplicates
    var result []string
    var prev string
    count := 1
    
    for i, line := range lines {
        if line == prev {
            count++
        } else {
            if prev != "" {
                if count > 1 {
                    result = append(result, prev+" (x"+strconv.Itoa(count)+")")
                } else {
                    result = append(result, prev)
                }
            }
            prev = line
            count = 1
        }
        
        // Handle last line
        if i == len(lines)-1 {
            if count > 1 {
                result = append(result, line+" (x"+strconv.Itoa(count)+")")
            } else {
                result = append(result, line)
            }
        }
    }
    
    return strings.Join(result, "\n")
}
```

---

## Phase 3: Git Command Handlers

### 3.1 Git Status Handler

**File:** `internal/commands/git.go`

```go
package commands

import (
    "bytes"
    "fmt"
    "os/exec"
    "strings"
    
    "github.com/fatih/color"
)

// run_status from RTK git.rs
// Uses git status --porcelain -b for parsing
// Formats custom ASCII summary with emoji

type GitHandler struct{}

type GitStatus struct {
    Branch     string
    Ahead      int
    Behind     int
    Staged     []string
    Modified   []string
    Untracked  []string
    Conflicted []string
}

func (h *GitHandler) RunStatus() (string, error) {
    // Get porcelain output for parsing
    cmd := exec.Command("git", "status", "--porcelain", "-b")
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    if err != nil {
        return "", err
    }
    
    status := h.parsePorcelain(out.String())
    return h.formatStatus(status), nil
}

func (h *GitHandler) parsePorcelain(output string) *GitStatus {
    status := &GitStatus{}
    lines := strings.Split(strings.TrimSpace(output), "\n")
    
    for i, line := range lines {
        if i == 0 {
            // Branch line: "## main...origin/main [ahead 2, behind 1]"
            status.Branch = h.parseBranchLine(line)
            continue
        }
        
        if len(line) < 2 {
            continue
        }
        
        code := line[:2]
        file := strings.TrimSpace(line[3:])
        
        switch {
        case strings.Contains(code, "U"):
            status.Conflicted = append(status.Conflicted, file)
        case strings.Contains(code, "M"), strings.Contains(code, "A"), strings.Contains(code, "D"):
            if code[0] != ' ' {
                status.Staged = append(status.Staged, file)
            }
            if code[1] != ' ' {
                status.Modified = append(status.Modified, file)
            }
        case code == "??":
            status.Untracked = append(status.Untracked, file)
        }
    }
    
    return status
}

func (h *GitHandler) parseBranchLine(line string) string {
    // Parse: "## main...origin/main [ahead 2, behind 1]"
    line = strings.TrimPrefix(line, "## ")
    parts := strings.Split(line, "...")
    if len(parts) > 0 {
        return strings.Fields(parts[0])[0]
    }
    return line
}

func (h *GitHandler) formatStatus(status *GitStatus) string {
    var buf strings.Builder
    
    // Branch with emoji
    green := color.New(color.FgGreen).SprintFunc()
    buf.WriteString(green("📌 ") + status.Branch + "\n")
    
    // Staged
    if len(status.Staged) > 0 {
        buf.WriteString(green("✅ Staged:\n"))
        for _, f := range status.Staged {
            buf.WriteString(fmt.Sprintf("   %s\n", f))
        }
    }
    
    // Modified
    if len(status.Modified) > 0 {
        yellow := color.New(color.FgYellow).SprintFunc()
        buf.WriteString(yellow("📝 Modified:\n"))
        for _, f := range status.Modified {
            buf.WriteString(fmt.Sprintf("   %s\n", f))
        }
    }
    
    // Untracked
    if len(status.Untracked) > 0 {
        red := color.New(color.FgRed).SprintFunc()
        buf.WriteString(red("❓ Untracked:\n"))
        for _, f := range status.Untracked {
            buf.WriteString(fmt.Sprintf("   %s\n", f))
        }
    }
    
    // Clean state
    if len(status.Staged) == 0 && len(status.Modified) == 0 && len(status.Untracked) == 0 {
        buf.WriteString(green("✓ Clean working tree\n"))
    }
    
    return buf.String()
}
```

### 3.2 Git Diff Handler

**File:** `internal/commands/git_diff.go`

```go
package commands

import (
    "bytes"
    "os/exec"
    "strings"
)

// run_diff from RTK git.rs
// Strategy: Run git diff --stat first, then append compact diff
// Limit hunk lines to 30 (from RTK L280)

func (h *GitHandler) RunDiff(args []string) (string, error) {
    // Get stat first
    statCmd := exec.Command("git", "diff", "--stat")
    var statOut bytes.Buffer
    statCmd.Stdout = &statOut
    statCmd.Run()
    
    // Get compact diff
    diffCmd := exec.Command("git", "diff")
    if len(args) > 0 {
        diffCmd.Args = append(diffCmd.Args, args...)
    }
    var diffOut bytes.Buffer
    diffCmd.Stdout = &diffOut
    diffCmd.Run()
    
    compact := h.compactDiff(diffOut.String())
    
    return statOut.String() + "\n" + compact, nil
}

func (h *GitHandler) compactDiff(diff string) string {
    lines := strings.Split(diff, "\n")
    var result []string
    hunkLineCount := 0
    inHunk := false
    maxHunkLines := 30
    
    for _, line := range lines {
        if strings.HasPrefix(line, "@@") {
            inHunk = true
            hunkLineCount = 0
            result = append(result, line)
            continue
        }
        
        if inHunk {
            hunkLineCount++
            if hunkLineCount <= maxHunkLines {
                result = append(result, line)
            } else if hunkLineCount == maxHunkLines+1 {
                result = append(result, "... (truncated)")
            }
        } else {
            result = append(result, line)
        }
    }
    
    return strings.Join(result, "\n")
}
```

---

## Phase 4: Cargo/Test Aggregation

### 4.1 Double-Dash Restoration

**File:** `internal/commands/cargo.go`

```go
package commands

import (
    "os"
    "strings"
)

// Critical logic from RTK cargo_cmd.rs
// Clap strips the double-dash, so we need to restore it
// Example: cargo test -- --nocapture
//          Clap sees: cargo test --nocapture
//          We need to find and restore the --

func RestoreDoubleDash(args []string, originalArgs []string) []string {
    // Find position of -- in original os.Args
    dashPos := -1
    for i, arg := range originalArgs {
        if arg == "--" {
            dashPos = i
            break
        }
    }
    
    if dashPos == -1 {
        return args
    }
    
    // Rebuild args with -- in correct position
    // This preserves commands like: cargo test -- --nocapture
    result := make([]string, 0)
    result = append(result, args[:dashPos]...)
    result = append(result, "--")
    result = append(result, args[dashPos:]...)
    
    return result
}

// Filter cargo test output
// Aggregates multiple test suites into single line unless failures
// Example output: "✓ 137 passed (4 suites, 1.45s)"
func (h *CargoHandler) FilterTestOutput(output string) string {
    lines := strings.Split(output, "\n")
    
    type TestSuite struct {
        Passed  int
        Failed  int
        Ignored int
        Time    float64
    }
    
    suites := []TestSuite{}
    currentSuite := TestSuite{}
    hasFailures := false
    
    // Parse test output (state machine from RTK cargo_cmd.rs)
    for _, line := range lines {
        // Match: "test result: ok. 137 passed; 0 failed; 0 ignored; 0 measured"
        if strings.Contains(line, "test result:") {
            // Parse counts
            if strings.Contains(line, "passed") {
                // Extract numbers
                // Update currentSuite
            }
            if strings.Contains(line, "FAILED") {
                hasFailures = true
                currentSuite.Failed++
            }
            suites = append(suites, currentSuite)
            currentSuite = TestSuite{}
        }
    }
    
    // If failures, show full output
    if hasFailures {
        return output
    }
    
    // Aggregate all suites
    totalPassed := 0
    totalTime := 0.0
    for _, s := range suites {
        totalPassed += s.Passed
        totalTime += s.Time
    }
    
    return fmt.Sprintf("✓ %d passed (%d suites, %.2fs)", 
        totalPassed, len(suites), totalTime)
}
```

---

## Phase 5: LS Handler

### 5.1 Noise Directory Filtering

**File:** `internal/commands/ls.go`

```go
package commands

import (
    "bytes"
    "fmt"
    "os/exec"
    "sort"
    "strings"
    
    "github.com/fatih/color"
)

// From RTK ls.rs
// Default noise dirs: .git, node_modules, target, __pycache__, .venv, vendor

type LSHandler struct {
    noiseDirs map[string]bool
}

func NewLSHandler(noiseDirs []string) *LSHandler {
    noise := make(map[string]bool)
    for _, dir := range noiseDirs {
        noise[dir] = true
    }
    return &LSHandler{noiseDirs: noise}
}

func (h *LSHandler) Run() (string, error) {
    cmd := exec.Command("ls", "-la")
    var out bytes.Buffer
    cmd.Stdout = &out
    err := cmd.Run()
    if err != nil {
        return "", err
    }
    
    return h.filterOutput(out.String()), nil
}

func (h *LSHandler) filterOutput(output string) string {
    lines := strings.Split(output, "\n")
    
    var dirs []string
    var files []string
    
    for _, line := range lines {
        if len(line) == 0 {
            continue
        }
        
        // Parse ls -la output
        fields := strings.Fields(line)
        if len(fields) < 9 {
            continue
        }
        
        name := fields[8]
        
        // Skip noise directories
        if h.noiseDirs[name] {
            continue
        }
        
        // Group by type
        if strings.HasPrefix(fields[0], "d") {
            size := h.humanSize(fields[4])
            dirs = append(dirs, fmt.Sprintf("📁 %s (%s)", name, size))
        } else {
            size := h.humanSize(fields[4])
            files = append(files, fmt.Sprintf("📄 %s (%s)", name, size))
        }
    }
    
    // Sort and combine
    sort.Strings(dirs)
    sort.Strings(files)
    
    result := append(dirs, files...)
    return strings.Join(result, "\n")
}

func (h *LSHandler) humanSize(bytes string) string {
    // Convert to human readable: 1024 → 1K, 1048576 → 1M
    // Implementation from RTK ls.rs L137
    // ... conversion logic
    return bytes
}
```

---

## Phase 6: Shell Hooks & Command Rewriting

### 6.1 Command Registry

**File:** `internal/discover/registry.go`

```go
package discover

// From RTK discover/registry.rs
// Maps original commands to TokMan wrappers

type CommandMapping struct {
    Original   string
    TokManCmd  string
    Enabled    bool
}

var Registry = map[string]CommandMapping{
    "git status":    {Original: "git status", TokManCmd: "tokman git status", Enabled: true},
    "git diff":      {Original: "git diff", TokManCmd: "tokman git diff", Enabled: true},
    "git log":       {Original: "git log", TokManCmd: "tokman git log", Enabled: true},
    "ls":            {Original: "ls", TokManCmd: "tokman ls", Enabled: true},
    "ls -la":        {Original: "ls -la", TokManCmd: "tokman ls", Enabled: true},
    "cargo test":    {Original: "cargo test", TokManCmd: "tokman cargo test", Enabled: true},
}

func Rewrite(command string) string {
    // Check registry for exact match
    if mapping, ok := Registry[command]; ok && mapping.Enabled {
        return mapping.TokManCmd
    }
    
    // Check for prefix match
    for original, mapping := range Registry {
        if strings.HasPrefix(command, original) && mapping.Enabled {
            return strings.Replace(command, original, mapping.TokManCmd, 1)
        }
    }
    
    return command // No rewrite
}
```

### 6.2 Shell Hook Script

**File:** `hooks/tokman-rewrite.sh`

```bash
#!/bin/bash
# TokMan command rewriter for Claude Code integration
# Intercepts tool_input JSON and rewrites commands

_TOKMAN_BIN="tokman"

# Read JSON from stdin
read -r JSON_INPUT

# Extract command using jq
COMMAND=$(echo "$JSON_INPUT" | jq -r '.command // empty')

if [ -z "$COMMAND" ]; then
    # No command field, return as-is
    echo "$JSON_INPUT"
    exit 0
fi

# Ask TokMan for rewrite
REWRITTEN=$("$_TOKMAN_BIN" rewrite "$COMMAND" 2>/dev/null)

if [ -n "$REWRITTEN" ] && [ "$REWRITTEN" != "$COMMAND" ]; then
    # Update JSON with rewritten command
    echo "$JSON_INPUT" | jq --arg cmd "$REWRITTEN" '.command = $cmd'
else
    # No rewrite, return original
    echo "$JSON_INPUT"
fi
```

---

## Phase 7: CLI Commands

### 7.1 Root Command

**File:** `internal/commands/root.go`

```go
package commands

import (
    "fmt"
    "os"
    
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    cfgFile string
    verbose bool
    dryRun  bool
)

var rootCmd = &cobra.Command{
    Use:   "tokman",
    Short: "Token-aware CLI proxy",
    Long: `TokMan intercepts CLI commands and filters verbose output
to reduce token usage in LLM interactions.`,
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}

func init() {
    cobra.OnInitialize(initConfig)
    
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", 
        "config file (default is ~/.config/tokman/config.toml)")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, 
        "verbose output")
    rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, 
        "show what would be filtered without executing")
    
    viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

func initConfig() {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        viper.AddConfigPath("$HOME/.config/tokman")
        viper.SetConfigName("config")
        viper.SetConfigType("toml")
    }
    
    viper.AutomaticEnv()
    viper.SetEnvPrefix("TOKMAN")
    
    viper.ReadInConfig()
}
```

### 7.2 Init Command

**File:** `internal/commands/init.go`

```go
package commands

import (
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/spf13/cobra"
    
    "tokman/internal/config"
    "tokman/internal/tracking"
)

var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize TokMan",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("🌸 Initializing TokMan...")
        
        // Create directories
        dirs := []string{
            filepath.Dir(config.ConfigPath()),
            config.DataPath(),
        }
        
        for _, dir := range dirs {
            if err := os.MkdirAll(dir, 0755); err != nil {
                fmt.Fprintf(os.Stderr, "Error creating directory: %v\n", err)
                os.Exit(1)
            }
        }
        
        // Initialize database
        tracker, err := tracking.NewTracker(config.DatabasePath())
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error initializing database: %v\n", err)
            os.Exit(1)
        }
        defer tracker.Close()
        
        // Create default config
        if _, err := os.Stat(config.ConfigPath()); os.IsNotExist(err) {
            // Write default config file
            // ...
        }
        
        fmt.Println("✓ Database initialized at:", config.DatabasePath())
        fmt.Println("✓ Config created at:", config.ConfigPath())
        fmt.Println("\nTo enable shell hooks, add to your .bashrc:")
        fmt.Println("  source ~/.local/share/tokman/hooks/tokman-rewrite.sh")
    },
}
```

### 7.3 Status Command

**File:** `internal/commands/status.go`

```go
package commands

import (
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/spf13/cobra"
    "github.com/fatih/color"
    
    "tokman/internal/config"
    "tokman/internal/tracking"
)

var statusCmd = &cobra.Command{
    Use:   "status",
    Short: "Show token savings summary",
    Run: func(cmd *cobra.Command, args []string) {
        tracker, err := tracking.NewTracker(config.DatabasePath())
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        defer tracker.Close()
        
        cwd, _ := os.Getwd()
        canonicalPath, _ := filepath.EvalSymlinks(cwd)
        
        summary, err := tracker.GetSavings(canonicalPath)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        
        green := color.New(color.FgGreen).SprintFunc()
        cyan := color.New(color.FgCyan).SprintFunc()
        
        fmt.Printf("\n%s\n", green("TokMan Status"))
        fmt.Println(strings.Repeat("─", 40))
        fmt.Printf("Commands executed: %s\n", cyan(summary.TotalCommands))
        fmt.Printf("Tokens saved:      %s\n", green(fmt.Sprintf("%d", summary.TotalSaved)))
        fmt.Printf("Reduction:         %.1f%%\n", 
            float64(summary.TotalSaved)/float64(summary.TotalOriginal)*100)
    },
}
```

---

## Dependencies Summary

| Purpose | Go Package | Version |
|---------|-----------|---------|
| CLI Framework | `github.com/spf13/cobra` | v1.8.0+ |
| Configuration | `github.com/spf13/viper` | v1.18.0+ |
| SQLite | `modernc.org/sqlite` | v1.28.0+ |
| TOML Parsing | `github.com/BurntSushi/toml` | v1.3.2+ |
| Terminal Colors | `github.com/fatih/color` | v1.16.0+ |
| JSON Processing | `encoding/json` | Standard library |
| Logging | `log/slog` | Standard library (Go 1.21+) |
| Testing | `testing` + `github.com/stretchr/testify` | v1.8.0+ |

---

## Milestones

### Milestone 1: Core Infrastructure (Week 1-2)
- [ ] Initialize Go module & dependencies
- [ ] Create directory structure
- [ ] Implement config system (TOML + defaults)
- [ ] SQLite migrations & tracker
- [ ] Logging system
- [ ] `init` command

### Milestone 2: Filtering Engine (Week 3-4)
- [ ] ANSI stripping
- [ ] Comment filter (multi-language)
- [ ] Import filter
- [ ] Brace depth tracker (aggressive mode)
- [ ] Log aggregator
- [ ] Filter engine integration

### Milestone 3: Git Commands (Week 5-6)
- [ ] Git status with porcelain parser
- [ ] Git diff with hunk limiting
- [ ] Git log filtering
- [ ] Token tracking per command

### Milestone 4: Additional Handlers (Week 7-8)
- [ ] LS handler with noise filtering
- [ ] Cargo/test aggregation
- [ ] Double-dash restoration
- [ ] Fallback executor

### Milestone 5: Shell Integration (Week 9-10)
- [ ] Command registry
- [ ] Rewrite logic
- [ ] Shell hook script
- [ ] Hook installation command

### Milestone 6: Polish & Release (Week 11-12)
- [ ] Comprehensive tests (80%+ coverage)
- [ ] Performance benchmarks
- [ ] Documentation
- [ ] Release binaries (Linux, macOS, Windows)

---

## Testing Strategy

### Unit Tests
- Filter functions with various input samples
- Token estimation accuracy
- Brace depth tracking edge cases
- Porcelain parser with fixture data

### Integration Tests
- End-to-end command execution
- Database persistence
- Config reload scenarios

### Benchmarks
```go
func BenchmarkFilterEngine(b *testing.B) {
    engine := filter.NewEngine(filter.ModeMinimal)
    input := largeTestOutput
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        engine.Process(input)
    }
}
```

Target: < 10ms per command filtering overhead

---

## Notes & Considerations

1. **Canonical Paths**: Always resolve symlinks for project path matching (as RTK does)
2. **Session IDs**: Generate per-shell session for tracking
3. **Fallback Handling**: Unknown commands should execute raw and log as parse failure
4. **Error Recovery**: Never block user's command, always return something
5. **Cross-Platform**: Use `path/filepath` for Windows compatibility

---

## File Paths (XDG Specification)

| Resource | Path |
|----------|------|
| Config | `~/.config/tokman/config.toml` |
| Database | `~/.local/share/tokman/history.db` |
| Logs | `~/.local/share/tokman/tokman.log` |
| Hooks | `~/.local/share/tokman/hooks/` |

Override with environment variables:
- `XDG_CONFIG_HOME`
- `XDG_DATA_HOME`
- `TOKMAN_DATABASE_PATH`
- `TOKMAN_CONFIG_PATH`

---

**End of Implementation Plan**
