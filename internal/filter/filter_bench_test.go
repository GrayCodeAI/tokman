package filter

import (
	"strings"
	"testing"
)

func BenchmarkFilterShort(b *testing.B) {
	input := "short line"
	engine := NewEngine(ModeMinimal)
	for i := 0; i < b.N; i++ {
		engine.Process(input)
	}
}

func BenchmarkFilterLong(b *testing.B) {
	input := strings.Repeat("This is a test line with some content\n", 100)
	engine := NewEngine(ModeMinimal)
	for i := 0; i < b.N; i++ {
		engine.Process(input)
	}
}

func BenchmarkFilterGitStatus(b *testing.B) {
	input := "On branch main\n" +
		"Your branch is up to date with 'origin/main'.\n\n" +
		"Changes not staged for commit:\n" +
		"  (use \"git add <file>...\" to update what will be committed)\n" +
		"  (use \"git restore <file>...\" to discard changes in working directory)\n" +
		"	modified:   internal/commands/smart.go\n" +
		"	modified:   internal/utils/utils.go\n\n" +
		"Untracked files:\n" +
		"  (use \"git add <file>...\" to include in what will be committed)\n" +
		"	internal/ccusage/ccusage_test.go\n" +
		"	internal/commands/smart_test.go\n"
	engine := NewEngine(ModeMinimal)
	for i := 0; i < b.N; i++ {
		engine.Process(input)
	}
}

func BenchmarkFilterNpmOutput(b *testing.B) {
	input := "npm WARN deprecated @types/request@2.48.8: request has been deprecated\n" +
		"npm WARN deprecated har-validator@5.1.5: this library is no longer supported\n" +
		"npm WARN deprecated request@2.88.2: request has been deprecated\n\n" +
		"added 1423 packages in 32s\n" +
		"152 packages are looking for funding\n" +
		"  run npm fund for details\n"
	engine := NewEngine(ModeMinimal)
	for i := 0; i < b.N; i++ {
		engine.Process(input)
	}
}

func BenchmarkFilterLargeOutput(b *testing.B) {
	// Simulate a large command output (e.g., npm install with many packages)
	var sb strings.Builder
	for i := 0; i < 1000; i++ {
		sb.WriteString("npm WARN deprecated package@1.0.0: deprecated\n")
		sb.WriteString("added 1 package in 1s\n")
	}
	input := sb.String()
	engine := NewEngine(ModeMinimal)
	for i := 0; i < b.N; i++ {
		engine.Process(input)
	}
}

func BenchmarkFilterAggressive(b *testing.B) {
	input := strings.Repeat("This is a test line with some content\n", 100)
	engine := NewEngine(ModeAggressive)
	for i := 0; i < b.N; i++ {
		engine.Process(input)
	}
}

func BenchmarkEstimateTokens(b *testing.B) {
	input := strings.Repeat("This is a test line with some content\n", 100)
	for i := 0; i < b.N; i++ {
		EstimateTokens(input)
	}
}

func BenchmarkIsCode(b *testing.B) {
	input := "package main\n\nfunc main() {\n    println(\"hello\")\n}\n"
	for i := 0; i < b.N; i++ {
		IsCode(input)
	}
}
