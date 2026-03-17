package rates

import (
	"math"
)

// RateConverter holds effective interest rates
type RateConverter struct {
	EffectiveRate float64 // 'i'
}

// V calculates the standard discount factor: v = 1 / (1 + i)
func (r RateConverter) V() float64 {
	return 1 / (1 + r.EffectiveRate)
}

// VStar implements the class joke: v* = (1 + j) / v
// Where j is a compounding growth rate.
func (r RateConverter) VStar(j float64) float64 {
	return (1 + j) / r.V()
}

// NominalToEffective converts a nominal rate compounded 'm' times per annum
func NominalToEffective(im float64, m int) float64 {
	return math.Pow(1+(im/float64(m)), float64(m)) - 1
}
