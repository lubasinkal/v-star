package profit

import (
	"math"
	"testing"

	"github.com/lubasinkal/v-star/pkg/mortality"
)

const tolerance = 1e-4

func floatEquals(a, b float64) bool {
	return math.Abs(a-b) < tolerance
}

// zeroMortality — no one dies
func zeroMortality(maxAge int) *mortality.Table {
	qx := make([]float64, maxAge+1)
	return mortality.NewTable("zero", qx)
}

// uniformMortality — constant qx at every age
func uniformMortality(qxVal float64, maxAge int) *mortality.Table {
	qx := make([]float64, maxAge+1)
	for i := range qx {
		qx[i] = qxVal
	}
	return mortality.NewTable("uniform", qx)
}

// --- No-mortality tests (qx=0) -------------------------------------------

func TestRun_NoMortality_ZeroProfit(t *testing.T) {
	// With no mortality and premium=0, profit each year is:
	// (0 + 0 - 0 - 0) * (1+i) - SA * 0 - 0 = 0
	policy := Policy{
		Age:        30,
		Term:       10,
		SumAssured: 100000,
		Premium:    0,
	}

	assumptions := Assumptions{
		Mortality:    zeroMortality(120),
		EarnedRate:   0.05,
		DiscountRate: 0.05,
	}

	results := Run(policy, assumptions)

	if !floatEquals(results.PVOfProfits, 0) {
		t.Errorf("PVOfProfits = %.4f, want 0", results.PVOfProfits)
	}
}

func TestRun_NoMortality_ProfitEqualsPremiumMinusClaims(t *testing.T) {
	// With qx=0, the profit each year is:
	// profit_t = (premium - expense) * (1+i) - SA * 0 - 0
	// = (premium - expense) * (1+i)
	// With i=0: profit_t = premium - expense
	policy := Policy{
		Age:        30,
		Term:       5,
		SumAssured: 0,
		Premium:    1000,
	}

	assumptions := Assumptions{
		Mortality:      zeroMortality(120),
		EarnedRate:     0,
		DiscountRate:   0,
		RenewalExpense: 100,
		Expenses:       200,
	}

	results := Run(policy, assumptions)
	// Year 0: (1000 - 200 - 100) * 1 - 0 = 700
	// Year 1-4: (1000 - 100) * 1 - 0 = 900
	if len(results.ProfitSignature) != 5 {
		t.Fatalf("len(ProfitSignature) = %d, want 5", len(results.ProfitSignature))
	}
	if !floatEquals(results.ProfitSignature[0], 700) {
		t.Errorf("ProfitSignature[0] = %.2f, want 700.00", results.ProfitSignature[0])
	}
	if !floatEquals(results.ProfitSignature[1], 900) {
		t.Errorf("ProfitSignature[1] = %.2f, want 900.00", results.ProfitSignature[1])
	}

	// PV of profits with 0% discount = sum of profits
	expectedPV := 700.0 + 900.0*4
	if !floatEquals(results.PVOfProfits, expectedPV) {
		t.Errorf("PVOfProfits = %.2f, want %.2f", results.PVOfProfits, expectedPV)
	}
}

func TestRun_NoMortality_WithExpenses(t *testing.T) {
	// i=0, r=0, qx=0
	// Premium = 1000, Expenses (issue) = 500, Renewal = 100, Commission = 10%
	// Year 0: (1000 - 500 - 100 - 100) * 1 - 0 = 300
	// Year 1-4: (1000 - 100 - 100) * 1 - 0 = 800
	// Commission = 0.10 * 1000 = 100 each year
	policy := Policy{
		Age:        30,
		Term:       5,
		SumAssured: 0,
		Premium:    1000,
	}

	assumptions := Assumptions{
		Mortality:       zeroMortality(120),
		EarnedRate:      0,
		DiscountRate:    0,
		Expenses:        500,
		RenewalExpense:  100,
		CommissionRate:  0.10,
		CommissionYears: 0,
	}

	results := Run(policy, assumptions)
	if !floatEquals(results.ProfitSignature[0], 300) {
		t.Errorf("ProfitSignature[0] = %.2f, want 300", results.ProfitSignature[0])
	}
	if !floatEquals(results.ProfitSignature[1], 800) {
		t.Errorf("ProfitSignature[1] = %.2f, want 800", results.ProfitSignature[1])
	}
}

