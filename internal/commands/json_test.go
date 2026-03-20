package commands

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateSchema(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"object", `{"name": "test", "age": 30}`, "string"},
		{"array", `[1, 2, 3]`, "number"},
		{"null", `null`, "null"},
		{"string", `"hello"`, "string"},
		{"number", `42`, "number"},
		{"bool", `true`, "bool"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var v interface{}
			json.Unmarshal([]byte(tt.input), &v)
			got := generateSchema(v, 0, 3)
			if !strings.Contains(got, tt.want) {
				t.Errorf("generateSchema(%q) = %q, want to contain %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateSchemaDepthLimit(t *testing.T) {
	input := `{"a": {"b": {"c": {"d": 1}}}}`
	var v interface{}
	json.Unmarshal([]byte(input), &v)
	got := generateSchema(v, 0, 2)
	if !strings.Contains(got, "...") {
		t.Errorf("generateSchema with depth limit should contain '...', got: %s", got)
	}
}
