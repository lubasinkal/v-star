package rates

import "math"

// DiscountFactor is the interface for discount factor calculations.
// Implementations must return the discount factor v^term for a given term.
type DiscountFactor interface {
	Discount(term int) float64
}

// RateConverter performs interest rate conversions and present value calculations.
// It pre-computes a discount table for terms 0 through 100 for fast lookups.
type RateConverter struct {
	EffectiveRate float64
	discountTable []float64
}

// NewRateConverter creates a RateConverter for the given effective annual rate.
// It pre-computes discount factors v^t for t in [0, 100] where v = 1/(1+i).
func NewRateConverter(effectiveRate float64) *RateConverter {
	v := 1 / (1 + effectiveRate)
	table := make([]float64, 101)
	table[0] = 1.0
	for i := 1; i <= 100; i++ {
		table[i] = table[i-1] * v
	}
	return &RateConverter{
		EffectiveRate: effectiveRate,
		discountTable: table,
	}
}

// Discount returns the discount factor v^term.
// Uses a pre-computed table for terms 0-100, falls back to loop for larger terms.
func (r *RateConverter) Discount(term int) float64 {
	if term <= 0 {
		return 1
	}
	if term < len(r.discountTable) {
		return r.discountTable[term]
	}
	v := r.V()
	result := 1.0
	for range term {
		result *= v
	}
	return result
}

// V returns the one-period discount factor v = 1/(1+i).
func (r *RateConverter) V() float64 {
	return 1 / (1 + r.EffectiveRate)
}

// VStar returns the v-star factor v* = (1+j) * v,
// used when premiums compound at rate j while being discounted at rate i.
func (r *RateConverter) VStar(j float64) float64 {
	return (1 + j) * r.V()
}

// PresentValue returns sumAssured * v^term.
func (r *RateConverter) PresentValue(sumAssured float64, term int) float64 {
	if term <= 0 {
		return sumAssured
	}
	if term < len(r.discountTable) {
		return sumAssured * r.discountTable[term]
	}
	v := r.V()
	result := sumAssured
	for range term {
		result *= v
	}
	return result
}

// PresentValueStar returns sumAssured * (v*)^term using the v-star discount factor.
func (r *RateConverter) PresentValueStar(sumAssured float64, term int, j float64) float64 {
	if term <= 0 {
		return sumAssured
	}
	vStar := r.VStar(j)
	return sumAssured * math.Pow(vStar, float64(term))
}

// NominalToEffective converts a nominal rate compounded m times per period
// to an effective annual rate: i = (1 + im/m)^m - 1.
func NominalToEffective(im float64, m int) float64 {
	return math.Pow(1+(im/float64(m)), float64(m)) - 1
}
