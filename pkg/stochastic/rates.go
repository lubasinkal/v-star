package stochastic

import (
	"math/rand/v2"
)

// RatePath represents a sequence of interest rates over time
type RatePath []float64

// RateGenerator generates stochastic interest rate paths
type RateGenerator struct {
	initialRate float64
	mu          float64 // drift
	sigma       float64 // volatility
}

// NewRateGenerator creates a new rate generator for geometric Brownian motion
func NewRateGenerator(initialRate, mu, sigma float64) *RateGenerator {
	return &RateGenerator{
		initialRate: initialRate,
		mu:          mu,
		sigma:       sigma,
	}
}

// GeneratePath generates a single interest rate path using geometric Brownian motion
// S(t+1) = S(t) * exp((mu - 0.5*sigma^2)*dt + sigma*sqrt(dt)*Z)
// where Z is a standard normal random variable
func (rg *RateGenerator) GeneratePath(steps int, dt float64) RatePath {
	path := make(RatePath, steps+1)
	path[0] = rg.initialRate

	for i := 1; i <= steps; i++ {
		// Generate standard normal random variable using Box-Muller transform
		u1 := rand.Float64()
		u2 := rand.Float64()
		z := sqrt(-2*log(u1)) * cos(2*3.14159265359*u2)

		// Geometric Brownian motion
		drift := (rg.mu - 0.5*rg.sigma*rg.sigma) * dt
		diffusion := rg.sigma * sqrt(dt) * z
		path[i] = path[i-1] * exp(drift+diffusion)
	}

	return path
}

// GeneratePaths generates multiple interest rate paths
func (rg *RateGenerator) GeneratePaths(numPaths, steps int, dt float64) []RatePath {
	paths := make([]RatePath, numPaths)
	for i := 0; i < numPaths; i++ {
		paths[i] = rg.GeneratePath(steps, dt)
	}
	return paths
}

// Helper math functions (since we're using zero dependencies)
func sqrt(x float64) float64 {
	// Simple Newton-Raphson approximation for sqrt
	if x < 0 {
		return 0
	}
	if x == 0 {
		return 0
	}

	// Initial guess
	result := x / 2

	// Iterate to improve approximation
	for i := 0; i < 10; i++ {
		result = 0.5 * (result + x/result)
	}

	return result
}

func log(x float64) float64 {
	// Simple approximation for natural log
	if x <= 0 {
		return -1e9 // Large negative number for invalid input
	}

	// Using the fact that log(x) = log(2^k * y) = k*log(2) + log(y)
	// where y is in [1, 2)

	// Extract exponent
	var k int
	for x >= 2 {
		x /= 2
		k++
	}
	for x < 1 {
		x *= 2
		k--
	}

	// Taylor series approximation for log(y) where y is in [1, 2)
	// log(y) = (y-1) - (y-1)^2/2 + (y-1)^3/3 - ...
	y := x
	yMinus1 := y - 1
	result := yMinus1 - yMinus1*yMinus1/2 + yMinus1*yMinus1*yMinus1/3 - yMinus1*yMinus1*yMinus1*yMinus1/4

	return float64(k)*0.69314718056 + result // 0.69314718056 = log(2)
}

func exp(x float64) float64 {
	// Simple approximation for exponential function
	// Using the fact that e^x = 2^(x/log(2))

	if x > 700 {
		return 1e308 // Large number
	}
	if x < -700 {
		return 0
	}

	// Using Taylor series expansion
	// e^x = 1 + x + x^2/2! + x^3/3! + ...
	result := 1.0
	term := 1.0
	for i := 1; i < 20; i++ {
		term *= x / float64(i)
		result += term
	}

	return result
}

func cos(x float64) float64 {
	// Simple approximation for cosine using Taylor series
	// cos(x) = 1 - x^2/2! + x^4/4! - x^6/6! + ...

	x = x - 2*3.14159265359*float64(int(x/(2*3.14159265359))) // Normalize to [0, 2π)

	result := 1.0
	term := 1.0
	for i := 1; i < 10; i++ {
		term *= -x * x / (float64(2*i) * float64(2*i-1))
		result += term
	}

	return result
}
