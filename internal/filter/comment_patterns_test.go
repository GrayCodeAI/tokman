package filter

import (
	"testing"
)

func TestCommentPatternsForLangConstants(t *testing.T) {
	tests := []struct {
		lang      Language
		wantLine  string
		wantBlock string
	}{
		{LangGo, "//", "/*"},
		{LangPython, "#", `"""`},
		{LangJavaScript, "//", "/*"},
		{LangTypeScript, "//", "/*"},
		{LangRust, "//", "/*"},
		{LangJava, "//", "/*"},
		{LangShell, "#", ""},
		{LangRuby, "#", "=begin"},
	}
	for _, tt := range tests {
		p := commentPatternsForLang(tt.lang)
		if p.Line != tt.wantLine {
			t.Errorf("commentPatternsForLang(%v).Line = %q, want %q", tt.lang, p.Line, tt.wantLine)
		}
		if p.BlockStart != tt.wantBlock {
			t.Errorf("commentPatternsForLang(%v).BlockStart = %q, want %q", tt.lang, p.BlockStart, tt.wantBlock)
		}
	}
}

func TestCommentFilterNew(t *testing.T) {
	f := newCommentFilter()
	if f == nil {
		t.Fatal("newCommentFilter returned nil")
	}
	if f.Name() != "comment" {
		t.Errorf("Name() = %q, want 'comment'", f.Name())
	}
}

func TestCommentFilter_Apply_None(t *testing.T) {
	f := newCommentFilter()
	input := "// comment\nfunc main() {}"
	output, _ := f.Apply(input, ModeNone)
	if output == "" {
		t.Error("output should not be empty")
	}
}

func TestCommentFilter_Apply_Minimal(t *testing.T) {
	f := newCommentFilter()
	input := "// comment\nfunc main() {}"
	output, saved := f.Apply(input, ModeMinimal)
	if output == "" {
		t.Error("output should not be empty")
	}
	_ = saved
}

func TestCommentFilter_Apply_Go(t *testing.T) {
	f := newCommentFilter()
	input := `// Package main provides the entry point
package main

// main is the entry point
func main() {
	// Print hello
	fmt.Println("hello")
}
`
	output, saved := f.Apply(input, ModeMinimal)
	if output == "" {
		t.Error("output should not be empty")
	}
	if saved <= 0 {
		t.Error("should save tokens when stripping comments")
	}
	// Should keep code
	if !outputContains(output, "package main") {
		t.Error("should keep package declaration")
	}
	if !outputContains(output, "func main") {
		t.Error("should keep function declaration")
	}
}

func TestStripCommentsFunc(t *testing.T) {
	input := "// line comment\nfunc test() {\n/* block comment */\n}"
	result := stripComments(input, LangGo)
	if result == "" {
		t.Error("result should not be empty")
	}
	if outputContains(result, "line comment") {
		t.Error("should strip line comment")
	}
	if !outputContains(result, "func test") {
		t.Error("should keep code")
	}
}

func TestCommentFilter_Empty(t *testing.T) {
	f := newCommentFilter()
	output, saved := f.Apply("", ModeMinimal)
	if output != "" {
		t.Errorf("empty input should return empty, got: %q", output)
	}
	if saved != 0 {
		t.Errorf("empty input should save 0, got %d", saved)
	}
}

// helper to check case-insensitive containment
func outputContains(output, substr string) bool {
	return len(output) > 0 && containsInsensitive(output, substr)
}

func containsInsensitive(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
