// Package annuities computes annuity values and insurance net single premiums.
// Uses a DiscountFactor and a MortalityTable to value life-contingent cash flows.
//
// # AnnuityCalculator
//
//	discount := rates.NewRateConverter(0.05)
//	mort, err := mortality.LoadCSV("mortality.csv")
//	ann := annuities.NewAnnuityCalculator(discount, mort)
//
//	// Whole life annuity: $1,000/year at age 65
//	value := ann.WholeLifeImmediate(65, 1000)
//	value = ann.WholeLifeDue(65, 1000)
//
//	// Term annuity: $1,000/year for 20 years at age 40
//	value = ann.TermImmediate(40, 20, 1000)
//	value = ann.TermDue(40, 20, 1000)
//
//	// Deferred annuity: payments start after delay
//	value = ann.DeferredWholeLife(50, 10, 1000)
//	value = ann.DeferredTerm(40, 5, 15, 1000)
//
//	// Life insurance net single premiums
//	ax := ann.WholeLifeNSP(30, 100000)
//	aterm := ann.TermNSP(30, 20, 100000)
//	aend := ann.EndowmentNSP(30, 20, 100000)
//
//	// Quick approximation (no mortality table)
//	approx := annuities.ApproxWholeLifeImmediate(65, 30, 1000, 0.05, mort)
//
// # ContingencyCalculator interface
//
// Implement ContingencyCalculator to provide alternative methods
// (fractional durations, select tables). AnnuityCalculator satisfies it.
package annuities
