// Package rates provides interest rate calculations, discount factors,
// and duration/convexity for fixed income analysis.
//
// # Present value
//
//	converter := rates.NewRateConverter(0.05)
//	pv := converter.PresentValue(100000, 20) // $100k in 20 years at 5%
//	fmt.Printf("PV: $%.2f\n", pv)             // PV: $37,688.95
//
// # Discount factor
//
//	converter := rates.NewRateConverter(0.05)
//	v := converter.V()          // 1/(1+0.05) = 0.952381
//	v10 := converter.Discount(10) // v^10 = 0.613913
//
// # V-star factor (premiums growing at rate j, discounted at rate i)
//
//	converter := rates.NewRateConverter(0.05)
//	vStar := converter.VStar(0.02) // (1+0.02) * 0.952381 = 0.971429
//
// # Rate conversions
//
//	effective := rates.NominalToEffective(0.048, 12) // 4.8% nominal, monthly compounding
//	nominal := rates.EffectiveToNominal(0.05, 12)   // convert back
//	delta := rates.ForceOfInterest(0.05)            // ln(1+0.05) = 0.048790
//
// # Annuity-certain (no mortality)
//
//	a := rates.AnnuityCertainImmediate(0.05, 20) // PV of $1/yr for 20 years
//	aDue := rates.AnnuityCertainDue(0.05, 20)    // annuity-due variant
//
// # Bond duration and convexity
//
//	cashFlows := []float64{50, 50, 50, 50, 1050} // 5% coupon bond, 5 years
//	macDur := rates.MacaulayDuration(0.05, cashFlows)
//	modDur := rates.ModifiedDuration(0.05, cashFlows)
//	conv := rates.Convexity(0.05, cashFlows)
//	fmt.Printf("Macaulay: %.2f, Modified: %.2f, Convexity: %.2f\n", macDur, modDur, conv)
package rates
