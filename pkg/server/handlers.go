package server

import (
	"encoding/json"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/lubasinkal/v-star/pkg/concurrency"
	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
	"github.com/lubasinkal/v-star/pkg/risk"
	"github.com/lubasinkal/v-star/pkg/stochastic"
	"github.com/lubasinkal/v-star/pkg/writer"
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

type MonteCarloRequest struct {
	InitialRate  float64 `json:"initial_rate"`
	Drift        float64 `json:"drift"`
	Volatility   float64 `json:"volatility"`
	NumPaths     int     `json:"num_paths"`
	Steps        int     `json:"steps"`
	Seed         int64   `json:"seed,omitempty"`
	IncludePaths bool    `json:"include_paths,omitempty"`
}

type MonteCarloResponse struct {
	Paths        []stochastic.RatePath `json:"paths,omitempty"`
	Mean         float64               `json:"mean"`
	StdDev       float64               `json:"std_dev"`
	VaR95        float64               `json:"var_95"`
	CTE95        float64               `json:"cte_95"`
	ProcessingMs int64                 `json:"processing_ms"`
}

type ConvertRateRequest struct {
	FromRate    float64 `json:"from_rate"`
	FromType    string  `json:"from_type"`   // "effective" or "nominal"
	Compounding int     `json:"compounding"` // 1, 2, 4, 12
}

type ConvertRateResponse struct {
	EffectiveRate float64 `json:"effective_rate"`
	NominalRate   float64 `json:"nominal_rate"`
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

func (s *Server) monteCarloHandler(w http.ResponseWriter, r *http.Request) {
	var req MonteCarloRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.NumPaths == 0 {
		req.NumPaths = 10000
	}
	if req.Steps == 0 {
		req.Steps = 10
	}

	var rg *stochastic.RateGenerator
	if req.Seed > 0 {
		rg = stochastic.NewRateGeneratorWithSeed(req.InitialRate, req.Drift, req.Volatility, uint64(req.Seed))
	} else {
		rg = stochastic.NewRateGenerator(req.InitialRate, req.Drift, req.Volatility)
	}

	start := time.Now()
	paths := rg.GeneratePaths(req.NumPaths, req.Steps, 1.0)

	losses := make([]float64, len(paths))
	for i, path := range paths {
		losses[i] = path[req.Steps]
	}

	report := risk.ComputeReport(losses)

	w.Header().Set("Content-Type", "application/json")
	resp := MonteCarloResponse{
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

func (s *Server) convertRateHandler(w http.ResponseWriter, r *http.Request) {
	var req ConvertRateRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var effective, nominal float64
	switch req.FromType {
	case "nominal":
		nominal = req.FromRate
		effective = rates.NominalToEffective(req.FromRate, req.Compounding)
	default:
		effective = req.FromRate
		nominal = rates.EffectiveToNominal(req.FromRate, req.Compounding)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ConvertRateResponse{
		EffectiveRate: effective,
		NominalRate:   nominal,
	})
}

func (s *Server) uploadCSVHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	rate := 0.05
	if v := r.FormValue("rate"); v != "" {
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			rate = parsed
		}
	}

	converter := rates.NewRateConverter(rate)
	var records []writer.CSVRecord

	reader.StreamCensusFromReader(file, reader.CSVOptions{Header: true}, func(rec reader.CensusRecord) {
		records = append(records, writer.CSVRecord{
			Sex:          rec.Sex,
			PolicyType:   rec.PolicyType,
			Age:          rec.Age,
			SumAssured:   rec.SumAssured,
			Term:         rec.Term,
			PresentValue: converter.PresentValue(rec.SumAssured, rec.Term),
		})
	})

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="results.csv"`)
	writer.StreamCSV(records, w)
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
