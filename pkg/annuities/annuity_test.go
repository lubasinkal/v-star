package annuities

import (
	"fmt"
	"math"
	"testing"

	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
)

const tolerance = 1e-6

func floatEquals(got, want float64) bool {
	return math.Abs(got-want) < tolerance
}

// Zero-mortality table (all qx=0, everyone survives forever)
func zeroMortalityTable(maxAge int) *mortality.Table {
	qx := make([]float64, maxAge+1)
	return mortality.NewTable("zero-mort", qx)
}

func TestTermImmediate_ExactValues(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	// TermImmediate(age=30, term=3, amount=1000)
	// = 1000 * (v + v^2 + v^3) where v = 1/1.05
	expected := 1000.0 * (1/1.05 + 1/(1.05*1.05) + 1/(1.05*1.05*1.05))
	got := calc.TermImmediate(30, 3, 1000)
	if !floatEquals(got, expected) {
		t.Errorf("TermImmediate(30,3,1000) = %.6f, want %.6f", got, expected)
	}
}

func TestTermDue_ExactValues(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	// TermDue(age=30, term=3, amount=1000)
	// = 1000 * (1 + v + v^2) where v = 1/1.05
	v := 1.0 / 1.05
	expected := 1000.0 * (1.0 + v + v*v)
	got := calc.TermDue(30, 3, 1000)
	if !floatEquals(got, expected) {
		t.Errorf("TermDue(30,3,1000) = %.6f, want %.6f", got, expected)
	}
}

func TestWholeLifeImmediate_ZeroMortality(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	// WholeLifeImmediate at age=118 with maxAge=120:
	// Px(118,t) returns 0 when 118+t > 120 (i.e., t >= 3)
	// Loop adds v^1 + v^2 only
	v := 1.0 / 1.05
	expected := 1000.0 * (v + v*v)
	got := calc.WholeLifeImmediate(118, 1000)
	if !floatEquals(got, expected) {
		t.Errorf("WholeLifeImmediate(118,1000) = %.6f, want %.6f", got, expected)
	}
}

func TestWholeLifeDue_ZeroMortality(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	// WholeDue at age=118 with maxAge=120:
	// t=0: Px(118,0)=1 (always), add 1*v^0 = 1
	// t=1: Px(118,1)=1, add v
	// t=2: Px(118,2)=1, add v^2
	// t=3: Px(118,3)=0 (121 > 120), break
	v := 1.0 / 1.05
	expected := 1000.0 * (1.0 + v + v*v)
	got := calc.WholeLifeDue(118, 1000)
	if !floatEquals(got, expected) {
		t.Errorf("WholeLifeDue(118,1000) = %.6f, want %.6f", got, expected)
	}
}

func TestWholeLifeImmediate_LargeAge(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	// At age=30 with maxAge=120: loop runs for t=1..90 (90 terms)
	// Px(30,t)=1 for t <= 90, 0 for t >= 91
	v := 1.0 / 1.05
	expected := 0.0
	for t := 1; t <= 90; t++ {
		expected += math.Pow(v, float64(t))
	}
	expected *= 1000
	got := calc.WholeLifeImmediate(30, 1000)
	if !floatEquals(got, expected) {
		t.Errorf("WholeLifeImmediate(30,1000) = %.6f, want %.6f", got, expected)
	}
}

func TestTermImmediate_WithMortality(t *testing.T) {
	// Uniform 10% mortality per age
	qx := []float64{0.1, 0.1, 0.1, 0.1, 0.1, 0.1}
	mort := mortality.NewTable("uniform-10", qx)
	converter := rates.NewRateConverter(0.0)
	calc := NewAnnuityCalculator(converter, mort)

	// i=0 means v=1. Px(0,1)=0.9, Px(0,2)=0.81, Px(0,3)=0.729
	// TermImmediate(0, 3, 1000) = 1000 * (0.9 + 0.81 + 0.729) = 2439.0
	expected := 1000.0 * (0.9 + 0.81 + 0.729)
	got := calc.TermImmediate(0, 3, 1000)
	if !floatEquals(got, expected) {
		t.Errorf("TermImmediate(0,3,1000) = %.6f, want %.6f", got, expected)
	}
}

