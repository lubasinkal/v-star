package reserves

import (
	"math"
	"testing"

	"github.com/lubasinkal/v-star/pkg/annuities"
	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
)

const tolerance = 1e-4

func floatEquals(got, want float64) bool {
	return math.Abs(got-want) < tolerance
}

func zeroMortalityTable(maxAge int) *mortality.Table {
	qx := make([]float64, maxAge+1)
	return mortality.NewTable("zero-mort", qx)
}

func TestProspectiveReserve_ExactValues(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := annuities.NewAnnuityCalculator(converter, mort)

	policy := PolicySpec{Age: 30, Term: 3, SumAssured: 1000, Premium: 300}

	// ProspectiveReserve = futureBenefits - futurePremiums
	// = sa * A_x:n (NSP) - prem * ä_x:n (annuity-due)
	// With zero mortality, A_x:n = 0 (no deaths, no claims)
	futureBenefits := calc.TermNSP(30, 3, 1000.0)  // 0 with qx=0
	futurePremiums := calc.TermDue(30, 3, 300.0)    // premiums at start of year
	expected := futureBenefits - futurePremiums      // = -futurePremiums
	got := ProspectiveReserve(policy, converter, mort)
	if !floatEquals(got, expected) {
		t.Errorf("ProspectiveReserve = %.6f, want %.6f", got, expected)
	}
	// With zero mortality, term insurance has no claims, so prospective
	// reserve is negative (premiums collected with no offsetting benefits)
	if got >= 0 {
		t.Errorf("ProspectiveReserve = %.6f, expected negative (zero mortality, no claims)", got)
	}
}

func TestProspectiveReserve_NetPremiumEquivalence(t *testing.T) {
	// Under the equivalence principle, when premium = net premium,
	// the reserve at issue (t=0) should be 0.
	// netPremium = SA * A_x:n / ä_x:n
	// Use non-zero mortality so A_x:n > 0
	qx := make([]float64, 121)
	for i := 50; i <= 120; i++ {
		qx[i] = 0.02 // 2% mortality from age 50
	}
	mort := mortality.NewTable("nonzero-mort", qx)
	converter := rates.NewRateConverter(0.05)
	calc := annuities.NewAnnuityCalculator(converter, mort)

	sa := 100000.0
	nsp := calc.TermNSP(30, 10, sa)
	due := calc.TermDue(30, 10, 1.0)
	netPrem := nsp / due

	policy := PolicySpec{Age: 30, Term: 10, SumAssured: sa, Premium: netPrem}
	got := ProspectiveReserve(policy, converter, mort)
	if !floatEquals(got, 0) {
		t.Errorf("ProspectiveReserve at issue with net premium = %.6f, want 0", got)
	}
}

type mockDiscount struct {
	rate float64
}

func (m mockDiscount) Discount(term int) float64 {
	if term <= 0 {
		return 1
	}
	v := 1 / (1 + m.rate)
	result := 1.0
	for range term {
		result *= v
	}
	return result
}

func TestProspectiveReserve_GenericPath(t *testing.T) {
	mort := zeroMortalityTable(120)
	discount := mockDiscount{rate: 0.05}

	policy := PolicySpec{Age: 30, Term: 3, SumAssured: 1000, Premium: 300}

	got := ProspectiveReserve(policy, discount, mort)
	// With zero mortality there are no death claims, so the reserve is
	// negative (premiums collected exceed expected benefits of 0).
	if got >= 0 {
		t.Errorf("ProspectiveReserve = %v, want < 0 (zero mortality, no claims)", got)
	}
}

func TestNetPremiumReserve_GenericPath(t *testing.T) {
	mort := zeroMortalityTable(120)
	discount := mockDiscount{rate: 0.05}

	policy := PolicySpec{Age: 30, Term: 3, SumAssured: 1000, Premium: 300}
	npr := NetPremiumReserve(policy, discount, mort)
	if npr <= 0 {
		t.Errorf("NetPremiumReserve = %v, want > 0", npr)
	}
}

func TestNetPremiumReserve_GenericPathFinite(t *testing.T) {
	mort := zeroMortalityTable(120)
	discount := mockDiscount{rate: 0.05}

	policy := PolicySpec{Age: 30, Term: 3, SumAssured: 1000, Premium: 300}
	npr := NetPremiumReserve(policy, discount, mort)
	if math.IsNaN(npr) || math.IsInf(npr, 0) {
		t.Errorf("NetPremiumReserve = %v, want finite", npr)
	}
}

func TestGrossPremiumReserve_GenericPath(t *testing.T) {
	mort := zeroMortalityTable(120)
	discount := mockDiscount{rate: 0.05}

	policy := PolicySpec{Age: 30, Term: 3, SumAssured: 1000, Premium: 300}
	gpr := GrossPremiumReserve(policy, 100, discount, mort)
	npr := NetPremiumReserve(policy, discount, mort)
	if gpr < npr {
		t.Errorf("GPR(%v) < NPR(%v)", gpr, npr)
	}
}

