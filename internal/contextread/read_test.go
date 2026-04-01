package contextread

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildAutoModeUsesStructuredSummary(t *testing.T) {
	content := strings.Repeat("func a() {}\n", 500)
	output, original, final, err := Build("main.go", content, Options{Mode: "auto"})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if output == "" {
		t.Fatal("expected non-empty output")
	}
	if final > original {
		t.Fatalf("expected final tokens <= original tokens (%d > %d)", final, original)
	}
}

func TestBuildDeltaModePersistsSnapshot(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	filePath := filepath.Join(t.TempDir(), "main.go")

	first, _, _, err := Build(filePath, "package main\nfunc a() {}\n", Options{
		Mode:         "delta",
		SaveSnapshot: true,
	})
	if err != nil {
		t.Fatalf("Build() first error = %v", err)
	}
	if !strings.Contains(first, "No previous snapshot found") {
		t.Fatalf("expected bootstrap message, got %q", first)
	}

	second, _, _, err := Build(filePath, "package main\nfunc b() {}\n", Options{
		Mode:         "delta",
		SaveSnapshot: true,
	})
	if err != nil {
		t.Fatalf("Build() second error = %v", err)
	}
	if !strings.Contains(second, "Delta:") {
		t.Fatalf("expected delta output, got %q", second)
	}
}

func TestBuildAppliesTokenBudget(t *testing.T) {
	content := strings.Repeat("line with useful text\n", 100)
	output, _, final, err := Build("notes.txt", content, Options{
		Mode:      "full",
		MaxTokens: 20,
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if output == "" {
		t.Fatal("expected output")
	}
	if final <= 0 {
		t.Fatalf("expected positive final token count, got %d", final)
	}
}

func TestBuildGraphModeIncludesRelatedFiles(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("WriteFile(go.mod) error = %v", err)
	}

	mainPath := filepath.Join(projectDir, "main.go")
	mainContent := "package main\n\nfunc main() { helper() }\n"
	if err := os.WriteFile(mainPath, []byte(mainContent), 0o644); err != nil {
		t.Fatalf("WriteFile(main.go) error = %v", err)
	}

	helperPath := filepath.Join(projectDir, "helper.go")
	helperContent := "package main\n\nfunc helper() {}\n"
	if err := os.WriteFile(helperPath, []byte(helperContent), 0o644); err != nil {
		t.Fatalf("WriteFile(helper.go) error = %v", err)
	}

	out, _, _, err := Build(mainPath, mainContent, Options{Mode: "graph", RelatedFiles: 2})
	if err != nil {
		t.Fatalf("Build(graph) error = %v", err)
	}
	if !strings.Contains(out, "# Target File") {
		t.Fatalf("expected target section, got %q", out)
	}
	if !strings.Contains(out, "# Related Files") {
		t.Fatalf("expected related section, got %q", out)
	}
	if !strings.Contains(out, "helper.go") {
		t.Fatalf("expected helper.go in graph output, got %q", out)
	}
}

func TestBuildCachesRepeatedReads(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "main.go")
	content := "package main\n\nfunc alpha() {}\n"

	before := CacheStats()
	if _, _, _, err := Build(filePath, content, Options{Mode: "auto"}); err != nil {
		t.Fatalf("Build() first error = %v", err)
	}
	mid := CacheStats()
	if _, _, _, err := Build(filePath, content, Options{Mode: "auto"}); err != nil {
		t.Fatalf("Build() second error = %v", err)
	}
	after := CacheStats()

	if mid.Misses <= before.Misses {
		t.Fatalf("expected cache miss count to increase")
	}
	if after.Hits <= mid.Hits {
		t.Fatalf("expected cache hit count to increase")
	}
}
