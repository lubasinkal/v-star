package stochastic

import (
	"fmt"
	"testing"
)

func TestRateGenerator_GeneratePath(t *testing.T) {
	rg := NewRateGenerator(0.05, 0.02, 0.1)

	path := rg.GeneratePath(10, 1.0)

	if len(path) != 11 {
		t.Errorf("Expected path length 11, got %d", len(path))
	}

	if path[0] != 0.05 {
		t.Errorf("Expected initial rate 0.05, got %f", path[0])
	}

	for i, rate := range path {
		if rate <= 0 {
			t.Errorf("Rate at step %d is not positive: %f", i, rate)
		}
	}
}

func TestRateGenerator_GeneratePaths(t *testing.T) {
	rg := NewRateGenerator(0.05, 0.02, 0.1)

	paths := rg.GeneratePaths(5, 10, 1.0)

	if len(paths) != 5 {
		t.Errorf("Expected 5 paths, got %d", len(paths))
	}

	for i, path := range paths {
		if len(path) != 11 {
			t.Errorf("Path %d: expected length 11, got %d", i, len(path))
		}
	}
}

func TestDeterministicSeed(t *testing.T) {
	seed := uint64(42)
	steps := 10
	numPaths := 3

	rg1 := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, seed)
	paths1 := rg1.GeneratePaths(numPaths, steps, 1.0)

	rg2 := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, seed)
	paths2 := rg2.GeneratePaths(numPaths, steps, 1.0)

	for i := range numPaths {
		for j := 0; j <= steps; j++ {
			if paths1[i][j] != paths2[i][j] {
				t.Errorf("Path %d step %d: got %f, want %f (non-deterministic with same seed)",
					i, j, paths1[i][j], paths2[i][j])
			}
		}
	}
}

func TestDifferentSeeds(t *testing.T) {
	steps := 10

	rg1 := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42)
	path1 := rg1.GeneratePath(steps, 1.0)

	rg2 := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 99)
	path2 := rg2.GeneratePath(steps, 1.0)

	same := true
	for i := 1; i <= steps; i++ {
		if path1[i] != path2[i] {
			same = false
			break
		}
	}
	if same {
		t.Error("Different seeds produced identical paths")
	}
}

func TestRateGenerator_ZeroVolatility(t *testing.T) {
	// GBM uses continuous compounding: r(t+1) = r(t) * exp((mu - sigma^2/2)*dt)
	// With sigma=0: r(t+1) = r(t) * exp(mu*dt)
	rg := NewRateGenerator(0.05, 0.02, 0)
	path := rg.GeneratePath(5, 1.0)
	expected := 0.05
	for i := 1; i <= 5; i++ {
		expected *= 1.0202013400267558 // exp(0.02)
		if path[i] != expected {
			t.Errorf("Step %d: expected %.10f, got %.10f", i, expected, path[i])
		}
	}
}

func TestRateGenerator_ZeroDrift(t *testing.T) {
	rg := NewRateGenerator(0.05, 0, 0.1)
	path := rg.GeneratePath(100, 1.0)
	if path[0] != 0.05 {
		t.Errorf("Initial rate should be 0.05, got %f", path[0])
	}
}

func TestRateGenerator_NegativeDrift(t *testing.T) {
	rg := NewRateGenerator(0.05, -0.1, 0.1)
	path := rg.GeneratePath(50, 1.0)
	if path[0] != 0.05 {
		t.Errorf("Initial rate should be 0.05, got %f", path[0])
	}
}

func TestRateGenerator_SinglePath(t *testing.T) {
	rg := NewRateGenerator(0.03, 0.01, 0.2)
	path := rg.GeneratePath(1, 1.0)
	if len(path) != 2 {
		t.Errorf("Single step path should have length 2, got %d", len(path))
	}
}

func TestRateGenerator_ExtremeVolatility(t *testing.T) {
	rg := NewRateGenerator(0.05, 0, 0.5)
	path := rg.GeneratePath(10, 1.0)
	if path[0] != 0.05 {
		t.Errorf("Initial rate should be 0.05, got %f", path[0])
	}
}

