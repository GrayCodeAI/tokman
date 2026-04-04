package llm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

// PromptTemplate represents a custom prompt template for LLM summarization
type PromptTemplate struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	SystemPrompt string            `json:"system_prompt"`
	UserPrompt   string            `json:"user_prompt"`
	Intent       string            `json:"intent"`    // debug, review, test, build, general
	Variables    map[string]string `json:"variables"` // Custom variables for template
	MaxTokens    int               `json:"max_tokens"`
	Temperature  float64           `json:"temperature"`
}

// PromptTemplateManager manages custom prompt templates
type PromptTemplateManager struct {
	templatesDir string
	templates    map[string]PromptTemplate
	cache        map[string]string // Compiled prompt cache
	mu           sync.RWMutex
}

// NewPromptTemplateManager creates a new template manager
func NewPromptTemplateManager(templatesDir string) *PromptTemplateManager {
	m := &PromptTemplateManager{
		templatesDir: templatesDir,
		templates:    make(map[string]PromptTemplate),
		cache:        make(map[string]string),
	}

	// Load built-in templates
	m.loadBuiltinTemplates()

	// Load custom templates from disk
	if templatesDir != "" {
		m.loadTemplates()
	}

	return m
}

// loadBuiltinTemplates loads the default templates
func (m *PromptTemplateManager) loadBuiltinTemplates() {
	m.templates["debug"] = PromptTemplate{
		Name:         "debug",
		Description:  "Focus on errors, stack traces, and failure causes",
		SystemPrompt: "You are a code context compressor for debugging. Extract only critical debugging information.",
		UserPrompt: `Focus on:
- Error messages and types
- Stack traces and file locations
- Line numbers and function names
- Root cause indicators

Ignore:
- Passing test output
- Progress messages
- Success indicators

Content:
{{content}}

Summary:`,
		Intent:      "debug",
		MaxTokens:   300,
		Temperature: 0.2,
	}

	m.templates["review"] = PromptTemplate{
		Name:         "review",
		Description:  "Focus on code changes and potential issues",
		SystemPrompt: "You are a code review assistant. Extract key changes and potential concerns.",
		UserPrompt: `Focus on:
- Modified functions and classes
- Added/removed code
- Potential issues or concerns
- API changes

Content:
{{content}}

Review Summary:`,
		Intent:      "review",
		MaxTokens:   400,
		Temperature: 0.3,
	}

	m.templates["test"] = PromptTemplate{
		Name:         "test",
		Description:  "Focus on test results and coverage",
		SystemPrompt: "You are a test result summarizer. Extract test outcomes and failures.",
		UserPrompt: `Focus on:
- Pass/fail counts
- Failed test names and reasons
- Coverage information
- Test duration

Content:
{{content}}

Test Summary:`,
		Intent:      "test",
		MaxTokens:   250,
		Temperature: 0.2,
	}

	m.templates["build"] = PromptTemplate{
		Name:         "build",
		Description:  "Focus on build status and errors",
		SystemPrompt: "You are a build output summarizer. Extract build status and issues.",
		UserPrompt: `Focus on:
- Build success/failure status
- Compilation errors
- Warning counts and types
- Generated artifacts

Content:
{{content}}

Build Summary:`,
		Intent:      "build",
		MaxTokens:   250,
		Temperature: 0.2,
	}

	m.templates["deploy"] = PromptTemplate{
		Name:         "deploy",
		Description:  "Focus on deployment status and health",
		SystemPrompt: "You are a deployment status summarizer. Extract deployment information.",
		UserPrompt: `Focus on:
- Deployment status and version
- Service health
- Resource utilization
- Potential issues

Content:
{{content}}

Deployment Summary:`,
		Intent:      "deploy",
		MaxTokens:   200,
		Temperature: 0.2,
	}

	m.templates["search"] = PromptTemplate{
		Name:         "search",
		Description:  "Focus on file names and definitions",
		SystemPrompt: "You are a code search assistant. Extract relevant identifiers.",
		UserPrompt: `Focus on:
- File names and paths
- Function/class definitions
- Variable declarations
- Import statements

Content:
{{content}}

Search Results:`,
		Intent:      "search",
		MaxTokens:   300,
		Temperature: 0.2,
	}

	m.templates["concise"] = PromptTemplate{
		Name:         "concise",
		Description:  "General concise summary",
		SystemPrompt: "You are a concise summarizer. Create brief, accurate summaries.",
		UserPrompt: `Summarize the key points in 3-5 sentences:

{{content}}

Summary:`,
		Intent:      "general",
		MaxTokens:   200,
		Temperature: 0.3,
	}

	m.templates["detailed"] = PromptTemplate{
		Name:         "detailed",
		Description:  "Detailed technical summary",
		SystemPrompt: "You are a technical summarizer. Create detailed, accurate summaries preserving all critical information.",
		UserPrompt: `Create a detailed summary preserving:
- All file paths and line numbers
- Error messages exactly
- Key identifiers and values
- Action items

Content:
{{content}}

Detailed Summary:`,
		Intent:      "general",
		MaxTokens:   500,
		Temperature: 0.2,
	}
}

// loadTemplates loads custom templates from disk
func (m *PromptTemplateManager) loadTemplates() error {
	if m.templatesDir == "" {
		return nil
	}

	// Check if directory exists
	if _, err := os.Stat(m.templatesDir); os.IsNotExist(err) {
		return nil
	}

	// Read all template files
	files, err := filepath.Glob(filepath.Join(m.templatesDir, "*.json"))
	if err != nil {
		return err
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var template PromptTemplate
		if err := json.Unmarshal(data, &template); err != nil {
			continue
		}

		// Override built-in with custom
		m.templates[template.Name] = template
	}

	return nil
}

