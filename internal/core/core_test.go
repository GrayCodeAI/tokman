package core

import (
	"context"
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		input       string
		minExpected int // BPE is more accurate than heuristic
	}{
		{"", 0},
		{"a", 1},
		{"abcd", 1},
		{"abcde", 1},
		{"abcdefgh", 1},
		{"abcdefghi", 1},
		{"hello world", 2},
	}

	for _, tt := range tests {
		got := EstimateTokens(tt.input)
		if got < tt.minExpected {
			t.Errorf("EstimateTokens(%q) = %d, want >= %d", tt.input, got, tt.minExpected)
		}
	}
}

func TestCalculateTokensSaved(t *testing.T) {
	tests := []struct {
		original string
		filtered string
		minSaved int
	}{
		{"hello world", "hello", 1},
		{"same", "same", 0},
		{"short", "longer than original", 0}, // Should return 0 when filtered is longer
		{"a b c d e f g h", "a c e g", 1},
	}

	for _, tt := range tests {
		got := CalculateTokensSaved(tt.original, tt.filtered)
		if got < tt.minSaved {
			t.Errorf("CalculateTokensSaved(%q, %q) = %d, want >= %d",
				tt.original, tt.filtered, got, tt.minSaved)
		}
	}
}

func TestHeuristicEstimator(t *testing.T) {
	e := newHeuristicEstimator()

	if e.Encoding() != "heuristic" {
		t.Errorf("Encoding() = %q, want %q", e.Encoding(), "heuristic")
	}

	got := e.Estimate("hello world")
	if got <= 0 {
		t.Errorf("Estimate returned %d, want > 0", got)
	}

	h, a, ratio := e.Compare("test")
	if h <= 0 || a <= 0 {
		t.Errorf("Compare returned invalid values: h=%d, a=%d", h, a)
	}
	if ratio != 0 {
		t.Errorf("HeuristicEstimator ratio should be 0, got %f", ratio)
	}
}

func TestMockCommandRunner(t *testing.T) {
	runner := newMockCommandRunner()
	runner.Outputs["echo"] = "hello"
	runner.ExitCodes["echo"] = 0

	output, exitCode, err := runner.Run(context.Background(), []string{"echo", "test"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if output != "hello" {
		t.Errorf("output = %q, want %q", output, "hello")
	}
	if exitCode != 0 {
		t.Errorf("exitCode = %d, want 0", exitCode)
	}
	if len(runner.Calls) != 1 {
		t.Errorf("expected 1 call, got %d", len(runner.Calls))
	}
}

func TestBufferedOutputWriter(t *testing.T) {
	w := newBufferedOutputWriter()

	w.WriteOutput([]byte("hello "))
	w.WriteOutput([]byte("world"))
	w.WriteDiagnostic([]byte("debug info"))

	if string(w.Output()) != "hello world" {
		t.Errorf("Output = %q, want %q", string(w.Output()), "hello world")
	}
	if string(w.Diagnostic()) != "debug info" {
		t.Errorf("Diagnostic = %q, want %q", string(w.Diagnostic()), "debug info")
	}
}

func TestCalculateSavings(t *testing.T) {
	savings := CalculateSavings(1000000, "gpt-4o")
	if savings <= 0 {
		t.Errorf("CalculateSavings returned %f, want > 0", savings)
	}

	formatted := formatSavings(1000, "gpt-4o")
	if formatted == "" {
		t.Errorf("formatSavings returned empty string")
	}
}
