package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/lubasinkal/v-star/pkg/annuities"
	"github.com/lubasinkal/v-star/pkg/concurrency"
	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/profit"
	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reserves"
	"github.com/lubasinkal/v-star/pkg/risk"
	"github.com/lubasinkal/v-star/pkg/stochastic"
)

// --- Request / Response types ------------------------------------------------

type PVRequest struct {
	InterestRate float64    `json:"interest_rate"`
	RateJ        float64    `json:"rate_j,omitempty"`
	Records      []PVRecord `json:"records"`
	Parallel     bool       `json:"parallel,omitempty"`
	Workers      int        `json:"workers,omitempty"`
}

type PVRecord struct {
	SumAssured float64 `json:"sum_assured"`
	Term       int     `json:"term"`
	Age        int     `json:"age,omitempty"`
	Sex        string  `json:"sex,omitempty"`
	PolicyType string  `json:"policy_type,omitempty"`
}

type PVResponse struct {
	TotalPV      float64 `json:"total_pv"`
	RecordCount  int     `json:"record_count"`
	ProcessingMs int64   `json:"processing_ms"`
}

type SimulateRequest struct {
	Model         string  `json:"model"` // "gbm" or "vasicek"
	InitialRate   float64 `json:"initial_rate"`
	Drift         float64 `json:"drift,omitempty"` // GBM μ
	Volatility    float64 `json:"volatility"`
	LongTermMean  float64 `json:"long_term_mean,omitempty"` // Vasicek b
	MeanReversion float64 `json:"mean_reversion,omitempty"` // Vasicek a
	NumPaths      int     `json:"num_paths"`
	Steps         int     `json:"steps"`
	Dt            float64 `json:"dt,omitempty"`
	Seed          int64   `json:"seed,omitempty"`
	IncludePaths  bool    `json:"include_paths,omitempty"`
	NumWorkers    int     `json:"num_workers,omitempty"`
}

type SimulateResponse struct {
	Paths        []stochastic.RatePath `json:"paths,omitempty"`
	Mean         float64               `json:"mean"`
	StdDev       float64               `json:"std_dev"`
	VaR95        float64               `json:"var_95"`
	CTE95        float64               `json:"cte_95"`
	ProcessingMs int64                 `json:"processing_ms"`
}

type AnnuityRequest struct {
	InterestRate float64   `json:"interest_rate"`
	Qxs          []float64 `json:"qxs"` // qx values indexed by age, starting at 0
	Age          int       `json:"age"`
	Term         int       `json:"term,omitempty"` // 0 = whole life where applicable
	Deferment    int       `json:"deferment,omitempty"`
	Amount       float64   `json:"amount"`
	Computation  string    `json:"computation"` // see validComputations
}

type AnnuityResponse struct {
	PresentValue float64 `json:"present_value"`
	ProcessingMs int64   `json:"processing_ms"`
}

type ReserveRequest struct {
	InterestRate float64   `json:"interest_rate"`
	Qxs          []float64 `json:"qxs"`
	Age          int       `json:"age"`
	Term         int       `json:"term"`
	SumAssured   float64   `json:"sum_assured"`
	Premium      float64   `json:"premium,omitempty"`
	Expenses     float64   `json:"expenses,omitempty"`
	Method       string    `json:"method"` // "net_premium", "gross_premium", "prospective", "retrospective"
}

type ReserveResponse struct {
	Reserve      float64 `json:"reserve"`
	ProcessingMs int64   `json:"processing_ms"`
}

var validComputations = map[string]bool{
	"whole_life_immediate": true,
	"whole_life_due":       true,
	"term_immediate":       true,
	"term_due":             true,
	"deferred_whole_life":  true,
	"deferred_term":        true,
	"whole_life_nsp":       true,
	"term_nsp":             true,
	"endowment_nsp":        true,
}

type ProfitRequest struct {
	EarnedRate      float64   `json:"earned_rate"`
	DiscountRate    float64   `json:"discount_rate"`
	Qxs             []float64 `json:"qxs"`
	Age             int       `json:"age"`
	Term            int       `json:"term"`
	SumAssured      float64   `json:"sum_assured"`
	Premium         float64   `json:"premium"`
	Expenses        float64   `json:"expenses,omitempty"`
	RenewalExpense  float64   `json:"renewal_expense,omitempty"`
	CommissionRate  float64   `json:"commission_rate,omitempty"`
	CommissionYears int       `json:"commission_years,omitempty"`
	ReserveEnabled  bool      `json:"reserve_enabled,omitempty"`
}

