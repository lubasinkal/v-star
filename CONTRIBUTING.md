# Contributing to v-star

Thanks for your interest in contributing! This guide will help you get started.

## Development Setup

```bash
# Clone the repository
git clone https://github.com/lubasinkal/v-star.git
cd v-star

# Build all packages
go build ./...

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Format code
go fmt ./...

# Static analysis
go vet ./...
```

## Project Structure

```
cmd/v-star/         # CLI application
cmd/generate/       # Test data generator
pkg/rates/          # Interest rate calculations
pkg/mortality/      # Mortality tables
pkg/annuities/      # Annuity calculations
pkg/reserves/       # Policy reserves
pkg/stochastic/     # Monte Carlo simulations
pkg/risk/           # Risk measures (VaR, CTE)
pkg/reader/         # CSV parsing and CensusRecord
pkg/writer/         # JSON output
pkg/concurrency/    # Worker pool
examples/           # Runnable examples
docs/               # Documentation and roadmap
```

## How to Contribute

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Write tests for new functionality
4. Ensure all tests pass: `go test ./...`
5. Ensure code is formatted: `go fmt ./...`
6. Commit with a clear message
7. Push to your fork and open a Pull Request

## Code Style

- Follow standard Go conventions
- Run `go fmt ./...` before committing
- Add tests for all exported functions
- Use table-driven tests where appropriate
- Add doc comments for exported functions

## Adding a New Actuarial Model

1. Create a new package under `pkg/` if needed
2. Define clear interfaces and types
3. Implement with zero external dependencies
4. Add comprehensive tests with known values
5. Add benchmarks for performance-critical code
6. Update README with API documentation

## Reporting Issues

Open an issue with:
- Description of the problem
- Steps to reproduce
- Expected vs actual behavior
- Go version and OS
