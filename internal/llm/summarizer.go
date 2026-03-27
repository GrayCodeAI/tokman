package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Summarizer provides LLM-powered summarization capabilities.
// Supports multiple backends: Ollama, LM Studio, and other OpenAI-compatible APIs.
type Summarizer struct {
	provider   Provider
	model      string
	baseURL    string
	timeout    time.Duration
	maxRetries int
}

// Provider represents an LLM provider type
type Provider string

const (
	ProviderOllama   Provider = "ollama"
	ProviderLMStudio Provider = "lmstudio"
	ProviderOpenAI   Provider = "openai"
	ProviderNone     Provider = "none"
)

// Config holds summarizer configuration
type Config struct {
	Provider   Provider
	Model      string
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		Provider:   ProviderOllama,
		Model:      "llama3.2:3b",
		BaseURL:    "http://localhost:11434",
		Timeout:    30 * time.Second,
		MaxRetries: 2,
	}
}

// NewSummarizer creates a new LLM summarizer
func NewSummarizer(cfg Config) *Summarizer {
	return &Summarizer{
		provider:   cfg.Provider,
		model:      cfg.Model,
		baseURL:    cfg.BaseURL,
		timeout:    cfg.Timeout,
		maxRetries: cfg.MaxRetries,
	}
}

// NewSummarizerFromEnv creates a summarizer from environment variables
func NewSummarizerFromEnv() *Summarizer {
	cfg := DefaultConfig()

	// Check for explicit provider
	if provider := os.Getenv("TOKMAN_LLM_PROVIDER"); provider != "" {
		cfg.Provider = Provider(provider)
	}

	// Check for model
	if model := os.Getenv("TOKMAN_LLM_MODEL"); model != "" {
		cfg.Model = model
	}

	// Check for base URL
	if baseURL := os.Getenv("TOKMAN_LLM_BASE_URL"); baseURL != "" {
		cfg.BaseURL = baseURL
	}

	// Auto-detect provider if not set
	if cfg.Provider == ProviderNone {
		cfg.Provider = detectProvider()
	}

	return NewSummarizer(cfg)
}

// detectProvider auto-detects available LLM provider
func detectProvider() Provider {
	// Try Ollama first (most common for local LLM)
	if isServiceRunning("http://localhost:11434/api/tags") {
		return ProviderOllama
	}

	// Try LM Studio
	if isServiceRunning("http://localhost:1234/v1/models") {
		return ProviderLMStudio
	}

	return ProviderNone
}

// isServiceRunning checks if a service endpoint is available
func isServiceRunning(url string) bool {
	client := http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// IsAvailable returns true if an LLM provider is available
func (s *Summarizer) IsAvailable() bool {
	return s.provider != ProviderNone && isServiceRunning(s.getHealthEndpoint())
}

// getHealthEndpoint returns the health check endpoint for the provider
func (s *Summarizer) getHealthEndpoint() string {
	switch s.provider {
	case ProviderOllama:
		return s.baseURL + "/api/tags"
	case ProviderLMStudio, ProviderOpenAI:
		return s.baseURL + "/v1/models"
	default:
		return ""
	}
}

// SummaryRequest represents a summarization request
type SummaryRequest struct {
	Content   string
	MaxTokens int
	Intent    string // "debug", "review", "search", etc.
}

// SummaryResponse represents a summarization result
type SummaryResponse struct {
	Summary     string
	TokensUsed  int
	ModelUsed   string
	FromCache   bool
	Compression float64 // Ratio of original to compressed
}

// Summarize generates a summary of the content
func (s *Summarizer) Summarize(req SummaryRequest) (*SummaryResponse, error) {
	if !s.IsAvailable() {
		return nil, fmt.Errorf("LLM provider not available")
	}

	// Build prompt based on intent
	prompt := s.buildPrompt(req)

	// Call the LLM
	response, err := s.callLLM(prompt, req.MaxTokens)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// Calculate compression ratio
	originalTokens := len(req.Content) / 4
	compression := float64(originalTokens) / float64(response.TokensUsed)
	if response.TokensUsed == 0 {
		compression = 1.0
	}

	return &SummaryResponse{
		Summary:     response.Content,
		TokensUsed:  response.TokensUsed,
		ModelUsed:   s.model,
		FromCache:   false,
		Compression: compression,
	}, nil
}

// buildPrompt constructs the summarization prompt
func (s *Summarizer) buildPrompt(req SummaryRequest) string {
	var prompt strings.Builder

	// System context
	prompt.WriteString("You are a code context compressor. Your task is to create concise summaries that preserve all critical information for an AI coding agent.\n\n")

	// Intent-specific instructions
	switch req.Intent {
	case "debug":
		prompt.WriteString("Focus on: errors, stack traces, file paths, line numbers, and failure causes.\n")
	case "review":
		prompt.WriteString("Focus on: code changes, modified functions, diff hunks, and potential issues.\n")
	case "search":
		prompt.WriteString("Focus on: file names, function definitions, and key identifiers.\n")
	case "deploy":
		prompt.WriteString("Focus on: build status, deployment targets, and configuration changes.\n")
	case "test":
		prompt.WriteString("Focus on: test results, pass/fail counts, and failing test names.\n")
	default:
		prompt.WriteString("Focus on: key information, actionable items, and important context.\n")
	}

	// Format instructions
	prompt.WriteString("\nFormat rules:\n")
	prompt.WriteString("- Keep file paths and line numbers exact\n")
	prompt.WriteString("- Preserve error messages verbatim when critical\n")
	prompt.WriteString("- Use bullet points for lists\n")
	prompt.WriteString("- Maximum 3-5 sentences total\n\n")

	// Add the content
	prompt.WriteString("Content to summarize:\n```\n")
	maxContent := 8000 // Limit context window
	if len(req.Content) > maxContent {
		prompt.WriteString(req.Content[:maxContent])
		prompt.WriteString("\n... [truncated]")
	} else {
		prompt.WriteString(req.Content)
	}
	prompt.WriteString("\n```\n\nSummary:")

	return prompt.String()
}

// llmResponse represents a response from the LLM
type llmResponse struct {
	Content    string
	TokensUsed int
}

// callLLM makes the actual API call to the LLM
func (s *Summarizer) callLLM(prompt string, maxTokens int) (*llmResponse, error) {
	switch s.provider {
	case ProviderOllama:
		return s.callOllama(prompt, maxTokens)
	case ProviderLMStudio, ProviderOpenAI:
		return s.callOpenAICompat(prompt, maxTokens)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", s.provider)
	}
}

// ollamaRequest represents an Ollama API request
type ollamaRequest struct {
	Model   string `json:"model"`
	Prompt  string `json:"prompt"`
	Stream  bool   `json:"stream"`
	Options struct {
		NumPredict  int     `json:"num_predict"`
		Temperature float64 `json:"temperature"`
	} `json:"options"`
}

// ollamaResponse represents an Ollama API response
type ollamaResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Response  string `json:"response"`
	Done      bool   `json:"done"`
	Context   []int  `json:"context"`
}

