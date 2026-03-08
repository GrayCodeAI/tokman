package filter

import "regexp"

// Language-specific comment patterns
var CommentPatterns = map[string]*regexp.Regexp{
	"go":       regexp.MustCompile(`(?m)//.*$|/\*[\s\S]*?\*/`),
	"rust":     regexp.MustCompile(`(?m)//.*$|/\*[\s\S]*?\*/`),
	"python":   regexp.MustCompile(`(?m)#.*$|"""[\s\S]*?"""|'''[\s\S]*?'''`),
	"javascript": regexp.MustCompile(`(?m)//.*$|/\*[\s\S]*?\*/`),
	"typescript": regexp.MustCompile(`(?m)//.*$|/\*[\s\S]*?\*/`),
	"java":     regexp.MustCompile(`(?m)//.*$|/\*[\s\S]*?\*/`),
	"c":        regexp.MustCompile(`(?m)//.*$|/\*[\s\S]*?\*/`),
	"cpp":      regexp.MustCompile(`(?m)//.*$|/\*[\s\S]*?\*/`),
	"sh":       regexp.MustCompile(`(?m)#.*$`),
	"bash":     regexp.MustCompile(`(?m)#.*$`),
	"ruby":     regexp.MustCompile(`(?m)#.*$|=begin[\s\S]*?=end`),
	"yaml":     regexp.MustCompile(`(?m)#.*$`),
	"toml":     regexp.MustCompile(`(?m)#.*$`),
	"dockerfile": regexp.MustCompile(`(?m)#.*$`),
	"makefile": regexp.MustCompile(`(?m)#.*$`),
}

// Import patterns for various languages (from RTK)
var ImportPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^use\s+`),           // Rust
	regexp.MustCompile(`^import\s+`),        // Python/JS/TS
	regexp.MustCompile(`^from\s+\S+\s+import`), // Python
	regexp.MustCompile(`^require\(`),        // JS (CommonJS)
	regexp.MustCompile(`^import\s*\(`),      // Go (multiple imports)
	regexp.MustCompile(`^import\s+"`),       // Go (single import)
	regexp.MustCompile(`#include\s*<`),      // C/C++
	regexp.MustCompile(`#include\s*"`),      // C/C++
	regexp.MustCompile(`^package\s+`),       // Go/Java
}

// Function/class signature patterns for aggressive filtering
var SignaturePatterns = []*regexp.Regexp{
	// Rust
	regexp.MustCompile(`^(pub\s+)?(async\s+)?fn\s+\w+`),
	regexp.MustCompile(`^(pub\s+)?struct\s+\w+`),
	regexp.MustCompile(`^(pub\s+)?enum\s+\w+`),
	regexp.MustCompile(`^(pub\s+)?trait\s+\w+`),
	regexp.MustCompile(`^(pub\s+)?type\s+\w+`),
	regexp.MustCompile(`^impl\s+`),
	
	// Go
	regexp.MustCompile(`^func\s+(\([^)]+\)\s+)?\w+`),
	regexp.MustCompile(`^type\s+\w+\s+(struct|interface)`),
	regexp.MustCompile(`^type\s+\w+\s+\w+`), // type aliases
	
	// Python
	regexp.MustCompile(`^def\s+\w+`),
	regexp.MustCompile(`^async\s+def\s+\w+`),
	regexp.MustCompile(`^class\s+\w+`),
	
	// JavaScript/TypeScript
	regexp.MustCompile(`^function\s+\w+`),
	regexp.MustCompile(`^(export\s+)?(async\s+)?function\s*\w*`),
	regexp.MustCompile(`^(export\s+)?(default\s+)?class\s+\w+`),
	regexp.MustCompile(`^(export\s+)?const\s+\w+\s*=\s*(async\s+)?\([^)]*\)\s*=>`),
	regexp.MustCompile(`^interface\s+\w+`),
	regexp.MustCompile(`^type\s+\w+\s*=`),
	
	// Java
	regexp.MustCompile(`^(public|private|protected)?\s*(static\s+)?(class|interface|enum)\s+\w+`),
	regexp.MustCompile(`^(public|private|protected)?\s*(static\s+)?(async\s+)?\w+\s+\w+\s*\(`),
}

// Block delimiters for brace tracking
var BlockDelimiters = map[rune]rune{
	'{': '}',
	'[': ']',
	'(': ')',
}

// Git porcelain status codes (from RTK git.rs)
var GitStatusCodes = map[string]string{
	"M ": "staged_modified",
	" M": "modified",
	"A ": "staged_new",
	"D ": "staged_deleted",
	" D": "deleted",
	"R ": "staged_renamed",
	"C ": "staged_copied",
	"??": "untracked",
	"!!": "ignored",
	"UU": "unmerged_both_modified",
	"AA": "unmerged_both_added",
	"DD": "unmerged_both_deleted",
}

// Test result patterns (from RTK cargo_cmd.rs)
var TestResultPatterns = []*regexp.Regexp{
	// Rust/Cargo
	regexp.MustCompile(`test result: (ok|FAILED|ignored)\.`),
	regexp.MustCompile(`(\d+) passed`),
	regexp.MustCompile(`(\d+) failed`),
	regexp.MustCompile(`(\d+) ignored`),
	// Go
	regexp.MustCompile(`PASS`),
	regexp.MustCompile(`FAIL`),
	regexp.MustCompile(`ok\s+\S+\s+[\d.]+s`),
	// Python pytest
	regexp.MustCompile(`(\d+) passed`),
	regexp.MustCompile(`(\d+) failed`),
	regexp.MustCompile(`(\d+) skipped`),
}

// Diff hunk pattern
var DiffHunkPattern = regexp.MustCompile(`^@@\s+-\d+(?:,\d+)?\s+\+\d+(?:,\d+)?\s+@@`)

// Log timestamp patterns
var LogTimestampPatterns = []*regexp.Regexp{
	regexp.MustCompile(`^\d{4}-\d{2}-\d{2}[T\s]\d{2}:\d{2}:\d{2}`), // ISO format
	regexp.MustCompile(`^\[\d{4}-\d{2}-\d{2}`),                     // Bracket format
	regexp.MustCompile(`^\d{2}:\d{2}:\d{2}`),                       // Time only
}
