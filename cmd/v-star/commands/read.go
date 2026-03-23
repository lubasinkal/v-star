package commands

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lubasinkal/v-star/pkg/annuities"
	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
	"github.com/lubasinkal/v-star/pkg/writer"
)

func Read(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: v-star read <file.csv> [--benchmark] [--limit=N] [--output=console|json]")
		fmt.Println("       v-star read <file.csv> --table=<mortality.csv> [--interest=0.05]")
		os.Exit(1)
	}

	filepath := args[1]

	var interest float64 = 0.05
	var benchmark bool
	var header bool = true
	var limit int
	var output string = "console"
	var tablePath string

	for i := 2; i < len(args); i++ {
		arg := args[i]
		if arg == "--benchmark" || arg == "-benchmark" {
			benchmark = true
		} else if strings.HasPrefix(arg, "--limit=") || strings.HasPrefix(arg, "-limit=") {
			if val := strings.Split(arg, "=")[1]; val != "" {
				if n, err := strconv.Atoi(val); err == nil {
					limit = n
				}
			}
		} else if strings.HasPrefix(arg, "--header=") || strings.HasPrefix(arg, "-header=") {
			if val := strings.Split(arg, "=")[1]; val != "" {
				header = (val == "true")
			}
		} else if strings.HasPrefix(arg, "--output=") || strings.HasPrefix(arg, "-output=") {
			if val := strings.Split(arg, "=")[1]; val != "" {
				output = val
			}
		} else if strings.HasPrefix(arg, "--interest=") || strings.HasPrefix(arg, "-interest=") {
			if val := strings.Split(arg, "=")[1]; val != "" {
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					interest = f
				}
			}
		} else if strings.HasPrefix(arg, "--table=") || strings.HasPrefix(arg, "-table=") {
			tablePath = strings.Split(arg, "=")[1]
		}
	}

	start := time.Now()

	if tablePath != "" {
		readWithMortality(filepath, interest, tablePath, header, limit, output, benchmark, start)
	} else {
		readWithoutMortality(filepath, interest, header, limit, output, benchmark, start)
	}

	os.Exit(0)
}

func readWithMortality(filepath string, interest float64, tablePath string, header bool, limit int, output string, benchmark bool, start time.Time) {
	mort, err := mortality.LoadCSV(tablePath)
	if err != nil {
		fmt.Printf("Error loading mortality table: %v\n", err)
		os.Exit(1)
	}

	calc := annuities.New(&rates.RateConverter{EffectiveRate: interest}, mort)

	type AnnuityResult struct {
		Age          int
		Sex          string
		PolicyType   string
		SumAssured   float64
		Term         int
		PresentValue float64
	}

	calcFn := func(age, term int, sumAssured float64) float64 {
		return calc.TermImmediate(age, term, sumAssured)
	}

	chunkSize := 50000
	if limit > 0 && limit < chunkSize {
		chunkSize = limit
	}

	processChunk := func(chunk []reader.CensusRecord) ([]AnnuityResult, error) {
		results := make([]AnnuityResult, len(chunk))
		for i, r := range chunk {
			pv := calcFn(r.Age, r.Term, r.SumAssured)
			results[i] = AnnuityResult{
				Age:          r.Age,
				Sex:          r.Sex,
				PolicyType:   r.PolicyType,
				SumAssured:   r.SumAssured,
				Term:         r.Term,
				PresentValue: pv,
			}
		}
		return results, nil
	}

	opts := reader.StreamOptions{
		Header:    header,
		Limit:     limit,
		ChunkSize: chunkSize,
		Workers:   runtime.NumCPU(),
	}

	var mu sync.Mutex
	var totalPV float64
	count := 0
	displayed := 0
	displayLimit := 5
	allResults := make([]AnnuityResult, 0, limit)
	if limit > 0 && limit < 1000 {
		displayLimit = limit
	}

	_, err = reader.StreamCensusChunked(filepath, opts, func(chunk []reader.CensusRecord) error {
		results, err := processChunk(chunk)
		if err != nil {
			return err
		}

		mu.Lock()
		for _, r := range results {
			totalPV += r.PresentValue
			if displayed < displayLimit {
				fmt.Printf("%d: Age=%d, Sex=%s, Type=%s, Sum=%.2f, Term=%d, PV=%.2f\n",
					displayed+1, r.Age, r.Sex, r.PolicyType, r.SumAssured, r.Term, r.PresentValue)
				displayed++
			}
			allResults = append(allResults, r)
		}
		count += len(results)
		mu.Unlock()
		return nil
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(start)

	if output == "json" {
		jsonRecords := make([]writer.JSONRecord, len(allResults))
		for i, r := range allResults {
			jsonRecords[i] = writer.JSONRecord{
				Age:          r.Age,
				Sex:          r.Sex,
				PolicyType:   r.PolicyType,
				SumAssured:   r.SumAssured,
				Term:         r.Term,
				PresentValue: r.PresentValue,
			}
		}
		if err := writer.StreamJSON(jsonRecords, os.Stdout); err != nil {
			fmt.Printf("Error writing JSON: %v\n", err)
			os.Exit(1)
		}
		fmt.Println()
	} else {
		fmt.Printf("Processed %d records with mortality table: %s\n", count, mort.Name())
		fmt.Printf("Total Present Value: %.2f\n", totalPV)
	}

	if benchmark {
		fmt.Printf("\n=== Benchmark Results ===\n")
		fmt.Printf("Total rows: %d\n", count)
		fmt.Printf("Duration: %v\n", duration)
		fmt.Printf("Throughput: %.0f rows/sec\n", float64(count)/duration.Seconds())
		fmt.Printf("Total Present Value: %.2f\n", totalPV)
	}
}

func readWithoutMortality(filepath string, interest float64, header bool, limit int, output string, benchmark bool, start time.Time) {
	converter := rates.NewRateConverter(interest)

	opts := reader.StreamOptions{
		Header: header,
		Limit:  limit,
	}

	totalPV, count := reader.StreamCensusWithPV(filepath, opts, converter.PresentValue)

	duration := time.Since(start)

	if benchmark {
		fmt.Printf("\n=== Benchmark Results ===\n")
		fmt.Printf("Total rows: %d\n", count)
		fmt.Printf("Duration: %v\n", duration)
		fmt.Printf("Throughput: %.0f rows/sec\n", float64(count)/duration.Seconds())
		fmt.Printf("Total Present Value: %.2f\n", totalPV)
	} else {
		fmt.Printf("Processed %d records\n", count)
		fmt.Printf("Total Present Value: %.2f\n", totalPV)
	}
}

var converter rates.RateConverter
