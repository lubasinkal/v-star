// Package profit provides profit testing and cashflow projection for
// life insurance policies.
//
// Profit testing projects the annual profit emergence of a policy over
// its term, allowing actuaries to assess profitability, set premiums,
// and evaluate sensitivity to assumptions.
//
// # Quick start
//
//	mort, _ := mortality.LoadCSV("utilsoc-qx.csv")
//	assumptions := profit.Assumptions{
//	    Mortality:     mort,
//	    EarnedRate:    0.05,
//	    DiscountRate:  0.08,
//	    Expenses:      500,    // per policy at issue
//	    RenewalExpense: 50,    // per policy per year
//	    CommissionRate: 0.05,  // 5% of premium
//	}
//	policy := profit.Policy{
//	    Age:        30,
//	    Term:       20,
//	    SumAssured: 100000,
//	    Premium:    5000,
//	}
//	results := profit.Run(policy, assumptions)
//	fmt.Printf("Profit margin: %.2f%%\n", results.ProfitMargin*100)
//	fmt.Printf("Payback year: %d\n", results.PaybackYear)
//	fmt.Printf("PV of profits: %.2f\n", results.PVOfProfits)
package profit

import (
	"math"

	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
)

// Assumptions defines the economic and expense assumptions for a profit test.
type Assumptions struct {
	// Mortality table for survival / death probabilities.
	Mortality mortality.MortalityTable

	// EarnedRate is the rate of investment return earned on assets (i).
	// Used to accumulate cashflows within each projection year.
	EarnedRate float64

	// DiscountRate is the risk discount rate (r) used to compute
	// the present value of the profit signature.
	DiscountRate float64

	// Expenses is the acquisition expense per policy, incurred at issue (year 0).
	Expenses float64

	// RenewalExpense is the ongoing maintenance expense per policy per year.
	RenewalExpense float64

	// CommissionRate is the percentage of annual premium paid as commission.
	// Applied for CommissionYears years (0 = every year of the term).
	CommissionRate float64

	// CommissionYears limits commission to the first N years.
	// 0 means commission is paid every year the premium is received.
	CommissionYears int

	// ReserveEnabled determines whether reserves are projected.
	// When true, the net premium reserve is calculated each year and
	// the change in reserve is reflected in the profit signature.
	// Requires PremiumUSD to be set (and must be the net premium, or
	// a warning will be printed).
	ReserveEnabled bool
}

// Policy defines the contract parameters for a single policy.
type Policy struct {
	Age        int
	Term       int
	SumAssured float64
	Premium    float64 // annual premium (gross)
}

// Results holds the full output of a profit test run.
type Results struct {
	// ProfitSignature[t] is the net cashflow emerging at end of year t+1,
	// per policy issued. Positive means profit; negative means loss.
	ProfitSignature []float64

	// CumulativeProfit[t] is the sum of ProfitSignature[0..t].
	CumulativeProfit []float64

	// ProfitVector[t] is the profit at end of year t+1 per policy in
	// force at the start of that year (before multiplying by survival).
	ProfitVector []float64

	// PVOfProfits is the present value of the profit signature
	// discounted at the risk discount rate.
	PVOfProfits float64

	// PVOfPremiums is the present value of all premium payments,
	// discounted at the risk discount rate with mortality.
	PVOfPremiums float64

	// ProfitMargin is PVOfProfits / PVOfPremiums.
	// A common target is 10–20% for term insurance.
	ProfitMargin float64

	// IRR is the discount rate that makes PVOfProfits = 0.
	// Computed via binary search. Set to -1 if not found.
	IRR float64

	// PaybackYear is the first year (1-indexed) where the cumulative
	// discounted profit becomes positive. 0 if never.
	PaybackYear int
}

