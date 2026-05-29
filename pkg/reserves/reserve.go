package reserves

import (
	"github.com/lubasinkal/v-star/pkg/annuities"
	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
)

// PolicySpec defines the parameters for a life insurance policy.
type PolicySpec struct {
	Age        int
	Term       int
	SumAssured float64
	Premium    float64
}

// NetPremiumReserve calculates the net premium reserve using the prospective method.
// The net annual premium is determined internally so that the policy is fair at inception;
// the Premium field of PolicySpec is not used by this function.
func NetPremiumReserve(policy PolicySpec, discount rates.DiscountFactor, mort mortality.MortalityTable) float64 {
	age := policy.Age
	term := policy.Term
	sa := policy.SumAssured

	if age < 0 || term <= 0 || sa <= 0 {
		return 0
	}

	v := discount.Discount(1)
	unitAnnuities := make([]float64, term+1)
	for k := term - 1; k >= 0; k-- {
		p := mort.Px(age+k, 1)
		unitAnnuities[k] = p * v * (1 + unitAnnuities[k+1])
	}

	if unitAnnuities[0] <= 0 {
		return 0
	}

	annualPremium := sa / unitAnnuities[0]

	reserve := 0.0
	for year := 1; year <= term; year++ {
		p := mort.Px(age+year-1, 1)
		if p <= 0 {
			break
		}
		netLiability := (sa - annualPremium) * unitAnnuities[year]
		reserve = (reserve+netLiability)*v/p - annualPremium
	}

	return reserve
}

// GrossPremiumReserve calculates the gross premium reserve (NPR + expense reserve).
func GrossPremiumReserve(policy PolicySpec, expenses float64, discount rates.DiscountFactor, mort mortality.MortalityTable) float64 {
	npr := NetPremiumReserve(policy, discount, mort)

	annuityCalc := annuities.NewAnnuityCalculator(discount, mort)
	expenseAnnuity := annuityCalc.TermImmediate(policy.Age, policy.Term, expenses)
	expenseReserve := expenseAnnuity - expenses

	return npr + expenseReserve
}

// ProspectiveReserve calculates the reserve as future benefits minus future premiums.
//
// Uses the term insurance NSP (death benefit weighted by mortality probability)
// for benefits and a term annuity-due (payments at start of year) for premiums.
func ProspectiveReserve(policy PolicySpec, discount rates.DiscountFactor, mort mortality.MortalityTable) float64 {
	age := policy.Age
	term := policy.Term
	sa := policy.SumAssured
	prem := policy.Premium

	if age < 0 || term <= 0 || sa <= 0 || prem <= 0 {
		return 0
	}

	calc := annuities.NewAnnuityCalculator(discount, mort)

	futureBenefits := calc.TermNSP(age, term, sa)
	futurePremiums := calc.TermDue(age, term, prem)

	return futureBenefits - futurePremiums
}

// RetrospectiveReserve calculates the reserve as accumulated premiums minus past liabilities.
//
// Uses the term insurance NSP (death benefit weighted by mortality probability)
// for the liability term instead of an annuity.
func RetrospectiveReserve(policy PolicySpec, discount rates.DiscountFactor, mort mortality.MortalityTable) float64 {
	age := policy.Age
	term := policy.Term
	sa := policy.SumAssured
	prem := policy.Premium

	if age < 0 || term <= 0 || sa <= 0 || prem <= 0 {
		return 0
	}

	v1 := discount.Discount(1)

	accumulated := 0.0
	currentAge := age
	for y := 1; y <= term; y++ {
		px := mort.Px(currentAge, 1)
		if px <= 0 {
			break
		}
		accumulated = (accumulated + prem) * v1 / px
		currentAge++
	}

	calc := annuities.NewAnnuityCalculator(discount, mort)
	futureLiability := calc.TermNSP(age, term, sa)

	return accumulated - futureLiability
}
