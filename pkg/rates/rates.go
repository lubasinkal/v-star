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

// EffectiveToNominal converts an effective annual rate to a nominal rate
// compounded m times per period: im = m * ((1+i)^(1/m) - 1).
func EffectiveToNominal(i float64, m int) float64 {
	return float64(m) * (math.Pow(1+i, 1/float64(m)) - 1)
}

// ForceOfInterest returns the force of interest delta = ln(1+i).
func ForceOfInterest(i float64) float64 {
	return math.Log(1 + i)
}

// InterestFromForce converts a force of interest to an effective annual rate: i = e^delta - 1.
func InterestFromForce(delta float64) float64 {
	return math.Exp(delta) - 1
}

// AnnuityCertainImmediate returns the present value of an annuity-certain-immediate:
// a_angle_n = (1 - v^n) / i.
func AnnuityCertainImmediate(i float64, n int) float64 {
	if n <= 0 || i <= 0 {
		return 0
	}
	v := 1 / (1 + i)
	return (1 - math.Pow(v, float64(n))) / i
}

// AnnuityCertainDue returns the present value of an annuity-certain-due:
// adbl_angle_n = (1 - v^n) / d where d = i/(1+i).
func AnnuityCertainDue(i float64, n int) float64 {
	if n <= 0 || i <= 0 {
		return 0
	}
	v := 1 / (1 + i)
	d := i / (1 + i)
	return (1 - math.Pow(v, float64(n))) / d
}

// MacaulayDuration computes the Macaulay duration of a cash flow stream.
// cashFlows[t] is the cash flow at time t (t=1..len(cashFlows)).
// Returns sum(t * PV_t) / sum(PV_t).
func MacaulayDuration(i float64, cashFlows []float64) float64 {
	if i <= 0 || len(cashFlows) == 0 {
		return 0
	}
	v := 1 / (1 + i)
	pvTotal := 0.0
	duration := 0.0
	for t, cf := range cashFlows {
		if cf <= 0 {
			continue
		}
		pv := cf * math.Pow(v, float64(t+1))
		pvTotal += pv
		duration += float64(t+1) * pv
	}
	if pvTotal <= 0 {
		return 0
	}
	return duration / pvTotal
}

// ModifiedDuration computes the modified duration: MacaulayDuration / (1+i).
// This measures the sensitivity of bond price to interest rate changes.
func ModifiedDuration(i float64, cashFlows []float64) float64 {
	md := MacaulayDuration(i, cashFlows)
	return md / (1 + i)
}

// Convexity computes the convexity of a cash flow stream.
// Returns sum(t*(t+1)*PV_t) / ((1+i)^2 * PV_total).
func Convexity(i float64, cashFlows []float64) float64 {
	if i <= 0 || len(cashFlows) == 0 {
		return 0
	}
	v := 1 / (1 + i)
	pvTotal := 0.0
	conv := 0.0
	for t, cf := range cashFlows {
		if cf <= 0 {
			continue
		}
		pv := cf * math.Pow(v, float64(t+1))
		pvTotal += pv
		conv += float64(t+1) * float64(t+2) * pv
	}
	if pvTotal <= 0 {
		return 0
	}
	return conv / (pvTotal * (1 + i) * (1 + i))
}
