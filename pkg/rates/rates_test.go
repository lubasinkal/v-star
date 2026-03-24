package rates

import (
	"math"
	"testing"
)

func TestNominalToEffective(t *testing.T) {
	tests := []struct {
		name        string
		nominal     float64
		compounding int
		expected    float64
	}{
		{"annual compounding", 0.05, 1, 0.05},
		{"monthly compounding (12%)", 0.12, 12, 0.12682503013196977},
		{"quarterly compounding (8%)", 0.08, 4, 0.08243216},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NominalToEffective(tt.nominal, tt.compounding)
			// Using 1e-9 tolerance for floating point comparison
			if math.Abs(got-tt.expected) > 1e-9 {
				t.Errorf("%s: got %v, want %v", tt.name, got, tt.expected)
			}
		})
	}
}
func TestV(t *testing.T) {
	tests := []struct {
		name     string
		rate     float64
		expected float64
	}{
		{"zero rate", 0, 1.0},
		{"5% rate", 0.05, 1.0 / 1.05},
		{"10% rate", 0.10, 1.0 / 1.10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rc := RateConverter{EffectiveRate: tt.rate}
			got := rc.V()
			if math.Abs(got-tt.expected) > 1e-9 {
				t.Errorf("V() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestVStar(t *testing.T) {
	rc := RateConverter{EffectiveRate: 0.05}
	j := 0.02
	// Logic: v* = (1+j) * v = (1+0.02) * (1/1.05) = 1.02 * 0.95238 = 0.97142
	expected := (1 + j) * (1 / (1 + 0.05))
	got := rc.VStar(j)

	if math.Abs(got-expected) > 1e-9 {
		t.Errorf("VStar() = %v, want %v", got, expected)
	}
}

func TestDiscount(t *testing.T) {
	rc := RateConverter{EffectiveRate: 0.05}
	v := rc.V()

	tests := []struct {
		term int
		want float64
	}{
		{0, 1.0},
		{1, v},
		{5, v * v * v * v * v},
		{10, math.Pow(v, 10)},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := rc.Discount(tt.term)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("Discount(%d) = %v, want %v", tt.term, got, tt.want)
			}
		})
	}
}

var _ DiscountFactor = (*RateConverter)(nil)

// Benchmark tests for the "1 Billion Row Challenge" mindset
func BenchmarkV(b *testing.B) {
	rc := RateConverter{EffectiveRate: 0.05}
	for b.Loop() {
		rc.V()
	}
}

func BenchmarkVStar(b *testing.B) {
	rc := RateConverter{EffectiveRate: 0.05}
	for b.Loop() {
		rc.VStar(0.02)
	}
}
