package core

import "testing"

func TestPercentOf(t *testing.T) {
	tests := []struct {
		name  string
		part  int
		total int
		want  float64
	}{
		{name: "normal", part: 25, total: 100, want: 25},
		{name: "zero total", part: 10, total: 0, want: 0},
		{name: "negative total", part: 10, total: -1, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := percentOf(tt.part, tt.total); got != tt.want {
				t.Fatalf("percentOf(%d, %d) = %v, want %v", tt.part, tt.total, got, tt.want)
			}
		})
	}
}
