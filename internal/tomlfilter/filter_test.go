package tomlfilter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

// TestFilterCompilation tests regex compilation for filters
func TestFilterCompilation(t *testing.T) {
	tests := []struct {
		name    string
		filter  TomlFilter
		wantErr bool
	}{
		{
			name: "valid match_command",
			filter: TomlFilter{
				MatchCommand: "^git\\s+status",
			},
			wantErr: false,
		},
		{
			name: "invalid match_command regex",
			filter: TomlFilter{
				MatchCommand: "[invalid(",
			},
			wantErr: true,
		},
		{
			name: "valid strip_lines_matching",
			filter: TomlFilter{
				StripLinesMatching: []string{"^\\s*$", "^INFO:"},
			},
			wantErr: false,
		},
		{
			name: "invalid strip_lines_matching regex",
			filter: TomlFilter{
				StripLinesMatching: []string{"[invalid("},
			},
			wantErr: true,
		},
		{
			name: "complex filter",
			filter: TomlFilter{
				MatchCommand:       "^cargo\\s+test",
				StripANSI:          true,
				StripLinesMatching: []string{"^running \\d+ tests", "^test .* ok$"},
				MaxLines:           50,
				OnEmpty:            "cargo test: ok",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.filter.compileRegex()
			if (err != nil) != tt.wantErr {
				t.Errorf("compileRegex() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFilterMatching tests command matching
func TestFilterMatching(t *testing.T) {
	filter := &TomlFilter{
		MatchCommand: "^git\\s+(status|diff|log)",
	}
	if err := filter.compileRegex(); err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	tests := []struct {
		command string
		matches bool
	}{
		{"git status", true},
		{"git diff", true},
		{"git log --oneline -5", true},
		{"git commit", false},
		{"cargo test", false},
		{"rtk git status", false}, // doesn't match ^git
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			if got := filter.Matches(tt.command); got != tt.matches {
				t.Errorf("Matches(%q) = %v, want %v", tt.command, got, tt.matches)
			}
		})
	}
}

// TestFilterApply tests the Apply function
func TestFilterApply(t *testing.T) {
	tests := []struct {
		name   string
		filter TomlFilter
		input  string
		want   string
	}{
		{
			name: "strip_empty_lines",
			filter: TomlFilter{
				StripLinesMatching: []string{"^\\s*$"},
			},
			input: "line1\n\nline2\n\n\nline3",
			want:  "line1\nline2\nline3",
		},
		{
			name: "strip_ansi",
			filter: TomlFilter{
				StripANSI: true,
			},
			input: "\x1b[32mgreen\x1b[0m text",
			want:  "green text",
		},
		{
			name: "max_lines",
			filter: TomlFilter{
				MaxLines: 2,
			},
			input: "line1\nline2\nline3\nline4",
			want:  "line1\nline2",
		},
		{
			name: "tail_lines",
			filter: TomlFilter{
				TailLines: 2,
			},
			input: "line1\nline2\nline3\nline4",
			want:  "line3\nline4",
		},
		{
			name: "on_empty",
			filter: TomlFilter{
				StripLinesMatching: []string{".*"},
				OnEmpty:            "no output",
			},
			input: "anything",
			want:  "no output",
		},
		{
			name: "keep_lines_matching",
			filter: TomlFilter{
				KeepLinesMatching: []string{"^ERROR:", "^WARN:"},
			},
			input: "INFO: info\nERROR: error\nDEBUG: debug\nWARN: warning",
			want:  "ERROR: error\nWARN: warning",
		},
		{
			name: "truncate_lines_at",
			filter: TomlFilter{
				TruncateLinesAt: 10,
			},
			input: "short\nthis is a very long line that should be truncated\nalso short",
			want:  "short\nthis is a \nalso short", // 10 chars includes trailing space
		},
		{
			name: "combined_filters",
			filter: TomlFilter{
				StripLinesMatching: []string{"^\\s*$", "^INFO:"},
				MaxLines:           3,
			},
			input: "INFO: starting\nline1\n\nline2\n\nline3\nline4",
			want:  "line1\nline2\nline3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.filter.compileRegex(); err != nil {
				t.Fatalf("failed to compile: %v", err)
			}
			got := tt.filter.Apply(tt.input)
			if got != tt.want {
				t.Errorf("Apply() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestReplaceRules tests regex replacement
func TestReplaceRules(t *testing.T) {
	filter := TomlFilter{
		Replace: []ReplaceRule{
			{Pattern: "\\b\\d+\\b", Replacement: "N"},
			{Pattern: "error", Replacement: "ERROR"},
		},
	}
	if err := filter.compileRegex(); err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	input := "Found 123 errors in 456 files"
	want := "Found N ERRORs in N files"
	got := filter.Apply(input)

	if got != want {
		t.Errorf("Apply() = %q, want %q", got, want)
	}
}

// TestMatchOutputRules tests short-circuit match_output rules
func TestMatchOutputRules(t *testing.T) {
	filter := TomlFilter{
		MatchOutput: []MatchRule{
			{Pattern: "(?i)already up to date", Message: "ok: up-to-date"},
			{Pattern: "nothing to commit", Message: "ok: clean"},
		},
	}
	if err := filter.compileRegex(); err != nil {
		t.Fatalf("failed to compile: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"Already up to date.", "ok: up-to-date"},
		{"nothing to commit, working tree clean", "ok: clean"},
		{"some other output", "some other output"},
	}

	for _, tt := range tests {
		got := filter.Apply(tt.input)
		if got != tt.want {
			t.Errorf("Apply(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// TestRegistryLoadFilters tests loading filters from directory
func TestRegistryLoadFilters(t *testing.T) {
	// Create temp directory with test filters
	tmpDir, err := os.MkdirTemp("", "tomlfilter-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Write test filter file
	filterContent := `
[filters.gcc]
description = "Compact gcc output"
match_command = "^g(cc|\\+\\+)\\b"
strip_lines_matching = ["^\\s*$"]
max_lines = 50
on_empty = "gcc: ok"

[filters.git-status]
description = "Compact git status"
match_command = "^git\\s+status"
strip_lines_matching = ["^\\s*$"]
`
	filterPath := filepath.Join(tmpDir, "test.toml")
	if err := os.WriteFile(filterPath, []byte(filterContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Load filters
	registry := NewRegistry()
	if err := registry.LoadFilters(tmpDir); err != nil {
		t.Fatalf("LoadFilters() error = %v", err)
	}

	// Check filters loaded
	if registry.Count() != 2 {
		t.Errorf("Count() = %d, want 2", registry.Count())
	}

	// Test matching
	gccFilter := registry.MatchFilter("gcc main.c -o main")
	if gccFilter == nil {
		t.Error("MatchFilter(gcc) returned nil")
	}

	gitFilter := registry.MatchFilter("git status")
	if gitFilter == nil {
		t.Error("MatchFilter(git status) returned nil")
	}

	// Test non-matching
	noMatch := registry.MatchFilter("cargo build")
	if noMatch != nil {
		t.Error("MatchFilter(cargo build) should return nil")
	}
}

// TestRegistryLoadFromEmbedded tests loading embedded filters
func TestRegistryLoadFromEmbedded(t *testing.T) {
	embedded := `
[filters.make]
description = "Compact make output"
match_command = "^make\\b"
strip_lines_matching = ["^make\\[\\d+\\]:"]

[filters.pnpm]
description = "Compact pnpm output"  
match_command = "^pnpm\\s+(install|add|update)"
strip_lines_matching = ["^Progress:", "^\\s+$"]
`

	registry := NewRegistry()
	if err := registry.LoadFromEmbedded(embedded); err != nil {
		t.Fatalf("LoadFromEmbedded() error = %v", err)
	}

	if registry.Count() != 2 {
		t.Errorf("Count() = %d, want 2", registry.Count())
	}
}

// TestApplyToCommand tests the convenience method
func TestApplyToCommand(t *testing.T) {
	registry := NewRegistry()
	filter := &TomlFilter{
		Name:          "test",
		MatchCommand:  "^echo\\s+test",
		StripANSI:     true,
		MaxLines:      1,
		OnEmpty:       "ok",
	}
	if err := filter.compileRegex(); err != nil {
		t.Fatal(err)
	}
	registry.filters = append(registry.filters, filter)

	tests := []struct {
		command string
		input   string
		want    string
		applied bool
	}{
		{"echo test", "line1\nline2", "line1", true},
		{"echo other", "input", "input", false},
		{"other command", "test", "test", false},
	}

	for _, tt := range tests {
		got, applied := registry.ApplyToCommand(tt.command, tt.input)
		if got != tt.want || applied != tt.applied {
			t.Errorf("ApplyToCommand(%q, %q) = (%q, %v), want (%q, %v)",
				tt.command, tt.input, got, applied, tt.want, tt.applied)
		}
	}
}

// TestInlineFilterTests tests the inline test runner
func TestInlineFilterTests(t *testing.T) {
	filterContent := `
[filters.gcc]
description = "Test filter"
match_command = "^gcc"
strip_lines_matching = ["^INFO:"]

[[tests.gcc]]
name = "strips info lines"
input = "INFO: starting\nERROR: failed"
expected = "ERROR: failed"

[[tests.gcc]]
name = "empty result uses on_empty"
input = "INFO: only"
expected = ""
`

	var filterFile TomlFilterFile
	if _, err := toml.Decode(filterContent, &filterFile); err != nil {
		t.Fatal(err)
	}

	// Check tests loaded
	if len(filterFile.Tests["gcc"]) != 2 {
		t.Errorf("Expected 2 tests, got %d", len(filterFile.Tests["gcc"]))
	}

	// Run tests manually
	filter := filterFile.Filters["gcc"]
	if err := filter.compileRegex(); err != nil {
		t.Fatal(err)
	}

	for _, test := range filterFile.Tests["gcc"] {
		got := filter.Apply(test.Input)
		if got != test.Expected {
			t.Errorf("Test %q: Apply() = %q, want %q", test.Name, got, test.Expected)
		}
	}
}

// BenchmarkFilterApply benchmarks filter application
func BenchmarkFilterApply(b *testing.B) {
	filter := TomlFilter{
		StripLinesMatching: []string{"^\\s*$", "^DEBUG:", "^TRACE:"},
		MaxLines:           100,
		StripANSI:          true,
	}
	if err := filter.compileRegex(); err != nil {
		b.Fatal(err)
	}

	// Generate test input with 1000 lines
	var lines []string
	for i := 0; i < 1000; i++ {
		if i%3 == 0 {
			lines = append(lines, "")
		} else if i%5 == 0 {
			lines = append(lines, "DEBUG: some debug message")
		} else {
			lines = append(lines, "\x1b[32mINFO: processing item "+string(rune(i))+"\x1b[0m")
		}
	}
	input := strings.Join(lines, "\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Apply(input)
	}
}

// BenchmarkRegistryMatchFilter benchmarks command matching
func BenchmarkRegistryMatchFilter(b *testing.B) {
	registry := NewRegistry()

	// Add multiple filters
	filters := []struct {
		name   string
		pattern string
	}{
		{"git-status", "^git\\s+status"},
		{"git-diff", "^git\\s+diff"},
		{"git-log", "^git\\s+log"},
		{"cargo-test", "^cargo\\s+test"},
		{"cargo-build", "^cargo\\s+build"},
		{"npm-test", "^npm\\s+test"},
		{"npm-run", "^npm\\s+run"},
		{"pytest", "^pytest"},
		{"go-test", "^go\\s+test"},
		{"docker", "^docker\\s+ps"},
	}

	for _, f := range filters {
		filter := &TomlFilter{
			Name:         f.name,
			MatchCommand: f.pattern,
		}
		if err := filter.compileRegex(); err != nil {
			b.Fatal(err)
		}
		registry.filters = append(registry.filters, filter)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.MatchFilter("git status --short")
	}
}
