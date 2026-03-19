package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lubasinkal/v-star/pkg/stochastic"
)

// MonteCarlo generates interest rate paths using geometric Brownian motion.
func MonteCarlo(args []string, interest float64) {
	numPaths := 100000
	steps := 10
	drift := 0.02
	volatility := 0.15
	var seed int64 = -1 // -1 means random

	for i := 1; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--paths=") {
			if val := strings.Split(arg, "=")[1]; val != "" {
				if n, err := strconv.Atoi(val); err == nil {
					numPaths = n
				}
			}
		} else if strings.HasPrefix(arg, "--steps=") {
			if val := strings.Split(arg, "=")[1]; val != "" {
				if n, err := strconv.Atoi(val); err == nil {
					steps = n
				}
			}
		} else if strings.HasPrefix(arg, "--drift=") {
			if val := strings.Split(arg, "=")[1]; val != "" {
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					drift = f
				}
			}
		} else if strings.HasPrefix(arg, "--volatility=") {
			if val := strings.Split(arg, "=")[1]; val != "" {
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					volatility = f
				}
			}
		} else if strings.HasPrefix(arg, "--seed=") {
			if val := strings.Split(arg, "=")[1]; val != "" {
				if n, err := strconv.ParseInt(val, 10, 64); err == nil {
					seed = n
				}
			}
		}
	}

	fmt.Printf("Generating %d Monte Carlo interest rate paths...\n", numPaths)
	fmt.Printf("Parameters: Initial Rate=%.2f%%, Drift=%.2f%%, Volatility=%.2f%%, Steps=%d\n",
		interest*100, drift*100, volatility*100, steps)
	if seed >= 0 {
		fmt.Printf("Seed: %d (deterministic)\n", seed)
	}

	start := time.Now()

	var rg *stochastic.RateGenerator
	if seed >= 0 {
		rg = stochastic.NewRateGeneratorWithSeed(interest, drift, volatility, uint64(seed))
	} else {
		rg = stochastic.NewRateGenerator(interest, drift, volatility)
	}
	paths := rg.GeneratePaths(numPaths, steps, 1.0)

	duration := time.Since(start)

	var totalRate float64
	var minRate, maxRate float64 = 1e9, -1e9

	for _, path := range paths {
		rate := path[steps]
		totalRate += rate
		if rate < minRate {
			minRate = rate
		}
		if rate > maxRate {
			maxRate = rate
		}
	}

	avgRate := totalRate / float64(numPaths)

	fmt.Printf("\n=== Monte Carlo Results ===\n")
	fmt.Printf("Paths Generated: %d\n", numPaths)
	fmt.Printf("Duration: %v\n", duration)
	fmt.Printf("Throughput: %.0f paths/sec\n", float64(numPaths)/duration.Seconds())
	fmt.Printf("\nFinal Rate Statistics:\n")
	fmt.Printf("  Average: %.4f%%\n", avgRate*100)
	fmt.Printf("  Minimum: %.4f%%\n", minRate*100)
	fmt.Printf("  Maximum: %.4f%%\n", maxRate*100)

	os.Exit(0)
}