func TestDeferredWholeLife_ExactValues(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	// DeferredWholeLife(age=30, deferment=2, amount=1000)
	// = Px(30,2) * v^2 * WholeLifeImmediate(32, 1000)
	// With zero mortality, Px(30,2) = 1
	// WholeLifeImmediate(32, 1000): maxAge=120, age=32
	//   Px(32,t)=1 for t <= 88, 0 for t >= 89
	//   sum = sum(v^t, t=1..88)
	v := 1.0 / 1.05
	annuityPV := 0.0
	for t := 1; t <= 88; t++ {
		annuityPV += math.Pow(v, float64(t))
	}
	expected := v * v * 1000.0 * annuityPV
	got := calc.DeferredWholeLife(30, 2, 1000)
	if !floatEquals(got, expected) {
		t.Errorf("DeferredWholeLife(30,2,1000) = %.6f, want %.6f", got, expected)
	}
}

func TestWholeLifeImmediate_AgeExceedsMaxAge(t *testing.T) {
	qx := []float64{0.01, 0.02, 0.03}
	mort := mortality.NewTable("small", qx)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	// maxAge=2, age=3 > maxAge => Px(3,t)=0 for all t, returns 0
	got := calc.WholeLifeImmediate(3, 1000)
	if got != 0 {
		t.Errorf("WholeLifeImmediate(3,1000) with maxAge=2 = %v, want 0", got)
	}
}

func TestWholeLifeDue_NoInfiniteLoop(t *testing.T) {
	// Zero mortality with large maxAge should terminate properly
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	got := calc.WholeLifeDue(30, 1000)
	if got <= 0 {
		t.Errorf("WholeLifeDue(30,1000) = %v, want > 0", got)
	}
}

func TestEdgeCases_ZeroAmount(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	tests := []struct {
		fn   func() float64
		name string
	}{
		{func() float64 { return calc.WholeLifeImmediate(30, 0) }, "WholeLifeImmediate"},
		{func() float64 { return calc.TermImmediate(30, 10, 0) }, "TermImmediate"},
		{func() float64 { return calc.WholeLifeDue(30, 0) }, "WholeLifeDue"},
		{func() float64 { return calc.TermDue(30, 10, 0) }, "TermDue"},
		{func() float64 { return calc.DeferredWholeLife(30, 5, 0) }, "DeferredWholeLife"},
		{func() float64 { return calc.DeferredTerm(30, 5, 10, 0) }, "DeferredTerm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fn(); got != 0 {
				t.Errorf("%s with zero amount = %v, want 0", tt.name, got)
			}
		})
	}
}

func TestEdgeCases_NegativeAge(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	if got := calc.WholeLifeImmediate(-1, 1000); got != 0 {
		t.Errorf("WholeLifeImmediate(-1,1000) = %v, want 0", got)
	}
	if got := calc.TermImmediate(-1, 10, 1000); got != 0 {
		t.Errorf("TermImmediate(-1,10,1000) = %v, want 0", got)
	}
}

func TestEdgeCases_TermZero(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	if got := calc.TermImmediate(30, 0, 1000); got != 0 {
		t.Errorf("TermImmediate(30,0,1000) = %v, want 0", got)
	}
}

func TestWholeLifeDue_TerminatesAtMaxAge(t *testing.T) {
	// Mortality table where qx=1.0 at last age (everyone dies at maxAge)
	qx := []float64{0.01, 0.02, 0.05, 0.1, 1.0}
	mort := mortality.NewTable("certain-death", qx)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	got := calc.WholeLifeDue(0, 1000)
	if got <= 0 {
		t.Errorf("WholeLifeDue(0,1000) = %v, want > 0", got)
	}
}

func BenchmarkWholeLifeImmediate(b *testing.B) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("bench", qx)
	converter := rates.NewRateConverter(0.04)
	calc := NewAnnuityCalculator(converter, mort)

	for b.Loop() {
		_ = calc.WholeLifeImmediate(30, 1000)
	}
}

