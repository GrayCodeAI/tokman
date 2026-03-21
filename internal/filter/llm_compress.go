package filter

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// LLMCompressor provides LLM-based intelligent compression
// Uses local or API-based LLMs for context-aware token reduction
type LLMCompressor struct {
	Provider     string        // openai, anthropic, ollama, local
	Model        string        // gpt-4, claude-3-sonnet, llama3, etc.
	APIKey       string        // API key for cloud providers
	BaseURL      string        // Custom endpoint (for Ollama, etc.)
	Timeout      time.Duration // Request timeout
	MaxTokens    int           // Max output tokens
	Temperature  float64       // Sampling temperature
	CacheEnabled bool          // Enable response caching
	Cache        map[string]string
}

// LLMCompressConfig holds configuration for LLM compression
type LLMCompressConfig struct {
	Provider     string
	Model        string
	APIKey       string
	BaseURL      string
	MaxTokens    int
	Temperature  float64
	CacheEnabled bool
}

// CompressionRequest represents a compression request
type CompressionRequest struct {
	Input       string
	QueryIntent string
	Budget      int
	Context     []string // Additional context files
}

// CompressionResult represents the compression result
type CompressionResult struct {
	Output           string
	OriginalTokens   int
	CompressedTokens int
	Reduction        float64
	Method           string
	Cached           bool
}

// NewLLMCompressor creates a new LLM compressor
func NewLLMCompressor(cfg LLMCompressConfig) *LLMCompressor {
	if cfg.MaxTokens == 0 {
		cfg.MaxTokens = 1000
	}
	if cfg.Temperature == 0 {
		cfg.Temperature = 0.3
	}

	return &LLMCompressor{
		Provider:     cfg.Provider,
		Model:        cfg.Model,
		APIKey:       cfg.APIKey,
		BaseURL:      cfg.BaseURL,
		Timeout:      30 * time.Second,
		MaxTokens:    cfg.MaxTokens,
		Temperature:  cfg.Temperature,
		CacheEnabled: cfg.CacheEnabled,
		Cache:        make(map[string]string),
	}
}

// Compress performs LLM-based compression
func (l *LLMCompressor) Compress(req CompressionRequest) (*CompressionResult, error) {
	// Check cache first
	if l.CacheEnabled {
		cacheKey := l.cacheKey(req.Input, req.QueryIntent)
		if cached, ok := l.Cache[cacheKey]; ok {
			origTokens := estimateTokens(req.Input)
			compTokens := estimateTokens(cached)
			return &CompressionResult{
				Output:           cached,
				OriginalTokens:   origTokens,
				CompressedTokens: compTokens,
				Reduction:        float64(origTokens-compTokens) / float64(origTokens) * 100,
				Method:           "llm-cache",
				Cached:           true,
			}, nil
		}
	}

	// Build prompt based on query intent
	prompt := l.buildPrompt(req)

	// Call LLM
	output, err := l.callLLM(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Calculate metrics
	origTokens := estimateTokens(req.Input)
	compTokens := estimateTokens(output)

	result := &CompressionResult{
		Output:           output,
		OriginalTokens:   origTokens,
		CompressedTokens: compTokens,
		Reduction:        float64(origTokens-compTokens) / float64(origTokens) * 100,
		Method:           "llm-" + l.Provider,
		Cached:           false,
	}

	// Cache result
	if l.CacheEnabled {
		cacheKey := l.cacheKey(req.Input, req.QueryIntent)
		l.Cache[cacheKey] = output
	}

	return result, nil
}

// buildPrompt creates the compression prompt based on intent
func (l *LLMCompressor) buildPrompt(req CompressionRequest) string {
	var intentInstruction string

	switch req.QueryIntent {
	case "debug":
		intentInstruction = `Focus on: errors, warnings, stack traces, and relevant code paths.
Preserve: exact error messages, line numbers, and function names.`
	case "understand":
		intentInstruction = `Focus on: structure, key concepts, and relationships.
Preserve: function signatures, type definitions, and control flow.`
	case "modify":
		intentInstruction = `Focus on: specific code areas mentioned in the query.
Preserve: exact code blocks that may need modification.`
	case "search":
		intentInstruction = `Focus on: matches and their surrounding context.
Preserve: exact match lines and file locations.`
	default:
		intentInstruction = `Focus on: essential information for understanding.
Preserve: key facts, names, and structural elements.`
	}

	budget := req.Budget
	if budget == 0 {
		budget = l.MaxTokens
	}

	return fmt.Sprintf(`You are a token optimization expert. Compress the following content for an AI coding assistant.

INTENT: %s

%s

BUDGET: Maximum %d tokens in output.

RULES:
1. Remove ALL boilerplate, comments (unless critical), and repetitive content
2. Use dense notation: "fn name(args) -> ret" instead of full signatures
3. Group related items: "imports: {a, b, c}" 
4. Keep exact values for errors, IDs, and paths
5. Preserve critical context only

INPUT:
%s

COMPRESSED OUTPUT:`, req.QueryIntent, intentInstruction, budget, req.Input)
}

// callLLM makes the actual LLM API call
func (l *LLMCompressor) callLLM(prompt string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), l.Timeout)
	defer cancel()

	switch l.Provider {
	case "openai":
		return l.callOpenAI(ctx, prompt)
	case "anthropic":
		return l.callAnthropic(ctx, prompt)
	case "ollama":
		return l.callOllama(ctx, prompt)
	case "local":
		return l.callLocal(ctx, prompt)
	default:
		// Try to detect available provider
		if _, err := exec.LookPath("ollama"); err == nil {
			return l.callOllama(ctx, prompt)
		}
		return "", fmt.Errorf("no LLM provider configured")
	}
}

