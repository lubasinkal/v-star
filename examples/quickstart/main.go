package main

import (
	"fmt"

	"github.com/lubasinkal/v-star/pkg/rates"
)

func main() {
	// Create a rate converter at 5% effective annual interest
	converter := rates.NewRateConverter(0.05)

	// Present value of $100,000 payable in 20 years
	pv := converter.PresentValue(100000, 20)
	fmt.Printf("PV of $100,000 in 20 years at 5%%: $%.2f\n", pv)

	// Standard discount factor v = 1/(1+i)
	fmt.Printf("Discount factor (v):  %.6f\n", converter.V())

	// V-Star factor: v* = (1+j) * v where j = premium growth rate
	j := 0.02
	fmt.Printf("V-Star factor (j=2%%): %.6f\n", converter.VStar(j))

	// Annuity-certain: present value of $1,000/year for 20 years
	annuity := rates.AnnuityCertainImmediate(0.05, 20)
	fmt.Printf("Annuity-certain ($1/yr, 20yr): $%.4f\n", annuity*1000)

	// Force of interest
	delta := rates.ForceOfInterest(0.05)
	fmt.Printf("Force of interest (δ): %.6f\n", delta)

	// Duration of a cash flow stream (e.g., 5-year bond, 5% coupon, 5% yield)
	cashFlows := []float64{50, 50, 50, 50, 1050}
	macDur := rates.MacaulayDuration(0.05, cashFlows)
	modDur := rates.ModifiedDuration(0.05, cashFlows)
	fmt.Printf("Macaulay Duration: %.4f years\n", macDur)
	fmt.Printf("Modified Duration: %.4f\n", modDur)
}