// Run executes a profit test for the given policy and assumptions.
// Returns a Results struct with the full profit signature and metrics.
func Run(policy Policy, assumptions Assumptions) Results {
	term := policy.Term
	if term <= 0 || policy.Age < 0 || assumptions.Mortality == nil {
		return Results{}
	}

	mort := assumptions.Mortality
	earnedRate := assumptions.EarnedRate
	discountRate := assumptions.DiscountRate

	// Pre-compute discount factors
	disc := rates.NewRateConverter(discountRate)

	// Pre-compute survival probabilities
	px := make([]float64, term+1)
	px[0] = 1.0 // probability in force at issue
	for t := 1; t <= term; t++ {
		px[t] = px[t-1] * mort.Px(policy.Age+t-1, 1)
	}

	// Pre-compute reserve values per year (net premium reserve)
	reserve := make([]float64, term+1)
	reserve[term] = 0 // no reserve after term ends
	if assumptions.ReserveEnabled && policy.Premium > 0 {
		r := rates.NewRateConverter(earnedRate)
		reserve = computeReserves(policy, r, mort)
	}

	// Build profit vector and signature
	profitVector := make([]float64, term)
	profitSignature := make([]float64, term)
	cumulative := make([]float64, term)
	pvProfits := 0.0
	pvPremiums := 0.0

	for t := range term {
		age := policy.Age + t
		qx := mort.Qx(age)
		p := mort.Px(age, 1) // 1-year survival

		// --- Cashflows at start of year t ---

		// Premium received (start of year, if policy in force)
		premiumIncome := policy.Premium

		// Commission on premium
		commissionRate := assumptions.CommissionRate
		if assumptions.CommissionYears > 0 && t >= assumptions.CommissionYears {
			commissionRate = 0
		}
		commission := premiumIncome * commissionRate

		// Acquisition expense (year 0 only) or renewal expense
		yearExpense := assumptions.RenewalExpense
		if t == 0 {
			yearExpense += assumptions.Expenses // acquisition at issue
		}

		// Opening reserve
		openingReserve := reserve[t]

		// Cash at start of year, before interest
		startCash := openingReserve + premiumIncome - commission - yearExpense

		// Investment income earned during year
		investmentIncome := startCash * earnedRate

		// --- Cashflows at end of year t ---

		// Claims paid
		claims := policy.SumAssured * qx

		// Closing reserve for survivors
		closingReserve := reserve[t+1] * p

		// Profit at end of year (per policy in force at start of year)
		profitEndYear := startCash + investmentIncome - claims - closingReserve

		// Per-policy-in-force profit
		profitVector[t] = profitEndYear

		// Profit per policy issued (discounted for survivorship)
		profitSig := profitEndYear * px[t]
		profitSignature[t] = profitSig

		// Cumulative undiscounted
		if t == 0 {
			cumulative[t] = profitSig
		} else {
			cumulative[t] = cumulative[t-1] + profitSig
		}

		// Discounted profit at risk discount rate (end of year)
		pvProfits += profitSig * disc.Discount(t+1)

		// PV of premium (at start of year = discounted 1 less period)
		pvPremiums += policy.Premium * px[t] * disc.Discount(t)
	}

	// Profit margin
	profitMargin := 0.0
	if pvPremiums > 0 {
		profitMargin = pvProfits / pvPremiums
	}

	// Payback year (discounted)
	paybackYear := 0
	cumDiscounted := 0.0
	for t := range term {
		cumDiscounted += profitSignature[t] * disc.Discount(t+1)
		if cumDiscounted > 0 && paybackYear == 0 {
			paybackYear = t + 1
		}
	}

	// IRR (binary search)
	irr := findIRR(profitSignature, px, assumptions.DiscountRate)

	return Results{
		ProfitSignature:  profitSignature,
		CumulativeProfit: cumulative,
		ProfitVector:     profitVector,
		PVOfProfits:      pvProfits,
		PVOfPremiums:     pvPremiums,
		ProfitMargin:     profitMargin,
		IRR:              irr,
		PaybackYear:      paybackYear,
	}
}

// computeReserves calculates the net premium reserve for each policy year.
// Returns a slice where reserve[t] is the reserve per policy in force at
// the start of year t (0-indexed, t=0 is issue, t=term is 0).
func computeReserves(policy Policy, earned *rates.RateConverter, mort mortality.MortalityTable) []float64 {
	term := policy.Term
	age := policy.Age
	sa := policy.SumAssured

	// Net single premium (NSP) for term insurance
	nsp := nspTerm(age, term, sa, earned, mort)

	// Net annual premium = NSP / annuity-due
	annuityDue := annuityDueTerm(age, term, 1.0, earned, mort)
	netPremium := 0.0
	if annuityDue > 0 {
		netPremium = nsp / annuityDue
	}

	// Prospective reserve: V_t = NSP(age+t, term-t) - netPremium * annuity(age+t, term-t)
	reserve := make([]float64, term+1)
	reserve[term] = 0
	for t := range term {
		if t >= term {
			break
		}
		remaining := term - t
		// For a pure endowment, NSP_remaining = discount * px
		// For term insurance, NSP_remaining = sum of discounted death benefits
		nspRemaining := nspTerm(age+t, remaining, sa, earned, mort)
		annuityRemaining := annuityDueTerm(age+t, remaining, 1.0, earned, mort)
		reserve[t] = nspRemaining - netPremium*annuityRemaining
		if reserve[t] < 0 {
			reserve[t] = 0
		}
	}

	return reserve
}

