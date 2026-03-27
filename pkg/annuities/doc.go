// Package annuities computes annuity values using a discount factor and mortality table.
//
// # Create an annuity calculator
//
//	converter := rates.NewRateConverter(0.05)
//	table, _ := mortality.LoadCSV("mortality.csv")
//	ann := annuities.NewAnnuityCalculator(converter, table)
//
// # Whole life annuity (payments while alive)
//
//	// Annuity-immediate: payments at end of each year
//	value := ann.WholeLifeImmediate(65, 1000) // age 65, $1000/year
//
//	// Annuity-due: payments at start of each year
//	value = ann.WholeLifeDue(65, 1000)
//
// # Term annuity (payments for fixed term)
//
//	// 20-year term annuity-immediate at age 40
//	value = ann.TermImmediate(40, 20, 1000)
//
//	// 20-year term annuity-due
//	value = ann.TermDue(40, 20, 1000)
//
// # Deferred annuity (payments start after delay)
//
//	// Payments start after 10 years, then continue for life
//	value = ann.DeferredWholeLife(50, 10, 1000)
//
//	// Payments start after 5 years, then continue for 15 years
//	value = ann.DeferredTerm(40, 5, 15, 1000)
//
// # Life insurance net single premiums
//
//	ax := ann.WholeLifeNSP(30, 100000)       // A_30: whole life
//	aterm := ann.TermNSP(30, 20, 100000)     // A^1_{30:20}: 20-year term
//	aend := ann.EndowmentNSP(30, 20, 100000) // A_{30:20}: endowment
//
// # Quick approximation (no mortality table needed)
//
//	value := annuities.ApproxWholeLifeImmediate(65, 30, 1000, 0.05, table)
package annuities
