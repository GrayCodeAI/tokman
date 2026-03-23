package filter

import (
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "plain text",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "ansi color codes",
			input:    "\x1b[32mgreen\x1b[0m text",
			expected: "green text",
		},
		{
			name:     "multiple ansi codes",
			input:    "\x1b[1;31mred bold\x1b[0m \x1b[34mblue\x1b[0m",
			expected: "red bold blue",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("StripANSI() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFilterEngine(t *testing.T) {
	engine := NewEngine(ModeMinimal)

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "ansi codes",
			input: "\x1b[31mred\x1b[0m",
		},
		{
			name:  "plain text",
			input: "hello world",
		},
		{
			name:  "mixed content",
			input: "\x1b[1mbold\x1b[0m text\nwith newlines",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, saved := engine.Process(tt.input)
			if saved < 0 {
				t.Errorf("Process() saved tokens = %d, should be >= 0", saved)
			}
			if result == "" && tt.input != "" {
				t.Errorf("Process() returned empty for non-empty input")
			}
		})
	}
}

func TestFilterMode(t *testing.T) {
	if ModeMinimal != "minimal" {
		t.Errorf("ModeMinimal = %q, want 'minimal'", ModeMinimal)
	}
	if ModeAggressive != "aggressive" {
		t.Errorf("ModeAggressive = %q, want 'aggressive'", ModeAggressive)
	}
	if ModeNone != "none" {
		t.Errorf("ModeNone = %q, want 'none'", ModeNone)
	}
}

func TestIsCode(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"func main() {}", true},
		{"function foo() {}", true},
		{"def hello(): pass", true},
		{"class MyClass {}", true},
		{"struct Point {}", true},
		{"import os", true},
		{"package main", true},
		{"use std::io", true},
		{"require('fs')", true},
		{"pub fn main() {}", true},
		{"pub struct Foo {}", true},
		{"pub async fn run() {}", true},
		{"// comment", true},
		{"/* block */", true},
		{"#!/bin/bash", true},
		{"just plain text", false},
		{"12345", false},
		{"", false},
		{"hello world", false},
	}
	for _, tt := range tests {
		got := IsCode(tt.input)
		if got != tt.expected {
			t.Errorf("IsCode(%q) = %v, want %v", tt.input, got, tt.expected)
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"package main\nfunc foo() {}", "go"},
		{"func helper() int { return 1 }", "go"},
		{"fn main() {}", "rust"},
		{"pub fn run() {}", "rust"},
		{"def hello(): pass", "python"},
		{"import os\ndef main(): pass", "python"},
		{"function foo() {}", "javascript"},
		{"const x = 1", "javascript"},
		{"random text with no code", "unknown"},
		{"", "unknown"},
	}
	for _, tt := range tests {
		got := DetectLanguage(tt.input)
		if got != tt.expected {
			t.Errorf("DetectLanguage(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestDetectLanguageFromInput(t *testing.T) {
	tests := []struct {
		input    string
		expected Language
	}{
		{"package main\nfunc foo() {}", LangGo},
		{"fn main() {}", LangRust},
		{"pub fn run() {}", LangRust},
		{"def hello(): pass", LangPython},
		{"import os\ndef main(): pass", LangPython},
		{"function foo() {}", LangJavaScript},
		{"const x = 1", LangJavaScript},
		{"SELECT * FROM users", LangSQL},
		{"INSERT INTO t VALUES(1)", LangSQL},
		{"UPDATE t SET x=1", LangSQL},
		{"WHERE id = 1", LangSQL},
		{"no code here", LangUnknown},
	}
	for _, tt := range tests {
		got := DetectLanguageFromInput(tt.input)
		if got != tt.expected {
			t.Errorf("DetectLanguageFromInput(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestEngineSetMode(t *testing.T) {
	engine := NewEngine(ModeMinimal)
	engine.SetMode(ModeAggressive)
	// Process should use aggressive mode (includes body filter)
	input := "func main() {\n\t// body\n}\n"
	result, _ := engine.Process(input)
	if result == "" {
		t.Error("Process returned empty result after SetMode")
	}
}

func TestEngineProcessModes(t *testing.T) {
	input := "\x1b[31mred\x1b[0m func main() {}"

	modes := []Mode{ModeNone, ModeMinimal, ModeAggressive}
	for _, mode := range modes {
		engine := NewEngine(mode)
		result, saved := engine.Process(input)
		if result == "" {
			t.Errorf("Process(%q) with mode %q returned empty", input, mode)
		}
		if saved < 0 {
			t.Errorf("Process(%q) with mode %q saved = %d, want >= 0", input, mode, saved)
		}
	}
}

func TestEngineWithQuery(t *testing.T) {
	engine := NewEngineWithQuery(ModeMinimal, "debug")
	input := "error: connection failed\nline1\nline2\n"
	result, saved := engine.Process(input)
	if result == "" {
		t.Error("Process with query intent returned empty")
	}
	if saved < 0 {
		t.Errorf("saved = %d, want >= 0", saved)
	}
}
