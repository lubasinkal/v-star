package stochastic

import (
	"math"
	"math/rand/v2"
	"runtime"
	"sync"
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
	seed          uint64
	hasSeed       bool
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
		hasSeed:       false,
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
		seed:          seed,
		hasSeed:       true,
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

// GeneratePathsParallel generates multiple independent rate paths in parallel.
// Each worker goroutine has its own RNG to avoid contention.
// If numWorkers <= 0, runtime.NumCPU() is used.
func (vg *VasicekGenerator) GeneratePathsParallel(numPaths, steps, numWorkers int, dt float64) []RatePath {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if numWorkers > numPaths {
		numWorkers = numPaths
	}
	if numWorkers == 1 {
		return vg.GeneratePaths(numPaths, steps, dt)
	}

	stride := steps + 1
	buf := make([]float64, numPaths*stride)
	paths := make([]RatePath, numPaths)
	for i := range numPaths {
		paths[i] = buf[i*stride : i*stride+stride : i*stride+stride]
	}

	sqrtDt := math.Sqrt(dt)
	longTermMean := vg.longTermMean
	meanReversion := vg.meanReversion
	volatility := vg.volatility
	initialRate := vg.initialRate

	var wg sync.WaitGroup
	chunkSize := (numPaths + numWorkers - 1) / numWorkers

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		end := min(start+chunkSize, numPaths)
		if start >= numPaths {
			break
		}

		var workerRng *rand.Rand
		if vg.hasSeed {
			workerSeed := deriveWorkerSeed(vg.seed, w)
			workerRng = rand.New(rand.NewPCG(workerSeed, workerSeed))
		} else {
			workerRng = rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64()))
		}

		wg.Add(1)
		go func(rng *rand.Rand, start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				path := paths[i]
				r := initialRate
				path[0] = r
				for j := 1; j <= steps; j++ {
					drift := meanReversion * (longTermMean - r) * dt
					r = r + drift + volatility*sqrtDt*rng.NormFloat64()
					if r < 0 {
						r = 0
					}
					path[j] = r
				}
			}
		}(workerRng, start, end)
	}

	wg.Wait()
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

// compile-time check: *VasicekGenerator satisfies PathGenerator
var _ PathGenerator = (*VasicekGenerator)(nil)
