package configcmd

import (
	"reflect"
	"testing"
)

func TestNormalizeConfigKey(t *testing.T) {
	tests := map[string]string{
		"tracking.enabled":   "tracking.enabled",
		"max_context_tokens": "maxcontexttokens",
		"max-context-tokens": "maxcontexttokens",
		"MaxContextTokens":   "maxcontexttokens",
		"tracking_enabled":   "trackingenabled",
		"tracking-enabled":   "trackingenabled",
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			got := normalizeConfigKey(input)
			if got != want {
				t.Fatalf("normalizeConfigKey(%q) = %q, want %q", input, got, want)
			}
		})
	}
}

func TestIndirectValue(t *testing.T) {
	value := "hello"
	ptr := &value

	got := indirectValue(reflect.ValueOf(ptr))
	if !got.IsValid() {
		t.Fatal("indirectValue() returned invalid value")
	}
	if got.String() != value {
		t.Fatalf("indirectValue() = %q, want %q", got.String(), value)
	}
}
