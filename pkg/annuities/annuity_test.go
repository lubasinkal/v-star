package annuities

import (
	"math"
	"testing"

	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
)

func TestWholeLifeImmediate(t *testing.T) {
	qx := []float64{0.01, 0.02, 0.03, 0.04, 0.05}
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.05}
	calc := New(&converter, mort)

	got := calc.WholeLifeImmediate(0, 1000)
	if got <= 0 {
		t.Errorf("WholeLifeImmediate(0, 1000) = %v, want > 0", got)
	}
}

func TestWholeLifeImmediateWithZeroMortality(t *testing.T) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.05}
	calc := New(&converter, mort)

	got := calc.WholeLifeImmediate(30, 1000)
	if got <= 0 {
		t.Errorf("WholeLifeImmediate = %v, want > 0", got)
	}
}

func TestTermImmediate(t *testing.T) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.05}
	calc := New(&converter, mort)

	got := calc.TermImmediate(30, 10, 1000)
	if got <= 0 {
		t.Errorf("TermImmediate = %v, want > 0", got)
	}
}

func TestTermDue(t *testing.T) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.05}
	calc := New(&converter, mort)

	got := calc.TermDue(30, 10, 1000)
	if got <= 0 {
		t.Errorf("TermDue = %v, want > 0", got)
	}
}

func TestWholeLifeDue(t *testing.T) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.05}
	calc := New(&converter, mort)

	got := calc.WholeLifeDue(30, 1000)
	if got <= 0 {
		t.Errorf("WholeLifeDue = %v, want > 0", got)
	}
}

func TestDeferredWholeLife(t *testing.T) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.05}
	calc := New(&converter, mort)

	got := calc.DeferredWholeLife(30, 10, 1000)
	if got <= 0 {
		t.Errorf("DeferredWholeLife = %v, want > 0", got)
	}
}

func TestDeferredTerm(t *testing.T) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.05}
	calc := New(&converter, mort)

	got := calc.DeferredTerm(30, 10, 15, 1000)
	if got <= 0 {
		t.Errorf("DeferredTerm = %v, want > 0", got)
	}
}

func TestTermImmediateWithMortality(t *testing.T) {
	qx := []float64{0.1, 0.1, 0.1, 0.1, 0.1}
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.0}
	calc := New(&converter, mort)

	got := calc.TermImmediate(0, 3, 1000)
	px0 := mort.Px(0, 1)
	px1 := mort.Px(0, 2)
	px2 := mort.Px(0, 3)
	expected := 1000 * (px0 + px1 + px2)
	if math.Abs(got-expected) > 1 {
		t.Errorf("TermImmediate = %v, want %v", got, expected)
	}
}

func BenchmarkWholeLifeImmediate(b *testing.B) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("bench", qx)
	converter := rates.RateConverter{EffectiveRate: 0.04}
	calc := New(&converter, mort)

	for i := 0; i < b.N; i++ {
		_ = calc.WholeLifeImmediate(30, 1000)
	}
}

func BenchmarkTermImmediate(b *testing.B) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("bench", qx)
	converter := rates.RateConverter{EffectiveRate: 0.04}
	calc := New(&converter, mort)

	for i := 0; i < b.N; i++ {
		_ = calc.TermImmediate(30, 20, 1000)
	}
}
