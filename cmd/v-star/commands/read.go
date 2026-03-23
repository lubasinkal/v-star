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
	"github.com/lubasinkal/v-star/pkg/concurrency"
	"github.com/lubasinkal/v-star/pkg/mortality"
	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
	"github.com/lubasinkal/v-star/pkg/writer"
)

// Read processes a CSV file and calculates present values.
func Read(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: v-star read <file.csv> [--benchmark] [--limit=N] [--output=console|json]")
		fmt.Println("       v-star read <file.csv> --table=<mortality.csv> [--interest=0.05] [--type=annuity|reserve|pv]")
		os.Exit(1)
	}

	filepath := args[1]

	var interest float64 = 0.05
	var benchmark bool
	var header bool = true
	var limit int
	var output string = "console"
	var tablePath string
	var valuationType string = "pv"

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
		} else if strings.HasPrefix(arg, "--type=") || strings.HasPrefix(arg, "-type=") {
			valuationType = strings.Split(arg, "=")[1]
		}
	}

	var count int
	var records []reader.CensusRecord

	start := time.Now()

	err := reader.StreamCensus(filepath, reader.CSVOptions{Header: header, Limit: limit}, func(r reader.CensusRecord) {
		count++
		records = append(records, r)
	})

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	duration := time.Since(start)

	converter := rates.RateConverter{EffectiveRate: interest}

	if tablePath != "" && (valuationType == "annuity" || valuationType == "pv") {
		mort, err := mortality.LoadCSV(tablePath)
		if err != nil {
			fmt.Printf("Error loading mortality table: %v\n", err)
			os.Exit(1)
		}

		calc := annuities.New(&converter, mort)

		type AnnuityResult struct {
			Index        int
			Age          int
			Sex          string
			PolicyType   string
			SumAssured   float64
			Term         int
			PresentValue float64
		}

		results := make([]AnnuityResult, len(records))

		processRecords := func(start, end int, wg *sync.WaitGroup, results chan AnnuityResult) {
			defer wg.Done()
			for i := start; i < end; i++ {
				r := records[i]
				var pv float64

				switch r.PolicyType {
				case "term":
					pv = calc.TermImmediate(r.Age, r.Term, r.SumAssured)
				case "whole_life", "whole":
					pv = calc.WholeLifeImmediate(r.Age, r.SumAssured)
				case "endowment":
					pv = calc.TermDue(r.Age, r.Term, r.SumAssured)
				default:
					pv = calc.TermImmediate(r.Age, r.Term, r.SumAssured)
				}

				results <- AnnuityResult{
					Index:        i,
					Age:          r.Age,
					Sex:          r.Sex,
					PolicyType:   r.PolicyType,
					SumAssured:   r.SumAssured,
					Term:         r.Term,
					PresentValue: pv,
				}
			}
		}

		resultsChan := make(chan AnnuityResult, len(records))
		var wg sync.WaitGroup
		numWorkers := runtime.NumCPU()
		if numWorkers > 8 {
			numWorkers = 8
		}
		if numWorkers < 1 {
			numWorkers = 1
		}

		chunkSize := (len(records) + numWorkers - 1) / numWorkers
		for w := 0; w < numWorkers; w++ {
			startIdx := w * chunkSize
			endIdx := startIdx + chunkSize
			if endIdx > len(records) {
				endIdx = len(records)
			}
			if startIdx >= len(records) {
				break
			}
			wg.Add(1)
			go processRecords(startIdx, endIdx, &wg, resultsChan)
		}

		go func() {
			wg.Wait()
			close(resultsChan)
		}()

		var totalPV float64
		for result := range resultsChan {
			results[result.Index] = result
			totalPV += result.PresentValue
		}

		duration = time.Since(start)

		if output == "json" {
			jsonRecords := make([]writer.JSONRecord, len(results))
			for i, r := range results {
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

			limitDisplay := 5
			if len(results) < limitDisplay {
				limitDisplay = len(results)
			}
			for i := 0; i < limitDisplay; i++ {
				r := results[i]
				fmt.Printf("%d: Age=%d, Sex=%s, Type=%s, Sum=%.2f, Term=%d, PV=%.2f\n",
					i+1, r.Age, r.Sex, r.PolicyType, r.SumAssured, r.Term, r.PresentValue)
			}
		}

		if benchmark {
			fmt.Printf("\n=== Benchmark Results ===\n")
			fmt.Printf("Total rows: %d\n", count)
			fmt.Printf("Duration: %v\n", duration)
			fmt.Printf("Throughput: %.0f rows/sec\n", float64(count)/duration.Seconds())
			fmt.Printf("Total Present Value: %.2f\n", totalPV)
		}

	} else {
		wp := concurrency.NewWorkerPool(runtime.NumCPU(), converter)
		totalPV := wp.ProcessBatch(records)

		if output == "json" {
			jsonRecords := make([]writer.JSONRecord, len(records))
			for i, r := range records {
				pv := converter.PresentValue(r.SumAssured, r.Term)
				jsonRecords[i] = writer.JSONRecord{
					Age:          r.Age,
					Sex:          r.Sex,
					PolicyType:   r.PolicyType,
					SumAssured:   r.SumAssured,
					Term:         r.Term,
					PresentValue: pv,
				}
			}
			if err := writer.StreamJSON(jsonRecords, os.Stdout); err != nil {
				fmt.Printf("Error writing JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println()
		} else {
			fmt.Printf("Processed %d records\n", count)
			fmt.Printf("Total Present Value: %.2f\n", totalPV)

			limitDisplay := 5
			if len(records) < limitDisplay {
				limitDisplay = len(records)
			}
			for i := 0; i < limitDisplay; i++ {
				r := records[i]
				pv := converter.PresentValue(r.SumAssured, r.Term)
				fmt.Printf("%d: Age=%d, Sex=%s, Type=%s, Sum=%.2f, Term=%d, PV=%.2f\n",
					i+1, r.Age, r.Sex, r.PolicyType, r.SumAssured, r.Term, pv)
			}
		}

		if benchmark {
			fmt.Printf("\n=== Benchmark Results ===\n")
			fmt.Printf("Total rows: %d\n", count)
			fmt.Printf("Duration: %v\n", duration)
			fmt.Printf("Throughput: %.0f rows/sec\n", float64(count)/duration.Seconds())
			fmt.Printf("Total Present Value: %.2f\n", totalPV)
		}
	}

	os.Exit(0)
}
