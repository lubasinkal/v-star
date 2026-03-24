package rates

import (
	"math"
	"testing"
)

const testTolerance = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < testTolerance
}

func TestForceOfInterest(t *testing.T) {
	tests := []struct {
		i    float64
		want float64
	}{
		{0.05, math.Log(1.05)},
		{0.0, 0.0},
		{0.10, math.Log(1.10)},
	}
	for _, tt := range tests {
		got := ForceOfInterest(tt.i)
		if !approxEqual(got, tt.want) {
			t.Errorf("ForceOfInterest(%v) = %v, want %v", tt.i, got, tt.want)
		}
	}
}

func TestInterestFromForce_Roundtrip(t *testing.T) {
	original := 0.05
	delta := ForceOfInterest(original)
	recovered := InterestFromForce(delta)
	if !approxEqual(recovered, original) {
		t.Errorf("roundtrip: i=%v -> delta=%v -> i=%v", original, delta, recovered)
	}
}

func TestEffectiveToNominal_Roundtrip(t *testing.T) {
	tests := []struct {
		i float64
		m int
	}{
		{0.05, 12},
		{0.08, 4},
		{0.10, 2},
		{0.06, 365},
	}
	for _, tt := range tests {
		nominal := EffectiveToNominal(tt.i, tt.m)
		recovered := NominalToEffective(nominal, tt.m)
		if !approxEqual(recovered, tt.i) {
			t.Errorf("roundtrip: i=%v, m=%d -> nominal=%v -> effective=%v", tt.i, tt.m, nominal, recovered)
		}
	}
}

func TestAnnuityCertainImmediate(t *testing.T) {
	// a_angle_10 at 5% = (1 - v^10) / i
	// v = 1/1.05, v^10 = 0.613913254
	// (1 - 0.613913254) / 0.05 = 7.7217349
	got := AnnuityCertainImmediate(0.05, 10)
	want := (1 - math.Pow(1/1.05, 10)) / 0.05
	if !approxEqual(got, want) {
		t.Errorf("AnnuityCertainImmediate(0.05, 10) = %v, want %v", got, want)
	}
}

func TestAnnuityCertainDue(t *testing.T) {
	// adbl_angle_10 at 5% = (1 - v^10) / d where d = i/(1+i) = 0.05/1.05
	i := 0.05
	n := 10
	got := AnnuityCertainDue(i, n)
	v := 1 / (1 + i)
	d := i / (1 + i)
	want := (1 - math.Pow(v, float64(n))) / d
	if !approxEqual(got, want) {
		t.Errorf("AnnuityCertainDue(0.05, 10) = %v, want %v", got, want)
	}

	// adbl = a * (1+i)
	a := AnnuityCertainImmediate(i, n)
	if !approxEqual(got, a*(1+i)) {
		t.Errorf("AnnuityCertainDue != AnnuityCertainImmediate * (1+i)")
	}
}

func TestAnnuityCertain_EdgeCases(t *testing.T) {
	if got := AnnuityCertainImmediate(0.05, 0); got != 0 {
		t.Errorf("AnnuityCertainImmediate with n=0 = %v, want 0", got)
	}
	if got := AnnuityCertainDue(-1, 10); got != 0 {
		t.Errorf("AnnuityCertainDue with negative i = %v, want 0", got)
	}
}

func TestMacaulayDuration(t *testing.T) {
	// Zero-coupon bond: 100 at time 5, i=5%
	// Duration should be exactly 5 (single cash flow at time 5)
	cashFlows := []float64{0, 0, 0, 0, 100}
	got := MacaulayDuration(0.05, cashFlows)
	if !approxEqual(got, 5.0) {
		t.Errorf("MacaulayDuration(zero-coupon 5yr) = %v, want 5.0", got)
	}
}

func TestModifiedDuration(t *testing.T) {
	cashFlows := []float64{0, 0, 0, 0, 100}
	md := ModifiedDuration(0.05, cashFlows)
	mac := MacaulayDuration(0.05, cashFlows)
	if !approxEqual(md, mac/1.05) {
		t.Errorf("ModifiedDuration = %v, want Macaulay/1.05 = %v", md, mac/1.05)
	}
}

func TestConvexity(t *testing.T) {
	// Zero-coupon bond at time 5: t*(t+1)*v^t / ((1+i)^2 * v^5)
	// = 5*6 / 1.05^2 = 30 / 1.1025 = 27.2108844
	cashFlows := []float64{0, 0, 0, 0, 100}
	got := Convexity(0.05, cashFlows)
	want := 30.0 / (1.05 * 1.05)
	if !approxEqual(got, want) {
		t.Errorf("Convexity(zero-coupon 5yr) = %v, want %v", got, want)
	}
}

func TestDurationConvexity_EdgeCases(t *testing.T) {
	if got := MacaulayDuration(0.05, nil); got != 0 {
		t.Errorf("MacaulayDuration(nil) = %v, want 0", got)
	}
	if got := Convexity(0.05, nil); got != 0 {
		t.Errorf("Convexity(nil) = %v, want 0", got)
	}
	if got := MacaulayDuration(-1, []float64{100}); got != 0 {
		t.Errorf("MacaulayDuration negative rate = %v, want 0", got)
	}
}