// callOpenAI calls OpenAI API
func (l *LLMCompressor) callOpenAI(ctx context.Context, prompt string) (string, error) {
	apiKey := l.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENAI_API_KEY")
	}
	if apiKey == "" {
		return "", fmt.Errorf("OpenAI API key not set")
	}

	model := l.Model
	if model == "" {
		model = "gpt-4o-mini"
	}

	reqBody := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  l.MaxTokens,
		"temperature": l.Temperature,
	}

	reqJSON, _ := json.Marshal(reqBody)

	baseURL := l.BaseURL
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	cmd := exec.CommandContext(ctx, "curl", "-s", "-X", "POST",
		baseURL+"/chat/completions",
		"-H", "Content-Type: application/json",
		"-H", "Authorization: Bearer "+apiKey,
		"-d", string(reqJSON))

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Parse response
	var resp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.Unmarshal(output, &resp); err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	return strings.TrimSpace(resp.Choices[0].Message.Content), nil
}

// callAnthropic calls Anthropic API
func (l *LLMCompressor) callAnthropic(ctx context.Context, prompt string) (string, error) {
	apiKey := l.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if apiKey == "" {
		return "", fmt.Errorf("Anthropic API key not set")
	}

	model := l.Model
	if model == "" {
		model = "claude-3-haiku-20240307"
	}

	reqBody := map[string]interface{}{
		"model":      model,
		"max_tokens": l.MaxTokens,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}

	reqJSON, _ := json.Marshal(reqBody)

	cmd := exec.CommandContext(ctx, "curl", "-s", "-X", "POST",
		"https://api.anthropic.com/v1/messages",
		"-H", "Content-Type: application/json",
		"-H", "x-api-key: "+apiKey,
		"-H", "anthropic-version: 2023-06-01",
		"-d", string(reqJSON))

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var resp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}

	if err := json.Unmarshal(output, &resp); err != nil {
		return "", err
	}

	if len(resp.Content) == 0 {
		return "", fmt.Errorf("no response from Anthropic")
	}

	return strings.TrimSpace(resp.Content[0].Text), nil
}

// callOllama calls local Ollama API
func (l *LLMCompressor) callOllama(ctx context.Context, prompt string) (string, error) {
	model := l.Model
	if model == "" {
		model = "llama3"
	}

	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
		"options": map[string]interface{}{
			"num_predict": l.MaxTokens,
			"temperature": l.Temperature,
		},
	}

	reqJSON, _ := json.Marshal(reqBody)

	baseURL := l.BaseURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	cmd := exec.CommandContext(ctx, "curl", "-s", "-X", "POST",
		baseURL+"/api/generate",
		"-H", "Content-Type: application/json",
		"-d", string(reqJSON))

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	var resp struct {
		Response string `json:"response"`
	}

	if err := json.Unmarshal(output, &resp); err != nil {
		return "", err
	}

	return strings.TrimSpace(resp.Response), nil
}

// callLocal calls a local command-line LLM
func (l *LLMCompressor) callLocal(ctx context.Context, prompt string) (string, error) {
	// Try llm CLI tool if available
	if _, err := exec.LookPath("llm"); err == nil {
		cmd := exec.CommandContext(ctx, "llm", "-m", l.Model, prompt)
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(output)), nil
	}

	return "", fmt.Errorf("no local LLM available")
}

// cacheKey generates a cache key
func (l *LLMCompressor) cacheKey(input, intent string) string {
	return fmt.Sprintf("%s:%s", intent, hashString(input))
}

// hashString creates a simple hash
func hashString(s string) string {
	if len(s) > 64 {
		s = s[:64]
	}
	return fmt.Sprintf("%x", len(s)^0xDEADBEEF)
}

// estimateTokens estimates token count
func estimateTokens(s string) int {
	// Rough approximation: ~4 chars per token
	return len(s) / 4
}

// Close cleans up resources
func (l *LLMCompressor) Close() error {
	l.Cache = nil
	return nil
}
