package rates

import (
	"math"
)

type DiscountFactor interface {
	Discount(term int) float64
}

type RateConverter struct {
	EffectiveRate float64 // 'i'
}

func (r RateConverter) Discount(term int) float64 {
	if term <= 0 {
		return 1
	}
	v := r.V()
	result := 1.0
	for i := 0; i < term; i++ {
		result *= v
	}
	return result
}

// V calculates the standard discount factor: v = 1 / (1 + i)
func (r RateConverter) V() float64 {
	return 1 / (1 + r.EffectiveRate)
}

// VStar implements the class joke: v* = (1 + j) * v
// Where j is a compounding growth rate.
func (r RateConverter) VStar(j float64) float64 {
	return (1 + j) * r.V()
}

// NominalToEffective converts a nominal rate compounded 'm' times per annum
func NominalToEffective(im float64, m int) float64 {
	return math.Pow(1+(im/float64(m)), float64(m)) - 1
}
