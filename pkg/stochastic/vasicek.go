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
	longTermMean  float64 // b
	meanReversion float64 // a
	volatility    float64 // sigma
	initialRate   float64
	rng           *rand.Rand
}

// NewVasicekGenerator creates a new VasicekGenerator with the given parameters.
// initialRate is the starting rate, longTermMean (b) is the long-term average level,
// meanReversion (a) is the speed at which rates revert to the mean,
// and volatility (sigma) is the standard deviation of rate changes.
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
// Parameters: initialRate, longTermMean (b, long-term average), meanReversion (a, speed),
// volatility (sigma), and a seed for reproducible simulations.
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
	vg.generatePathInto(path, n, dt)
	return path
}

func (vg *VasicekGenerator) generatePathInto(path RatePath, n int, dt float64) {
	path[0] = vg.initialRate
	r := vg.initialRate
	sqrtDt := math.Sqrt(dt)

	for i := 1; i <= n; i++ {
		drift := vg.meanReversion * (vg.longTermMean - r) * dt
		r = r + drift + vg.volatility*sqrtDt*vg.rng.NormFloat64()
		if r < 0 {
			r = 0
		}
		path[i] = r
	}
}

// GeneratePaths generates multiple independent rate paths using a single flat buffer.
func (vg *VasicekGenerator) GeneratePaths(numPaths, steps int, dt float64) []RatePath {
	paths := make([]RatePath, numPaths)
	buf := make([]float64, numPaths*(steps+1))
	stride := steps + 1
	for i := range numPaths {
		paths[i] = buf[i*stride : i*stride+stride : i*stride+stride]
		vg.generatePathInto(paths[i], steps, dt)
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
