package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandleHealth(t *testing.T) {
	s := New(Config{Port: 8080})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	s.handleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("Expected status 'ok', got %s", resp.Status)
	}

	t.Logf("Health response: %+v", resp)
}

func TestHandleCompress(t *testing.T) {
	s := New(Config{Port: 8080})

	body := CompressRequest{
		Input:  "This is a test string to compress for the REST API endpoint.",
		Mode:   "minimal",
		Budget: 100,
	}

	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/compress", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.handleCompress(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var resp CompressResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Compress response: tokens %d -> %d (%.1f%% saved)",
		resp.OriginalTokens, resp.FinalTokens, resp.ReductionPercent)
}

func TestHandleCompressAggressive(t *testing.T) {
	s := New(Config{Port: 8080})

	body := CompressRequest{
		Input:  "This is a longer test string that should be compressed aggressively when we set the mode to aggressive mode for the API.",
		Mode:   "aggressive",
		Budget: 50,
	}

	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/compress", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.handleCompress(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var resp CompressResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Aggressive compress: tokens %d -> %d (%.1f%% saved)",
		resp.OriginalTokens, resp.FinalTokens, resp.ReductionPercent)
}

func TestHandleCompressAdaptive(t *testing.T) {
	s := New(Config{Port: 8080})

	body := CompressRequest{
		Input: `func main() {
	fmt.Println("Hello, World!")
}`,
		Mode: "minimal",
	}

	jsonBody, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/compress/adaptive", bytes.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	s.handleCompressAdaptive(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var resp CompressResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	t.Logf("Adaptive compress: tokens %d -> %d", resp.OriginalTokens, resp.FinalTokens)
}

func TestHandleAnalyze(t *testing.T) {
	s := New(Config{Port: 8080})

	tests := []struct {
		input    string
		expected string
	}{
		{"func main() { }", "code"},
		{"User: Hello\nAssistant: Hi", "conversation"},
		{"[INFO] Server started", "logs"},
	}

	for _, tt := range tests {
		body := CompressRequest{Input: tt.input}
		jsonBody, _ := json.Marshal(body)

		req := httptest.NewRequest(http.MethodPost, "/analyze", bytes.NewReader(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()

		s.handleAnalyze(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var resp AnalyzeResponse
		if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		preview := tt.input
		if len(preview) > 20 {
			preview = preview[:20]
		}
		t.Logf("Analyze %q -> %s", preview, resp.ContentType)
	}
}

func TestHandleStats(t *testing.T) {
	s := New(Config{Port: 8080})

	req := httptest.NewRequest(http.MethodGet, "/stats", nil)
	rec := httptest.NewRecorder()

	s.handleStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rec.Code)
	}

	var resp StatsResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.LayerCount != 14 {
		t.Errorf("Expected 14 layers, got %d", resp.LayerCount)
	}

	t.Logf("Stats: %+v", resp)
}
