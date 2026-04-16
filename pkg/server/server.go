// Package server provides an HTTP API for v-star actuarial calculations.
// This allows Python, R, Excel, and other non-Go users to access v-star functionality.
package server

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
	"github.com/lubasinkal/v-star/pkg/risk"
	"github.com/lubasinkal/v-star/pkg/stochastic"
)

type Server struct {
	addr string
}

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
}

type PVResponse struct {
	TotalPV      float64 `json:"total_pv"`
	RecordCount  int     `json:"record_count"`
	ProcessingMs int64   `json:"processing_ms"`
}

type MonteCarloRequest struct {
	InitialRate float64 `json:"initial_rate"`
	Drift       float64 `json:"drift"`
	Volatility  float64 `json:"volatility"`
	NumPaths    int     `json:"num_paths"`
	Steps       int     `json:"steps"`
	Seed        int64   `json:"seed,omitempty"`
}

type MonteCarloResponse struct {
	Paths        []stochastic.RatePath `json:"paths"`
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

func New(addr string) *Server {
	return &Server{addr: addr}
}

func (s *Server) Start() error {
	http.HandleFunc("/health", s.healthHandler)
	http.HandleFunc("/value", s.pvHandler)
	http.HandleFunc("/montecarlo", s.monteCarloHandler)
	http.HandleFunc("/convert-rate", s.convertRateHandler)
	http.HandleFunc("/mortality/", s.mortalityHandler)
	return http.ListenAndServe(s.addr, nil)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) pvHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req PVRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	converter := rates.NewRateConverter(req.InterestRate)
	var totalPV float64

	if req.Parallel && len(req.Records) > 1000 {
		totalPV = processParallelPV(req.Records, converter, req.Workers)
	} else {
		for _, rec := range req.Records {
			if req.RateJ > 0 {
				totalPV += converter.PresentValueStar(rec.SumAssured, rec.Term, req.RateJ)
			} else {
				totalPV += converter.PresentValue(rec.SumAssured, rec.Term)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PVResponse{
		TotalPV:     totalPV,
		RecordCount: len(req.Records),
	})
}

func (s *Server) monteCarloHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req MonteCarloRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	paths := rg.GeneratePaths(req.NumPaths, req.Steps, 1.0)

	losses := make([]float64, len(paths))
	for i, path := range paths {
		losses[i] = path[req.Steps]
	}

	report := risk.ComputeReport(losses)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MonteCarloResponse{
		Paths:  paths,
		Mean:   report.Mean,
		StdDev: report.StdDev,
		VaR95:  report.VaR95,
		CTE95:  report.CTE95,
	})
}

func (s *Server) convertRateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req ConvertRateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var effective, nominal float64

	if req.FromType == "nominal" {
		nominal = req.FromRate
		effective = rates.NominalToEffective(req.FromRate, req.Compounding)
	} else {
		effective = req.FromRate
		nominal = rates.EffectiveToNominal(req.FromRate, req.Compounding)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ConvertRateResponse{
		EffectiveRate: effective,
		NominalRate:   nominal,
	})
}

func (s *Server) mortalityHandler(w http.ResponseWriter, r *http.Request) {
	tableName := r.URL.Path[len("/mortality/"):]
	if tableName == "" {
		http.Error(w, "table name required", http.StatusBadRequest)
		return
	}

	table, err := mortality.LoadCSV("mortality/" + tableName + ".csv")
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":   table.Name(),
		"maxAge": table.MaxAge(),
	})
}

func processParallelPV(records []PVRecord, converter *rates.RateConverter, workers int) float64 {
	if workers <= 0 {
		workers = 4
	}

	type result struct{ pv float64 }
	results := make(chan result, len(records))

	chunkSize := len(records) / workers
	if chunkSize < 100 {
		chunkSize = len(records)
		workers = 1
	}

	for w := 0; w < workers; w++ {
		start := w * chunkSize
		end := start + chunkSize
		if w == workers-1 {
			end = len(records)
		}

		go func(recs []PVRecord) {
			var total float64
			for _, rec := range recs {
				total += converter.PresentValue(rec.SumAssured, rec.Term)
			}
			results <- result{total}
		}(records[start:end])
	}

	var totalPV float64
	for range workers {
		totalPV += (<-results).pv
	}
	return totalPV
}

func StreamCSVHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	rateStr := r.FormValue("rate")
	rate, _ := strconv.ParseFloat(rateStr, 64)
	if rate == 0 {
		rate = 0.05
	}

	converter := rates.NewRateConverter(rate)
	opts := reader.CSVOptions{Header: true}

	var totalPV float64
	var count int

	tempFile, err := os.CreateTemp("", "upload-*.csv")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer os.Remove(tempFile.Name())

	io.Copy(tempFile, file)
	tempFile.Close()

	reader.StreamCensus(tempFile.Name(), opts, func(rec reader.CensusRecord) {
		totalPV += converter.PresentValue(rec.SumAssured, rec.Term)
		count++
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PVResponse{
		TotalPV:     totalPV,
		RecordCount: count,
	})
}
