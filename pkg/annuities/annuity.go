package annuities

import (
	"math"

	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
)

// AnnuityCalculator computes annuity values using a discount factor and mortality table.
type AnnuityCalculator struct {
	discount rates.DiscountFactor
	mort     mortality.MortalityTable
}

// NewAnnuityCalculator creates an AnnuityCalculator from a DiscountFactor and MortalityTable.
func NewAnnuityCalculator(discount rates.DiscountFactor, mort mortality.MortalityTable) *AnnuityCalculator {
	return &AnnuityCalculator{
		discount: discount,
		mort:     mort,
	}
}

// WholeLifeImmediate computes the present value of a whole life annuity-immediate.
// Payments of amount are made at the end of each year while the annuitant is alive.
func (a *AnnuityCalculator) WholeLifeImmediate(age int, amount float64) float64 {
	if age < 0 || amount <= 0 {
		return 0
	}
	maxAge := a.mort.MaxAge()
	sum := 0.0
	for t := 1; age+t <= maxAge+1; t++ {
		probSurvive := a.mort.Px(age, t)
		if probSurvive <= 0 {
			break
		}
		discount := a.discount.Discount(t)
		sum += probSurvive * discount
	}
	return amount * sum
}

// TermImmediate computes the present value of a term annuity-immediate.
// Payments of amount are made at the end of each year for the specified term.
func (a *AnnuityCalculator) TermImmediate(age int, term int, amount float64) float64 {
	if age < 0 || term <= 0 || amount <= 0 {
		return 0
	}
	sum := 0.0
	for t := 1; t <= term; t++ {
		probSurvive := a.mort.Px(age, t)
		discount := a.discount.Discount(t)
		sum += probSurvive * discount
	}
	return amount * sum
}

// WholeLifeDue computes the present value of a whole life annuity-due.
// Payments of amount are made at the start of each year while the annuitant is alive.
func (a *AnnuityCalculator) WholeLifeDue(age int, amount float64) float64 {
	if age < 0 || amount <= 0 {
		return 0
	}
	maxAge := a.mort.MaxAge()
	sum := 0.0
	for t := 0; age+t <= maxAge+1; t++ {
		probSurvive := a.mort.Px(age, t)
		if t > 0 && probSurvive <= 0 {
			break
		}
		discount := a.discount.Discount(t)
		sum += probSurvive * discount
	}
	return amount * sum
}

// TermDue computes the present value of a term annuity-due.
// Payments of amount are made at the start of each year for the specified term.
func (a *AnnuityCalculator) TermDue(age int, term int, amount float64) float64 {
	if age < 0 || term <= 0 || amount <= 0 {
		return 0
	}
	sum := 0.0
	for t := range term {
		probSurvive := a.mort.Px(age, t)
		discount := a.discount.Discount(t)
		sum += probSurvive * discount
	}
	return amount * sum
}

// DeferredWholeLife computes the present value of a deferred whole life annuity.
// Payments begin after deferment years, contingent on survival to that point.
func (a *AnnuityCalculator) DeferredWholeLife(age int, deferment int, amount float64) float64 {
	if age < 0 || deferment <= 0 || amount <= 0 {
		return 0
	}
	probAliveAtDeferment := a.mort.Px(age, deferment)
	if probAliveAtDeferment <= 0 {
		return 0
	}
	discountAtDeferment := a.discount.Discount(deferment)
	pvDeferred := probAliveAtDeferment * discountAtDeferment
	annuityPV := a.WholeLifeImmediate(age+deferment, amount)
	return pvDeferred * annuityPV
}

// DeferredTerm computes the present value of a deferred term annuity.
// Payments begin after deferment years and continue for the specified term.
func (a *AnnuityCalculator) DeferredTerm(age int, deferment int, term int, amount float64) float64 {
	if age < 0 || deferment <= 0 || term <= 0 || amount <= 0 {
		return 0
	}
	probAliveAtDeferment := a.mort.Px(age, deferment)
	if probAliveAtDeferment <= 0 {
		return 0
	}
	discountAtDeferment := a.discount.Discount(deferment)
	pvDeferred := probAliveAtDeferment * discountAtDeferment
	annuityPV := a.TermImmediate(age+deferment, term, amount)
	return pvDeferred * annuityPV
}

// ApproxWholeLifeImmediate computes an approximate term annuity using a direct
// interest rate rather than a RateConverter. Useful for quick estimates.
func ApproxWholeLifeImmediate(age int, term int, amount float64, i float64, mort mortality.MortalityTable) float64 {
	if age < 0 || amount <= 0 || i < 0 || term <= 0 {
		return 0
	}
	v := 1 / (1 + i)
	ax := 0.0
	for t := 1; t <= term; t++ {
		px := mort.Px(age, t)
		ax += px * math.Pow(v, float64(t))
	}
	return amount * ax
}
