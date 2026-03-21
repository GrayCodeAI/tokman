package filter

import (
	"testing"
)

func TestAdaptiveLayerSelector_Code(t *testing.T) {
	selector := NewAdaptiveLayerSelector()

	code := `package main

func main() {
	fmt.Println("Hello")
}

type User struct {
	Name string
}`

	ct := selector.AnalyzeContent(code)
	if ct != ContentTypeCode {
		t.Errorf("Expected ContentTypeCode, got %v", ct)
	}
	t.Logf("Detected: %s", ct)
}

func TestAdaptiveLayerSelector_Logs(t *testing.T) {
	selector := NewAdaptiveLayerSelector()

	logs := `[2024-01-15 10:23:45] INFO: Starting server
[2024-01-15 10:23:46] DEBUG: Connected to database
[2024-01-15 10:23:47] WARN: High memory usage
[2024-01-15 10:23:48] ERROR: Connection timeout`

	ct := selector.AnalyzeContent(logs)
	if ct != ContentTypeLogs {
		t.Errorf("Expected ContentTypeLogs, got %v", ct)
	}
	t.Logf("Detected: %s", ct)
}

func TestAdaptiveLayerSelector_Conversation(t *testing.T) {
	selector := NewAdaptiveLayerSelector()

	conv := `User: How do I compress tokens?
Assistant: You can use TokMan's 14-layer pipeline.
User: What's the reduction?
Assistant: 95-99% depending on content type.`

	ct := selector.AnalyzeContent(conv)
	if ct != ContentTypeConversation {
		t.Errorf("Expected ContentTypeConversation, got %v", ct)
	}
	t.Logf("Detected: %s", ct)
}

func TestAdaptiveLayerSelector_Git(t *testing.T) {
	selector := NewAdaptiveLayerSelector()

	git := `On branch main
Changes not staged for commit:
	modified:   internal/filter/pipeline.go
	modified:   README.md`

	ct := selector.AnalyzeContent(git)
	if ct != ContentTypeGitOutput {
		t.Errorf("Expected ContentTypeGitOutput, got %v", ct)
	}
	t.Logf("Detected: %s", ct)
}

func TestAdaptiveLayerSelector_TestOutput(t *testing.T) {
	selector := NewAdaptiveLayerSelector()

	testOut := `=== RUN   TestFilter
--- PASS: TestFilter (0.00s)
=== RUN   TestPipeline
--- PASS: TestPipeline (0.01s)
PASS`

	ct := selector.AnalyzeContent(testOut)
	if ct != ContentTypeTestOutput {
		t.Errorf("Expected ContentTypeTestOutput, got %v", ct)
	}
	t.Logf("Detected: %s", ct)
}

func TestAdaptiveLayerSelector_Docker(t *testing.T) {
	selector := NewAdaptiveLayerSelector()

	docker := `CONTAINER ID   IMAGE          COMMAND         STATUS
abc123         nginx:latest   "/entrypoint"   Up 2 hours`

	ct := selector.AnalyzeContent(docker)
	if ct != ContentTypeDockerOutput {
		t.Errorf("Expected ContentTypeDockerOutput, got %v", ct)
	}
	t.Logf("Detected: %s", ct)
}

func TestAdaptiveLayerSelector_RecommendedConfig(t *testing.T) {
	selector := NewAdaptiveLayerSelector()

	tests := []struct {
		name string
		ct   ContentType
		mode Mode
	}{
		{"code", ContentTypeCode, ModeMinimal},
		{"logs", ContentTypeLogs, ModeAggressive},
		{"conversation", ContentTypeConversation, ModeMinimal},
		{"git", ContentTypeGitOutput, ModeMinimal},
		{"test", ContentTypeTestOutput, ModeMinimal},
		{"docker", ContentTypeDockerOutput, ModeMinimal},
		{"mixed", ContentTypeMixed, ModeAggressive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := selector.RecommendedConfig(tt.ct, tt.mode)
			t.Logf("%s: compaction=%v, h2o=%v, sink=%v",
				tt.name, config.EnableCompaction, config.EnableH2O, config.EnableAttentionSink)
		})
	}
}

func TestAdaptiveLayerSelector_OptimizePipeline(t *testing.T) {
	selector := NewAdaptiveLayerSelector()

	code := `func main() { fmt.Println("test") }`
	p := selector.OptimizePipeline(code, ModeMinimal)

	if p == nil {
		t.Error("Expected non-nil pipeline")
		return
	}

	// Process the input
	_, stats := p.Process(code)
	t.Logf("Input: %d, Output: %d, Saved: %d",
		stats.OriginalTokens, stats.FinalTokens, stats.TotalSaved)
}
