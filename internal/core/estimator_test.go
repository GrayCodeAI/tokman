package core

import "testing"

func TestEstimateTokensExact(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"a", 1},
		{"ab", 1},
		{"abc", 1},
		{"abcd", 1},
		{"abcde", 2},
		{"abcd ef", 2},
		{"hello world", 3},
		{"abcdefghijklmnop", 4},
	}
	for _, tt := range tests {
		got := EstimateTokens(tt.input)
		if got != tt.expected {
			t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestCalculateTokensSavedPositive(t *testing.T) {
	saved := CalculateTokensSaved("hello world test", "hello")
	if saved <= 0 {
		t.Errorf("expected positive savings, got %d", saved)
	}
}

func TestCalculateTokensSavedZero(t *testing.T) {
	zero := CalculateTokensSaved("hi", "hello world test more")
	if zero != 0 {
		t.Errorf("expected 0 for negative savings, got %d", zero)
	}
}

func TestCalculateTokensSavedEqual(t *testing.T) {
	saved := CalculateTokensSaved("same", "same")
	if saved != 0 {
		t.Errorf("expected 0 for equal length, got %d", saved)
	}
}

func TestEstimateTokensConsistency(t *testing.T) {
	text := "repeated calculation test"
	for i := 0; i < 100; i++ {
		a := EstimateTokens(text)
		b := EstimateTokens(text)
		if a != b {
			t.Fatalf("EstimateTokens not deterministic: %d != %d", a, b)
		}
	}
}
