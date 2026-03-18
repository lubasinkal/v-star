package rates

// PresentValue calculates the present value of a sum assured
// PV = SumAssured * v^term where v = 1/(1+i)
func (r RateConverter) PresentValue(sumAssured float64, term int) float64 {
	if term <= 0 {
		return sumAssured
	}
	v := r.V()
	return sumAssured * pow(v, term)
}

// PresentValueStar calculates the present value using the v-star discount factor
// PV = SumAssured * v*^term where v* = (1+j)*v
func (r RateConverter) PresentValueStar(sumAssured float64, term int, j float64) float64 {
	if term <= 0 {
		return sumAssured
	}
	vStar := r.VStar(j)
	return sumAssured * pow(vStar, term)
}

// pow is a helper to calculate power without math import (already imported in rates.go)
func pow(base float64, exp int) float64 {
	result := 1.0
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}
