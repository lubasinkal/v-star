package reserves

import (
	"math"
	"testing"

	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
)

func TestNetPremiumReserve(t *testing.T) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.05}

	policy := PolicySpec{
		Age:        30,
		Term:       20,
		SumAssured: 100000,
		Premium:    5000,
	}

	reserve := NetPremiumReserve(policy, &converter, mort)
	if reserve <= 0 {
		t.Errorf("NetPremiumReserve = %v, want > 0", reserve)
	}
}

func TestNetPremiumReserveZeroMortality(t *testing.T) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.0}

	policy := PolicySpec{
		Age:        30,
		Term:       10,
		SumAssured: 10000,
		Premium:    1000,
	}

	reserve := NetPremiumReserve(policy, &converter, mort)
	if math.IsInf(reserve, 0) || math.IsNaN(reserve) {
		t.Errorf("NetPremiumReserve = %v, want finite value", reserve)
	}
}

func TestGrossPremiumReserve(t *testing.T) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.05}

	policy := PolicySpec{
		Age:        30,
		Term:       20,
		SumAssured: 100000,
		Premium:    5000,
	}

	reserve := GrossPremiumReserve(policy, 100, &converter, mort)
	if reserve <= 0 {
		t.Errorf("GrossPremiumReserve = %v, want > 0", reserve)
	}
}

func TestProspectiveReserve(t *testing.T) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.05}

	reserve := ProspectiveReserve(30, 20, 100000, 5000, &converter, mort)
	if reserve <= 0 {
		t.Errorf("ProspectiveReserve = %v, want > 0", reserve)
	}
}

func TestRetrospectiveReserve(t *testing.T) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("test", qx)
	converter := rates.RateConverter{EffectiveRate: 0.05}

	reserve := RetrospectiveReserve(30, 20, 100000, 5000, &converter, mort)
	if math.IsInf(reserve, 0) || math.IsNaN(reserve) {
		t.Errorf("RetrospectiveReserve = %v, want finite value", reserve)
	}
}

func BenchmarkNetPremiumReserve(b *testing.B) {
	qx := make([]float64, 120)
	mort := mortality.NewTable("bench", qx)
	converter := rates.RateConverter{EffectiveRate: 0.04}

	policy := PolicySpec{
		Age:        30,
		Term:       20,
		SumAssured: 100000,
		Premium:    5000,
	}

	for i := 0; i < b.N; i++ {
		_ = NetPremiumReserve(policy, &converter, mort)
	}
}
