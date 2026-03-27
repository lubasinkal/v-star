package risk

import (
	"math"
	"testing"
)

func TestVaR_EmptySlice(t *testing.T) {
	if got := VaR(nil, 0.95); got != 0 {
		t.Errorf("VaR(nil) = %v, want 0", got)
	}
	if got := VaR([]float64{}, 0.95); got != 0 {
		t.Errorf("VaR([]) = %v, want 0", got)
	}
}

func TestVaR_InvalidConfidence(t *testing.T) {
	losses := []float64{1, 2, 3}
	if got := VaR(losses, 0); got != 0 {
		t.Errorf("VaR(confidence=0) = %v, want 0", got)
	}
	if got := VaR(losses, 1); got != 0 {
		t.Errorf("VaR(confidence=1) = %v, want 0", got)
	}
}

func TestVaR_KnownValues(t *testing.T) {
	// 100 losses: 1, 2, 3, ..., 100
	losses := make([]float64, 100)
	for i := range 100 {
		losses[i] = float64(i + 1)
	}

	// VaR at 95% confidence = 5th percentile = index 4 = value 5
	got := VaR(losses, 0.95)
	if got != 5 {
		t.Errorf("VaR(0.95) = %v, want 5", got)
	}

	// VaR at 99% confidence = 1st percentile = index 0 = value 1
	got = VaR(losses, 0.99)
	if got != 1 {
		t.Errorf("VaR(0.99) = %v, want 1", got)
	}

	// VaR at 90% confidence = 10th percentile = index 9 = value 10
	got = VaR(losses, 0.90)
	if got != 10 {
		t.Errorf("VaR(0.90) = %v, want 10", got)
	}
}

func TestCTE_KnownValues(t *testing.T) {
	// losses: 1, 2, 3, ..., 10
	losses := make([]float64, 10)
	for i := range 10 {
		losses[i] = float64(i + 1)
	}

	// VaR(0.95) = index 0 = 1, all values >= 1, so CTE = mean = 5.5
	got := CTE(losses, 0.95)
	expected := 5.5
	if math.Abs(got-expected) > 1e-9 {
		t.Errorf("CTE(0.95) = %v, want %v", got, expected)
	}

	// VaR(0.50) = index 4 = 5, values >= 5: {5,6,7,8,9,10}, mean = 7.5
	got = CTE(losses, 0.50)
	expected = 7.5
	if math.Abs(got-expected) > 1e-9 {
		t.Errorf("CTE(0.50) = %v, want %v", got, expected)
	}
}

func TestExpectedShortfall_IsCTE(t *testing.T) {
	losses := []float64{10, 20, 30, 40, 50}
	if got := ExpectedShortfall(losses, 0.90); got != CTE(losses, 0.90) {
		t.Errorf("ExpectedShortfall != CTE: %v vs %v", got, CTE(losses, 0.90))
	}
}

func TestComputeReport(t *testing.T) {
	losses := []float64{10, 20, 30, 40, 50, 60, 70, 80, 90, 100}
	report := ComputeReport(losses)

	if math.Abs(report.Mean-55.0) > 1e-9 {
		t.Errorf("Mean = %v, want 55", report.Mean)
	}
	if report.Min != 10 {
		t.Errorf("Min = %v, want 10", report.Min)
	}
	if report.Max != 100 {
		t.Errorf("Max = %v, want 100", report.Max)
	}
	if report.StdDev <= 0 {
		t.Errorf("StdDev = %v, want > 0", report.StdDev)
	}
	if report.VaR95 == 0 {
		t.Error("VaR95 should not be 0")
	}
	if report.CTE95 == 0 {
		t.Error("CTE95 should not be 0")
	}
}

func TestComputeReport_Empty(t *testing.T) {
	report := ComputeReport(nil)
	if report.Mean != 0 || report.StdDev != 0 || report.VaR95 != 0 {
		t.Error("Empty report should have all zeros")
	}
}

func BenchmarkVaR(b *testing.B) {
	losses := make([]float64, 100000)
	for i := range losses {
		losses[i] = float64(i)
	}
	b.ResetTimer()
	for b.Loop() {
		VaR(losses, 0.99)
	}
}

func BenchmarkCTE(b *testing.B) {
	losses := make([]float64, 100000)
	for i := range losses {
		losses[i] = float64(i)
	}
	b.ResetTimer()
	for b.Loop() {
		CTE(losses, 0.99)
	}
}

func BenchmarkComputeReport(b *testing.B) {
	losses := make([]float64, 100000)
	for i := range losses {
		losses[i] = float64(i)
	}
	b.ResetTimer()
	for b.Loop() {
		ComputeReport(losses)
	}
}
