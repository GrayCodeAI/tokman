package llm

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptTemplateManager_GetTemplate(t *testing.T) {
	m := NewPromptTemplateManager("")

	tests := []string{"debug", "review", "test", "build", "deploy", "search", "concise", "detailed"}

	for _, name := range tests {
		template, ok := m.GetTemplate(name)
		if !ok {
			t.Errorf("expected to find template %q", name)
		}
		if template.Name != name {
			t.Errorf("expected name %q, got %q", name, template.Name)
		}
	}
}

func TestPromptTemplateManager_GetTemplateForIntent(t *testing.T) {
	m := NewPromptTemplateManager("")

	template := m.GetTemplateForIntent("debug")
	if template.Name != "debug" {
		t.Errorf("expected debug template, got %q", template.Name)
	}

	// Unknown intent should fall back to concise
	template = m.GetTemplateForIntent("unknown")
	if template.Name != "concise" {
		t.Errorf("expected concise fallback, got %q", template.Name)
	}
}

func TestPromptTemplateManager_ListTemplates(t *testing.T) {
	m := NewPromptTemplateManager("")

	templates := m.ListTemplates()
	if len(templates) < 8 {
		t.Errorf("expected at least 8 built-in templates, got %d", len(templates))
	}
}

func TestPromptTemplateManager_AddTemplate(t *testing.T) {
	m := NewPromptTemplateManager("")

	custom := PromptTemplate{
		Name:        "custom_test",
		Description: "Custom test template",
		UserPrompt:  "Test: {{content}}",
		Intent:      "test",
	}

	err := m.AddTemplate(custom)
	if err != nil {
		t.Errorf("failed to add template: %v", err)
	}

	// Verify it was added
	template, ok := m.GetTemplate("custom_test")
	if !ok {
		t.Error("expected to find added template")
	}
	if template.Description != "Custom test template" {
		t.Errorf("expected description 'Custom test template', got %q", template.Description)
	}
}

func TestPromptTemplateManager_RemoveTemplate(t *testing.T) {
	m := NewPromptTemplateManager("")

	// Add a custom template
	custom := PromptTemplate{
		Name:       "to_remove",
		UserPrompt: "Test: {{content}}",
	}
	m.AddTemplate(custom)

	// Remove it
	err := m.RemoveTemplate("to_remove")
	if err != nil {
		t.Errorf("failed to remove template: %v", err)
	}

	// Verify it was removed
	_, ok := m.GetTemplate("to_remove")
	if ok {
		t.Error("expected template to be removed")
	}
}

func TestPromptTemplateManager_CannotRemoveBuiltin(t *testing.T) {
	m := NewPromptTemplateManager("")

	err := m.RemoveTemplate("debug")
	if err == nil {
		t.Error("expected error when removing built-in template")
	}
}

func TestPromptTemplateManager_BuildPrompt(t *testing.T) {
	m := NewPromptTemplateManager("")

	template, _ := m.GetTemplate("debug")
	content := "error: something went wrong"

	prompt := m.BuildPrompt(template, content, nil)

	// Should contain the content
	if !strings.Contains(prompt, content) {
		t.Error("expected prompt to contain content")
	}

	// Should contain system prompt
	if !strings.Contains(prompt, "debugging") {
		t.Error("expected prompt to contain system context")
	}
}

func TestPromptTemplateManager_BuildPromptWithVariables(t *testing.T) {
	m := NewPromptTemplateManager("")

	template := PromptTemplate{
		Name:       "var_test",
		UserPrompt: "Language: {{lang}}\nContent: {{content}}",
		Variables:  map[string]string{"lang": "Go"},
	}

	prompt := m.BuildPrompt(template, "test content", map[string]string{
		"lang": "Rust", // Override
	})

	if !strings.Contains(prompt, "Rust") {
		t.Error("expected prompt to contain variable override")
	}
	if !strings.Contains(prompt, "test content") {
		t.Error("expected prompt to contain content")
	}
}

func TestPromptTemplateManager_Validation(t *testing.T) {
	m := NewPromptTemplateManager("")

	// Missing name
	err := m.AddTemplate(PromptTemplate{UserPrompt: "test"})
	if err != ErrTemplateNameRequired {
		t.Error("expected ErrTemplateNameRequired")
	}

	// Missing prompt
	err = m.AddTemplate(PromptTemplate{Name: "test"})
	if err != ErrPromptRequired {
		t.Error("expected ErrPromptRequired")
	}
}

func TestPromptTemplateManager_SaveAndLoad(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "tokman-prompts-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create manager and add custom template
	m := NewPromptTemplateManager(tmpDir)

	custom := PromptTemplate{
		Name:        "saved_template",
		Description: "This should persist",
		UserPrompt:  "Saved: {{content}}",
	}

	err = m.AddTemplate(custom)
	if err != nil {
		t.Errorf("failed to add template: %v", err)
	}

	// Create new manager to test loading
	m2 := NewPromptTemplateManager(tmpDir)

	template, ok := m2.GetTemplate("saved_template")
	if !ok {
		t.Error("expected to find saved template after reload")
	}
	if template.Description != "This should persist" {
		t.Errorf("expected description to persist, got %q", template.Description)
	}
}

func TestCreateCustomTemplate(t *testing.T) {
	template := CreateCustomTemplate(
		"my_custom",
		"My custom template",
		"You are a helpful assistant",
		"Summarize: {{content}}",
		"general",
	)

	if template.Name != "my_custom" {
		t.Errorf("expected name 'my_custom', got %q", template.Name)
	}
	if template.MaxTokens != 300 {
		t.Errorf("expected default MaxTokens 300, got %d", template.MaxTokens)
	}
}

func TestDefaultTemplatesDir(t *testing.T) {
	dataHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dataHome)

	dir := DefaultTemplatesDir()
	want := filepath.Join(dataHome, "tokman", "prompts")
	if dir != want {
		t.Errorf("DefaultTemplatesDir() = %q, want %q", dir, want)
	}
}
