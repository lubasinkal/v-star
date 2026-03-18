package stochastic

import (
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

	// Check that all rates are positive
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

func TestSqrt(t *testing.T) {
	tests := []struct {
		input     float64
		expected  float64
		tolerance float64
	}{
		{4, 2, 0.01},
		{9, 3, 0.01},
		{16, 4, 0.01},
		{2, 1.414, 0.01},
	}

	for _, tt := range tests {
		result := sqrt(tt.input)
		if result < tt.expected-tt.tolerance || result > tt.expected+tt.tolerance {
			t.Errorf("sqrt(%f) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestLog(t *testing.T) {
	tests := []struct {
		input     float64
		expected  float64
		tolerance float64
	}{
		{1, 0, 0.01},
		{2, 0.693, 0.01},
		{10, 2.302, 0.1},
	}

	for _, tt := range tests {
		result := log(tt.input)
		if result < tt.expected-tt.tolerance || result > tt.expected+tt.tolerance {
			t.Errorf("log(%f) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestExp(t *testing.T) {
	tests := []struct {
		input     float64
		expected  float64
		tolerance float64
	}{
		{0, 1, 0.01},
		{1, 2.718, 0.1},
		{2, 7.389, 0.2},
	}

	for _, tt := range tests {
		result := exp(tt.input)
		if result < tt.expected-tt.tolerance || result > tt.expected+tt.tolerance {
			t.Errorf("exp(%f) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestCos(t *testing.T) {
	tests := []struct {
		input     float64
		expected  float64
		tolerance float64
	}{
		{0, 1, 0.01},
		{3.14159, -1, 0.1},
		{1.5708, 0, 0.1},
	}

	for _, tt := range tests {
		result := cos(tt.input)
		if result < tt.expected-tt.tolerance || result > tt.expected+tt.tolerance {
			t.Errorf("cos(%f) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}