func TestRateGenerator_GeneratePathsParallel(t *testing.T) {
	rg := NewRateGenerator(0.05, 0.02, 0.1)

	paths := rg.GeneratePathsParallel(5, 10, 4, 1.0)

	if len(paths) != 5 {
		t.Errorf("Expected 5 paths, got %d", len(paths))
	}

	for i, path := range paths {
		if len(path) != 11 {
			t.Errorf("Path %d: expected length 11, got %d", i, len(path))
		}
		if path[0] != 0.05 {
			t.Errorf("Path %d: expected initial rate 0.05, got %f", i, path[0])
		}
	}
}

func TestRateGenerator_ParallelDeterministic(t *testing.T) {
	seed := uint64(42)
	numPaths := 100
	steps := 10

	rg1 := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, seed)
	paths1 := rg1.GeneratePathsParallel(numPaths, steps, 4, 1.0)

	rg2 := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, seed)
	paths2 := rg2.GeneratePathsParallel(numPaths, steps, 4, 1.0)

	for i := range numPaths {
		for j := 0; j <= steps; j++ {
			if paths1[i][j] != paths2[i][j] {
				t.Errorf("Path %d step %d: got %f, want %f (parallel deterministic mismatch)",
					i, j, paths1[i][j], paths2[i][j])
			}
		}
	}
}

func TestRateGenerator_ParallelStatisticalMatch(t *testing.T) {
	// Parallel uses independent RNGs per worker, so paths differ.
	// But the distribution must be statistically equivalent.
	seed := uint64(42)
	numPaths := 10000
	steps := 10

	rgSeq := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, seed)
	seqPaths := rgSeq.GeneratePaths(numPaths, steps, 1.0)

	rgPar := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, seed)
	parPaths := rgPar.GeneratePathsParallel(numPaths, steps, 4, 1.0)

	var seqMean, parMean float64
	for i := range numPaths {
		seqMean += seqPaths[i][steps]
		parMean += parPaths[i][steps]
	}
	seqMean /= float64(numPaths)
	parMean /= float64(numPaths)

	diff := seqMean - parMean
	if diff < 0 {
		diff = -diff
	}
	if diff > 0.002 { // 0.2% tolerance on mean
		t.Errorf("Mean mismatch: sequential=%.6f, parallel=%.6f", seqMean, parMean)
	}
}

func TestRateGenerator_ParallelZeroVolatility(t *testing.T) {
	rg := NewRateGeneratorWithSeed(0.05, 0.02, 0, 42)
	paths := rg.GeneratePathsParallel(10, 5, 4, 1.0)

	expected := 0.05
	for j := 0; j <= 5; j++ {
		for i := range paths {
			if paths[i][j] != expected {
				t.Errorf("Path %d step %d: expected %.10f, got %.10f",
					i, j, expected, paths[i][j])
			}
		}
		expected *= 1.0202013400267558
	}
}

func TestRateGenerator_ParallelSingleWorkerFallback(t *testing.T) {
	seed := uint64(42)
	numPaths := 10
	steps := 10

	rg1 := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, seed)
	seqPaths := rg1.GeneratePaths(numPaths, steps, 1.0)

	rg2 := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, seed)
	parPaths := rg2.GeneratePathsParallel(numPaths, steps, 1, 1.0)

	for i := range numPaths {
		for j := 0; j <= steps; j++ {
			if seqPaths[i][j] != parPaths[i][j] {
				t.Errorf("Path %d step %d: sequential=%f, parallel(1 worker)=%f",
					i, j, seqPaths[i][j], parPaths[i][j])
			}
		}
	}
}

func BenchmarkGeneratePaths(b *testing.B) {
	rg := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42)
	for b.Loop() {
		rg.GeneratePaths(1000, 10, 1.0)
	}
}

func BenchmarkGeneratePathsParallel(b *testing.B) {
	rg := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42)
	for b.Loop() {
		rg.GeneratePathsParallel(1000, 10, 0, 1.0)
	}
}

func ExampleNewRateGeneratorWithSeed() {
	rg := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42)
	path := rg.GeneratePath(5, 1.0)
	fmt.Printf("%.2f%%\n", path[5]*100)
	// Output: 5.56%
}
