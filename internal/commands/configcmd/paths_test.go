package configcmd

import (
	"path/filepath"
	"testing"

	"github.com/GrayCodeAI/tokman/internal/commands/shared"
	"github.com/GrayCodeAI/tokman/internal/config"
)

func TestEffectiveConfigPathUsesSharedOverride(t *testing.T) {
	oldCfgFile := shared.CfgFile
	shared.CfgFile = filepath.Join(t.TempDir(), "override.toml")
	t.Cleanup(func() {
		shared.CfgFile = oldCfgFile
	})

	if got := effectiveConfigPath(); got != shared.CfgFile {
		t.Fatalf("effectiveConfigPath() = %q, want %q", got, shared.CfgFile)
	}
	if got := effectiveConfigDir(); got != filepath.Dir(shared.CfgFile) {
		t.Fatalf("effectiveConfigDir() = %q, want %q", got, filepath.Dir(shared.CfgFile))
	}
}

func TestLookupConfigValueSupportsNestedKeys(t *testing.T) {
	cfg := config.Defaults()
	cfg.Filter.Mode = "aggressive"
	cfg.Tracking.Enabled = true
	cfg.Pipeline.MaxContextTokens = 1234

	tests := map[string]string{
		"filter.mode":                 "aggressive",
		"tracking.enabled":            "true",
		"pipeline.max_context_tokens": "1234",
	}

	for key, want := range tests {
		got, ok := lookupConfigValue(cfg, key)
		if !ok {
			t.Fatalf("lookupConfigValue(%q) reported missing", key)
		}
		if got != want {
			t.Fatalf("lookupConfigValue(%q) = %q, want %q", key, got, want)
		}
	}
}