// nspTerm computes the net single premium for a term insurance:
// NSP = SA * sum_{k=0}^{term-1} qx(age+k) * px(age, k) * v^(k+1)
func nspTerm(age, term int, sa float64, disc *rates.RateConverter, mort mortality.MortalityTable) float64 {
	if age < 0 || term <= 0 || sa <= 0 {
		return 0
	}
	nsp := 0.0
	px := 1.0
	for k := range term {
		q := mort.Qx(age + k)
		nsp += sa * q * px * disc.Discount(k+1)
		px *= (1 - q)
	}
	return nsp
}

// annuityDueTerm computes the present value of an annuity-due:
// ad = sum_{k=0}^{term-1} px(age, k) * v^k
func annuityDueTerm(age, term int, amount float64, disc *rates.RateConverter, mort mortality.MortalityTable) float64 {
	if age < 0 || term <= 0 || amount <= 0 {
		return 0
	}
	pv := 0.0
	px := 1.0
	for k := range term {
		pv += amount * px * disc.Discount(k)
		px *= (1 - mort.Qx(age+k))
	}
	return pv
}

// findIRR computes the internal rate of return using Newton-Raphson
// with analytical derivative, guarded by bisection.
//
// The IRR is the discount rate r that makes the net present value of
// the profit signature zero:
//
//	NPV(r) = Σ CF_t / (1+r)^(t+1) = 0
//
// Uses the analytical derivative (Newton's method):
//
//	NPV'(r) = -Σ CF_t * (t+1) / (1+r)^(t+2)
//
// Returns -1 when no finite IRR exists (all cash flows same sign,
// or NPV does not cross zero).
func findIRR(profitSignature []float64, px []float64, guessRate float64) float64 {
	const (
		maxIter = 100
		tol     = 1e-10
		maxRate = 1e6 // 100,000,000% — effectively infinite
	)

	// Find first sign change to determine if IRR is possible.
	// If all cash flows have the same sign, no finite IRR exists.
	hasPositive := false
	hasNegative := false
	for _, cf := range profitSignature {
		if cf > 0 {
			hasPositive = true
		}
		if cf < 0 {
			hasNegative = true
		}
	}
	if !hasPositive || !hasNegative {
		return -1 // No sign change — no finite IRR
	}

	// Compute NPV and its analytical derivative at a given rate.
	fn := func(rate float64) (f, fp float64) {
		v := 1.0 / (1.0 + rate) // v = 1/(1+r)
		vPow := v               // v^(t+1) for t=0
		for t, cf := range profitSignature {
			f += cf * vPow
			fp -= cf * float64(t+1) * vPow / (1.0 + rate)
			vPow *= v
		}
		return f, fp
	}

	// Find a bracket [low, high] where NPV changes sign.
	// Start from the guess rate and expand outward exponentially.
	low := 0.0
	high := guessRate
	if high <= 0 {
		high = 0.05 // default 5%
	}

	fLow, _ := fn(low)
	fHigh, _ := fn(high)

	// Guard: if our initial bracket straddles zero, we're done bracketing.
	// Otherwise expand high until sign changes or we hit maxRate.
	if fLow*fHigh > 0 {
		// fLow and fHigh have the same sign — expand high.
		// fLow > 0 by construction (hasPositive && hasNegative, and
		// sum(CF_t) > 0 means NPV(0) > 0).
		for math.Abs(fHigh) > tol && high < maxRate && fLow*fHigh > 0 {
			high *= 2
			fHigh, _ = fn(high)
		}
		if high >= maxRate || fLow*fHigh > 0 {
			return -1 // Could not find bracket
		}
	}

	// Newton-Raphson with bisection guarding.
	// At each step, Newton proposes x_new. If it leaves the bracket
	// or overshoots, we bisect instead.
	x := (low + high) / 2
	for range maxIter {
		f, fp := fn(x)
		if math.Abs(f) < tol {
			return x
		}
		if math.Abs(fp) < 1e-16 {
			break // Derivative too small — can't Newton
		}

		xNew := x - f/fp

		// Guard: Newton step must stay inside bracket; if not, bisect.
		if xNew <= low || xNew >= high {
			xNew = (low + high) / 2
		}

		fNew, _ := fn(xNew)

		// Update bracket: one side keeps the sign, one side has the root.
		if fNew*fLow > 0 {
			low = xNew
			fLow = fNew
		} else {
			high = xNew
			fHigh = fNew
		}

		x = xNew

		if high-low < tol {
			return (low + high) / 2
		}
	}

	return -1 // Not converged
}
