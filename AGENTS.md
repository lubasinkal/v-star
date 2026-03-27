# AGENTS.md - v-star Development Guide

## Overview

v-star is a high-performance, zero-dependency actuarial engine for Concurrent Financial Simulations built in Go 1.26.1. It uses only the Go Standard Library.

## Build Commands

```bash
# Build all packages
go build ./...

# Build the CLI binary
go build ./cmd/v-star

# Run the CLI
./v-star.exe -i 0.05 -j 0.02

# Format code (required before commits)
go fmt ./...

# Static analysis
go vet ./...

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test ./pkg/rates/...
go test ./pkg/reader/...
go test ./pkg/stochastic/...

# Run a single test function
go test -v -run TestPresentValue ./pkg/rates/...
go test -v -run TestV ./pkg/rates/...

# Run tests with benchmark output
go test -bench=. ./pkg/rates/...

# Run tests with coverage
go test -cover ./...
```

## Project Structure

```
cmd/
  v-star/          # Main CLI application
    main.go        # Entry point, flag parsing
    commands/      # Command implementations (read.go, montecarlo.go, bench.go)
  generate/        # Data generation utilities

pkg/
  rates/           # Interest rate calculations, discount factors, DiscountFactor interface
  stochastic/      # Monte Carlo simulations, geometric Brownian motion
  mortality/       # Mortality tables, qx/px calculations, CSV loading
  annuities/       # Annuity calculations (whole life, term, deferred)
  reserves/        # Policy reserves (net premium, gross premium, prospective)
  risk/            # Risk measures (VaR, CTE, Expected Shortfall)
  reader/          # CSV parsing (streaming, parallel), CensusRecord struct
  writer/          # JSON output streaming
  concurrency/     # Worker pool for parallel processing

examples/
  quickstart/      # Basic present value and duration demo
  monte_carlo_risk/ # Monte Carlo + VaR/CTE risk analysis
  csv_valuation/   # CSV streaming with valuation
  python_bridge/   # Python wrapper and Jupyter notebook
```

## Code Style Guidelines

### Formatting and Style
- Run `go fmt ./...` before committing code
- Use the `gofmt` tool for consistent formatting
- Standard Go indentation (tabs, not spaces)

### Naming Conventions
- **Types/Structs**: PascalCase (e.g., `RateConverter`, `CensusRecord`, `WorkerPool`)
- **Functions/Methods**: PascalCase (e.g., `PresentValue`, `GeneratePath`, `StreamCSV`)
- **Variables/Parameters**: camelCase (e.g., `effectiveRate`, `numWorkers`, `sumAssured`)
- **Constants**: camelCase or PascalCase depending on scope
- **Package names**: Short, lowercase, no underscores (e.g., `rates`, `reader`, `stochastic`)
- **Files**: lowercase with underscores for multi-word names (e.g., `rates_test.go`, `csv.go`)

### Imports
- Group imports: standard library first, then third-party, then internal packages
- Use blank line between groups
- Example:
```go
import (
    "bufio"
    "bytes"
    "io"
    "os"

    "github.com/lubasinkal/v-star/pkg/rates"
)
```

### Types and Structs
- Use struct tags for serialization (e.g., `json:"field_name"`, `csv:"field_name"`)
- Export fields that need serialization; keep internal fields lowercase
- Example:
```go
type JSONRecord struct {
    Age          int     `json:"age"`
    Sex          string  `json:"sex"`
    PolicyType   string  `json:"policy_type"`
    SumAssured   float64 `json:"sum_assured"`
    PresentValue float64 `json:"present_value"`
}
```

### Error Handling
- Return errors from functions when possible
- Handle errors early with clear error messages
- Use `fmt.Printf("Error: %v\n", err)` followed by `os.Exit(1)` in CLI code
- Example pattern:
```go
file, err := os.Open(filepath)
if err != nil {
    return err
}
defer file.Close()
```

### Concurrency Patterns
- Use goroutines with WaitGroups for parallel processing
- Channels for collecting results
- Always use `defer wg.Done()` in goroutines
- Example pattern:
```go
func (wp *WorkerPool) processParallel(records []RecordType) float64 {
    results := make(chan float64, wp.workers)
    for w := 0; w < wp.workers; w++ {
        wp.wg.Add(1)
        go func(chunk []RecordType) {
            defer wp.wg.Done()
            // work...
            results <- partial
        }(records[start:end])
    }
    go func() {
        wp.wg.Wait()
        close(results)
    }()
}
```

### Testing Conventions
- Test files: `*_test.go` naming pattern
- Table-driven tests with descriptive names
- Use `t.Run()` for subtests
- Floating point comparisons use tolerance (typically `1e-9` for high precision)
- Benchmark tests use `b.Loop()` (Go 1.21+)
- Example test structure:
```go
func TestPresentValue(t *testing.T) {
    converter := RateConverter{EffectiveRate: 0.05}
    tests := []struct {
        name       string
        sumAssured float64
        term       int
        want       float64
    }{
        {"Zero term", 1000, 0, 1000},
        {"Simple PV", 1000, 1, 952.38},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := converter.PresentValue(tt.sumAssured, tt.term)
            if got < tt.want-0.01 || got > tt.want+0.01 {
                t.Errorf("PresentValue() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Performance Considerations
- Zero-allocation streaming for CSV parsing
- Use `bufio.Scanner` with custom buffer sizes for large files
- Pre-allocate slices with capacity hints (`make([]Type, 0, capacity)`)
- Use `bytes.Buffer` and `bytes.IndexByte` for fast string parsing
- Fallback to sequential processing for small workloads

### CLI Design
- Use `flag` package for command-line flags
- Use `flag.Args()` for subcommands
- Provide `--help` flag with clear usage information
- Exit codes: 0 for success, 1 for errors

## Key Dependencies and Features

- **Go Standard Library Only**: No external dependencies
- **CSV Parsing**: Custom zero-allocation streaming parser with parallel support
- **Actuarial Calculations**: Present value, v-star discount factor, nominal-to-effective conversion
- **Monte Carlo**: Geometric Brownian motion for interest rate simulation
- **Risk Measures**: VaR, CTE (Expected Shortfall), full risk reports
- **Concurrency**: Worker pool pattern with configurable goroutines

## Documentation

- Add doc comments for exported functions and types (will appear in `go doc`)
- Keep code self-documenting through clear naming
- Document mathematical formulas in comments when relevant
