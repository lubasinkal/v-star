// Package server provides an HTTP API for v-star actuarial calculations.
// This allows Python, R, Excel, and other non-Go users to access v-star functionality.
package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/lubasinkal/v-star/pkg/concurrency"
	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
	"github.com/lubasinkal/v-star/pkg/risk"
	"github.com/lubasinkal/v-star/pkg/server/middleware"
	"github.com/lubasinkal/v-star/pkg/stochastic"
	"github.com/lubasinkal/v-star/pkg/writer"
)

type Server struct {
	addr              string
	MortalityTableDir string
	server            *http.Server
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
	return &Server{addr: addr, MortalityTableDir: "mortality"}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.healthHandler)
	mux.HandleFunc("/value", s.pvHandler)
	mux.HandleFunc("/montecarlo", s.monteCarloHandler)
	mux.HandleFunc("/convert-rate", s.convertRateHandler)
	mux.HandleFunc("/mortality/", s.mortalityHandler)
	mux.HandleFunc("/export/csv", s.exportCSVHandler)
	mux.HandleFunc("/export/report", s.exportReportHandler)
	mux.HandleFunc("/upload/csv", s.StreamCSVHandler)

	handler := middleware.CreateStack(
		middleware.Logging,
		middleware.CORS,
	)(mux)

	s.server = &http.Server{
		Addr:         s.addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s.server.ListenAndServe()
}

// StartWithGracefulShutdown starts the server and blocks until SIGINT/SIGTERM.
func (s *Server) StartWithGracefulShutdown() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("v-star server listening on %s", s.addr)
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			log.Printf("server error: %v", err)
		}
	}()

	<-ctx.Done()
	stop()
	log.Println("shutting down gracefully...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return s.server.Shutdown(shutdownCtx)
}

func (s *Server) healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// pvHandler computes present value for a batch of records.
func (s *Server) pvHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req PVRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	start := time.Now()
	totalPV := computePresentValues(req)
	elapsed := time.Since(start)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PVResponse{
		TotalPV:      totalPV,
		RecordCount:  len(req.Records),
		ProcessingMs: elapsed.Milliseconds(),
	})
}

func (s *Server) monteCarloHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

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
	elapsed := time.Since(start)

	w.Header().Set("Content-Type", "application/json")
	resp := MonteCarloResponse{
		Mean:         report.Mean,
		StdDev:       report.StdDev,
		VaR95:        report.VaR95,
		CTE95:        report.CTE95,
		ProcessingMs: elapsed.Milliseconds(),
	}
	if req.IncludePaths {
		resp.Paths = paths
	}
	json.NewEncoder(w).Encode(resp)
}

func (s *Server) convertRateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

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
	default: // "effective"
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

	table, err := mortality.LoadCSV(filepath.Join(s.MortalityTableDir, tableName+".csv"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"name":   table.Name(),
		"maxAge": table.MaxAge(),
	})
}

// computePresentValues returns the sum of present values for all records.
// Uses parallel processing for batches over 1000 records.
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

// pvForRecord computes the present value for a single record, optionally using v*.
func pvForRecord(rec PVRecord, converter *rates.RateConverter, rateJ float64) float64 {
	if rateJ > 0 {
		return converter.PresentValueStar(rec.SumAssured, rec.Term, rateJ)
	}
	return converter.PresentValue(rec.SumAssured, rec.Term)
}

func (s *Server) StreamCSVHandler(w http.ResponseWriter, r *http.Request) {
	file, _, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	rate, _ := parseRate(r.FormValue("rate"))
	if rate == 0 {
		rate = 0.05
	}

	converter := rates.NewRateConverter(rate)
	var csvRecords []writer.CSVRecord

	reader.StreamCensusFromReader(file, reader.CSVOptions{Header: true}, func(rec reader.CensusRecord) {
		pv := converter.PresentValue(rec.SumAssured, rec.Term)
		csvRecords = append(csvRecords, writer.CSVRecord{
			Sex:          rec.Sex,
			PolicyType:   rec.PolicyType,
			Age:          rec.Age,
			SumAssured:   rec.SumAssured,
			Term:         rec.Term,
			PresentValue: pv,
		})
	})

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"export.csv\"")
	if err := writer.StreamCSV(csvRecords, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// parseRate parses a rate string, returning 0 on empty input.
func parseRate(s string) (float64, error) {
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, nil
	}
	return v, nil
}

func (s *Server) exportCSVHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req PVRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	converter := rates.NewRateConverter(req.InterestRate)
	records := make([]writer.CSVRecord, 0, len(req.Records))
	for _, rec := range req.Records {
		records = append(records, writer.CSVRecord{
			Sex:          rec.Sex,
			PolicyType:   rec.PolicyType,
			Age:          rec.Age,
			SumAssured:   rec.SumAssured,
			Term:         rec.Term,
			PresentValue: pvForRecord(rec, converter, req.RateJ),
		})
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"export.csv\"")
	if err := writer.StreamCSV(records, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) exportReportHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}

	var req PVRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 10<<20)).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	totalPV := computePresentValues(req)
	assumptions := writer.FormatAssumptions(req.InterestRate, "", nil)
	data := writer.ReportData{
		Title:             "Actuarial Valuation Report",
		InterestRate:      req.InterestRate,
		RecordCount:       len(req.Records),
		TotalPresentValue: totalPV,
		Assumptions:       assumptions,
	}

	w.Header().Set("Content-Type", "text/plain")
	if err := writer.StreamTextReport(data, w); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