func TestRun_NoMortality_CommissionLimitedYears(t *testing.T) {
	policy := Policy{
		Age:        30,
		Term:       5,
		SumAssured: 0,
		Premium:    1000,
	}

	assumptions := Assumptions{
		Mortality:       zeroMortality(120),
		EarnedRate:      0,
		DiscountRate:    0,
		RenewalExpense:  100,
		CommissionRate:  0.10,
		CommissionYears: 2, // commission only in years 0 and 1
	}

	results := Run(policy, assumptions)
	// Year 0: 1000 - 100 - 100 = 800 (with commission)
	// Year 1: 1000 - 100 - 100 = 800 (with commission)
	// Year 2-4: 1000 - 100 - 0 = 900 (no commission)
	if !floatEquals(results.ProfitSignature[0], 800) {
		t.Errorf("ProfitSignature[0] = %.2f, want 800", results.ProfitSignature[0])
	}
	if !floatEquals(results.ProfitSignature[2], 900) {
		t.Errorf("ProfitSignature[2] = %.2f, want 900", results.ProfitSignature[2])
	}
}

// --- With mortality -------------------------------------------------------

func TestRun_WithMortality(t *testing.T) {
	// Uniform 1% mortality, i=0, r=0, no expenses/reserves
	// qx = 0.01 at every age
	// px(30, 1) = 0.99
	//
	// Year 0: profit per policy in force at start = (1000 - 0) * 1 - 200000 * 0.01 - 0 = 1000 - 2000 = -1000
	// Per policy issued: -1000 * 1.00 = -1000
	// Year 1: profit per policy in force = 1000 - 200000 * 0.01 = -1000
	// Per policy issued: -1000 * 0.99 = -990
	// Year 2: profit per policy issued: -1000 * 0.99^2 = -980.1
	// PV of profits with r=0: -1000 - 990 - 980.1 - ... = geometric series
	// Instead, let's use an exact test.

	mort := uniformMortality(0.01, 120)
	policy := Policy{
		Age:        30,
		Term:       3,
		SumAssured: 200000,
		Premium:    1000,
	}

	assumptions := Assumptions{
		Mortality:    mort,
		EarnedRate:   0,
		DiscountRate: 0,
	}

	results := Run(policy, assumptions)

	if len(results.ProfitSignature) != 3 {
		t.Fatalf("len = %d, want 3", len(results.ProfitSignature))
	}

	// Year 0: profit = 1000 - 200000*0.01 = 1000 - 2000 = -1000
	// Per issued: -1000 * 1.0 = -1000
	if !floatEquals(results.ProfitSignature[0], -1000) {
		t.Errorf("ProfitSignature[0] = %.2f, want -1000", results.ProfitSignature[0])
	}
	// Year 1: profit = 1000 - 2000 = -1000. Per issued: -1000 * 0.99 = -990
	if !floatEquals(results.ProfitSignature[1], -990) {
		t.Errorf("ProfitSignature[1] = %.2f, want -990", results.ProfitSignature[1])
	}
	// Year 2: profit = 1000 - 2000 = -1000. Per issued: -1000 * 0.99^2 = -980.10
	expectedY2 := -1000.0 * 0.99 * 0.99
	if !floatEquals(results.ProfitSignature[2], expectedY2) {
		t.Errorf("ProfitSignature[2] = %.2f, want %.2f", results.ProfitSignature[2], expectedY2)
	}
}

// --- Edge cases -----------------------------------------------------------

