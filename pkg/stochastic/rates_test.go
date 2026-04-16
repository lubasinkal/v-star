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

func BenchmarkGeneratePaths(b *testing.B) {
	rg := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42)
	for b.Loop() {
		rg.GeneratePaths(1000, 10, 1.0)
	}
}

func ExampleNewRateGeneratorWithSeed() {
	rg := NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42)
	path := rg.GeneratePath(5, 1.0)
	fmt.Printf("%.2f%%\n", path[5]*100)
	// Output: 3.82%
}
