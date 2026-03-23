package analysis

import (
	"strings"
	"testing"

	"github.com/GrayCodeAI/tokman/internal/filter"
)

// BenchmarkCommandOverhead measures the overhead of command processing
// Target: <10ms for typical outputs

// BenchmarkGitStatusOverhead measures git status filtering overhead
func BenchmarkGitStatusOverhead(b *testing.B) {
	input := generateGitStatusOutput(50) // 50 files
	engine := filter.NewEngine(filter.ModeMinimal)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Process(input)
	}
}

// BenchmarkGoTestOverhead measures go test output filtering overhead
func BenchmarkGoTestOverhead(b *testing.B) {
	input := generateGoTestOutput(100) // 100 tests
	engine := filter.NewEngine(filter.ModeMinimal)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Process(input)
	}
}

// BenchmarkNpmTestOverhead measures npm test output filtering overhead
func BenchmarkNpmTestOverhead(b *testing.B) {
	input := generateNpmTestOutput(200) // 200 tests
	engine := filter.NewEngine(filter.ModeMinimal)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Process(input)
	}
}

// BenchmarkDockerPsOverhead measures docker ps output filtering overhead
func BenchmarkDockerPsOverhead(b *testing.B) {
	input := generateDockerPsOutput(100) // 100 containers
	engine := filter.NewEngine(filter.ModeMinimal)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Process(input)
	}
}

// BenchmarkKubectlGetOverhead measures kubectl get output filtering overhead
func BenchmarkKubectlGetOverhead(b *testing.B) {
	input := generateKubectlGetOutput(50) // 50 pods
	engine := filter.NewEngine(filter.ModeMinimal)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.Process(input)
	}
}

// BenchmarkLargeOutput measures performance with very large outputs
func BenchmarkLargeOutput(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}
	for _, size := range sizes {
		b.Run(string(rune(size)), func(b *testing.B) {
			input := generateLargeOutput(size)
			engine := filter.NewEngine(filter.ModeMinimal)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				engine.Process(input)
			}
		})
	}
}

// BenchmarkUltraCompactMode measures ultra-compact mode performance
func BenchmarkUltraCompactMode(b *testing.B) {
	input := generateGitStatusOutput(100)
	engine := filter.NewEngine(filter.ModeMinimal)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate ultra-compact processing
		result, _ := engine.Process(input)
		// Additional compact formatting
		lines := strings.Split(result, "\n")
		compactResult := strings.Join(lines[:min(10, len(lines))], "\n")
		_ = compactResult
	}
}

// BenchmarkTokenEstimation measures token estimation performance
func BenchmarkTokenEstimation(b *testing.B) {
	inputs := []string{
		"short line",
		strings.Repeat("medium line with content\n", 10),
		strings.Repeat("long line with lots of content for testing\n", 100),
	}

	for i, input := range inputs {
		b.Run(string(rune('A'+i)), func(b *testing.B) {
			for j := 0; j < b.N; j++ {
				filter.EstimateTokens(input)
			}
		})
	}
}

// BenchmarkMemoryAllocation measures memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	input := generateLargeOutput(1000)
	engine := filter.NewEngine(filter.ModeMinimal)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Process(input)
	}
}

// Helper functions to generate test data

func generateGitStatusOutput(files int) string {
	var sb strings.Builder
	sb.WriteString("On branch main\n")
	sb.WriteString("Your branch is up to date with 'origin/main'.\n\n")
	sb.WriteString("Changes not staged for commit:\n")
	sb.WriteString("  (use \"git add <file>...\" to update what will be committed)\n")
	sb.WriteString("  (use \"git restore <file>...\" to discard changes in working directory)\n")

	for i := 0; i < files; i++ {
		sb.WriteString("\tmodified:   src/file")
		sb.WriteString(strings.Repeat("x", i%10))
		sb.WriteString(".go\n")
	}

	sb.WriteString("\nUntracked files:\n")
	for i := 0; i < files/2; i++ {
		sb.WriteString("\tinternal/new/file")
		sb.WriteString(strings.Repeat("y", i%5))
		sb.WriteString(".go\n")
	}

	return sb.String()
}

