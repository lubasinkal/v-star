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
	futureBenefits := calc.TermNSP(30, 3, 1000.0) // 0 with qx=0
	futurePremiums := calc.TermDue(30, 3, 300.0)  // premiums at start of year
	expected := futureBenefits - futurePremiums   // = -futurePremiums
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

	policy := PolicySpec{Age: 30, Term: 1, SumAssured: 1000, Premium: 300}

	// RetrospectiveReserve for term=1, zero mortality, i=5%:
	// V_1 = [(0 + 300) * (1+0.05) - 1000 * 0] / 1 = 315.00
	expected := 300.0 * 1.05
	got := RetrospectiveReserve(policy, converter, mort)
	if !floatEquals(got, expected) {
		t.Errorf("RetrospectiveReserve = %.6f, want %.6f", got, expected)
	}
}

func TestRetrospectiveReserve_WithMortality(t *testing.T) {
	// Uniform 1% mortality, i=5%, term=2, P=300, SA=1000
	qx := make([]float64, 121)
	for i := range qx {
		qx[i] = 0.01
	}
	mort := mortality.NewTable("uniform-1pct", qx)
	converter := rates.NewRateConverter(0.05)

	policy := PolicySpec{Age: 30, Term: 2, SumAssured: 1000, Premium: 300}

	// V_1 = [(0 + 300) * 1.05 - 1000 * 0.01] / 0.99
	//     = (315 - 10) / 0.99 = 305 / 0.99 = 308.0808
	// V_2 = [(308.0808 + 300) * 1.05 - 1000 * 0.01] / 0.99
	//     = (608.0808 * 1.05 - 10) / 0.99
	//     = (638.4848 - 10) / 0.99 = 628.4848 / 0.99 = 634.8329
	expectedY1 := (300.0*1.05 - 1000.0*0.01) / 0.99
	expectedY2 := ((expectedY1+300.0)*1.05 - 1000.0*0.01) / 0.99

	got := RetrospectiveReserve(policy, converter, mort)
	if !floatEquals(got, expectedY2) {
		t.Errorf("RetrospectiveReserve(term=2) = %.6f, want %.6f", got, expectedY2)
	}

	// Also verify the reserve after 1 year by computing from first principles
	policy1 := PolicySpec{Age: 30, Term: 1, SumAssured: 1000, Premium: 300}
	got1 := RetrospectiveReserve(policy1, converter, mort)
	if !floatEquals(got1, expectedY1) {
		t.Errorf("RetrospectiveReserve(term=1) = %.6f, want %.6f", got1, expectedY1)
	}
}

func TestRetrospectiveReserve_MatchesProspective(t *testing.T) {
	// Under the equivalence principle (premium = net premium), the
	// prospective and retrospective methods give the same reserve at
	// the same valuation date.
	//
	// This tests the identity: V_t(pros) = V_t(retro) for t = 5.
	//
	// Note: ProspectiveReserve({age, term}) returns V_0 (at issue),
	// while RetrospectiveReserve({age, term}) returns V_term (at end
	// of term). These are at DIFFERENT time points. To compare at the
	// same t, you must offset age+term accordingly.
	qx := make([]float64, 121)
	for i := 40; i <= 120; i++ {
		qx[i] = 0.01
	}
	mort := mortality.NewTable("age40-plus", qx)
	converter := rates.NewRateConverter(0.05)
	calc := annuities.NewAnnuityCalculator(converter, mort)

	sa := 100000.0
	term := 10
	age := 35

	// Net premium from equivalence principle: P = SA * A_x:n / ä_x:n
	nsp := calc.TermNSP(age, term, sa)
	due := calc.TermDue(age, term, 1.0)
	netPrem := nsp / due

	// --- Mid-term (t=5): same time point, should match ---
	//   Retrospective: accumulate issue→t=5: {age=35, term=5}
	//   Prospective: value remaining t=5→term: {age=40, term=5}
	retro5 := RetrospectiveReserve(
		PolicySpec{Age: age, Term: 5, SumAssured: sa, Premium: netPrem}, converter, mort)
	pros5 := ProspectiveReserve(
		PolicySpec{Age: age + 5, Term: term - 5, SumAssured: sa, Premium: netPrem}, converter, mort)
	if !floatEquals(pros5, retro5) {
		t.Errorf("At t=5: Prospective(age+5) = %.4f, Retrospective(term=5) = %.4f, want equal", pros5, retro5)
	}

	// --- At issue (t=0): only Prospective gives V_0 ---
	// ProspectiveReserve({age, term}) = V_0. With net premium, V_0 = 0.
	// RetrospectiveReserve({age, term}) = V_term (different time point!).
	v0 := ProspectiveReserve(
		PolicySpec{Age: age, Term: term, SumAssured: sa, Premium: netPrem}, converter, mort)
	if !floatEquals(v0, 0) {
		t.Errorf("V_0(pros) with net premium = %.4f, want 0", v0)
	}

	// --- Terminal (t=term): only Retrospective gives V_term ---
	// With net premium over the full term, accumulated premiums = accumulated
	// claims, so V_term = 0. Prospective at t=term would need term=0 (returns 0).
	vTerm := RetrospectiveReserve(
		PolicySpec{Age: age, Term: term, SumAssured: sa, Premium: netPrem}, converter, mort)
	if !floatEquals(vTerm, 0) {
		t.Errorf("V_term(retro) with net premium = %.4f, want 0", vTerm)
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
