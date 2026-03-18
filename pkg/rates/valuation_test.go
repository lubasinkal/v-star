package rates

import (
	"testing"
)

func TestPresentValue(t *testing.T) {
	converter := RateConverter{EffectiveRate: 0.05}

	tests := []struct {
		name       string
		sumAssured float64
		term       int
		want       float64
	}{
		{"Zero term", 1000, 0, 1000},
		{"Simple PV", 1000, 1, 952.38},
		{"Longer term", 1000, 5, 783.53},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := converter.PresentValue(tt.sumAssured, tt.term)
			if got < tt.want-0.01 || got > tt.want+0.01 {
				t.Errorf("PresentValue() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPresentValueStar(t *testing.T) {
	converter := RateConverter{EffectiveRate: 0.05}
	j := 0.02

	tests := []struct {
		name       string
		sumAssured float64
		term       int
		want       float64
	}{
		{"Zero term", 1000, 0, 1000},
		{"Simple PV with v-star", 1000, 1, 971.43},
		{"Longer term with v-star", 1000, 5, 865.08},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := converter.PresentValueStar(tt.sumAssured, tt.term, j)
			if got < tt.want-0.01 || got > tt.want+0.01 {
				t.Errorf("PresentValueStar() = %v, want %v", got, tt.want)
			}
		})
	}
}

func BenchmarkPresentValue(b *testing.B) {
	converter := RateConverter{EffectiveRate: 0.05}

	for i := 0; i < b.N; i++ {
		_ = converter.PresentValue(1000, 10)
	}
}
