package main

import (
	"fmt"
	"math"

	"github.com/lubasinkal/v-star/pkg/risk"
	"github.com/lubasinkal/v-star/pkg/stochastic"
)

func main() {
	fmt.Println("=== v-star Risk Analysis Demo ===")
	fmt.Println()

	// Simulate 100,000 interest rate paths using geometric Brownian motion
	rg := stochastic.NewRateGeneratorWithSeed(0.05, 0.20, 0.20, 42)
	numPaths := 100000
	steps := 10

	fmt.Printf("Generating %d Monte Carlo paths (%d steps each)...\n", numPaths, steps)
	paths := rg.GeneratePaths(numPaths, steps, 1.0)

	// Compute portfolio losses
	// Model: $1M portfolio invested at initial rate
	// Portfolio value = notional * (initial_rate / final_rate)
	// This approximates the inverse relationship between rates and bond prices
	notional := 1000000.0
	initialRate := 0.05
	losses := make([]float64, numPaths)
	for i, path := range paths {
		finalRate := path[steps]
		if finalRate <= 0 {
			finalRate = 0.001 // prevent division by zero
		}
		portfolioValue := notional * (initialRate / finalRate)
		loss := math.Max(0, notional-portfolioValue)
		losses[i] = loss
	}

	// Generate risk report
	report := risk.ComputeReport(losses)

	fmt.Println()
	fmt.Println("=== Risk Report ===")
	fmt.Printf("Mean Loss:     $%.2f\n", report.Mean)
	fmt.Printf("Std Deviation: $%.2f\n", report.StdDev)
	fmt.Printf("Min Loss:      $%.2f\n", report.Min)
	fmt.Printf("Max Loss:      $%.2f\n", report.Max)
	fmt.Println()
	fmt.Println("=== Value at Risk ===")
	fmt.Printf("VaR (95%%): $%.2f\n", report.VaR95)
	fmt.Printf("VaR (99%%): $%.2f\n", report.VaR99)
	fmt.Println()
	fmt.Println("=== Conditional Tail Expectation ===")
	fmt.Printf("CTE (95%%): $%.2f\n", report.CTE95)
	fmt.Printf("CTE (99%%): $%.2f\n", report.CTE99)
	fmt.Println()
	fmt.Println("Interpretation:")
	fmt.Printf("  With 95%% confidence, loss does not exceed $%.0f\n", report.VaR95)
	fmt.Printf("  In the worst 5%% of scenarios, average loss is $%.0f\n", report.CTE95)
	fmt.Printf("  In the worst 1%% of scenarios, average loss is $%.0f\n", report.CTE99)
	fmt.Println()
	fmt.Println("This models a $1M portfolio where value moves inversely to interest rates.")
	fmt.Println("Loss occurs when rates rise above the initial 5% level.")
}
