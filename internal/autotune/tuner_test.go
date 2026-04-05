package autotune

import "testing"

func TestNewTuner(t *testing.T) {
	tr := NewTuner()
	if tr == nil {
		t.Fatal("NewTuner returned nil")
	}
}

func TestTuner_AddConfig(t *testing.T) {
	tr := NewTuner()
	cfg := TuningConfig{
		Name:       "test-config",
		Parameters: map[string]interface{}{"ratio": 0.3},
	}
	tr.AddConfig(cfg)

	best := tr.GetBestConfig()
	if best == nil || best.Name != "test-config" {
		t.Errorf("GetBestConfig = %+v, want test-config", best)
	}
}

func TestTuner_UpdateScore(t *testing.T) {
	tr := NewTuner()
	tr.AddConfig(TuningConfig{Name: "a"})
	tr.AddConfig(TuningConfig{Name: "b"})
	tr.UpdateScore("a", 0.7)
	tr.UpdateScore("b", 0.9)

	best := tr.GetBestConfig()
	if best == nil || best.Name != "b" {
		t.Errorf("best config = %+v, want b", best)
	}
}

func TestTuner_GenerateConfigs(t *testing.T) {
	tr := NewTuner()
	configs := tr.GenerateConfigs(5)
	if len(configs) != 5 {
		t.Errorf("GenerateConfigs(5) = %d, want 5", len(configs))
	}
}