func TestRetrospectiveReserve_ExactValues(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)
	calc := annuities.NewAnnuityCalculator(converter, mort)

	policy := PolicySpec{Age: 30, Term: 1, SumAssured: 1000, Premium: 300}

	// RetrospectiveReserve for term=1, zero mortality:
	// accumulated = (0 + prem) * v / Px(30,1) = prem * v / 1 = prem * v
	// futureLiability = sa * A^1_{30:1} = sa * qx * v = 0 (qx=0)
	// reserve = prem*v - 0 = prem*v
	v := 1.0 / 1.05
	expected := 300.0 * v
	got := RetrospectiveReserve(policy, converter, mort)
	if !floatEquals(got, expected) {
		t.Errorf("RetrospectiveReserve = %.6f, want %.6f", got, expected)
	}
	// With zero mortality, term insurance has no claims, so retrospective
	// reserve is positive (accumulated premiums minus nothing owed)
	nsp := calc.TermNSP(30, 1, 1000.0)
	if !floatEquals(nsp, 0) {
		t.Errorf("TermNSP with zero mortality = %.6f, want 0", nsp)
	}
}

func TestRetrospectiveReserve_GenericPath(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)

	policy := PolicySpec{Age: 30, Term: 3, SumAssured: 1000, Premium: 300}
	got := RetrospectiveReserve(policy, converter, mort)
	if math.IsNaN(got) || math.IsInf(got, 0) {
		t.Errorf("RetrospectiveReserve = %v, want finite", got)
	}
}

func TestNetPremiumReserve_PremiumIgnored(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)

	// NetPremiumReserve computes the net premium internally and ignores the
	// Premium field. Both calls should return the same result.
	policy0 := PolicySpec{Age: 30, Term: 10, SumAssured: 100000, Premium: 0}
	policy1 := PolicySpec{Age: 30, Term: 10, SumAssured: 100000, Premium: 1000}

	npr0 := NetPremiumReserve(policy0, converter, mort)
	npr1 := NetPremiumReserve(policy1, converter, mort)

	if !floatEquals(npr0, npr1) {
		t.Errorf("NetPremiumReserve(Pre=0) = %v, NetPremiumReserve(Pre=1000) = %v, want equal", npr0, npr1)
	}
	if math.IsNaN(npr0) || math.IsInf(npr0, 0) {
		t.Errorf("NetPremiumReserve = %v, want finite", npr0)
	}
}

func TestNetPremiumReserve_TerminalValue(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)

	// For zero mortality, longer terms accumulate a higher terminal reserve
	policy10 := PolicySpec{Age: 30, Term: 10, SumAssured: 100000, Premium: 5000}
	policy20 := PolicySpec{Age: 30, Term: 20, SumAssured: 100000, Premium: 5000}

	npr10 := NetPremiumReserve(policy10, converter, mort)
	npr20 := NetPremiumReserve(policy20, converter, mort)

	if npr10 <= 0 || npr20 <= 0 {
		t.Errorf("NetPremiumReserve should be positive for zero-mort policies")
	}
	if npr20 <= npr10 {
		t.Errorf("NPR(20yr=%v) should be > NPR(10yr=%v)", npr20, npr10)
	}
}

func TestGrossPremiumReserve_GreaterThanNPR(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)

	policy := PolicySpec{Age: 30, Term: 10, SumAssured: 100000, Premium: 5000}
	npr := NetPremiumReserve(policy, converter, mort)
	gpr := GrossPremiumReserve(policy, 500, converter, mort)

	// Gross reserve should be >= net premium reserve (includes expense loading)
	if gpr < npr {
		t.Errorf("GPR(%v) < NPR(%v)", gpr, npr)
	}
}

func TestProspectiveReserve_EdgeCases(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)

	tests := []struct {
		name   string
		policy PolicySpec
		want   float64
	}{
		{"zero term", PolicySpec{Age: 30, Term: 0, SumAssured: 1000, Premium: 100}, 0},
		{"zero sa", PolicySpec{Age: 30, Term: 10, SumAssured: 0, Premium: 100}, 0},
		{"zero premium", PolicySpec{Age: 30, Term: 10, SumAssured: 1000, Premium: 0}, 0},
		{"negative age", PolicySpec{Age: -1, Term: 10, SumAssured: 1000, Premium: 100}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ProspectiveReserve(tt.policy, converter, mort)
			if got != tt.want {
				t.Errorf("ProspectiveReserve = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRetrospectiveReserve_EdgeCases(t *testing.T) {
	mort := zeroMortalityTable(120)
	converter := rates.NewRateConverter(0.05)

	tests := []struct {
		name   string
		policy PolicySpec
	}{
		{"zero term", PolicySpec{Age: 30, Term: 0, SumAssured: 1000, Premium: 100}},
		{"zero sa", PolicySpec{Age: 30, Term: 10, SumAssured: 0, Premium: 100}},
		{"zero premium", PolicySpec{Age: 30, Term: 10, SumAssured: 1000, Premium: 0}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RetrospectiveReserve(tt.policy, converter, mort)
			if got != 0 {
				t.Errorf("RetrospectiveReserve = %v, want 0", got)
			}
		})
	}
}

func BenchmarkNetPremiumReserve(b *testing.B) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("bench", qx)
	converter := rates.NewRateConverter(0.04)
	policy := PolicySpec{Age: 30, Term: 20, SumAssured: 100000, Premium: 0}

	for b.Loop() {
		_ = NetPremiumReserve(policy, converter, mort)
	}
}