type ProfitResponse struct {
	ProfitSignature  []float64 `json:"profit_signature"`
	CumulativeProfit []float64 `json:"cumulative_profit"`
	ProfitVector     []float64 `json:"profit_vector"`
	PVOfProfits      float64   `json:"pv_of_profits"`
	PVOfPremiums     float64   `json:"pv_of_premiums"`
	ProfitMargin     float64   `json:"profit_margin"`
	IRR              float64   `json:"irr"`
	PaybackYear      int       `json:"payback_year"`
	ProcessingMs     int64     `json:"processing_ms"`
}

var validReserveMethods = map[string]bool{
	"net_premium":   true,
	"gross_premium": true,
	"prospective":   true,
	"retrospective": true,
}

// --- Handlers ----------------------------------------------------------------

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) pvHandler(w http.ResponseWriter, r *http.Request) {
	var req PVRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	start := time.Now()
	totalPV := computePresentValues(req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PVResponse{
		TotalPV:      totalPV,
		RecordCount:  len(req.Records),
		ProcessingMs: time.Since(start).Milliseconds(),
	})
}

func (s *Server) simulateHandler(w http.ResponseWriter, r *http.Request) {
	var req SimulateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.Model == "" {
		req.Model = "gbm"
	}
	if req.NumPaths <= 0 {
		req.NumPaths = 10000
	}
	if req.Steps <= 0 {
		req.Steps = 10
	}
	if req.Dt <= 0 {
		req.Dt = 1.0
	}

	start := time.Now()

	var paths []stochastic.RatePath

	switch req.Model {
	case "vasicek":
		vg := stochastic.NewVasicekGenerator(req.InitialRate, req.LongTermMean, req.MeanReversion, req.Volatility)
		if req.NumWorkers > 0 || req.NumPaths > 1000 {
			workers := req.NumWorkers
			if workers <= 0 {
				workers = runtime.NumCPU()
			}
			paths = vg.GeneratePathsParallel(req.NumPaths, req.Steps, workers, req.Dt)
		} else {
			paths = vg.GeneratePaths(req.NumPaths, req.Steps, req.Dt)
		}

	default: // "gbm"
		var rg *stochastic.RateGenerator
		if req.Seed > 0 {
			rg = stochastic.NewRateGeneratorWithSeed(req.InitialRate, req.Drift, req.Volatility, uint64(req.Seed))
		} else {
			rg = stochastic.NewRateGenerator(req.InitialRate, req.Drift, req.Volatility)
		}
		if req.NumWorkers > 0 || req.NumPaths > 1000 {
			workers := req.NumWorkers
			if workers <= 0 {
				workers = runtime.NumCPU()
			}
			paths = rg.GeneratePathsParallel(req.NumPaths, req.Steps, workers, req.Dt)
		} else {
			paths = rg.GeneratePaths(req.NumPaths, req.Steps, req.Dt)
		}
	}

	losses := make([]float64, len(paths))
	for i, path := range paths {
		losses[i] = path[req.Steps]
	}

	report := risk.ComputeReport(losses)

	w.Header().Set("Content-Type", "application/json")
	resp := SimulateResponse{
		Mean:         report.Mean,
		StdDev:       report.StdDev,
		VaR95:        report.VaR95,
		CTE95:        report.CTE95,
		ProcessingMs: time.Since(start).Milliseconds(),
	}
	if req.IncludePaths {
		resp.Paths = paths
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) annuityHandler(w http.ResponseWriter, r *http.Request) {
	var req AnnuityRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !validComputations[req.Computation] {
		http.Error(w, fmt.Sprintf("invalid computation %q", req.Computation), http.StatusBadRequest)
		return
	}
	if len(req.Qxs) == 0 {
		http.Error(w, "qxs array is required", http.StatusBadRequest)
		return
	}
	if req.InterestRate <= 0 {
		http.Error(w, "interest_rate must be positive", http.StatusBadRequest)
		return
	}
	if req.Amount <= 0 {
		http.Error(w, "amount must be positive", http.StatusBadRequest)
		return
	}

	start := time.Now()

	discount := rates.NewRateConverter(req.InterestRate)
	mort := mortality.NewTable("inline", req.Qxs)
	calc := annuities.NewAnnuityCalculator(discount, mort)

	var result float64
	switch req.Computation {
	case "whole_life_immediate":
		result = calc.WholeLifeImmediate(req.Age, req.Amount)
	case "whole_life_due":
		result = calc.WholeLifeDue(req.Age, req.Amount)
	case "term_immediate":
		result = calc.TermImmediate(req.Age, req.Term, req.Amount)
	case "term_due":
		result = calc.TermDue(req.Age, req.Term, req.Amount)
	case "deferred_whole_life":
		result = calc.DeferredWholeLife(req.Age, req.Deferment, req.Amount)
	case "deferred_term":
		result = calc.DeferredTerm(req.Age, req.Deferment, req.Term, req.Amount)
	case "whole_life_nsp":
		result = calc.WholeLifeNSP(req.Age, req.Amount)
	case "term_nsp":
		result = calc.TermNSP(req.Age, req.Term, req.Amount)
	case "endowment_nsp":
		result = calc.EndowmentNSP(req.Age, req.Term, req.Amount)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(AnnuityResponse{
		PresentValue: result,
		ProcessingMs: time.Since(start).Milliseconds(),
	})
}

func (s *Server) reserveHandler(w http.ResponseWriter, r *http.Request) {
	var req ReserveRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !validReserveMethods[req.Method] {
		http.Error(w, fmt.Sprintf("invalid method %q", req.Method), http.StatusBadRequest)
		return
	}
	if len(req.Qxs) == 0 {
		http.Error(w, "qxs array is required", http.StatusBadRequest)
		return
	}
	if req.InterestRate <= 0 {
		http.Error(w, "interest_rate must be positive", http.StatusBadRequest)
		return
	}
	if req.SumAssured <= 0 {
		http.Error(w, "sum_assured must be positive", http.StatusBadRequest)
		return
	}

	start := time.Now()

	discount := rates.NewRateConverter(req.InterestRate)
	mort := mortality.NewTable("inline", req.Qxs)

	policy := reserves.PolicySpec{
		Age:        req.Age,
		Term:       req.Term,
		SumAssured: req.SumAssured,
		Premium:    req.Premium,
	}

	var result float64
	switch req.Method {
	case "net_premium":
		result = reserves.NetPremiumReserve(policy, discount, mort)
	case "gross_premium":
		result = reserves.GrossPremiumReserve(policy, req.Expenses, discount, mort)
	case "prospective":
		result = reserves.ProspectiveReserve(policy, discount, mort)
	case "retrospective":
		result = reserves.RetrospectiveReserve(policy, discount, mort)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ReserveResponse{
		Reserve:      result,
		ProcessingMs: time.Since(start).Milliseconds(),
	})
}

func (s *Server) profitHandler(w http.ResponseWriter, r *http.Request) {
	var req ProfitRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if len(req.Qxs) == 0 {
		http.Error(w, "qxs array is required", http.StatusBadRequest)
		return
	}
	if req.Term <= 0 {
		http.Error(w, "term must be positive", http.StatusBadRequest)
		return
	}
	if req.Age < 0 {
		http.Error(w, "age must be non-negative", http.StatusBadRequest)
		return
	}

	start := time.Now()

	mort := mortality.NewTable("inline", req.Qxs)
	assumptions := profit.Assumptions{
		Mortality:       mort,
		EarnedRate:      req.EarnedRate,
		DiscountRate:    req.DiscountRate,
		Expenses:        req.Expenses,
		RenewalExpense:  req.RenewalExpense,
		CommissionRate:  req.CommissionRate,
		CommissionYears: req.CommissionYears,
		ReserveEnabled:  req.ReserveEnabled,
	}
	policy := profit.Policy{
		Age:        req.Age,
		Term:       req.Term,
		SumAssured: req.SumAssured,
		Premium:    req.Premium,
	}

	results := profit.Run(policy, assumptions)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ProfitResponse{
		ProfitSignature:  results.ProfitSignature,
		CumulativeProfit: results.CumulativeProfit,
		ProfitVector:     results.ProfitVector,
		PVOfProfits:      results.PVOfProfits,
		PVOfPremiums:     results.PVOfPremiums,
		ProfitMargin:     results.ProfitMargin,
		IRR:              results.IRR,
		PaybackYear:      results.PaybackYear,
		ProcessingMs:     time.Since(start).Milliseconds(),
	})
}

// --- Helpers -----------------------------------------------------------------

func computePresentValues(req PVRequest) float64 {
	converter := rates.NewRateConverter(req.InterestRate)

	if req.Parallel || len(req.Records) > 1000 {
		workers := req.Workers
		if workers <= 0 {
			workers = runtime.NumCPU()
		}
		wp := concurrency.NewWorkerPool(workers, func(rec PVRecord) float64 {
			return pvForRecord(rec, converter, req.RateJ)
		})
		return wp.ProcessBatch(req.Records)
	}

	var total float64
	for _, rec := range req.Records {
		total += pvForRecord(rec, converter, req.RateJ)
	}
	return total
}

func pvForRecord(rec PVRecord, converter *rates.RateConverter, rateJ float64) float64 {
	if rateJ > 0 {
		return converter.PresentValueStar(rec.SumAssured, rec.Term, rateJ)
	}
	return converter.PresentValue(rec.SumAssured, rec.Term)
}