// callOllama calls the Ollama API
func (s *Summarizer) callOllama(prompt string, maxTokens int) (*llmResponse, error) {
	req := ollamaRequest{
		Model:  s.model,
		Prompt: prompt,
		Stream: false,
	}
	req.Options.NumPredict = maxTokens
	req.Options.Temperature = 0.3 // Low temperature for consistent summaries

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: s.timeout}
	resp, err := client.Post(s.baseURL+"/api/generate", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	var result ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &llmResponse{
		Content:    result.Response,
		TokensUsed: len(result.Response) / 4, // Approximate
	}, nil
}

// openAIRequest represents an OpenAI-compatible API request
type openAIRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIResponse represents an OpenAI-compatible API response
type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// callOpenAICompat calls an OpenAI-compatible API (LM Studio, etc.)
func (s *Summarizer) callOpenAICompat(prompt string, maxTokens int) (*llmResponse, error) {
	req := openAIRequest{
		Model: s.model,
		Messages: []message{
			{Role: "system", Content: "You are a code context compressor for AI coding agents."},
			{Role: "user", Content: prompt},
		},
		MaxTokens:   maxTokens,
		Temperature: 0.3,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: s.timeout}
	resp, err := client.Post(s.baseURL+"/v1/chat/completions", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var result openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no response from LLM")
	}

	return &llmResponse{
		Content:    result.Choices[0].Message.Content,
		TokensUsed: result.Usage.CompletionTokens,
	}, nil
}

// Streaming support for real-time summarization

// StreamSummary generates a summary with streaming output
func (s *Summarizer) StreamSummary(req SummaryRequest, callback func(string)) (*SummaryResponse, error) {
	if !s.IsAvailable() {
		return nil, fmt.Errorf("LLM provider not available")
	}

	if s.provider == ProviderOllama {
		return s.streamOllama(req, callback)
	}

	// Non-streaming fallback
	return s.Summarize(req)
}

// streamOllama streams from Ollama API
func (s *Summarizer) streamOllama(req SummaryRequest, callback func(string)) (*SummaryResponse, error) {
	prompt := s.buildPrompt(req)

	ollamaReq := ollamaRequest{
		Model:  s.model,
		Prompt: prompt,
		Stream: true,
	}
	ollamaReq.Options.NumPredict = req.MaxTokens
	ollamaReq.Options.Temperature = 0.3

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: s.timeout}
	resp, err := client.Post(s.baseURL+"/api/generate", "application/json", strings.NewReader(string(body)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var fullContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		var chunk ollamaResponse
		if err := json.Unmarshal(scanner.Bytes(), &chunk); err != nil {
			continue
		}

		if chunk.Response != "" {
			fullContent.WriteString(chunk.Response)
			callback(chunk.Response)
		}

		if chunk.Done {
			break
		}
	}

	content := fullContent.String()
	return &SummaryResponse{
		Summary:     content,
		TokensUsed:  len(content) / 4,
		ModelUsed:   s.model,
		FromCache:   false,
		Compression: float64(len(req.Content)/4) / float64(len(content)/4),
	}, nil
}

// ListModels returns available models for the provider
func (s *Summarizer) ListModels() ([]string, error) {
	if !s.IsAvailable() {
		return nil, fmt.Errorf("LLM provider not available")
	}

	switch s.provider {
	case ProviderOllama:
		return s.listOllamaModels()
	case ProviderLMStudio, ProviderOpenAI:
		return s.listOpenAIModels()
	default:
		return nil, fmt.Errorf("unsupported provider")
	}
}

// listOllamaModels lists available Ollama models
func (s *Summarizer) listOllamaModels() ([]string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(s.baseURL + "/api/tags")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// listOpenAIModels lists available OpenAI-compatible models
func (s *Summarizer) listOpenAIModels() ([]string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(s.baseURL + "/v1/models")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Data))
	for i, m := range result.Data {
		models[i] = m.ID
	}

	return models, nil
}

// GetModel returns the current model name
func (s *Summarizer) GetModel() string {
	return s.model
}

// GetProvider returns the current provider
func (s *Summarizer) GetProvider() Provider {
	return s.provider
}
