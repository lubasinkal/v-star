// Package stochastic generates interest rate paths using geometric Brownian motion (GBM).
//
// # Generate a single path
//
//	rg := stochastic.NewRateGenerator(0.05, 0.02, 0.15) // initial rate, drift, volatility
//	path := rg.GeneratePath(10, 1.0)                     // 10 annual steps
//	for t, rate := range path {
//	    fmt.Printf("Year %d: %.4f%%\n", t, rate*100)
//	}
//
// # Generate many paths for Monte Carlo analysis
//
//	rg := stochastic.NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42) // deterministic seed
//	paths := rg.GeneratePaths(100000, 10, 1.0)                      // 100k paths
//
// # Compute percentiles from paths
//
//	finalRates := make([]float64, len(paths))
//	for i, path := range paths {
//	    finalRates[i] = path[10]
//	}
//	slices.Sort(finalRates)
//	p5 := finalRates[int(0.05*float64(len(finalRates)))]  // 5th percentile
//	p50 := finalRates[len(finalRates)/2]                   // median
//	p95 := finalRates[int(0.95*float64(len(finalRates)))] // 95th percentile
//
// # Model: how GBM works
//
// Each step: S(t+1) = S(t) * exp((mu - 0.5*sigma^2)*dt + sigma*sqrt(dt)*Z)
// where Z is a standard normal random variable (Box-Muller transform).
//
// - mu = drift (expected growth rate)
// - sigma = volatility (standard deviation of returns)
// - dt = time step size
//
// # Reproducible simulations
//
// Use NewRateGeneratorWithSeed for deterministic output:
//
//	rg1 := stochastic.NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42)
//	rg2 := stochastic.NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42)
//	// rg1 and rg2 produce identical paths
package stochastic
