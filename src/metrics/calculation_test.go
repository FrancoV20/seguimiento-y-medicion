package metrics

import (
	"math/big"
	"testing"
)

func TestCalculateDeviation(t *testing.T) {
	tests := []struct {
		name      string
		estimated string
		actual    string
		want      string
	}{
		{
			name:      "positive deviation",
			estimated: "100",
			actual:    "110",
			want:      "10.00",
		},
		{
			name:      "negative deviation",
			estimated: "100",
			actual:    "90",
			want:      "-10.00",
		},
		{
			name:      "rounds positive half away from zero",
			estimated: "1000",
			actual:    "1010.05",
			want:      "1.01",
		},
		{
			name:      "rounds negative half away from zero",
			estimated: "1000",
			actual:    "989.95",
			want:      "-1.01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estimated := mustRat(t, tt.estimated)
			actual := mustRat(t, tt.actual)

			got, err := CalculateDeviation(estimated, actual)
			if err != nil {
				t.Fatalf("CalculateDeviation() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("CalculateDeviation() = %q, want %q", got, tt.want)
			}
		})
	}
}

func mustRat(t *testing.T, value string) *big.Rat {
	t.Helper()

	rat, ok := new(big.Rat).SetString(value)
	if !ok {
		t.Fatalf("invalid rational number %q", value)
	}
	return rat
}