func BenchmarkTermImmediate(b *testing.B) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("bench", qx)
	converter := rates.NewRateConverter(0.04)
	calc := NewAnnuityCalculator(converter, mort)

	for b.Loop() {
		_ = calc.TermImmediate(30, 20, 1000)
	}
}

func TestWholeLifeNSP_ExactValues(t *testing.T) {
	// Uniform 10% mortality, i=0
	qx := []float64{0.1, 0.1, 0.1, 0.1, 0.1, 0.1}
	mort := mortality.NewTable("uniform-10", qx)
	converter := rates.NewRateConverter(0.0)
	calc := NewAnnuityCalculator(converter, mort)

	// A_x = sum(q(x+t-1) * v^t)
	// With i=0, v=1. A_0 = 0.1*1 + 0.1*1 + 0.1*1 + 0.1*1 + 0.1*1 + 0.1*1 = 0.6
	got := calc.WholeLifeNSP(0, 1000)
	expected := 1000.0 * 0.6
	if !floatEquals(got, expected) {
		t.Errorf("WholeLifeNSP(0,1000) = %.6f, want %.6f", got, expected)
	}
}

func TestTermNSP_ExactValues(t *testing.T) {
	// Uniform 10% mortality, i=0
	qx := []float64{0.1, 0.1, 0.1, 0.1, 0.1, 0.1}
	mort := mortality.NewTable("uniform-10", qx)
	converter := rates.NewRateConverter(0.0)
	calc := NewAnnuityCalculator(converter, mort)

	// A^1_{0:3} = q0*1*1 + Px(0,1)*q1*1 + Px(0,2)*q2*1
	// = 0.1*1*1 + 0.9*0.1*1 + 0.81*0.1*1 = 0.1 + 0.09 + 0.081 = 0.271
	got := calc.TermNSP(0, 3, 1000)
	expected := 1000.0 * (0.1 + 0.9*0.1 + 0.81*0.1)
	if !floatEquals(got, expected) {
		t.Errorf("TermNSP(0,3,1000) = %.6f, want %.6f", got, expected)
	}
}

func TestEndowmentNSP_Decomposition(t *testing.T) {
	qx := []float64{0.1, 0.1, 0.1, 0.1, 0.1}
	mort := mortality.NewTable("test", qx)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	// Endowment = TermNSP + survival benefit
	termNSP := calc.TermNSP(0, 3, 1000)
	survival := 1000.0 * mort.Px(0, 3) * converter.Discount(3)
	got := calc.EndowmentNSP(0, 3, 1000)
	expected := termNSP + survival

	if !floatEquals(got, expected) {
		t.Errorf("EndowmentNSP = %.6f, want %.6f (term=%.6f + surv=%.6f)", got, expected, termNSP, survival)
	}
}

func TestLifeInsurance_EdgeCases(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	if got := calc.WholeLifeNSP(-1, 1000); got != 0 {
		t.Errorf("WholeLifeNSP negative age = %v, want 0", got)
	}
	if got := calc.TermNSP(30, 0, 1000); got != 0 {
		t.Errorf("TermNSP zero term = %v, want 0", got)
	}
	if got := calc.EndowmentNSP(30, 10, 0); got != 0 {
		t.Errorf("EndowmentNSP zero amount = %v, want 0", got)
	}

	// Zero mortality: no deaths, so NSP should be 0
	if got := calc.WholeLifeNSP(30, 1000); got != 0 {
		t.Errorf("WholeLifeNSP zero mortality = %v, want 0", got)
	}
}

func ExampleAnnuityCalculator_TermImmediate() {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	pv := calc.TermImmediate(30, 10, 1000)
	fmt.Printf("%.2f\n", pv)
	// Output: 7721.73
}

func ExampleAnnuityCalculator_WholeLifeImmediate() {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := NewAnnuityCalculator(converter, mort)

	pv := calc.WholeLifeImmediate(65, 10000)
	fmt.Printf("%.2f\n", pv)
	// Output: 186334.72
}
