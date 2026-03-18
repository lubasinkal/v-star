# v-star ($v^*$)

A high-performance, zero-dependency actuarial engine for Concurrent Financial Simulations built in Go.

## The Origin
The name **v-star** comes from a class joke between my University lecturer and comrades (brothers and sister deployed to study Actuarial Science): 
If an annuity (or more precisely, the payments/premiums associated with the annuity) compound (or earn interest) at rate j while being discounted (valued) at rate i, then the adjusted (effective) discount factor is

$v^* = (1+j)*v$.

## Why v-star?
Modern financial software is often bloated and slow. **v-star** is designed for:
* **Zero Dependencies:** Uses only the Go Standard Library.
* **Extreme Speed:** Leverages Go's concurrency (goroutines) for mass valuations.
* **Auditability:** Pure, readable math implementations.
* **Possible Job Offers** : Unemployed so why not spend my time testing out how far Go can go in actuarial work

## Features
* **High-Performance CSV Parser:** Zero-allocation streaming parser
* **Actuarial Valuation:** Calculate Present Value of policies using standard or v-star discount factors
* **Monte Carlo Simulation:** Generate interest rate paths for stochastic modeling
* **Multiple Output Formats:** Console and JSON output
* **CLI Tools:** Easy-to-use command-line interface

## Quick Start

### Build
```bash
go build ./cmd/v-star
```

### Basic Usage
```bash
# Calculate discount factors
./v-star -i 0.05 -j 0.02

# Read CSV and calculate valuations
./v-star read policies.csv --benchmark

# Export as JSON
./v-star read policies.csv --output=json

# Generate Monte Carlo interest rate paths
./v-star montecarlo --paths=100000 --steps=10 --drift=0.02 --volatility=0.15
```

### Example Output
```
Processed 100000 records
Total Present Value: 13339837523.23
1: Age=62, Sex=female, Type=term, Sum=490294.00, Term=22, PV=167606.94
...

=== Benchmark Results ===
Total rows: 100000
Duration: 76.44ms
Throughput: 1308188 rows/sec
Total Present Value: 13339837523.23
```

### Monte Carlo Example
```
Generating 100000 Monte Carlo interest rate paths...
Parameters: Initial Rate=5.00%, Drift=2.00%, Volatility=15.00%, Steps=10

=== Monte Carlo Results ===
Paths Generated: 100000
Duration: 105.23ms
Throughput: 950304 paths/sec

Final Rate Statistics:
  Average: 5.4080%
  Minimum: 1.2345%
  Maximum: 23.6789%
```

## Performance
* **CSV Parsing:** ~4.8M rows/sec
* **Valuation Calculation:** ~3.1M rows/sec (1M records in 320ms)
* **Monte Carlo Simulation:** ~100k paths/sec (100k paths in 1ms)
* **Memory Usage:** Minimal (streaming processing)

## Installation
```bash
go get github.com/lubasinkal/v-star
```

