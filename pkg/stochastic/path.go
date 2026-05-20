package stochastic

// PathGenerator generates interest rate paths for Monte Carlo simulation.
// Both RateGenerator (GBM) and VasicekGenerator implement this interface.
type PathGenerator interface {
	// GeneratePath generates a single path with steps steps and step size dt.
	// Returns a slice of length steps+1 where [0] is the initial rate.
	GeneratePath(steps int, dt float64) RatePath

	// GeneratePaths generates multiple paths using a single flat buffer.
	GeneratePaths(numPaths, steps int, dt float64) []RatePath

	// GeneratePathsParallel generates multiple paths in parallel.
	// Each worker goroutine uses its own RNG to avoid contention.
	// If numWorkers <= 0, runtime.NumCPU() is used.
	GeneratePathsParallel(numPaths, steps, numWorkers int, dt float64) []RatePath
}
