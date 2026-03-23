package annuities

import (
	"math"

	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
)

type AnnuityCalculator struct {
	discount rates.DiscountFactor
	mort     mortality.MortalityTable
}

func New(discount rates.DiscountFactor, mort mortality.MortalityTable) *AnnuityCalculator {
	return &AnnuityCalculator{
		discount: discount,
		mort:     mort,
	}
}

func (a *AnnuityCalculator) WholeLifeImmediate(age int, amount float64) float64 {
	if age < 0 || amount <= 0 {
		return 0
	}
	sum := 0.0
	for t := 1; ; t++ {
		probSurvive := a.mort.Px(age, t)
		if probSurvive <= 0 {
			break
		}
		discount := a.discount.Discount(t)
		sum += probSurvive * discount
	}
	return amount * sum
}

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

func (a *AnnuityCalculator) WholeLifeDue(age int, amount float64) float64 {
	if age < 0 || amount <= 0 {
		return 0
	}
	sum := 0.0
	for t := 0; ; t++ {
		probSurvive := a.mort.Px(age, t)
		if t > 0 && probSurvive <= 0 {
			break
		}
		discount := a.discount.Discount(t)
		sum += probSurvive * discount
	}
	return amount * sum
}

func (a *AnnuityCalculator) TermDue(age int, term int, amount float64) float64 {
	if age < 0 || term <= 0 || amount <= 0 {
		return 0
	}
	sum := 0.0
	for t := 0; t < term; t++ {
		probSurvive := a.mort.Px(age, t)
		discount := a.discount.Discount(t)
		sum += probSurvive * discount
	}
	return amount * sum
}

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

type DeferredAnnuityCalculator interface {
	DeferredWholeLife(age int, deferment int, amount float64) float64
	DeferredTerm(age int, deferment int, term int, amount float64) float64
}

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
