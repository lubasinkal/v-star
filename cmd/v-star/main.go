package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lubasinkal/v-star/pkg/concurrency"
	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
	"github.com/lubasinkal/v-star/pkg/stochastic"
	"github.com/lubasinkal/v-star/pkg/writer"
)

func main() {
	// Define CLI flags
	interest := flag.Float64("i", 0.05, "The effective annual interest rate (e.g., 0.05 for 5%)")
	growth := flag.Float64("j", 0.02, "The compounding growth rate for v-star logic")
	help := flag.Bool("h", false, "Show help")
	benchmark := flag.Bool("benchmark", false, "Show performance metrics")
	header := flag.Bool("header", true, "Treat first row as header")
	limit := flag.Int("limit", 0, "Limit number of rows to process")
	output := flag.String("output", "console", "Output format: console|json")

	flag.Parse()

	if *help {
		fmt.Println("v-star: High-performance actuarial rate converter")
		fmt.Println("\nSubcommands:")
		fmt.Println("  read <file.csv>    Read CSV and calculate valuations")
		fmt.Println("  montecarlo         Generate Monte Carlo interest rate paths")
		fmt.Println("  (default)          Calculate discount factors")
		fmt.Println("\nFlags:")
		flag.PrintDefaults()
		fmt.Println("\nExamples:")
		fmt.Println("  v-star -i 0.05 -j 0.02")
		fmt.Println("  v-star read policies.csv --benchmark")
		fmt.Println("  v-star read policies.csv --output=json")
		fmt.Println("  v-star montecarlo --paths=100000 --steps=10")
		os.Exit(0)
	}

	// Get subcommand from positional args
	args := flag.Args()

	if len(args) > 0 && args[0] == "read" {
		if len(args) < 2 {
			fmt.Println("Usage: v-star read <file.csv> [--benchmark] [--limit=N] [--output=console|json]")
			os.Exit(1)
		}

		filepath := args[1]
		var count int
		var totalPV float64
		var records []reader.CensusRecord

		// Parse additional flags that may appear after the file path
		// This handles cases like: v-star read file.csv --limit=10 --benchmark
		for i := 2; i < len(args); i++ {
			arg := args[i]
			if arg == "--benchmark" || arg == "-benchmark" {
				*benchmark = true
			} else if strings.HasPrefix(arg, "--limit=") || strings.HasPrefix(arg, "-limit=") {
				val := strings.Split(arg, "=")[1]
				if val != "" {
					limitVal, err := strconv.Atoi(val)
					if err == nil {
						*limit = limitVal
					}
				}
			} else if strings.HasPrefix(arg, "--header=") || strings.HasPrefix(arg, "-header=") {
				val := strings.Split(arg, "=")[1]
				if val != "" {
					*header = (val == "true")
				}
			} else if strings.HasPrefix(arg, "--output=") || strings.HasPrefix(arg, "-output=") {
				val := strings.Split(arg, "=")[1]
				if val != "" {
					*output = val
				}
			}
		}

		start := time.Now()

		// Collect records first for processing
		err := reader.StreamCSV(filepath, reader.StreamOptions{Header: *header, Limit: *limit}, func(r reader.CensusRecord) {
			count++
			records = append(records, r)
		})

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		// Process records with worker pool
		converter := rates.RateConverter{EffectiveRate: *interest}
		totalPV = concurrency.ProcessBatch(records, converter, 4) // Use 4 workers

		duration := time.Since(start)

		// Output results
		if *output == "json" {
			// Prepare JSON records
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

			// Write JSON output
			if err := writer.StreamJSON(jsonRecords, os.Stdout); err != nil {
				fmt.Printf("Error writing JSON: %v\n", err)
				os.Exit(1)
			}
			fmt.Println() // Add newline after JSON
		} else {
			// Console output
			fmt.Printf("Processed %d records\n", count)
			fmt.Printf("Total Present Value: %.2f\n", totalPV)

			// Show first 5 records
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

		if *benchmark {
			fmt.Printf("\n=== Benchmark Results ===\n")
			fmt.Printf("Total rows: %d\n", count)
			fmt.Printf("Duration: %v\n", duration)
			fmt.Printf("Throughput: %.0f rows/sec\n", float64(count)/duration.Seconds())
			fmt.Printf("Total Present Value: %.2f\n", totalPV)
		}

		os.Exit(0)
	}

	// Monte Carlo subcommand
	if len(args) > 0 && args[0] == "montecarlo" {
		// Default parameters
		numPaths := 100000
		steps := 10
		initialRate := *interest
		drift := 0.02
		volatility := 0.15

		// Parse additional flags
		for i := 1; i < len(args); i++ {
			arg := args[i]
			if strings.HasPrefix(arg, "--paths=") {
				val := strings.Split(arg, "=")[1]
				if val != "" {
					if n, err := strconv.Atoi(val); err == nil {
						numPaths = n
					}
				}
			} else if strings.HasPrefix(arg, "--steps=") {
				val := strings.Split(arg, "=")[1]
				if val != "" {
					if n, err := strconv.Atoi(val); err == nil {
						steps = n
					}
				}
			} else if strings.HasPrefix(arg, "--drift=") {
				val := strings.Split(arg, "=")[1]
				if val != "" {
					if f, err := strconv.ParseFloat(val, 64); err == nil {
						drift = f
					}
				}
			} else if strings.HasPrefix(arg, "--volatility=") {
				val := strings.Split(arg, "=")[1]
				if val != "" {
					if f, err := strconv.ParseFloat(val, 64); err == nil {
						volatility = f
					}
				}
			}
		}

		fmt.Printf("Generating %d Monte Carlo interest rate paths...\n", numPaths)
		fmt.Printf("Parameters: Initial Rate=%.2f%%, Drift=%.2f%%, Volatility=%.2f%%, Steps=%d\n",
			initialRate*100, drift*100, volatility*100, steps)

		start := time.Now()

		// Generate rate paths
		rg := stochastic.NewRateGenerator(initialRate, drift, volatility)
		paths := rg.GeneratePaths(numPaths, steps, 1.0)

		duration := time.Since(start)

		// Calculate statistics
		var totalRate float64
		var minRate, maxRate float64 = 1e9, -1e9

		for _, path := range paths {
			rate := path[steps] // Final rate
			totalRate += rate
			if rate < minRate {
				minRate = rate
			}
			if rate > maxRate {
				maxRate = rate
			}
		}

		avgRate := totalRate / float64(numPaths)

		fmt.Printf("\n=== Monte Carlo Results ===\n")
		fmt.Printf("Paths Generated: %d\n", numPaths)
		fmt.Printf("Duration: %v\n", duration)
		fmt.Printf("Throughput: %.0f paths/sec\n", float64(numPaths)/duration.Seconds())
		fmt.Printf("\nFinal Rate Statistics:\n")
		fmt.Printf("  Average: %.4f%%\n", avgRate*100)
		fmt.Printf("  Minimum: %.4f%%\n", minRate*100)
		fmt.Printf("  Maximum: %.4f%%\n", maxRate*100)

		os.Exit(0)
	}

	// Default: rate calculation
	converter := rates.RateConverter{EffectiveRate: *interest}

	fmt.Println("--- V-star Actuarial Engine ---")
	fmt.Printf("Effective Rate (i): %.2f%%\n", *interest*100)
	fmt.Printf("Growth Rate (j): %.2f%%\n", *growth*100)
	fmt.Printf("Standard Discount (v): %.6f\n", converter.V())
	fmt.Printf("V-Star (v*):           %.6f\n", converter.VStar(*growth))
	fmt.Println("-------------------------------")
}
