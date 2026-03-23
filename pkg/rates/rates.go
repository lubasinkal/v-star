package rates

import "math"

type DiscountFactor interface {
	Discount(term int) float64
}

type RateConverter struct {
	EffectiveRate float64
	discountTable []float64
}

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

func (r RateConverter) Discount(term int) float64 {
	if term <= 0 {
		return 1
	}
	if term < len(r.discountTable) {
		return r.discountTable[term]
	}
	v := r.V()
	result := 1.0
	for i := 0; i < term; i++ {
		result *= v
	}
	return result
}

func (r RateConverter) V() float64 {
	return 1 / (1 + r.EffectiveRate)
}

func (r RateConverter) VStar(j float64) float64 {
	return (1 + j) * r.V()
}

func (r RateConverter) PresentValue(sumAssured float64, term int) float64 {
	if term <= 0 {
		return sumAssured
	}
	if term < len(r.discountTable) {
		return sumAssured * r.discountTable[term]
	}
	v := r.V()
	result := sumAssured
	for i := 0; i < term; i++ {
		result *= v
	}
	return result
}

func (r RateConverter) PresentValueStar(sumAssured float64, term int, j float64) float64 {
	if term <= 0 {
		return sumAssured
	}
	vStar := r.VStar(j)
	return sumAssured * math.Pow(vStar, float64(term))
}

func NominalToEffective(im float64, m int) float64 {
	return math.Pow(1+(im/float64(m)), float64(m)) - 1
}
