package filter

import (
	"strings"
	"testing"
)

func TestLanguageDetection(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedLang Language
	}{
		{
			name:         "Go code",
			input:        "package main\nfunc main() { println(\"hello\") }",
			expectedLang: LangGo,
		},
		{
			name:         "Rust code",
			input:        "fn main() { println!(\"hello\"); }",
			expectedLang: LangRust,
		},
		{
			name:         "Python code",
			input:        "def main():\n    print('hello')",
			expectedLang: LangPython,
		},
		{
			name:         "JavaScript code",
			input:        "function main() { console.log('hello'); }",
			expectedLang: LangJavaScript,
		},
		{
			name:         "TypeScript code",
			input:        "function main(): void { console.log('hello'); }",
			expectedLang: LangTypeScript,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectLanguageFromInput(tt.input)
			if result != tt.expectedLang {
				t.Errorf("DetectLanguageFromInput() = %v, want %v", result, tt.expectedLang)
			}
		})
	}
}

func TestCommentPatternsForLang(t *testing.T) {
	tests := []struct {
		name          string
		lang          Language
		expectedLine  string
		expectedBlock string
	}{
		{
			name:          "Rust",
			lang:          LangRust,
			expectedLine:  "//",
			expectedBlock: "/*",
		},
		{
			name:          "Python",
			lang:          LangPython,
			expectedLine:  "#",
			expectedBlock: `"""`,
		},
		{
			name:          "JavaScript",
			lang:          LangJavaScript,
			expectedLine:  "//",
			expectedBlock: "/*",
		},
		{
			name:          "Go",
			lang:          LangGo,
			expectedLine:  "//",
			expectedBlock: "/*",
		},
		{
			name:          "Ruby",
			lang:          LangRuby,
			expectedLine:  "#",
			expectedBlock: "=begin",
		},
		{
			name:          "Shell",
			lang:          LangShell,
			expectedLine:  "#",
			expectedBlock: "",
		},
		{
			name:          "SQL",
			lang:          LangSQL,
			expectedLine:  "--",
			expectedBlock: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patterns := CommentPatternsForLang(tt.lang)
			if patterns.Line != tt.expectedLine {
				t.Errorf("Line pattern = %v, want %v", patterns.Line, tt.expectedLine)
			}
			if patterns.BlockStart != tt.expectedBlock {
				t.Errorf("BlockStart pattern = %v, want %v", patterns.BlockStart, tt.expectedBlock)
			}
		})
	}
}

func TestCommentFilter_Rust(t *testing.T) {
	input := `// This is a comment
fn main() {
    println!("Hello, world!");
}
/* Block comment */
fn other() {
    // Another comment
}`

	filter := NewCommentFilter()
	result, _ := filter.Apply(input, ModeMinimal)

	// Check that single-line comments are removed
	if strings.Contains(result, "// This is a comment") {
		t.Errorf("CommentFilter should remove single-line comments: %s", result)
	}

	// Check that block comments are removed
	if strings.Contains(result, "/* Block comment */") {
		t.Errorf("CommentFilter should remove block comments: %s", result)
	}

	// Check that code is preserved (not stripped)
	if !strings.Contains(result, "fn main()") {
		t.Errorf("CommentFilter should keep function declarations: %s", result)
	}
}

func TestCommentFilter_Python(t *testing.T) {
	input := `# This is a comment
def main():
    print("Hello, world!")
"""
Docstring
"""
def other():
    # Another comment
    pass`

	filter := NewCommentFilter()
	result, _ := filter.Apply(input, ModeMinimal)

	// Check that comments are removed
	if strings.Contains(result, "# This is a comment") {
		t.Errorf("CommentFilter should remove Python comments: %s", result)
	}

	// Check that code is preserved
	if !strings.Contains(result, "def main():") {
		t.Errorf("CommentFilter should keep Python code: %s", result)
	}
}

func TestCommentFilter_JavaScript(t *testing.T) {
	input := `// This is a comment
function main() {
    console.log("Hello, world!");
}
/* Block comment */
function other() {
    // Another comment
}`

	filter := NewCommentFilter()
	result, _ := filter.Apply(input, ModeMinimal)

	// Check that comments are removed
	if strings.Contains(result, "// This is a comment") {
		t.Errorf("CommentFilter should remove comments: %s", result)
	}
	if strings.Contains(result, "/* Block comment */") {
		t.Errorf("CommentFilter should remove block comments: %s", result)
	}

	// Check that code is preserved
	if !strings.Contains(result, "function main()") {
		t.Errorf("CommentFilter should keep code: %s", result)
	}
}

func TestCommentFilter_Go(t *testing.T) {
	input := `// This is a comment
package main

func main() {
    fmt.Println("Hello, world!")
}

/* Block comment */
func other() {
    // Another comment
}`

	filter := NewCommentFilter()
	result, _ := filter.Apply(input, ModeMinimal)

	// Check that comments are removed
	if strings.Contains(result, "// This is a comment") {
		t.Errorf("CommentFilter should remove comments: %s", result)
	}
	if strings.Contains(result, "/* Block comment */") {
		t.Errorf("CommentFilter should remove block comments: %s", result)
	}

	// Check that code is preserved
	if !strings.Contains(result, "func main()") {
		t.Errorf("CommentFilter should keep code: %s", result)
	}
}

func TestCommentFilter_Ruby(t *testing.T) {
	input := `# This is a comment
def main
  puts "Hello, world!"
end

=begin
Block comment
=end

def other
  # Another comment
end`

	filter := NewCommentFilter()
	result, _ := filter.Apply(input, ModeMinimal)

	// Check that comments are removed
	if strings.Contains(result, "# This is a comment") {
		t.Errorf("CommentFilter should remove comments: %s", result)
	}

	// Check that Ruby code uses # for comments (single line)
	// Ruby block comments =begin/=end should be removed too
}

func TestCommentFilter_Shell(t *testing.T) {
	input := `#!/bin/bash
# This is a comment
echo "Hello, world!"

# Another comment
ls -la`

	filter := NewCommentFilter()
	result, _ := filter.Apply(input, ModeMinimal)

	// Check that comments are removed
	if strings.Contains(result, "# This is a comment") {
		t.Errorf("CommentFilter should remove comments: %s", result)
	}

	// Check that code is preserved
	if !strings.Contains(result, "echo") || !strings.Contains(result, "ls") {
		t.Errorf("CommentFilter should keep shell commands: %s", result)
	}
}

func TestCommentFilter_SQL(t *testing.T) {
	input := `-- This is a comment
SELECT * FROM users;

-- Another comment
DELETE FROM users WHERE id = 1;`

	filter := NewCommentFilter()
	result, _ := filter.Apply(input, ModeMinimal)

	// Check that comments are removed
	if strings.Contains(result, "-- This is a comment") {
		t.Errorf("CommentFilter should remove comments: %s", result)
	}

	// Check that SQL code is preserved
	if !strings.Contains(result, "SELECT") || !strings.Contains(result, "DELETE") {
		t.Errorf("CommentFilter should keep SQL: %s", result)
	}
}

func TestCommentFilter_TokenSavings(t *testing.T) {
	input := `// Comment 1
// Comment 2
fn main() {
    println!("Hello");
}
// Comment 3`

	filter := NewCommentFilter()
	result, saved := filter.Apply(input, ModeMinimal)

	// Should save tokens (comments removed)
	if saved <= 0 {
		t.Errorf("CommentFilter should save tokens, got %d", saved)
	}

	// Result should be shorter than input
	if len(result) >= len(input) {
		t.Errorf("Result should be shorter, input: %d, result: %d", len(input), len(result))
	}
}