// GetTemplate retrieves a template by name
func (m *PromptTemplateManager) GetTemplate(name string) (PromptTemplate, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	template, ok := m.templates[name]
	return template, ok
}

// GetTemplateForIntent retrieves a template matching an intent
func (m *PromptTemplateManager) GetTemplateForIntent(intent string) PromptTemplate {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Try exact match first
	if template, ok := m.templates[intent]; ok {
		return template
	}

	// Fall back to general
	if template, ok := m.templates["concise"]; ok {
		return template
	}

	// Return minimal default
	return PromptTemplate{
		Name:        "default",
		UserPrompt:  "{{content}}\n\nSummary:",
		MaxTokens:   300,
		Temperature: 0.3,
	}
}

// ListTemplates returns all available template names
func (m *PromptTemplateManager) ListTemplates() []PromptTemplate {
	m.mu.RLock()
	defer m.mu.RUnlock()

	templates := make([]PromptTemplate, 0, len(m.templates))
	for _, t := range m.templates {
		templates = append(templates, t)
	}

	return templates
}

// AddTemplate adds or updates a custom template
func (m *PromptTemplateManager) AddTemplate(template PromptTemplate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate template
	if template.Name == "" {
		return ErrTemplateNameRequired
	}
	if template.UserPrompt == "" {
		return ErrPromptRequired
	}

	// Set defaults
	if template.MaxTokens == 0 {
		template.MaxTokens = 300
	}
	if template.Temperature == 0 {
		template.Temperature = 0.3
	}

	// Save to memory
	m.templates[template.Name] = template

	// Invalidate cache
	delete(m.cache, template.Name)

	// Save to disk
	if m.templatesDir != "" {
		return m.saveTemplate(template)
	}

	return nil
}

// RemoveTemplate removes a custom template
func (m *PromptTemplateManager) RemoveTemplate(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Don't allow removing built-in templates
	builtin := []string{"debug", "review", "test", "build", "deploy", "search", "concise", "detailed"}
	for _, b := range builtin {
		if name == b {
			return ErrCannotRemoveBuiltin
		}
	}

	delete(m.templates, name)
	delete(m.cache, name)

	// Remove from disk
	if m.templatesDir != "" {
		file := filepath.Join(m.templatesDir, name+".json")
		if err := os.Remove(file); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to remove %s: %v\n", file, err)
		}
	}

	return nil
}

// saveTemplate saves a template to disk
func (m *PromptTemplateManager) saveTemplate(template PromptTemplate) error {
	if m.templatesDir == "" {
		return nil
	}

	// Ensure directory exists
	if err := os.MkdirAll(m.templatesDir, 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(template, "", "  ")
	if err != nil {
		return err
	}

	file := filepath.Join(m.templatesDir, template.Name+".json")
	return os.WriteFile(file, data, 0600)
}

// BuildPrompt creates a complete prompt from a template
func (m *PromptTemplateManager) BuildPrompt(template PromptTemplate, content string, variables map[string]string) string {
	// Check cache first
	cacheKey := template.Name + "_" + hashContent(content)
	m.mu.RLock()
	if cached, ok := m.cache[cacheKey]; ok {
		m.mu.RUnlock()
		return cached
	}
	m.mu.RUnlock()

	// Build prompt
	var sb strings.Builder

	// Add system prompt if present
	if template.SystemPrompt != "" {
		sb.WriteString(template.SystemPrompt)
		sb.WriteString("\n\n")
	}

	// Process user prompt with variable substitution
	prompt := template.UserPrompt

	// Replace {{content}}
	prompt = strings.ReplaceAll(prompt, "{{content}}", content)

	// Replace custom variables - function parameters take priority over template defaults
	// Apply function parameters first (overrides), then template defaults for remaining
	for key, value := range variables {
		prompt = strings.ReplaceAll(prompt, "{{"+key+"}}", value)
	}
	for key, value := range template.Variables {
		prompt = strings.ReplaceAll(prompt, "{{"+key+"}}", value)
	}

	sb.WriteString(prompt)

	result := sb.String()

	// Cache result
	m.mu.Lock()
	m.cache[cacheKey] = result
	m.mu.Unlock()

	return result
}

// hashContent creates a simple hash for caching
func hashContent(content string) string {
	if len(content) > 50 {
		return content[:50]
	}
	return content
}

// Custom error types
var (
	ErrTemplateNameRequired = &PromptError{Msg: "template name is required"}
	ErrPromptRequired       = &PromptError{Msg: "user prompt is required"}
	ErrCannotRemoveBuiltin  = &PromptError{Msg: "cannot remove built-in template"}
)

// PromptError represents a prompt template error
type PromptError struct {
	Msg string
}

func (e *PromptError) Error() string {
	return e.Msg
}

// CreateCustomTemplate creates a new custom template from scratch
func CreateCustomTemplate(name, description, systemPrompt, userPrompt, intent string) PromptTemplate {
	return PromptTemplate{
		Name:         name,
		Description:  description,
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Intent:       intent,
		MaxTokens:    300,
		Temperature:  0.3,
		Variables:    make(map[string]string),
	}
}

// DefaultTemplatesDir returns the default templates directory
func DefaultTemplatesDir() string {
	return filepath.Join(llmDataPath(), "prompts")
}

func llmDataPath() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "tokman")
	}

	if runtime.GOOS == "windows" {
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			return filepath.Join(localAppData, "tokman")
		}
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "tokman", "data")
		}
	}

	if homeDir, err := os.UserHomeDir(); err == nil {
		return filepath.Join(homeDir, ".local", "share", "tokman")
	}

	return filepath.Join(os.TempDir(), "tokman-data")
}
