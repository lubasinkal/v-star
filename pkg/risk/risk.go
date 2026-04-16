package risk

import (
	"slices"
)

// VaR computes Value at Risk at the given confidence level.
// Returns the loss threshold that is not exceeded with the specified probability.
// For example, VaR(0.95) returns the 95th percentile of losses (95% confidence).
// losses should contain simulated portfolio losses (positive values represent losses).
func VaR(losses []float64, confidence float64) float64 {
	if len(losses) == 0 || confidence <= 0 || confidence >= 1 {
		return 0
	}
	sorted := make([]float64, len(losses))
	copy(sorted, losses)
	slices.Sort(sorted)

	idx := int(confidence * float64(len(sorted)-1))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// CTE computes Conditional Tail Expectation (Expected Shortfall).
// Returns the average of losses exceeding the VaR threshold.
// CTE is more informative than VaR as it captures tail severity.
func CTE(losses []float64, confidence float64) float64 {
	if len(losses) == 0 || confidence <= 0 || confidence >= 1 {
		return 0
	}
	varThreshold := VaR(losses, confidence)

	sum := 0.0
	count := 0
	for _, loss := range losses {
		if loss >= varThreshold {
			sum += loss
			count++
		}
	}
	if count == 0 {
		return varThreshold
	}
	return sum / float64(count)
}

// ExpectedShortfall is an alias for CTE.
// Both measure the expected loss given that the loss exceeds VaR.
func ExpectedShortfall(losses []float64, confidence float64) float64 {
	return CTE(losses, confidence)
}

// RiskReport contains comprehensive risk metrics from a simulation.
type RiskReport struct {
	Mean   float64
	StdDev float64
	Min    float64
	Max    float64
	VaR95  float64
	VaR99  float64
	CTE95  float64
	CTE99  float64
}

// ComputeReport generates a full risk report from simulated losses.
func ComputeReport(losses []float64) RiskReport {
	n := float64(len(losses))
	if n == 0 {
		return RiskReport{}
	}

	mean := 0.0
	min, max := losses[0], losses[0]
	for _, l := range losses {
		mean += l
		if l < min {
			min = l
		}
		if l > max {
			max = l
		}
	}
	mean /= n

	variance := 0.0
	for _, l := range losses {
		d := l - mean
		variance += d * d
	}
	variance /= n

	return RiskReport{
		Mean:   mean,
		StdDev: sqrt(variance),
		Min:    min,
		Max:    max,
		VaR95:  VaR(losses, 0.95),
		VaR99:  VaR(losses, 0.99),
		CTE95:  CTE(losses, 0.95),
		CTE99:  CTE(losses, 0.99),
	}
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 100; i++ {
		z = (z + x/z) / 2
	}
	return z
}
