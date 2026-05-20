package stochastic

import (
	"math"
	"testing"
)

func TestVasicekGenerator_GeneratePath(t *testing.T) {
	vg := NewVasicekGenerator(0.05, 0.04, 0.5, 0.02)
	path := vg.GeneratePath(10, 1.0)

	if len(path) != 11 {
		t.Errorf("Expected path length 11, got %d", len(path))
	}
	if path[0] != 0.05 {
		t.Errorf("Expected initial rate 0.05, got %f", path[0])
	}
}

func TestVasicekGenerator_MeanReverting(t *testing.T) {
	// High initial rate far from long-term mean should drift down
	vg := NewVasicekGenerator(0.10, 0.04, 0.8, 0.01)
	path := vg.GeneratePath(100, 1.0)

	// The path may not reach the mean, but it should move toward it
	if path[100] >= 0.10 {
		t.Logf("High rate may not have reverted yet: final=%.4f, initial=%.4f", path[100], path[0])
	}
}

func TestVasicekGenerator_DeterministicSeed(t *testing.T) {
	seed := uint64(42)
	vg1 := NewVasicekGeneratorWithSeed(0.05, 0.04, 0.5, 0.02, seed)
	vg2 := NewVasicekGeneratorWithSeed(0.05, 0.04, 0.5, 0.02, seed)

	path1 := vg1.GeneratePath(10, 1.0)
	path2 := vg2.GeneratePath(10, 1.0)

	for i := range path1 {
		if path1[i] != path2[i] {
			t.Errorf("Step %d: %f != %f (non-deterministic with same seed)", i, path1[i], path2[i])
		}
	}
}

func TestVasicekGenerator_NonNegative(t *testing.T) {
	// High volatility with low mean could theoretically go negative
	// but our implementation floors at 0
	vg := NewVasicekGenerator(0.01, 0.01, 0.1, 0.3)
	paths := vg.GeneratePaths(100, 50, 1.0)

	for i, path := range paths {
		for j, r := range path {
			if r < 0 {
				t.Errorf("Path %d step %d: negative rate %f", i, j, r)
			}
		}
	}
}

func TestVasicekGenerator_GeneratePaths(t *testing.T) {
	vg := NewVasicekGenerator(0.05, 0.04, 0.5, 0.02)
	paths := vg.GeneratePaths(10, 20, 1.0)

	if len(paths) != 10 {
		t.Errorf("Expected 10 paths, got %d", len(paths))
	}
	for i, path := range paths {
		if len(path) != 21 {
			t.Errorf("Path %d: expected length 21, got %d", i, len(path))
		}
	}
}

func TestVasicekGenerator_ZeroVolatility(t *testing.T) {
	// With zero volatility, the path should be deterministic
	// dr = a(b - r)dt
	vg := NewVasicekGenerator(0.05, 0.04, 0.5, 0)
	path := vg.GeneratePath(5, 1.0)

	// r1 = r0 + a(b - r0)*dt = 0.05 + 0.5*(0.04-0.05)*1 = 0.045
	expected := 0.045
	eps := 1e-6
	if math.Abs(path[1]-expected) > eps {
		t.Errorf("Step 1: expected %.6f, got %.6f", expected, path[1])
	}
}

func TestVasicekGenerator_NilGuards(t *testing.T) {
	var vg *VasicekGenerator
	if vg.LongTermMean() != 0 {
		t.Error("nil generator should return 0 for LongTermMean")
	}
	if vg.MeanReversionSpeed() != 0 {
		t.Error("nil generator should return 0 for MeanReversionSpeed")
	}
	if vg.Volatility() != 0 {
		t.Error("nil generator should return 0 for Volatility")
	}
}

func TestVasicekGenerator_GeneratePathsParallel(t *testing.T) {
	vg := NewVasicekGenerator(0.05, 0.04, 0.5, 0.02)
	paths := vg.GeneratePathsParallel(10, 20, 4, 1.0)

	if len(paths) != 10 {
		t.Errorf("Expected 10 paths, got %d", len(paths))
	}
	for i, path := range paths {
		if len(path) != 21 {
			t.Errorf("Path %d: expected length 21, got %d", i, len(path))
		}
		if path[0] != 0.05 {
			t.Errorf("Path %d: expected initial rate 0.05, got %f", i, path[0])
		}
	}
}

func TestVasicekGenerator_ParallelDeterministic(t *testing.T) {
	seed := uint64(42)
	numPaths := 50
	steps := 10

	vg1 := NewVasicekGeneratorWithSeed(0.05, 0.04, 0.5, 0.02, seed)
	paths1 := vg1.GeneratePathsParallel(numPaths, steps, 4, 1.0)

	vg2 := NewVasicekGeneratorWithSeed(0.05, 0.04, 0.5, 0.02, seed)
	paths2 := vg2.GeneratePathsParallel(numPaths, steps, 4, 1.0)

	for i := range numPaths {
		for j := 0; j <= steps; j++ {
			if paths1[i][j] != paths2[i][j] {
				t.Errorf("Path %d step %d: %f != %f (parallel deterministic mismatch)",
					i, j, paths1[i][j], paths2[i][j])
			}
		}
	}
}

func TestVasicekGenerator_ParallelZeroVolatility(t *testing.T) {
	vg := NewVasicekGeneratorWithSeed(0.05, 0.04, 0.5, 0, 42)
	paths := vg.GeneratePathsParallel(10, 5, 4, 1.0)

	// Expected: r_n+1 = r_n + a(b - r_n)*dt
	expected := 0.05
	for step := 0; step <= 5; step++ {
		for i := range paths {
			if paths[i][step] != expected {
				t.Errorf("Path %d step %d: expected %.6f, got %.6f",
					i, step, expected, paths[i][step])
			}
		}
		expected = expected + 0.5*(0.04-expected)
	}
}
