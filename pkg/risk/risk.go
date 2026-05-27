package risk

import (
	"math"
	"slices"
)

// VaR computes Value at Risk at the given confidence level.
// Returns the loss threshold that is not exceeded with the specified probability.
// For example, VaR(0.95) returns the 95th percentile of losses (95% confidence).
// losses should contain simulated portfolio losses (positive values represent losses).
// The losses slice is sorted in place during computation.
func VaR(losses []float64, confidence float64) float64 {
	if len(losses) == 0 || confidence <= 0 || confidence >= 1 {
		return 0
	}
	slices.Sort(losses)
	return varSorted(losses, confidence)
}

// CTE computes Conditional Tail Expectation (Expected Shortfall).
// Returns the average of losses exceeding the VaR threshold.
// CTE is more informative than VaR as it captures tail severity.
// The losses slice is sorted in place during computation.
func CTE(losses []float64, confidence float64) float64 {
	if len(losses) == 0 || confidence <= 0 || confidence >= 1 {
		return 0
	}
	slices.Sort(losses)
	return cteSorted(losses, confidence)
}

func varSorted(sorted []float64, confidence float64) float64 {
	idx := int(confidence * float64(len(sorted)-1))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

func cteSorted(sorted []float64, confidence float64) float64 {
	idx := int(confidence * float64(len(sorted)-1))
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	sum := 0.0
	for i := idx; i < len(sorted); i++ {
		sum += sorted[i]
	}
	return sum / float64(len(sorted)-idx)
}

// RiskReport contains comprehensive risk metrics from a simulation.
type RiskReport struct {
	Mean           float64
	StdDev         float64
	Min            float64
	Max            float64
	VaR95          float64
	VaR99          float64
	CTE95          float64
	CTE99          float64
	StdError       float64
	Confidence95Lo float64
	Confidence95Hi float64
}

// ComputeReport generates a full risk report from simulated losses.
// The losses slice is sorted in place during computation.
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

	slices.Sort(losses)

	stdDev := math.Sqrt(variance)
	stdErr := stdDev / math.Sqrt(n)

	return RiskReport{
		Mean:           mean,
		StdDev:         stdDev,
		StdError:       stdErr,
		Confidence95Lo: mean - 1.96*stdErr,
		Confidence95Hi: mean + 1.96*stdErr,
		Min:            min,
		Max:            max,
		VaR95:          varSorted(losses, 0.95),
		VaR99:          varSorted(losses, 0.99),
		CTE95:          cteSorted(losses, 0.95),
		CTE99:          cteSorted(losses, 0.99),
	}
}
