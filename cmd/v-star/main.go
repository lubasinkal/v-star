package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lubasinkal/v-star/cmd/v-star/commands"
	"github.com/lubasinkal/v-star/pkg/rates"
)

const version = "0.2.0"

func main() {
	interest := flag.Float64("i", 0.05, "The effective annual interest rate (e.g., 0.05 for 5%)")
	growth := flag.Float64("j", 0.02, "The compounding growth rate for v-star logic")
	help := flag.Bool("h", false, "Show help")
	showVersion := flag.Bool("version", false, "Show version")

	flag.Parse()

	if *showVersion {
		fmt.Printf("v-star %s\n", version)
		os.Exit(0)
	}

	if *help {
		printHelp()
		os.Exit(0)
	}

	args := flag.Args()

	if len(args) > 0 {
		switch args[0] {
		case "read":
			commands.Read(args)
		case "montecarlo":
			commands.MonteCarlo(args, *interest)
		case "bench":
			commands.Bench()
		default:
			fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", args[0])
			printHelp()
			os.Exit(1)
		}
		return
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

func printHelp() {
	fmt.Println("v-star: High-performance actuarial rate converter")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  read <file.csv>    Read CSV and calculate valuations")
	fmt.Println("  montecarlo         Generate Monte Carlo interest rate paths")
	fmt.Println("  bench              Run performance benchmark suite")
	fmt.Println("  (default)          Calculate discount factors")
	fmt.Println()
	fmt.Println("Flags:")
	flag.PrintDefaults()
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  v-star -i 0.05 -j 0.02")
	fmt.Println("  v-star read policies.csv --benchmark")
	fmt.Println("  v-star read policies.csv --output=json")
	fmt.Println("  v-star montecarlo --paths=100000 --steps=10 --seed=42")
	fmt.Println("  v-star bench")
	fmt.Println("  v-star --version")
}
