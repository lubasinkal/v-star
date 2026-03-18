package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
)

func main() {
	// Define CLI flags
	interest := flag.Float64("i", 0.05, "The effective annual interest rate (e.g., 0.05 for 5%)")
	growth := flag.Float64("j", 0.02, "The compounding growth rate for v-star logic")
	help := flag.Bool("h", false, "Show help")

	flag.Parse()

	if *help {
		fmt.Println("v-star: High-performance actuarial rate converter")
		flag.PrintDefaults()
		os.Exit(0)
	}
	// In main(), after flag.Parse():
	if len(os.Args) > 1 && os.Args[1] == "read" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: v-star read <file.csv>")
			os.Exit(1)
		}

		// Use streaming for memory efficiency
		var count int
		var totalSum float64

		err := reader.StreamCSV(os.Args[2], func(r reader.CensusRecord) {
			count++
			totalSum += r.SumAssured
			if count <= 5 {
				fmt.Printf("%d: Age=%d, Sex=%s, Type=%s, Sum=%.2f, Term=%d\n",
					count, r.Age, r.Sex, r.PolicyType, r.SumAssured, r.Term)
			}
		})

		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nTotal: %d records, Total Sum Assured: %.2f\n", count, totalSum)
		os.Exit(0)
	}
	// Initialize our library logic
	converter := rates.RateConverter{EffectiveRate: *interest}

	// Output calculations
	fmt.Println("--- V-star Actuarial Engine ---")
	fmt.Printf("Effective Rate (i): %.2f%%\n", *interest*100)
	fmt.Printf("Growth Rate (j): %.2f%%\n", *growth*100)
	fmt.Printf("Standard Discount (v): %.6f\n", converter.V())
	fmt.Printf("V-Star (v*):           %.6f\n", converter.VStar(*growth))
	fmt.Println("-------------------------------")
}
