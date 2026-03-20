package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarizeGoDeps(t *testing.T) {
	tmpDir := t.TempDir()
	goMod := `module example.com/test

go 1.21

require (
	github.com/spf13/cobra v1.8.0
	github.com/fatih/color v1.16.0
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
)
`
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)

	result := summarizeGoDeps(tmpDir)

	if !strings.Contains(result, "github.com/spf13/cobra") {
		t.Error("missing cobra dependency")
	}
	if !strings.Contains(result, "github.com/fatih/color") {
		t.Error("missing color dependency")
	}
	if !strings.Contains(result, "Total: 3") {
		t.Errorf("expected 3 total deps, got: %s", result)
	}
}

func TestSummarizeGoDepsNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	result := summarizeGoDeps(tmpDir)
	if result != "" {
		t.Errorf("expected empty string for missing go.mod, got: %s", result)
	}
}

func TestSummarizeJSDeps(t *testing.T) {
	tmpDir := t.TempDir()
	pkgJSON := `{
  "name": "test",
  "dependencies": {
    "react": "^18.0.0",
    "react-dom": "^18.0.0"
  },
  "devDependencies": {
    "typescript": "^5.0.0",
    "eslint": "^8.0.0"
  }
}
`
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(pkgJSON), 0644)

	result := summarizeJSDeps(tmpDir)

	if !strings.Contains(result, "react") {
		t.Error("missing react dependency")
	}
	if !strings.Contains(result, "typescript") {
		t.Error("missing typescript dependency")
	}
	if !strings.Contains(result, "(dev)") {
		t.Error("missing dev marker for devDependencies")
	}
}

func TestSummarizeRustDeps(t *testing.T) {
	tmpDir := t.TempDir()
	cargoToml := `[package]
name = "test"
version = "0.1.0"

[dependencies]
serde = "1.0"
tokio = { version = "1.0", features = ["full"] }
clap = "4.0"

[dev-dependencies]
tempfile = "3.0"
`
	os.WriteFile(filepath.Join(tmpDir, "Cargo.toml"), []byte(cargoToml), 0644)

	result := summarizeRustDeps(tmpDir)

	if !strings.Contains(result, "serde") {
		t.Error("missing serde dependency")
	}
	if !strings.Contains(result, "tokio") {
		t.Error("missing tokio dependency")
	}
	if !strings.Contains(result, "clap") {
		t.Error("missing clap dependency")
	}
	if !strings.Contains(result, "Total: 3") {
		t.Errorf("expected 3 deps, got: %s", result)
	}
	// Should NOT include dev-dependencies (different section)
	if strings.Contains(result, "tempfile") {
		t.Error("should not include dev-dependencies")
	}
}

func TestSummarizePythonDeps(t *testing.T) {
	tmpDir := t.TempDir()
	reqs := `# Core dependencies
requests==2.31.0
flask>=2.3.0
sqlalchemy~=2.0
psycopg2-binary
numpy
`
	os.WriteFile(filepath.Join(tmpDir, "requirements.txt"), []byte(reqs), 0644)

	result := summarizePythonDeps(tmpDir)

	if !strings.Contains(result, "requests") {
		t.Error("missing requests dependency")
	}
	if !strings.Contains(result, "flask") {
		t.Error("missing flask dependency")
	}
	if !strings.Contains(result, "numpy") {
		t.Error("missing numpy dependency")
	}
}

func TestSummarizePythonDepsPyprojectToml(t *testing.T) {
	tmpDir := t.TempDir()
	pyproject := `[project]
name = "test"
dependencies = [
    "requests>=2.31.0",
    "pydantic>=2.0",
]
`
	os.WriteFile(filepath.Join(tmpDir, "pyproject.toml"), []byte(pyproject), 0644)

	result := summarizePythonDeps(tmpDir)

	if !strings.Contains(result, "requests") {
		t.Error("missing requests dependency")
	}
	if !strings.Contains(result, "pydantic") {
		t.Error("missing pydantic dependency")
	}
}

func TestHasFile(t *testing.T) {
	tmpDir := t.TempDir()
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(""), 0644)

	if !hasFile(tmpDir, "go.mod") {
		t.Error("expected hasFile to return true for existing file")
	}
	if hasFile(tmpDir, "package.json") {
		t.Error("expected hasFile to return false for missing file")
	}
}
