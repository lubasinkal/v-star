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
	initialRate float64
	mu          float64 // drift
	sigma       float64 // volatility
	rng         *rand.Rand
}

// NewRateGenerator creates a new rate generator with a random seed.
// Use NewRateGeneratorWithSeed for reproducible simulations.
func NewRateGenerator(initialRate, mu, sigma float64) *RateGenerator {
	return &RateGenerator{
		initialRate: initialRate,
		mu:          mu,
		sigma:       sigma,
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
		rng:         rand.New(rand.NewPCG(seed, 0)),
	}
}

// GeneratePath generates a single interest rate path using geometric Brownian motion.
// S(t+1) = S(t) * exp((mu - 0.5*sigma^2)*dt + sigma*sqrt(dt)*Z)
// where Z is a standard normal random variable via Box-Muller transform.
func (rg *RateGenerator) GeneratePath(steps int, dt float64) RatePath {
	path := make(RatePath, steps+1)
	path[0] = rg.initialRate

	for i := 1; i <= steps; i++ {
		u1 := rg.rng.Float64()
		u2 := rg.rng.Float64()
		z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)

		drift := (rg.mu - 0.5*rg.sigma*rg.sigma) * dt
		diffusion := rg.sigma * math.Sqrt(dt) * z
		path[i] = path[i-1] * math.Exp(drift+diffusion)
	}

	return path
}

// GeneratePaths generates multiple interest rate paths.
func (rg *RateGenerator) GeneratePaths(numPaths, steps int, dt float64) []RatePath {
	paths := make([]RatePath, numPaths)
	for i := 0; i < numPaths; i++ {
		paths[i] = rg.GeneratePath(steps, dt)
	}
	return paths
}
