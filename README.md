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
// Multi-core parsing for generic CSVs
opts := reader.CSVOptions{Header: true}
count, fieldCount := reader.StreamCSVWithCallback("data.csv", opts, func(fields []string) {
    // process fields
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

## Architecture

```
pkg/
├── reader/          # CSV parsing and streaming
│   ├── csv.go      # Generic CSV parser
│   ├── census.go   # CensusRecord parser
│   └── streaming.go # Parallel streaming
├── rates/          # Interest rate calculations
├── annuities/      # Annuity calculations
├── mortality/     # Mortality tables
├── reserves/      # Policy reserves
├── stochastic/    # Monte Carlo simulations
└── writer/        # JSON output
```

## Contributing

Contributions welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) file.
