package reserves

import (
	"github.com/lubasinkal/v-star/pkg/annuities"
	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
)

type PolicySpec struct {
	Age        int
	Term       int
	SumAssured float64
	Premium    float64
}

func NetPremiumReserve(policy PolicySpec, discount rates.DiscountFactor, mort mortality.MortalityTable) float64 {
	age := policy.Age
	term := policy.Term
	sa := policy.SumAssured
	prem := policy.Premium

	if age < 0 || term <= 0 || sa <= 0 || prem <= 0 {
		return 0
	}

	converter, ok := discount.(*rates.RateConverter)
	if !ok {
		return netPremiumReserveGeneric(policy, discount, mort)
	}
	annuityCalc := annuities.New(converter, mort)

	ax := annuityCalc.TermImmediate(age, term, 1.0)
	if ax <= 0 {
		return 0
	}

	annualPremium := sa / ax
	if annualPremium <= 0 {
		return 0
	}

	reserve := 0.0
	currentAge := age
	currentTerm := term

	for year := 1; year <= term; year++ {
		survivalProb := mort.Px(currentAge, 1)
		discountFactor := converter.Discount(1)

		futureLiability := annuityCalc.TermImmediate(currentAge+1, currentTerm-1, sa)
		futurePremium := annualPremium * annuityCalc.TermImmediate(currentAge+1, currentTerm-1, 1.0)

		netLiability := futureLiability - futurePremium
		reserve = (reserve+netLiability)*discountFactor/survivalProb - annualPremium

		currentAge++
		currentTerm--
	}

	return reserve
}

func netPremiumReserveGeneric(policy PolicySpec, discount rates.DiscountFactor, mort mortality.MortalityTable) float64 {
	age := policy.Age
	term := policy.Term
	sa := policy.SumAssured

	ax := 0.0
	for t := 1; t <= term; t++ {
		px := mort.Px(age, t)
		v := discount.Discount(t)
		ax += px * v
	}

	if ax <= 0 {
		return 0
	}

	annualPremium := sa / ax

	reserve := 0.0
	currentAge := age
	currentTerm := term

	for year := 1; year <= term; year++ {
		survivalProb := mort.Px(currentAge, 1)
		discountFactor := discount.Discount(1)

		futureAx := 0.0
		for t := 1; t <= currentTerm-1; t++ {
			px := mort.Px(currentAge+1, t)
			v := discount.Discount(t)
			futureAx += px * v
		}

		futurePremium := annualPremium * futureAx
		futureLiability := sa * futureAx

		netLiability := futureLiability - futurePremium
		if survivalProb > 0 && discountFactor > 0 {
			reserve = (reserve+netLiability)*discountFactor/survivalProb - annualPremium
		}

		currentAge++
		currentTerm--
	}

	return reserve
}

func GrossPremiumReserve(policy PolicySpec, expenses float64, discount rates.DiscountFactor, mort mortality.MortalityTable) float64 {
	npr := NetPremiumReserve(policy, discount, mort)

	converter, ok := discount.(*rates.RateConverter)
	if !ok {
		return npr + expenses
	}

	annuityCalc := annuities.New(converter, mort)
	expenseAnnuity := annuityCalc.TermImmediate(policy.Age, policy.Term, expenses)
	expenseReserve := expenseAnnuity - expenses

	return npr + expenseReserve
}

func ProspectiveReserve(age int, term int, sa float64, prem float64, discount rates.DiscountFactor, mort mortality.MortalityTable) float64 {
	if age < 0 || term <= 0 || sa <= 0 || prem <= 0 {
		return 0
	}

	converter, ok := discount.(*rates.RateConverter)
	if !ok {
		return 0
	}

	annuityCalc := annuities.New(converter, mort)

	futureBenefits := annuityCalc.TermImmediate(age, term, sa)
	futurePremiums := annuityCalc.TermImmediate(age, term, prem)

	return futureBenefits - futurePremiums
}

func RetrospectiveReserve(age int, term int, sa float64, prem float64, discount rates.DiscountFactor, mort mortality.MortalityTable) float64 {
	if age < 0 || term <= 0 || sa <= 0 || prem <= 0 {
		return 0
	}

	converter, ok := discount.(*rates.RateConverter)
	if !ok {
		return 0
	}

	annuityCalc := annuities.New(converter, mort)

	accumulated := 0.0
	currentAge := age

	for y := 1; y <= term; y++ {
		px := mort.Px(currentAge, 1)
		v := converter.Discount(1)
		accumulated = (accumulated + prem) * v / px
		currentAge++
	}

	futureLiability := annuityCalc.TermImmediate(age, term, sa)

	return accumulated - futureLiability
}
