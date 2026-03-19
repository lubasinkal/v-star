package commands

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/lubasinkal/v-star/pkg/concurrency"
	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
	"github.com/lubasinkal/v-star/pkg/writer"
)

// Read processes a CSV file and calculates present values.
func Read(args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: v-star read <file.csv> [--benchmark] [--limit=N] [--output=console|json]")
		os.Exit(1)
	}

	filepath := args[1]

	var interest float64 = 0.05
	var benchmark bool
	var header bool = true
	var limit int
	var output string = "console"

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
		}
	}

	var count int
	var totalPV float64
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

	converter := rates.RateConverter{EffectiveRate: interest}
	wp := concurrency.NewWorkerPool(runtime.NumCPU(), converter)
	totalPV = wp.ProcessBatch(records)

	duration := time.Since(start)

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

	os.Exit(0)
}
