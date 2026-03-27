// Package risk provides Value at Risk (VaR) and Conditional Tail Expectation (CTE)
// for quantifying financial uncertainty from Monte Carlo simulations.
//
// # Generate a full risk report
//
//	// losses is a []float64 of simulated portfolio losses
//	report := risk.ComputeReport(losses)
//	fmt.Printf("Mean: $%.2f\n", report.Mean)
//	fmt.Printf("VaR 95%%: $%.2f\n", report.VaR95)
//	fmt.Printf("CTE 95%%: $%.2f\n", report.CTE95)
//
// # Value at Risk at a specific confidence level
//
//	var95 := risk.VaR(losses, 0.95) // 95% confidence
//	var99 := risk.VaR(losses, 0.99) // 99% confidence
//	fmt.Printf("With 95%% confidence, loss does not exceed $%.2f\n", var95)
//
// # Conditional Tail Expectation (Expected Shortfall)
//
//	cte95 := risk.CTE(losses, 0.95)
//	fmt.Printf("In the worst 5%% of scenarios, average loss is $%.2f\n", cte95)
//
// # Complete example: Monte Carlo + VaR
//
//	rg := stochastic.NewRateGeneratorWithSeed(0.05, 0.20, 0.20, 42)
//	paths := rg.GeneratePaths(100000, 10, 1.0)
//
//	losses := make([]float64, len(paths))
//	for i, path := range paths {
//	    finalRate := path[10]
//	    loss := math.Max(0, finalRate-0.05) * 1000000 // $1M notional
//	    losses[i] = loss
//	}
//
//	report := risk.ComputeReport(losses)
//	fmt.Printf("VaR 95%%: $%.2f\n", report.VaR95)
//	fmt.Printf("CTE 95%%: $%.2f\n", report.CTE95)
package risk