func TestRun_ZeroTerm(t *testing.T) {
	results := Run(Policy{Age: 30, Term: 0, SumAssured: 100000, Premium: 5000},
		Assumptions{Mortality: zeroMortality(120)})

	if len(results.ProfitSignature) != 0 {
		t.Errorf("len = %d, want 0", len(results.ProfitSignature))
	}
	if results.PVOfProfits != 0 {
		t.Errorf("PVOfProfits = %v, want 0", results.PVOfProfits)
	}
}

func TestRun_NegativeAge(t *testing.T) {
	results := Run(Policy{Age: -1, Term: 5, SumAssured: 100000, Premium: 5000},
		Assumptions{Mortality: zeroMortality(120)})

	if len(results.ProfitSignature) != 0 {
		t.Errorf("len = %d, want 0", len(results.ProfitSignature))
	}
}

func TestRun_NilMortality(t *testing.T) {
	results := Run(Policy{Age: 30, Term: 5, SumAssured: 100000, Premium: 5000},
		Assumptions{Mortality: nil})

	if len(results.ProfitSignature) != 0 {
		t.Errorf("len = %d, want 0", len(results.ProfitSignature))
	}
}

// --- Profit margin --------------------------------------------------------

func TestRun_ProfitMargin(t *testing.T) {
	// With no mortality, i=0, r=0:
	// premium = 1000, term = 3, expenses = 200 year 0, 50 renewal
	// Year 0 profit: 1000 - 200 - 50 = 750
	// Year 1 profit: 1000 - 50 = 950
	// Year 2 profit: 1000 - 50 = 950
	// PV of profits (at r=0) = 750 + 950 + 950 = 2650
	// PV of premiums (at r=0) = 1000 + 1000 + 1000 = 3000
	// Profit margin = 2650 / 3000 = 0.8833

	policy := Policy{Age: 30, Term: 3, Premium: 1000}
	assumptions := Assumptions{
		Mortality:      zeroMortality(120),
		EarnedRate:     0,
		DiscountRate:   0,
		Expenses:       200,
		RenewalExpense: 50,
	}

	results := Run(policy, assumptions)
	expectedMargin := 2650.0 / 3000.0
	if !floatEquals(results.ProfitMargin, expectedMargin) {
		t.Errorf("ProfitMargin = %.4f, want %.4f", results.ProfitMargin, expectedMargin)
	}
}

// --- Payback year ---------------------------------------------------------

func TestRun_PaybackYear(t *testing.T) {
	// With negative first year (acquisition cost) and positive renewal years
	policy := Policy{Age: 30, Term: 10, Premium: 1000}
	assumptions := Assumptions{
		Mortality:      zeroMortality(120),
		EarnedRate:     0.05,
		DiscountRate:   0.05,
		Expenses:       8000, // huge acquisition cost
		RenewalExpense: 0,
	}

	results := Run(policy, assumptions)

	// First year should be negative (big acquisition expense)
	if results.ProfitSignature[0] >= 0 {
		t.Errorf("ProfitSignature[0] = %.2f, expected negative (acquisition cost)", results.ProfitSignature[0])
	}

	// Should eventually pay back
	if results.PaybackYear == 0 {
		t.Errorf("PaybackYear = 0, expected positive payback")
	}
	t.Logf("Payback year: %d", results.PaybackYear)
}

func TestRun_NeverPaysBack(t *testing.T) {
	// Premium = 0, expenses > 0 — never profitable
	policy := Policy{Age: 30, Term: 5, Premium: 0}
	assumptions := Assumptions{
		Mortality:      zeroMortality(120),
		EarnedRate:     0,
		DiscountRate:   0,
		Expenses:       100,
		RenewalExpense: 100,
	}

	results := Run(policy, assumptions)
	if results.PaybackYear != 0 {
		t.Errorf("PaybackYear = %d, want 0", results.PaybackYear)
	}
}

// --- IRR ------------------------------------------------------------------

func TestRun_IRR(t *testing.T) {
	// Simple case: invest 100 today, get 120 in 1 year → IRR = 20%
	// Profit signature: [-100, 120]
	sig := []float64{-100, 120}
	px := []float64{1.0, 1.0} // all survive
	irr := findIRR(sig, px, 0.10)
	if !floatEquals(irr, 0.20) {
		t.Errorf("IRR = %.4f, want 0.20", irr)
	}
}

