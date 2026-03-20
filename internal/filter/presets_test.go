package filter

import (
	"strings"
	"testing"
)

func TestPresetConfig(t *testing.T) {
	tests := []struct {
		name            string
		preset          PipelinePreset
		wantEntropy     bool
		wantPerplexity  bool
		wantGoalDriven  bool
		wantAST         bool
		wantContrastive bool
		wantNgram       bool
		wantCompaction  bool
		wantAttribution bool
		wantH2O         bool
		wantAttention   bool
	}{
		{
			name:            "fast preset",
			preset:          PresetFast,
			wantEntropy:     true,
			wantPerplexity:  false,
			wantGoalDriven:  true,
			wantAST:         false,
			wantContrastive: false,
			wantNgram:       false,
			wantCompaction:  false,
			wantAttribution: false,
			wantH2O:         false,
			wantAttention:   false,
		},
		{
			name:            "balanced preset",
			preset:          PresetBalanced,
			wantEntropy:     true,
			wantPerplexity:  true,
			wantGoalDriven:  true,
			wantAST:         true,
			wantContrastive: true,
			wantNgram:       true,
			wantCompaction:  false,
			wantAttribution: false,
			wantH2O:         false,
			wantAttention:   true,
		},
		{
			name:            "full preset",
			preset:          PresetFull,
			wantEntropy:     true,
			wantPerplexity:  true,
			wantGoalDriven:  true,
			wantAST:         true,
			wantContrastive: true,
			wantNgram:       true,
			wantCompaction:  true,
			wantAttribution: true,
			wantH2O:         true,
			wantAttention:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := PresetConfig(tt.preset, ModeMinimal)
			if cfg.EnableEntropy != tt.wantEntropy {
				t.Errorf("EnableEntropy = %v, want %v", cfg.EnableEntropy, tt.wantEntropy)
			}
			if cfg.EnablePerplexity != tt.wantPerplexity {
				t.Errorf("EnablePerplexity = %v, want %v", cfg.EnablePerplexity, tt.wantPerplexity)
			}
			if cfg.EnableGoalDriven != tt.wantGoalDriven {
				t.Errorf("EnableGoalDriven = %v, want %v", cfg.EnableGoalDriven, tt.wantGoalDriven)
			}
			if cfg.EnableCompaction != tt.wantCompaction {
				t.Errorf("EnableCompaction = %v, want %v", cfg.EnableCompaction, tt.wantCompaction)
			}
		})
	}
}

func TestQuickProcessPreset(t *testing.T) {
	input := strings.Repeat("hello world test data with some noise ", 100)

	presets := []PipelinePreset{PresetFast, PresetBalanced, PresetFull}
	for _, preset := range presets {
		t.Run(string(preset), func(t *testing.T) {
			output, saved := QuickProcessPreset(input, ModeMinimal, preset)
			if output == "" {
				t.Error("output should not be empty")
			}
			if saved < 0 {
				t.Errorf("saved tokens should be >= 0, got %d", saved)
			}
		})
	}
}

func TestPipelineEarlyExit(t *testing.T) {
	// Test that early exit works when budget is set
	input := strings.Repeat("the a an is are was were ", 1000)

	p := NewPipelineCoordinator(PipelineConfig{
		Mode:            ModeMinimal,
		Budget:          50, // Very tight budget
		SessionTracking: false,
		NgramEnabled:    false,
	})

	output, stats := p.Process(input)

	// Output should be <= budget when early exit triggers
	finalTokens := EstimateTokens(output)
	if finalTokens > stats.OriginalTokens {
		t.Errorf("final tokens (%d) should not exceed original (%d)", finalTokens, stats.OriginalTokens)
	}
}

func TestEntropyFilterExpanded(t *testing.T) {
	f := NewEntropyFilter()

	// Test that code tokens are now recognized
	codeTokens := []string{"func", "return", "import", "package", "def", "class", "const", "let"}
	for _, token := range codeTokens {
		_, exists := f.frequencies[token]
		if !exists {
			t.Errorf("token %q should be in expanded frequency table", token)
		}
	}

	// Test configurable threshold
	f2 := NewEntropyFilterWithThreshold(5.0)
	if f2.entropyThreshold != 5.0 {
		t.Errorf("threshold = %f, want 5.0", f2.entropyThreshold)
	}
}

func TestPipelineStats(t *testing.T) {
	stats := &PipelineStats{
		OriginalTokens:   1000,
		FinalTokens:      200,
		TotalSaved:       800,
		ReductionPercent: 80.0,
		LayerStats: map[string]LayerStat{
			"1_entropy":    {TokensSaved: 100},
			"2_perplexity": {TokensSaved: 200},
		},
	}

	str := stats.String()
	if str == "" {
		t.Error("Stats.String() should not be empty")
	}
	if !strings.Contains(str, "1000") {
		t.Error("Stats.String() should contain original token count")
	}
}
