package ccusage

import (
	"testing"
)

func TestParseJSON_Daily(t *testing.T) {
	jsonStr := `{
		"daily": [
			{"date": "2025-01-15", "inputTokens": 1000, "outputTokens": 500, "totalTokens": 1500, "totalCost": 0.05},
			{"date": "2025-01-16", "inputTokens": 2000, "outputTokens": 800, "totalTokens": 2800, "totalCost": 0.10}
		]
	}`

	periods, err := parseJSON(jsonStr, Daily)
	if err != nil {
		t.Fatalf("parseJSON failed: %v", err)
	}

	if len(periods) != 2 {
		t.Errorf("expected 2 periods, got %d", len(periods))
	}

	if periods[0].Key != "2025-01-15" {
		t.Errorf("expected key '2025-01-15', got %q", periods[0].Key)
	}

	if periods[0].Metrics.InputTokens != 1000 {
		t.Errorf("expected 1000 input tokens, got %d", periods[0].Metrics.InputTokens)
	}

	if periods[1].Metrics.TotalCost != 0.10 {
		t.Errorf("expected total cost 0.10, got %f", periods[1].Metrics.TotalCost)
	}
}

func TestParseJSON_Weekly(t *testing.T) {
	jsonStr := `{
		"weekly": [
			{"week": "2025-01-13", "inputTokens": 5000, "outputTokens": 2000, "totalTokens": 7000, "totalCost": 0.25}
		]
	}`

	periods, err := parseJSON(jsonStr, Weekly)
	if err != nil {
		t.Fatalf("parseJSON failed: %v", err)
	}

	if len(periods) != 1 {
		t.Errorf("expected 1 period, got %d", len(periods))
	}

	if periods[0].Key != "2025-01-13" {
		t.Errorf("expected key '2025-01-13', got %q", periods[0].Key)
	}
}

func TestParseJSON_Monthly(t *testing.T) {
	jsonStr := `{
		"monthly": [
			{"month": "2025-01", "inputTokens": 20000, "outputTokens": 8000, "totalTokens": 28000, "totalCost": 1.00}
		]
	}`

	periods, err := parseJSON(jsonStr, Monthly)
	if err != nil {
		t.Fatalf("parseJSON failed: %v", err)
	}

	if len(periods) != 1 {
		t.Errorf("expected 1 period, got %d", len(periods))
	}

	if periods[0].Key != "2025-01" {
		t.Errorf("expected key '2025-01', got %q", periods[0].Key)
	}
}

func TestParseJSON_InvalidJSON(t *testing.T) {
	jsonStr := `{invalid json}`

	_, err := parseJSON(jsonStr, Daily)
	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestParseJSON_EmptyResponse(t *testing.T) {
	jsonStr := `{"daily": []}`

	periods, err := parseJSON(jsonStr, Daily)
	if err != nil {
		t.Fatalf("parseJSON failed: %v", err)
	}

	if len(periods) != 0 {
		t.Errorf("expected 0 periods, got %d", len(periods))
	}
}

func TestParseJSON_AllTokenFields(t *testing.T) {
	jsonStr := `{
		"daily": [
			{
				"date": "2025-01-15",
				"inputTokens": 1000,
				"outputTokens": 500,
				"cacheCreationTokens": 200,
				"cacheReadTokens": 100,
				"totalTokens": 1800,
				"totalCost": 0.05
			}
		]
	}`

	periods, err := parseJSON(jsonStr, Daily)
	if err != nil {
		t.Fatalf("parseJSON failed: %v", err)
	}

	m := periods[0].Metrics
	if m.CacheCreationTokens != 200 {
		t.Errorf("expected 200 cache creation tokens, got %d", m.CacheCreationTokens)
	}
	if m.CacheReadTokens != 100 {
		t.Errorf("expected 100 cache read tokens, got %d", m.CacheReadTokens)
	}
}