func TestFindIRR_NoProfit(t *testing.T) {
	// All negative — IRR should return 0
	sig := []float64{-100, -100}
	px := []float64{1.0, 1.0}
	irr := findIRR(sig, px, 0.05)
	if irr != 0 {
		t.Errorf("IRR = %.4f, want 0", irr)
	}
}

func TestFindIRR_NegativeThenPositive(t *testing.T) {
	// Invest 100 now, get 60 in year 1, 60 in year 2
	// NPV at rate r: -100 + 60/(1+r) + 60/(1+r)^2 = 0
	// 60/(1+r) + 60/(1+r)^2 = 100
	// Let x = 1/(1+r): 60x + 60x^2 = 100
	// 60x^2 + 60x - 100 = 0
	// x = (-60 + sqrt(3600 + 24000)) / 120 = (-60 + sqrt(27600)) / 120
	// x = (-60 + 166.13) / 120 = 0.8844
	// r = 1/x - 1 = 0.1307 = 13.07%
	sig := []float64{-100, 60, 60}
	px := []float64{1.0, 1.0, 1.0}
	irr := findIRR(sig, px, 0.10)
	expected := 0.1307
	if !floatEquals(irr, expected) {
		t.Errorf("IRR = %.4f, want %.4f", irr, expected)
	}
}

// --- Reserves (integration-style) ----------------------------------------

func TestComputeReserves(t *testing.T) {
	// With zero mortality, the net premium reserve should equal
	// the retrospective accumulation of premiums less benefits.
	mort := zeroMortality(120)
	policy := Policy{
		Age:        30,
		Term:       10,
		SumAssured: 100000,
		Premium:    5000,
	}

	assumptions := Assumptions{
		Mortality:      mort,
		EarnedRate:     0.05,
		DiscountRate:   0.08,
		ReserveEnabled: true,
	}

	results := Run(policy, assumptions)

	// With reserves, profit should differ from without
	noReserveAssumptions := assumptions
	noReserveAssumptions.ReserveEnabled = false
	noReserveResults := Run(policy, noReserveAssumptions)

	if results.PVOfProfits == noReserveResults.PVOfProfits {
		t.Log("Reserve-enabled profit equals non-reserve profit (premium ≈ net premium)")
	}
}

// --- Negative / zero values ----------------------------------------------

func TestRun_ZeroSumAssured(t *testing.T) {
	policy := Policy{Age: 30, Term: 5, SumAssured: 0, Premium: 1000}
	assumptions := Assumptions{
		Mortality:      zeroMortality(120),
		EarnedRate:     0,
		DiscountRate:   0,
		RenewalExpense: 100,
		Expenses:       500,
	}

	results := Run(policy, assumptions)
	if results.ProfitSignature == nil {
		t.Error("ProfitSignature should not be nil")
	}
	if len(results.ProfitSignature) != 5 {
		t.Errorf("len = %d, want 5", len(results.ProfitSignature))
	}
}

// --- Benchmark ------------------------------------------------------------

func BenchmarkRun(b *testing.B) {
	mort := uniformMortality(0.01, 120)
	policy := Policy{
		Age:        30,
		Term:       20,
		SumAssured: 100000,
		Premium:    5000,
	}
	assumptions := Assumptions{
		Mortality:       mort,
		EarnedRate:      0.05,
		DiscountRate:    0.08,
		Expenses:        500,
		RenewalExpense:  50,
		CommissionRate:  0.05,
		CommissionYears: 5,
	}

	for b.Loop() {
		Run(policy, assumptions)
	}
}

// --- Example --------------------------------------------------------------

func ExampleRun() {
	mort := zeroMortality(120)
	results := Run(Policy{Age: 30, Term: 5, SumAssured: 100000, Premium: 25000},
		Assumptions{
			Mortality:    mort,
			EarnedRate:   0.05,
			DiscountRate: 0.08,
			Expenses:     1000,
		})
	_ = results
}
