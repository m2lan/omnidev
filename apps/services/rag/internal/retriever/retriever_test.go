package retriever

import (
	"testing"
)

func TestVectorToString(t *testing.T) {
	tests := []struct {
		name string
		vec  []float32
		want string
	}{
		{"empty", []float32{}, "[]"},
		{"nil", nil, "[]"},
		{"single", []float32{1.0}, "[1.000000]"},
		{"multiple", []float32{0.1, 0.2, 0.3}, "[0.100000,0.200000,0.300000]"},
		{"negative", []float32{-0.5, 0.5}, "[-0.500000,0.500000]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := vectorToString(tt.vec)
			if got != tt.want {
				t.Errorf("vectorToString(%v) = %q, want %q", tt.vec, got, tt.want)
			}
		})
	}
}

// Note: Full HybridRetriever.Search tests require a PostgreSQL database with pgvector.
// These would be integration tests run with `go test -tags=integration`.
// For unit tests, we test the helper functions and the Retriever interface contract.
