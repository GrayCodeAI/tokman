package anomaly

import (
	"testing"
	"time"
)

func TestNewDetector(t *testing.T) {
	d := NewDetector(2.0, time.Minute)
	if d == nil {
		t.Fatal("NewDetector returned nil")
	}
}

func TestCalculateStats(t *testing.T) {
	data := []DataPoint{
		{Timestamp: time.Now(), Value: 10},
		{Timestamp: time.Now(), Value: 20},
		{Timestamp: time.Now(), Value: 30},
	}

	mean, stddev := calculateStats(data)
	if mean != 20 {
		t.Errorf("mean = %f, want 20", mean)
	}
	if stddev <= 0 {
		t.Error("stddev should be positive")
	}
}

func TestDetectSuddenChange(t *testing.T) {
	d := NewDetector(2.0, time.Minute)

	// Create data with a sudden spike (>200% change to exceed threshold)
	data := []DataPoint{
		{Timestamp: time.Now().Add(-1 * time.Minute), Value: 100},
		{Timestamp: time.Now(), Value: 500}, // 400% increase, > 200% threshold
	}

	anomalies := d.DetectSuddenChange(data)
	if len(anomalies) == 0 {
		t.Error("expected to detect spike anomaly")
	}
}

func TestDetectTrendChange(t *testing.T) {
	d := NewDetector(2.0, time.Minute)

	// Need at least 10 data points
	data := make([]DataPoint, 12)
	base := time.Now().Add(-12 * time.Minute)
	for i := range data {
		data[i].Timestamp = base.Add(time.Duration(i) * time.Minute)
		// First half: steady increase
		if i < 6 {
			data[i].Value = float64(i) * 10
		} else {
			// Second half: sharp decrease (trend change)
			data[i].Value = float64(12-i) * 50
		}
	}

	anomalies := d.DetectTrendChange(data)
	if len(anomalies) == 0 {
		t.Error("expected trend change anomaly")
	}
}

func TestDetect_EmptyData(t *testing.T) {
	d := NewDetector(2.0, time.Minute)

	// Empty data returns empty slice
	if got := d.DetectSuddenChange([]DataPoint{}); len(got) != 0 {
		t.Errorf("DetectSuddenChange([]DataPoint{}) = %v, want empty slice", got)
	}

	// Single point - no change to detect (loop starts at i=1)
	if got := d.DetectSuddenChange([]DataPoint{{Value: 100}}); got != nil && len(got) > 0 {
		t.Errorf("DetectSuddenChange(1 point) should return empty, got %v", got)
	}
}

func TestAnomalyFields(t *testing.T) {
	d := NewDetector(2.0, time.Minute)

	data := []DataPoint{
		{Timestamp: time.Now().Add(-1 * time.Minute), Value: 100},
		{Timestamp: time.Now(), Value: 500},
	}

	anomalies := d.DetectSuddenChange(data)
	for _, a := range anomalies {
		if a.Type == "" {
			t.Error("anomaly should have Type")
		}
		if a.Severity == "" {
			t.Error("anomaly should have Severity")
		}
	}
}
