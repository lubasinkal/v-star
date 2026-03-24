# v-star (v*)

**A high-performance, zero-dependency actuarial engine for Concurrent Financial Simulations built in Go.**

[![Go Version](https://img.shields.io/badge/Go-1.26-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

## The Origin

The name **v-star** comes from a class joke between my University lecturer and comrades (brothers and sister deployed to study Actuarial Science): 
If an annuity (or more precisely, the payments/premiums associated with the annuity) compound (or earn interest) at rate j while being discounted (valued) at rate i, then the adjusted (effective) discount factor is:

```
v* = (1+j) * v
```

## Why v-star?

Modern financial software is often bloated and slow. **v-star** is designed for:

- **Zero Dependencies:** Uses only the Go Standard Library
- **Extreme Speed:** 11M+ rows/sec CSV parsing, parallel processing
- **Auditability:** Pure, readable math implementations
- **Flexibility:** Generic CSV parsing + specialized actuarial models
- **Open Source:** MIT Licensed, community-driven

## Features

| Feature | Description | Performance |
|---------|-------------|-------------|
| **Generic CSV Parser** | Stream any CSV format | ~2.5M rows/sec |
| **Parallel CSV Parser** | Multi-core CSV processing | ~11M rows/sec |
| **Actuarial CSV Parser** | Direct CensusRecord parsing | ~11M rows/sec |
| **Present Value** | Standard & v* discount factors | ~10M calcs/sec |
| **Monte Carlo** | GBM interest rate paths | ~100k paths/sec |

## Quick Start

### Installation

```bash
go get github.com/lubasinkal/v-star
```

### Library Usage

```go
package main

import (
    "fmt"
    "github.com/lubasinkal/v-star/pkg/reader"
    "github.com/lubasinkal/v-star/pkg/rates"
)

func main() {
    // Create rate converter
    converter := rates.NewRateConverter(0.05)
    
    // Calculate present value
    pv := converter.PresentValue(100000, 20)
    fmt.Printf("PV: %.2f\n", pv)
    
    // Stream CSV with parallel processing
    opts := reader.CSVOptions{Header: true}
    totalPV, count := reader.StreamCSVWithPV("policies.csv", opts, converter.PresentValue)
    fmt.Printf("Total PV: %.2f from %d records\n", totalPV, count)
}
```

### CLI Usage

```bash
# Build
go build -o v-star ./cmd/v-star

# Calculate discount factors
./v-star -i 0.05 -j 0.02

# Read CSV and calculate valuations
./v-star read policies.csv --benchmark

# Export as JSON
./v-star read policies.csv --output=json

# Monte Carlo simulation
./v-star montecarlo --paths=100000 --steps=10 --drift=0.02 --volatility=0.15
```

## CSV Streaming API

### Generic CSV Streaming

```go
// Stream any CSV format - returns []string per row
opts := reader.CSVOptions{Header: true, Limit: 1000000}
err := reader.StreamCSV("data.csv", opts, func(fields []string) {
    fmt.Println(fields)
})
```

### Parallel Generic Streaming

```go
// Zero-allocation parallel streaming with raw bytes
opts := reader.CSVOptions{Header: true}
err := reader.StreamCSVRaw("data.csv", opts, func(fields [][]byte) {
    // process raw bytes without string allocation
})
```

### Actuarial CSV Streaming

```go
// Direct CensusRecord parsing - fastest path
opts := reader.CSVOptions{Header: true}
err := reader.StreamCensus("policies.csv", opts, func(record reader.CensusRecord) {
    fmt.Printf("Age: %d, Sum: %.2f\n", record.Age, record.SumAssured)
})
```

### Parallel PV Calculation

```go
// Read + Calculate PV in one pass
converter := rates.NewRateConverter(0.05)
opts := reader.CSVOptions{Header: true}
totalPV, count := reader.StreamCSVWithPV("policies.csv", opts, converter.PresentValue)
```

## Performance Benchmarks

Tested on: Intel Core i5-8250U @ 1.60GHz (8 cores), 10M row CSV (~288MB)

| Operation | Throughput | Duration |
|-----------|-----------|----------|
| Generic CSV Streaming | ~2.5M rows/sec | 4.0s |
| Parallel Generic CSV | ~11M rows/sec | 0.9s |
| CensusRecord Streaming | ~11M rows/sec | 0.9s |
| PV Calculation (full) | ~10M rows/sec | 1.0s |
| Monte Carlo (100k paths) | ~100k paths/sec | 1.0s |

### Additional CLI Commands

```bash
# Read CSV with mortality table for annuity valuations
./v-star read policies.csv --table=mortality.csv --interest=0.05

# Run performance benchmarks
./v-star bench
```

#### CLI Flags Reference

**Global flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `-i` | float64 | `0.05` | Effective annual interest rate |
| `-j` | float64 | `0.02` | Compounding growth rate for v* |

**`read` subcommand:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--benchmark` | bool | `false` | Show timing and throughput |
| `--limit` | int | `0` | Max rows to process (0 = all) |
| `--header` | bool | `true` | CSV has header row |
| `--output` | string | `console` | Output format: `console` or `json` |
| `--interest` | float64 | `-1` | Override global interest rate |
| `--table` | string | `""` | Mortality table CSV for annuity calculations |

**`montecarlo` subcommand:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--paths` | int | `100000` | Number of simulation paths |
| `--steps` | int | `10` | Time steps per path |
| `--drift` | float64 | `0.02` | Drift parameter (mu) |
| `--volatility` | float64 | `0.15` | Volatility (sigma) |
| `--seed` | int | `-1` | Deterministic seed (-1 = random) |

## Architecture

```
cmd/
├── v-star/
│   ├── main.go              # Entry point, flag parsing
│   └── commands/
│       ├── read.go           # CSV read command
│       ├── montecarlo.go     # Monte Carlo command
│       └── bench.go          # Benchmark command
└── generate/
    └── main.go               # Test data generator

pkg/
├── reader/                   # CSV parsing and streaming
│   ├── models.go             # CensusRecord struct
│   ├── csv.go                # Generic CSV parser
│   ├── census.go             # CensusRecord parser, ColumnMap
│   └── streaming.go          # Parallel chunked streaming
├── rates/                    # Interest rate calculations
│   └── rates.go              # RateConverter, DiscountFactor interface
├── mortality/                # Mortality tables
│   └── table.go              # Table, MortalityTable interface
├── annuities/                # Annuity calculations
│   └── annuity.go            # AnnuityCalculator, deferred annuities
├── reserves/                 # Policy reserves
│   └── reserve.go            # Net/gross/prospective/retrospective reserves
├── stochastic/               # Monte Carlo simulations
│   └── rates.go              # RateGenerator, GBM paths
├── writer/                   # JSON output
│   └── json.go               # JSONWriter, streaming JSON
└── concurrency/              # Parallel processing
    └── worker_pool.go        # WorkerPool, ProcessBatch
```

## API Reference

### rates

Interest rate calculations, discount factors, and the v-star adjustment.

#### Types

```go
// DiscountFactor is the interface for discount factor calculations.
type DiscountFactor interface {
    Discount(term int) float64
}

// RateConverter performs interest rate conversions and present value calculations.
// Pre-computes a discount table for terms 0-100 for fast lookups.
type RateConverter struct {
    EffectiveRate float64
}
```

#### Functions

```go
// NewRateConverter creates a RateConverter for the given effective annual rate.
// Pre-computes discount factors for terms 0 through 100.
converter := rates.NewRateConverter(0.05)

// NominalToEffective converts a nominal rate compounded m times per period
// to an effective annual rate.
// Formula: i = (1 + im/m)^m - 1
effective := rates.NominalToEffective(0.048, 12) // 4.8% nominal, monthly
```

#### Methods

| Signature | Description |
|-----------|-------------|
| `V() float64` | One-period discount factor: v = 1/(1+i) |
| `VStar(j float64) float64` | v-star factor: v* = (1+j) * v |
| `Discount(term int) float64` | v^term; uses pre-computed table for terms 0-100 |
| `PresentValue(sumAssured float64, term int) float64` | sumAssured * v^term |
| `PresentValueStar(sumAssured float64, term int, j float64) float64` | sumAssured * (v*)^term |

```go
converter := rates.NewRateConverter(0.05)

v := converter.V()                           // 0.952381
vStar := converter.VStar(0.02)               // 0.971429
pv := converter.PresentValue(100000, 20)     // 37688.95
pvStar := converter.PresentValueStar(100000, 20, 0.02) // 42210.83

// DiscountFactor interface usage
var df rates.DiscountFactor = converter
d := df.Discount(10) // v^10
```

---

### mortality

Mortality tables with survival probability calculations.

#### Types

```go
// MortalityTable defines the interface for mortality data access.
type MortalityTable interface {
    Qx(age int) float64       // Probability of death within one year
    Px(age int, term int) float64 // Cumulative survival probability
    MaxAge() int              // Maximum age in table
}

// Table implements MortalityTable. Also has Name() and Lx() methods.
type Table struct { /* unexported fields */ }
```

#### Functions

```go
// NewTable constructs a Table from a slice of qx values.
// Computes lx internally using radix 100000.
qx := []float64{0.001, 0.0012, 0.0015, ...}
table := mortality.NewTable("CSO-80", qx)

// LoadCSV loads a mortality table from CSV.
// Supports both qx and px columns.
table, err := mortality.LoadCSV("mortality.csv")

// StreamCSV streams mortality data row by row.
err := mortality.StreamCSV("mortality.csv", func(age int, qx float64) {
    fmt.Printf("Age %d: qx=%.6f\n", age, qx)
})
```

#### Methods

| Signature | Description |
|-----------|-------------|
| `Qx(age int) float64` | Probability of death between age x and x+1 |
| `Px(age int, term int) float64` | Cumulative survival: Px(age, term) = product of (1 - Qx) |
| `Ex(age int) float64` | Curtate expectation of life: sum of Px(age, t) for t >= 1 |
| `MaxAge() int` | Maximum age defined in the table |
| `Name() string` | Table name (on concrete *Table type) |
| `Lx(age int) float64` | Number of lives surviving to age from radix 100000 (on concrete *Table type) |

```go
table, _ := mortality.LoadCSV("mortality.csv")

qx := table.Qx(30)              // 0.000812
px := table.Px(30, 20)          // 20-year survival from age 30
ex := table.Ex(65)              // Life expectancy at 65
fmt.Printf("%s: max age = %d\n", table.Name(), table.MaxAge())
```

---

### annuities

Whole life, term, and deferred annuity calculations.

#### Types

```go
// AnnuityCalculator computes annuity values using a discount factor and mortality table.
type AnnuityCalculator struct { /* unexported fields */ }
```

#### Functions

```go
// NewAnnuityCalculator creates an AnnuityCalculator from a DiscountFactor and MortalityTable.
converter := rates.NewRateConverter(0.05)
table, _ := mortality.LoadCSV("mortality.csv")
ann := annuities.NewAnnuityCalculator(converter, table)

// ApproxWholeLifeImmediate computes an approximate whole life immediate
// annuity using a direct interest rate (no RateConverter needed).
value := annuities.ApproxWholeLifeImmediate(65, 30, 1000, 0.05, table)
```

#### Methods

| Signature | Description |
|-----------|-------------|
| `WholeLifeImmediate(age int, amount float64) float64` | Whole life annuity-immediate; payments at end of each period |
| `WholeLifeDue(age int, amount float64) float64` | Whole life annuity-due; payments at start of each period |
| `TermImmediate(age int, term int, amount float64) float64` | Term annuity-immediate over specified years |
| `TermDue(age int, term int, amount float64) float64` | Term annuity-due over specified years |
| `DeferredWholeLife(age int, deferment int, amount float64) float64` | Deferred whole life annuity; payments start after deferment years |
| `DeferredTerm(age int, deferment int, term int, amount float64) float64` | Deferred term annuity |

```go
converter := rates.NewRateConverter(0.05)
table, _ := mortality.LoadCSV("mortality.csv")
ann := annuities.New(converter, table)

// Whole life annuity of 1000/year starting at age 65
wl := ann.WholeLifeImmediate(65, 1000)

// 20-year term annuity-due of 1000/year at age 40
term := ann.TermDue(40, 20, 1000)

// Deferred whole life: defer 10 years, then pay 1000/year
deferred := ann.DeferredWholeLife(50, 10, 1000)
```

---

### reserves

Policy reserve calculations using prospective and retrospective methods.

#### Types

```go
// PolicySpec defines the parameters for a policy.
type PolicySpec struct {
    Age        int
    Term       int
    SumAssured float64
    Premium    float64
}
```

#### Functions

| Signature | Description |
|-----------|-------------|
| `NetPremiumReserve(policy PolicySpec, discount DiscountFactor, mort MortalityTable) float64` | Net premium reserve using prospective method |
| `GrossPremiumReserve(policy PolicySpec, expenses float64, discount DiscountFactor, mort MortalityTable) float64` | Gross premium reserve (NPR + expense reserve) |
| `ProspectiveReserve(policy PolicySpec, discount DiscountFactor, mort MortalityTable) float64` | Future benefits minus future premiums |
| `RetrospectiveReserve(policy PolicySpec, discount DiscountFactor, mort MortalityTable) float64` | Accumulated premiums minus past claims |

```go
converter := rates.NewRateConverter(0.05)
table, _ := mortality.LoadCSV("mortality.csv")

policy := reserves.PolicySpec{
    Age:        30,
    Term:       20,
    SumAssured: 100000,
    Premium:    500,
}

npr := reserves.NetPremiumReserve(policy, converter, table)
gpr := reserves.GrossPremiumReserve(policy, 50, converter, table)

prosp := reserves.ProspectiveReserve(policy, converter, table)
retro := reserves.RetrospectiveReserve(policy, converter, table)
```

---

### stochastic

Monte Carlo interest rate simulations using geometric Brownian motion.

#### Types

```go
// RatePath is a sequence of simulated interest rates over time.
type RatePath []float64

// RateGenerator produces stochastic interest rate paths via GBM.
// Uses Box-Muller transform for normal variates.
type RateGenerator struct { /* unexported fields */ }
```

#### Functions

```go
// NewRateGenerator creates a generator with a random (PCG-based) seed.
rg := stochastic.NewRateGenerator(0.05, 0.02, 0.15)

// NewRateGeneratorWithSeed creates a generator with a deterministic seed
// for reproducible simulations.
rg := stochastic.NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42)
```

#### Methods

| Signature | Description |
|-----------|-------------|
| `GeneratePath(steps int, dt float64) RatePath` | Single GBM path: S(t+1) = S(t) * exp((mu - 0.5*sigma^2)*dt + sigma*sqrt(dt)*Z) |
| `GeneratePaths(numPaths, steps int, dt float64) []RatePath` | Multiple paths sequentially |

```go
rg := stochastic.NewRateGenerator(0.05, 0.02, 0.15)

// Single path: 10 steps, dt=1.0 (annual)
path := rg.GeneratePath(10, 1.0)
for t, rate := range path {
    fmt.Printf("t=%d: rate=%.6f\n", t, rate)
}

// 100000 paths for Monte Carlo aggregation
paths := rg.GeneratePaths(100000, 10, 1.0)
```

---

### reader

CSV streaming and parsing with parallel processing support.

#### Types

```go
// CensusRecord represents a policy record with core actuarial fields.
type CensusRecord struct {
    Age        int     `csv:"age"`
    Sex        string  `csv:"sex"`
    PolicyType string  `csv:"policy_type"`
    SumAssured float64 `csv:"sum_assured"`
    Term       int     `csv:"term"`
}

// CSVOptions configures CSV reading behavior.
type CSVOptions struct {
    Header    bool // First row contains column names
    Limit     int  // Max rows to read (0 = unlimited)
    Delimiter byte // Column delimiter (default ',')
}

// StreamOptions configures chunked parallel streaming.
type StreamOptions struct {
    CSVOptions            // Embeds Header, Limit, Delimiter
    ChunkSize int         // Records per chunk (default: auto)
    Workers   int         // Goroutine count (default: NumCPU)
}

// ColumnMap maps CSV column names to their indices.
type ColumnMap map[string]int

// ChunkProcessor is a callback for processing chunks of CensusRecords.
type ChunkProcessor func(chunk []CensusRecord) error
```

#### Functions

| Signature | Description |
|-----------|-------------|
| `StreamCSV(filepath string, opts CSVOptions, fn func([]string)) error` | Generic CSV reader; calls fn per row with string fields |
| `StreamCSVRaw(filepath string, opts CSVOptions, fn func([][]byte)) error` | Zero-allocation CSV reader; calls fn with raw byte slices |
| `StreamCSVWithPV(filepath string, opts CSVOptions, pvFn func(float64, int) float64) (float64, int)` | Read CSV as CensusRecords, calculate PV per row |
| `StreamCensus(filepath string, opts CSVOptions, fn func(CensusRecord)) error` | Fast CensusRecord streaming; byte-level parallel path for default columns |
| `StreamCensusChunked(filepath string, opts StreamOptions, processFn ChunkProcessor) (int, error)` | Parallel chunked CensusRecord processing |
| `StreamCensusWithPV(filepath string, opts StreamOptions, pvFn func(float64, int) float64) (float64, int)` | Parallel CensusRecord PV calculation |
| `ParseCensusRow(fields []string, colMap ColumnMap) (CensusRecord, error)` | Convert string fields to CensusRecord using column mapping |
| `GetHeaders(filepath string, delimiter byte) ([]string, error)` | Read header row and return column names |

```go
// Generic streaming
opts := reader.CSVOptions{Header: true, Limit: 1000}
reader.StreamCSV("data.csv", opts, func(fields []string) {
    fmt.Println(fields)
})

// Zero-allocation streaming
reader.StreamCSVRaw("data.csv", opts, func(fields [][]byte) {
    // process raw bytes without allocation
})

// Census record streaming
reader.StreamCensus("policies.csv", opts, func(rec reader.CensusRecord) {
    fmt.Printf("Age: %d, Sum: %.2f\n", rec.Age, rec.SumAssured)
})

// Chunked parallel processing
sopts := reader.StreamOptions{Header: true, ChunkSize: 10000, Workers: 8}
count, err := reader.StreamCensusChunked("policies.csv", sopts, func(chunk []reader.CensusRecord) error {
    // process 10000 records per chunk, in parallel
    return nil
})

// One-pass PV calculation
converter := rates.NewRateConverter(0.05)
totalPV, count := reader.StreamCSVWithPV("policies.csv", opts, converter.PresentValue)

// Get headers then parse manually
headers, _ := reader.GetHeaders("data.csv", ',')
colMap := reader.ColumnMap{}
for i, h := range headers {
    colMap[h] = i
}
rec, _ := reader.ParseCensusRow([]string{"30", "M", "term", "100000", "20"}, colMap)
```

---

### writer

Streaming JSON output for valuation results.

#### Types

```go
// JSONRecord is the output structure for valuation results.
type JSONRecord struct {
    Age          int     `json:"age"`
    Sex          string  `json:"sex"`
    PolicyType   string  `json:"policy_type"`
    SumAssured   float64 `json:"sum_assured"`
    Term         int     `json:"term"`
    PresentValue float64 `json:"present_value"`
}

// JSONWriter streams JSON arrays without buffering all records in memory.
type JSONWriter struct { /* unexported fields */ }
```

#### Functions

```go
// NewJSONWriter creates a streaming JSON writer wrapping any io.Writer.
f, _ := os.Create("output.json")
jw := writer.NewJSONWriter(f)
defer jw.Close()

// StreamJSON is a convenience function for writing a slice as JSON.
records := []writer.JSONRecord{{Age: 30, SumAssured: 100000, PresentValue: 37688.95}}
writer.StreamJSON(records, os.Stdout)
```

#### Methods

| Signature | Description |
|-----------|-------------|
| `WriteRecord(record JSONRecord) error` | Write a single record; opens JSON array on first call |
| `Close() error` | Finalize the JSON array; writes `[]` if no records |

```go
jw := writer.NewJSONWriter(os.Stdout)
jw.WriteRecord(writer.JSONRecord{Age: 30, SumAssured: 100000, PresentValue: 37688.95})
jw.WriteRecord(writer.JSONRecord{Age: 45, SumAssured: 200000, PresentValue: 78000.00})
jw.Close()
// Output: [{"age":30,...},{"age":45,...}]
```

---

### concurrency

Worker pool for parallel actuarial computations.

#### Types

```go
// WorkerPool distributes CensusRecord processing across goroutines.
type WorkerPool struct { /* unexported fields */ }
```

#### Functions

```go
// NewWorkerPool creates a pool with the given worker count.
// If workers <= 0, defaults to runtime.NumCPU().
wp := concurrency.NewWorkerPool(8, converter)

// ProcessBatch is a convenience function that creates a pool and processes records.
// Returns the total present value across all records.
totalPV := concurrency.ProcessBatch(records, converter, 8)
```

#### Methods

| Signature | Description |
|-----------|-------------|
| `ProcessBatch(records []reader.CensusRecord) float64` | Process records in parallel; falls back to sequential for <1000 records |

```go
converter := rates.NewRateConverter(0.05)
records := []reader.CensusRecord{
    {Age: 30, SumAssured: 100000, Term: 20},
    {Age: 45, SumAssured: 200000, Term: 10},
}

// Using convenience function
totalPV := concurrency.ProcessBatch(records, converter, 4)

// Using worker pool directly
wp := concurrency.NewWorkerPool(4, converter)
totalPV = wp.ProcessBatch(records)
```

## Contributing

Contributions welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) file.
