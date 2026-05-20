package stochastic

import (
	"math"
	"math/rand/v2"
	"runtime"
	"sync"
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
	seed        uint64
	hasSeed     bool
}

// NewRateGenerator creates a new rate generator with a random seed.
// initialRate is the starting interest rate, mu is the drift (expected growth rate),
// and sigma is the volatility (standard deviation of returns).
// Use NewRateGeneratorWithSeed for reproducible simulations.
func NewRateGenerator(initialRate, mu, sigma float64) *RateGenerator {
	return &RateGenerator{
		initialRate: initialRate,
		mu:          mu,
		sigma:       sigma,
		driftOffset: mu - 0.5*sigma*sigma,
		rng:         rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
		hasSeed:     false,
	}
}

// NewRateGeneratorWithSeed creates a new rate generator with a deterministic seed.
// initialRate is the starting interest rate, mu is the drift (expected growth rate),
// sigma is the volatility (standard deviation of returns).
// The seed enables reproducible Monte Carlo simulations for auditability.
func NewRateGeneratorWithSeed(initialRate, mu, sigma float64, seed uint64) *RateGenerator {
	return &RateGenerator{
		initialRate: initialRate,
		mu:          mu,
		sigma:       sigma,
		driftOffset: mu - 0.5*sigma*sigma,
		rng:         rand.New(rand.NewPCG(seed, 0)),
		seed:        seed,
		hasSeed:     true,
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

// deriveWorkerSeed produces a unique seed for each worker from a base seed.
func deriveWorkerSeed(baseSeed uint64, workerIdx int) uint64 {
	return baseSeed + uint64(workerIdx)*6364136223846793005
}

// GeneratePathsParallel generates multiple interest rate paths in parallel.
// Each worker goroutine has its own RNG to avoid contention.
// If numWorkers <= 0, runtime.NumCPU() is used.
func (rg *RateGenerator) GeneratePathsParallel(numPaths, steps, numWorkers int, dt float64) []RatePath {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if numWorkers > numPaths {
		numWorkers = numPaths
	}
	if numWorkers == 1 {
		return rg.GeneratePaths(numPaths, steps, dt)
	}

	stride := steps + 1
	buf := make([]float64, numPaths*stride)
	paths := make([]RatePath, numPaths)
	for i := range numPaths {
		paths[i] = buf[i*stride : i*stride+stride : i*stride+stride]
	}

	driftTerm := rg.driftOffset * dt
	diffusionFactor := rg.sigma * math.Sqrt(dt)
	initialRate := rg.initialRate

	var wg sync.WaitGroup
	chunkSize := (numPaths + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := min(start+chunkSize, numPaths)
		if start >= numPaths {
			break
		}

		var workerRng *rand.Rand
		if rg.hasSeed {
			workerSeed := deriveWorkerSeed(rg.seed, w)
			workerRng = rand.New(rand.NewPCG(workerSeed, workerSeed))
		} else {
			workerRng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
		}

		wg.Add(1)
		go func(rng *rand.Rand, start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				path := paths[i]
				path[0] = initialRate
				for j := 1; j <= steps; j++ {
					path[j] = path[j-1] * math.Exp(driftTerm+diffusionFactor*rng.NormFloat64())
				}
			}
		}(workerRng, start, end)
	}

	wg.Wait()
	return paths
}

// compile-time check: *RateGenerator satisfies PathGenerator
var _ PathGenerator = (*RateGenerator)(nil)
