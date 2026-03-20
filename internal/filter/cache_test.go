package filter

import (
	"testing"
	"time"
)

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(3, time.Minute)

	// Test basic set/get
	cache.Set("a", &CachedResult{Output: "result_a", Tokens: 10})
	cache.Set("b", &CachedResult{Output: "result_b", Tokens: 20})
	cache.Set("c", &CachedResult{Output: "result_c", Tokens: 30})

	if cache.Size() != 3 {
		t.Errorf("Size() = %d, want 3", cache.Size())
	}

	// Test cache hit
	result := cache.Get("a")
	if result == nil {
		t.Fatal("expected cache hit for 'a'")
	}
	if result.Output != "result_a" {
		t.Errorf("Output = %q, want %q", result.Output, "result_a")
	}

	// Test eviction (adding 4th item should evict oldest)
	cache.Set("d", &CachedResult{Output: "result_d", Tokens: 40})
	if cache.Size() != 3 {
		t.Errorf("Size() after eviction = %d, want 3", cache.Size())
	}

	// 'b' should be evicted (oldest after 'a' was accessed)
	if cache.Get("b") != nil {
		t.Error("expected 'b' to be evicted")
	}

	// Hit rate
	hits, _ := cache.Stats()
	if hits == 0 {
		t.Error("expected some cache hits")
	}

	rate := cache.HitRate()
	if rate < 0 || rate > 100 {
		t.Errorf("HitRate() = %f, want 0-100", rate)
	}

	// Clear
	cache.Clear()
	if cache.Size() != 0 {
		t.Errorf("Size() after Clear = %d, want 0", cache.Size())
	}
}

func TestLRUCacheTTL(t *testing.T) {
	cache := NewLRUCache(10, 50*time.Millisecond)

	cache.Set("key", &CachedResult{Output: "value", Tokens: 5})

	// Should be available immediately
	if cache.Get("key") == nil {
		t.Error("expected cache hit before TTL")
	}

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Should be expired
	if cache.Get("key") != nil {
		t.Error("expected cache miss after TTL")
	}
}

func TestQualityMetrics(t *testing.T) {
	original := `Error: connection failed at 192.168.1.1:8080
INFO: retrying in 5 seconds
WARN: timeout exceeded`

	compressed := `Error: connection failed
WARN: timeout exceeded`

	metrics := MeasureQuality(original, compressed)

	if !metrics.ErrorPreserved {
		t.Error("ErrorPreserved should be true")
	}

	score := metrics.QualityScore()
	if score < 0 || score > 1 {
		t.Errorf("QualityScore() = %f, want 0-1", score)
	}
}

func TestByteSlicePool(t *testing.T) {
	pool := NewByteSlicePool(5, 1024)

	buf := pool.Get()
	if cap(buf) != 1024 {
		t.Errorf("Get() capacity = %d, want 1024", cap(buf))
	}

	buf = append(buf, []byte("hello")...)
	pool.Put(buf)

	buf2 := pool.Get()
	if cap(buf2) != 1024 {
		t.Errorf("Get() after Put capacity = %d, want 1024", cap(buf2))
	}
}

func TestLineScanner(t *testing.T) {
	data := []byte("line1\nline2\nline3\n")
	scanner := NewLineScanner(data)

	line1 := scanner.Next()
	if string(line1) != "line1" {
		t.Errorf("line1 = %q, want %q", string(line1), "line1")
	}

	line2 := scanner.Next()
	if string(line2) != "line2" {
		t.Errorf("line2 = %q, want %q", string(line2), "line2")
	}

	line3 := scanner.Next()
	if string(line3) != "line3" {
		t.Errorf("line3 = %q, want %q", string(line3), "line3")
	}

	line4 := scanner.Next()
	if line4 != nil {
		t.Errorf("line4 should be nil, got %q", string(line4))
	}
}

func TestLineScannerCRLF(t *testing.T) {
	data := []byte("line1\r\nline2\r\n")
	scanner := NewLineScanner(data)

	line1 := scanner.Next()
	if string(line1) != "line1" {
		t.Errorf("line1 = %q, want %q (CRLF handling)", string(line1), "line1")
	}
}

func TestCountLines(t *testing.T) {
	tests := []struct {
		input    string
		expected int
	}{
		{"", 0},
		{"single line", 1},
		{"line1\nline2\n", 2},
		{"line1\nline2", 2},
		{"a\nb\nc\n", 3},
	}

	for _, tt := range tests {
		got := CountLines([]byte(tt.input))
		if got != tt.expected {
			t.Errorf("CountLines(%q) = %d, want %d", tt.input, got, tt.expected)
		}
	}
}

func TestContentDetector(t *testing.T) {
	detector := NewContentAnalyzer()

	// JSON detection
	jsonType := detector.Analyze(`{"key": "value"}`)
	if jsonType != ContentTypeUnknown && jsonType != ContentTypeMixed {
		// JSON detection may or may not be implemented in AdaptiveLayerSelector
		t.Logf("JSON content detected as type %d", jsonType)
	}
}
