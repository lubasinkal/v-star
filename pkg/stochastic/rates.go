package stochastic

import (
	"math"
	"math/rand/v2"
)

// RatePath represents a sequence of interest rates over time
type RatePath []float64

// RateGenerator generates stochastic interest rate paths using geometric Brownian motion.
// It supports deterministic seeding for reproducible actuarial simulations.
type RateGenerator struct {
	rng         *rand.Rand
	initialRate float64
	mu          float64
	sigma       float64
	driftOffset float64
}

// NewRateGenerator creates a new rate generator with a random seed.
// Use NewRateGeneratorWithSeed for reproducible simulations.
func NewRateGenerator(initialRate, mu, sigma float64) *RateGenerator {
	return &RateGenerator{
		initialRate: initialRate,
		mu:          mu,
		sigma:       sigma,
		driftOffset: mu - 0.5*sigma*sigma,
		rng:         rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}
}

// NewRateGeneratorWithSeed creates a new rate generator with a deterministic seed.
// This enables reproducible Monte Carlo simulations for auditability.
func NewRateGeneratorWithSeed(initialRate, mu, sigma float64, seed uint64) *RateGenerator {
	return &RateGenerator{
		initialRate: initialRate,
		mu:          mu,
		sigma:       sigma,
		driftOffset: mu - 0.5*sigma*sigma,
		rng:         rand.New(rand.NewPCG(seed, 0)),
	}
}

// GeneratePath generates a single interest rate path using geometric Brownian motion.
// S(t+1) = S(t) * exp((mu - 0.5*sigma^2)*dt + sigma*sqrt(dt)*Z)
// where Z is a standard normal random variable via the Ziggurat method.
func (rg *RateGenerator) GeneratePath(steps int, dt float64) RatePath {
	path := make(RatePath, steps+1)
	rg.generatePathInto(path, steps, dt)
	return path
}

func (rg *RateGenerator) generatePathInto(path RatePath, steps int, dt float64) {
	path[0] = rg.initialRate
	driftTerm := rg.driftOffset * dt
	diffusionFactor := rg.sigma * math.Sqrt(dt)
	for i := 1; i <= steps; i++ {
		path[i] = path[i-1] * math.Exp(driftTerm+diffusionFactor*rg.rng.NormFloat64())
	}
}

// GeneratePaths generates multiple interest rate paths using a single flat buffer.
func (rg *RateGenerator) GeneratePaths(numPaths, steps int, dt float64) []RatePath {
	paths := make([]RatePath, numPaths)
	buf := make([]float64, numPaths*(steps+1))
	stride := steps + 1
	for i := range numPaths {
		paths[i] = buf[i*stride : i*stride+stride : i*stride+stride]
		rg.generatePathInto(paths[i], steps, dt)
	}
	return paths
}
