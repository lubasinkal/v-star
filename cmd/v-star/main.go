package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
)

func main() {
	// Define CLI flags
	interest := flag.Float64("i", 0.05, "The effective annual interest rate (e.g., 0.05 for 5%)")
	growth := flag.Float64("j", 0.02, "The compounding growth rate for v-star logic")
	help := flag.Bool("h", false, "Show help")
	benchmark := flag.Bool("benchmark", false, "Show performance metrics")
	header := flag.Bool("header", true, "Treat first row as header")
	limit := flag.Int("limit", 0, "Limit number of rows to process")

	flag.Parse()

	if *help {
		fmt.Println("v-star: High-performance actuarial rate converter")
		flag.PrintDefaults()
		fmt.Println("\nUsage:")
		fmt.Println("  v-star read <file.csv> [--benchmark] [--limit=N]")
		os.Exit(0)
	}

	// Get subcommand from positional args
	args := flag.Args()

	if len(args) > 0 && args[0] == "read" {
		if len(args) < 2 {
			fmt.Println("Usage: v-star read <file.csv> [--benchmark] [--limit=N]")
			os.Exit(1)
		}

		filepath := args[1]
		var count int
		var totalSum float64

		start := time.Now()

		err := reader.StreamCSV(filepath, reader.StreamOptions{Header: *header, Limit: *limit}, func(r reader.CensusRecord) {
			count++
			totalSum += r.SumAssured
			if count <= 5 {
				fmt.Printf("%d: Age=%d, Sex=%s, Type=%s, Sum=%.2f, Term=%d\n",
					count, r.Age, r.Sex, r.PolicyType, r.SumAssured, r.Term)
			}
		})

		duration := time.Since(start)

		if *benchmark {
			fmt.Printf("\n=== Benchmark Results ===\n")
			fmt.Printf("Total rows: %d\n", count)
			fmt.Printf("Duration: %v\n", duration)
			fmt.Printf("Throughput: %.0f rows/sec\n", float64(count)/duration.Seconds())
			fmt.Printf("Total Sum Assured: %.2f\n", totalSum)
		}

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

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
