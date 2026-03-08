package commands

import (
	"testing"
)

func TestDetectSmartLanguage(t *testing.T) {
	tests := []struct {
		filePath string
		expected Language
	}{
		{"main.go", LangGo},
		{"lib.rs", LangRust},
		{"script.py", LangPython},
		{"index.js", LangJavaScript},
		{"app.ts", LangTypeScript},
		{"component.tsx", LangTypeScript},
		{"main.c", LangC},
		{"utils.cpp", LangCpp},
		{"App.java", LangJava},
		{"gem.rb", LangRuby},
		{"run.sh", LangShell},
		{"unknown.xyz", LangUnknown},
		{"noextension", LangUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.filePath, func(t *testing.T) {
			result := detectSmartLanguage(tt.filePath)
			if result != tt.expected {
				t.Errorf("detectSmartLanguage(%q) = %q, want %q", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestExtractFunctions_Go(t *testing.T) {
	content := `package main

func main() {}
func helper() string { return "" }
func (s *Struct) method() {}
`
	functions := extractFunctions(content, LangGo)

	if len(functions) < 2 {
		t.Errorf("expected at least 2 functions, got %d: %v", len(functions), functions)
	}

	// main should be filtered out
	for _, fn := range functions {
		if fn == "main" {
			t.Error("main should be filtered out from function list")
		}
	}
}

func TestExtractFunctions_Rust(t *testing.T) {
	content := `fn main() {}
pub fn public_func() {}
async fn async_func() {}
`
	functions := extractFunctions(content, LangRust)

	// Should exclude main
	for _, fn := range functions {
		if fn == "main" {
			t.Error("main should be filtered out from function list")
		}
	}
}

func TestExtractFunctions_Python(t *testing.T) {
	content := `def main():
    pass

def helper():
    pass

def test_something():
    pass
`
	functions := extractFunctions(content, LangPython)

	// test_ functions should be filtered out
	for _, fn := range functions {
		if fn == "test_something" {
			t.Error("test_ functions should be filtered out")
		}
	}
}

func TestExtractStructs_Go(t *testing.T) {
	content := `type Config struct {
    Name string
}

type Server struct {
    Port int
}
`
	structs := extractStructs(content, LangGo)

	if len(structs) != 2 {
		t.Errorf("expected 2 structs, got %d: %v", len(structs), structs)
	}
}

func TestExtractStructs_Rust(t *testing.T) {
	content := `pub struct Config {
    name: String,
}

enum Status {
    Active,
    Inactive,
}
`
	structs := extractStructs(content, LangRust)

	if len(structs) != 2 {
		t.Errorf("expected 2 structs/enums, got %d: %v", len(structs), structs)
	}
}

func TestExtractStructs_TypeScript(t *testing.T) {
	content := `interface User {
    name: string;
}

class Service {
    run() {}
}

type ID = string;
`
	structs := extractStructs(content, LangTypeScript)

	if len(structs) != 3 {
		t.Errorf("expected 3 interface/class/type, got %d: %v", len(structs), structs)
	}
}

func TestExtractImports_Go(t *testing.T) {
	content := `package main

import (
    "fmt"
    "github.com/spf13/cobra"
    "github.com/GrayCodeAI/tokman/internal/config"
)
`
	imports := extractImports(content, LangGo)

	if len(imports) == 0 {
		t.Error("expected imports to be extracted")
		return
	}

	// Should contain non-std imports
	found := false
	for _, imp := range imports {
		if imp == "github.com" {
			found = true
			break
		}
	}
	if !found {
		t.Logf("imports: %v", imports)
	}
}

func TestExtractImports_Rust(t *testing.T) {
	content := `use std::collections::HashMap;
use serde::Deserialize;
use tokio::runtime;
`
	imports := extractImports(content, LangRust)

	if len(imports) == 0 {
		t.Error("expected imports to be extracted")
	}
}

func TestDetectPatterns(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		lang     Language
		contains string
	}{
		{
			name:     "async pattern",
			content:  "async fn run() { await something(); }",
			lang:     LangRust,
			contains: "async",
		},
		{
			name:     "derive pattern",
			content:  "#[derive(Debug, Clone)]\nstruct Config {}",
			lang:     LangRust,
			contains: "derive",
		},
		{
			name:     "React hooks",
			content:  "const [state, setState] = useState(0);\nuseEffect(() => {}, []);",
			lang:     LangTypeScript,
			contains: "React hooks",
		},
		{
			name:     "Python dataclass",
			content:  "@dataclass\nclass Config:\n    name: str",
			lang:     LangPython,
			contains: "dataclass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := detectPatterns(tt.content, tt.lang)
			found := false
			for _, p := range patterns {
				if p == tt.contains {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected pattern %q in %v", tt.contains, patterns)
			}
		})
	}
}

func TestAnalyzeCode(t *testing.T) {
	content := `package main

import "fmt"

func helper() string {
    return "hello"
}

type Config struct {
    Name string
}
`
	summary := analyzeCode(content, LangGo)

	if summary.line1 == "" {
		t.Error("expected non-empty line1")
	}
	if summary.line2 == "" {
		t.Error("expected non-empty line2")
	}
}
