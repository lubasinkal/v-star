package stochastic

import (
	"math"
	"math/rand/v2"
)

// VasicekGenerator generates interest rate paths using the Vasicek model:
// dr = a(b - r)dt + sigma * dW
// where a is the speed of mean reversion, b is the long-term mean,
// and sigma is the volatility.
type VasicekGenerator struct {
	longTermMean    float64 // b
	meanReversion   float64 // a
	volatility      float64 // sigma
	initialRate     float64
	rng             *rand.Rand
}

// NewVasicekGenerator creates a new VasicekGenerator with the given parameters.
func NewVasicekGenerator(initialRate, longTermMean, meanReversion, volatility float64) *VasicekGenerator {
	return &VasicekGenerator{
		initialRate:   initialRate,
		longTermMean:  longTermMean,
		meanReversion: meanReversion,
		volatility:    volatility,
		rng:           rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}
}

// NewVasicekGeneratorWithSeed creates a VasicekGenerator with a deterministic seed.
func NewVasicekGeneratorWithSeed(initialRate, longTermMean, meanReversion, volatility float64, seed uint64) *VasicekGenerator {
	return &VasicekGenerator{
		initialRate:   initialRate,
		longTermMean:  longTermMean,
		meanReversion: meanReversion,
		volatility:    volatility,
		rng:           rand.New(rand.NewPCG(seed, seed)),
	}
}

// GeneratePath generates a single interest rate path with n steps and given dt.
// Returns a slice of length n+1 where [0] is the initial rate.
func (vg *VasicekGenerator) GeneratePath(n int, dt float64) RatePath {
	path := make(RatePath, n+1)
	path[0] = vg.initialRate
	r := vg.initialRate
	sqrtDt := math.Sqrt(dt)

	for i := 1; i <= n; i++ {
		drift := vg.meanReversion * (vg.longTermMean - r) * dt
		diffusion := vg.volatility * sqrtDt * vg.rng.NormFloat64()
		r = r + drift + diffusion
		if r < 0 {
			r = 0
		}
		path[i] = r
	}
	return path
}

// GeneratePaths generates multiple independent rate paths.
func (vg *VasicekGenerator) GeneratePaths(numPaths, steps int, dt float64) []RatePath {
	paths := make([]RatePath, numPaths)
	for i := range numPaths {
		paths[i] = vg.GeneratePath(steps, dt)
	}
	return paths
}

// LongTermMean returns the long-term mean rate (b).
func (vg *VasicekGenerator) LongTermMean() float64 {
	if vg == nil {
		return 0
	}
	return vg.longTermMean
}

// MeanReversionSpeed returns the speed of mean reversion (a).
func (vg *VasicekGenerator) MeanReversionSpeed() float64 {
	if vg == nil {
		return 0
	}
	return vg.meanReversion
}

// Volatility returns the volatility (sigma).
func (vg *VasicekGenerator) Volatility() float64 {
	if vg == nil {
		return 0
	}
	return vg.volatility
}
