package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lubasinkal/v-star/cmd/v-star/commands"
	"github.com/lubasinkal/v-star/pkg/rates"
)

const version = "0.5.0"

func main() {
	interest := flag.Float64("i", 0.05, "effective annual interest rate (e.g., 0.05 for 5%)")
	growth := flag.Float64("j", 0.02, "compounding growth rate for v-star discount")
	showVersion := flag.Bool("version", false, "show version and exit")

	flag.Usage = func() {
		printGeneralUsage()
	}

	flag.Parse()

	args := flag.Args()

	if *showVersion {
		fmt.Printf("v-star %s\n", version)
		os.Exit(0)
	}

	if len(args) == 0 {
		printGeneralUsage()
		runDefault(*interest, *growth)
		os.Exit(0)
	}

	subcommand := args[0]

	switch subcommand {
	case "help", "-h", "--help":
		if len(args) > 1 {
			printSubcommandHelp(args[1])
		} else {
			printGeneralUsage()
		}
	case "read":
		commands.Read(args)
	case "montecarlo":
		commands.MonteCarlo(args, *interest)
	case "bench":
		commands.Bench()
	case "serve":
		commands.Serve(args)
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown subcommand '%s'\n\n", subcommand)
		printGeneralUsage()
		os.Exit(1)
	}
}

func runDefault(interest, growth float64) {
	converter := rates.RateConverter{EffectiveRate: interest}
	fmt.Println("--- V-star Actuarial Engine ---")
	fmt.Printf("Effective Rate (i): %.2f%%\n", interest*100)
	fmt.Printf("Growth Rate (j): %.2f%%\n", growth*100)
	fmt.Printf("Standard Discount (v): %.6f\n", converter.V())
	fmt.Printf("V-Star (v*):           %.6f\n", converter.VStar(growth))
	fmt.Println("-------------------------------")
}

func printGeneralUsage() {
	fmt.Println(`v-star: High-performance zero-dependency actuarial engine

Usage: v-star [flags] [subcommand] [args]

Subcommands:
  read        Process CSV file and calculate valuations
  montecarlo  Generate Monte Carlo interest rate paths
  bench      Run performance benchmark suite
  serve      Start HTTP API server (for Python/R/Excel)

Flags:
  -i float    effective annual interest rate (default 0.05)
  -j float    compounding growth rate for v-star (default 0.02)
  -version    show version and exit
  -h, help   show this help message

Default behavior (no subcommand):
  Calculate discount factors using specified rates

Examples:
  v-star -i 0.05 -j 0.02
  v-star read policies.csv --benchmark
  v-star read policies.csv --table=mortality.csv --output=json
  v-star montecarlo --paths=100000 --steps=10 --seed=42
  v-star bench
  v-star serve --port=8080

Run 'v-star help <subcommand>' for detailed subcommand help.
`)
}

func printSubcommandHelp(subcommand string) {
	switch subcommand {
	case "read":
		fmt.Println(`Usage: v-star read <file.csv> [flags]

Process a CSV file and calculate present values for each record.

Arguments:
  file.csv    Path to CSV file (required)

Flags:
  -i, --interest=fLOAT   interest rate (default 0.05)
  -t, --table=PATH       mortality table CSV path
  -o, --output=STRING    output format: console, json (default console)
  -b, --benchmark        show benchmark results
  -l, --limit=N         limit number of rows to process
  -H, --header          file has header row (default true)
  -h, help             show this help

CSV Format (without mortality table):
  age,term,sum_assured

CSV Format (with mortality table):
  age,sex,policy_type,term,sum_assured

Examples:
  v-star read policies.csv
  v-star read policies.csv --benchmark
  v-star read policies.csv --table=mortality.csv --output=json
  v-star read policies.csv --interest=0.04 --limit=10000
`)
	case "montecarlo":
		fmt.Println(`Usage: v-star montecarlo [flags]

Generate Monte Carlo interest rate paths using Geometric Brownian Motion.

Flags:
  -p, --paths=N        number of paths to generate (default 100000)
  -s, --steps=N        number of time steps (default 10)
  -d, --drift=FLOAT    drift parameter (default 0.02)
  -v, --volatility=FLOAT  volatility (default 0.15)
  -S, --seed=N         random seed (default random, -1 for deterministic use 42)
  -i, --interest=FLOAT  initial interest rate (default 0.05)
  -h, help            show this help

Notes:
  Seed >= 0 produces deterministic output for reproducibility.
  Without seed, output varies each run.

Examples:
  v-star montecarlo
  v-star montecarlo --paths=100000 --steps=10 --seed=42
  v-star montecarlo --drift=0.03 --volatility=0.20
`)
	case "bench":
		fmt.Println(`Usage: v-star bench

Run performance benchmark suite including:
  - CSV parsing (streaming, parallel, raw)
  - Present value calculations
  - Monte Carlo path generation
  - Risk measure computation

No flags. Output includes detailed timing and throughput.

Examples:
  v-star bench
`)
	case "serve":
		fmt.Println(`Usage: v-star serve [flags]

Start HTTP API server for non-Go access (Python, R, Excel).

Flags:
  -p, --port=PORT   port to listen on (default 8080)
  -h, help          show this help

Endpoints:
  GET  /health              - Health check
  POST /value               - Calculate present value for CSV records
  POST /montecarlo         - Run Monte Carlo simulation, get VaR/CTE
  POST /convert-rate        - Convert between nominal and effective rates
  GET  /mortality/{table}   - Get mortality table info

Examples:
  v-star serve
  v-star serve --port=9000

API Usage (curl):
  curl -X POST http://localhost:8080/value -d '{"records":[...]}'
  curl -X POST http://localhost:8080/montecarlo -d '{"paths":100000,"steps":10}'
`)
	default:
		fmt.Printf("Error: unknown subcommand '%s'\n\n", subcommand)
		printGeneralUsage()
		os.Exit(1)
	}
}
