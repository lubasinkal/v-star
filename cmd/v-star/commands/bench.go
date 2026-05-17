package commands

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/lubasinkal/v-star/pkg/concurrency"
	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
	"github.com/lubasinkal/v-star/pkg/stochastic"
)

// Bench runs the full benchmark suite.
func Bench() {
	fmt.Println("Running v-star benchmark suite...")
	fmt.Println("=====================================")

	fmt.Println("1. CSV Parsing Benchmark")
	fmt.Println("------------------------")
	benchmarkCSV()

	fmt.Println("\n2. Valuation Calculation Benchmark")
	fmt.Println("----------------------------------")
	benchmarkValuation()

	fmt.Println("\n3. Monte Carlo Simulation Benchmark")
	fmt.Println("-----------------------------------")
	benchmarkMonteCarlo()

	fmt.Println("\n=====================================")
	fmt.Println("Benchmark suite completed!")
	os.Exit(0)
}

func benchmarkCSV() {
	filepath := "10M.csv"

	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		fmt.Println("  Skipping: 10M.csv not found")
		return
	}

	// Warmup run
	_ = reader.StreamCensus(filepath, reader.CSVOptions{Header: true, Limit: 10000000}, func(r reader.CensusRecord) {})

	// GC to get clean baseline
	runtime.GC()
	runtime.GC()

	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	start := time.Now()
	var count int

	err := reader.StreamCensus(filepath, reader.CSVOptions{Header: true, Limit: 10000000}, func(r reader.CensusRecord) {
		count++
	})

	duration := time.Since(start)

	// Read memory stats immediately without GC to capture actual allocations
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)

	var memoryUsed uint64
	if memStatsAfter.Alloc > memStatsBefore.Alloc {
		memoryUsed = (memStatsAfter.Alloc - memStatsBefore.Alloc) / 1024 / 1024
	}

	if err != nil {
		fmt.Printf("  Error: %v\n", err)
		return
	}

	fmt.Printf("  Rows Processed: %d\n", count)
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Throughput: %.0f rows/sec\n", float64(count)/duration.Seconds())
	fmt.Printf("  Memory Used: %d MB\n", memoryUsed)
}

func benchmarkValuation() {
	// Setup test data - 10M records
	numRecords := 10000000
	records := make([]reader.CensusRecord, numRecords)
	for i := range numRecords {
		records[i] = reader.CensusRecord{
			Age:        30 + i%50,
			Sex:        "male",
			PolicyType: "term",
			SumAssured: 100000.0 + float64(i%100)*1000.0,
			Term:       10 + i%20,
		}
	}

	converter := rates.NewRateConverter(0.05)

	// Warmup run
	wp := concurrency.NewWorkerPool(runtime.NumCPU(), func(r reader.CensusRecord) float64 {
		return converter.PresentValue(r.SumAssured, r.Term)
	})
	wp.ProcessBatch(records)

	// GC to get clean baseline
	runtime.GC()
	runtime.GC()

	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	start := time.Now()
	totalPV := wp.ProcessBatch(records)
	duration := time.Since(start)

	// Read memory stats immediately without GC to capture actual allocations
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)

	var memoryUsed uint64
	if memStatsAfter.Alloc > memStatsBefore.Alloc {
		memoryUsed = (memStatsAfter.Alloc - memStatsBefore.Alloc) / 1024 / 1024
	}

	fmt.Printf("  Policies Valuated: %d\n", len(records))
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Throughput: %.0f policies/sec\n", float64(len(records))/duration.Seconds())
	fmt.Printf("  Total PV: %.2f\n", totalPV)
	fmt.Printf("  Memory Used: %d MB\n", memoryUsed)
}

func benchmarkMonteCarlo() {
	// Warmup run
	rg := stochastic.NewRateGenerator(0.05, 0.02, 0.15)
	_ = rg.GeneratePaths(1000000, 10, 1.0)

	// GC to get clean baseline
	runtime.GC()
	runtime.GC()

	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	start := time.Now()

	paths := rg.GeneratePaths(1000000, 10, 1.0)

	duration := time.Since(start)

	// Read memory stats immediately without GC to capture actual allocations
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)

	var memoryUsed uint64
	if memStatsAfter.Alloc > memStatsBefore.Alloc {
		memoryUsed = (memStatsAfter.Alloc - memStatsBefore.Alloc) / 1024 / 1024
	}

	var totalRate float64
	for _, path := range paths {
		totalRate += path[10]
	}
	avgRate := totalRate / float64(len(paths))

	fmt.Printf("  Paths Generated: %d\n", len(paths))
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Throughput: %.0f paths/sec\n", float64(len(paths))/duration.Seconds())
	fmt.Printf("  Average Final Rate: %.4f%%\n", avgRate*100)
	fmt.Printf("  Memory Used: %d MB\n", memoryUsed)
}