func generateGoTestOutput(tests int) string {
	var sb strings.Builder
	sb.WriteString("=== RUN   TestPackage\n")

	for i := 0; i < tests; i++ {
		sb.WriteString("=== RUN   TestPackage/Test")
		sb.WriteString(strings.Repeat("A", i%5+1))
		sb.WriteString("\n")
		if i%10 == 0 {
			sb.WriteString("    file_test.go:")
			sb.WriteString(strings.Repeat("12", i%3+1))
			sb.WriteString(": some test output\n")
		}
		sb.WriteString("--- ")
		if i%20 == 0 {
			sb.WriteString("FAIL")
		} else {
			sb.WriteString("PASS")
		}
		sb.WriteString(": TestPackage/Test")
		sb.WriteString(strings.Repeat("A", i%5+1))
		sb.WriteString(" (0.00s)\n")
	}

	sb.WriteString("--- PASS: TestPackage (")
	sb.WriteString(strings.Repeat("0", 2))
	sb.WriteString(".00s)\n")
	sb.WriteString("PASS\n")

	return sb.String()
}

func generateNpmTestOutput(tests int) string {
	var sb strings.Builder
	sb.WriteString("PASS src/components/Button.test.tsx\n")
	sb.WriteString("PASS src/utils/helpers.test.ts\n")

	for i := 0; i < tests; i++ {
		sb.WriteString("✓ Test")
		sb.WriteString(strings.Repeat("X", i%5+1))
		sb.WriteString(" (")
		if i%10 == 0 {
			sb.WriteString("15")
		} else {
			sb.WriteString("5")
		}
		sb.WriteString(" ms)\n")

		if i%20 == 0 {
			sb.WriteString("  ✕ Failed test")
			sb.WriteString(strings.Repeat("Y", i%3+1))
			sb.WriteString("\n")
			sb.WriteString("  expect(received).toBe(expected)\n")
		}
	}

	sb.WriteString("\nTest Suites: ")
	sb.WriteString(strings.Repeat("1", 2))
	sb.WriteString(" passed, 1 total\n")
	sb.WriteString("Tests:       ")
	sb.WriteString(strings.Repeat("1", 2))
	sb.WriteString(" passed, ")
	sb.WriteString(strings.Repeat("1", 2))
	sb.WriteString(" total\n")

	return sb.String()
}

func generateDockerPsOutput(containers int) string {
	var sb strings.Builder
	sb.WriteString("CONTAINER ID   IMAGE                              COMMAND                  CREATED         STATUS         PORTS                    NAMES\n")

	for i := 0; i < containers; i++ {
		sb.WriteString("abc")
		sb.WriteString(strings.Repeat("1", i%5+1))
		sb.WriteString("def")
		sb.WriteString(strings.Repeat("2", i%5+1))
		sb.WriteString("   myapp-service:")
		sb.WriteString(strings.Repeat("v", i%3+1))
		sb.WriteString("   \"/app/start.sh\"   ")
		sb.WriteString(strings.Repeat("2", 2))
		sb.WriteString(" hours ago     Up ")
		sb.WriteString(strings.Repeat("2", 2))
		sb.WriteString(" hours       0.0.0.0:")
		sb.WriteString(strings.Repeat("3", 4))
		sb.WriteString("->80/tcp   myapp-")
		sb.WriteString(strings.Repeat("x", i%3+1))
		sb.WriteString("\n")
	}

	return sb.String()
}

func generateKubectlGetOutput(pods int) string {
	var sb strings.Builder
	sb.WriteString("NAME                          READY   STATUS    RESTARTS   AGE\n")

	for i := 0; i < pods; i++ {
		sb.WriteString("myapp-")
		sb.WriteString(strings.Repeat("p", i%5+1))
		sb.WriteString("-")
		sb.WriteString(strings.Repeat("a", 5))
		sb.WriteString("   1/1     Running   ")
		if i%10 == 0 {
			sb.WriteString("3")
		} else {
			sb.WriteString("0")
		}
		sb.WriteString("          ")
		sb.WriteString(strings.Repeat("2", 1))
		sb.WriteString("d\n")
	}

	return sb.String()
}

func generateLargeOutput(lines int) string {
	var sb strings.Builder
	for i := 0; i < lines; i++ {
		sb.WriteString("Line ")
		sb.WriteString(strings.Repeat("X", i%10+1))
		sb.WriteString(": ")
		sb.WriteString(strings.Repeat("content ", 5))
		sb.WriteString("\n")
	}
	return sb.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
