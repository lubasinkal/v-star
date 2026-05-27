# Changelog

All notable changes to v-star are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

For future plans, see [ROADMAP.md](./ROADMAP.md).

## [Unreleased]

### Added
- Core interfaces: `PathGenerator` (stochastic), `ContingencyCalculator` (annuities), `RecordWriter` (writer)
- `StreamCensusFromReader` for streaming census data from any `io.Reader`
- Unified `Record` type (replaces `JSONRecord`/`CSVRecord`)
- `Ex()` and `Lx()` methods on `MortalityTable`
- Knowledge graph with `graphify` for codebase navigation
- `GEMINI.md` with graphify usage instructions
- `AGENTS.md` with development guide for AI coding agents
- `CensusSource` interface with 3 implementations (`SliceCensusSource`, `FileCensusSource`, `ReaderCensusSource`)
- `pkg/profit` — profit testing / cashflow projection with profit signature, PV of profits, profit margin, IRR, payback year

### Changed
- Reader coverage: 51% → 75% (StreamCensus, StreamCensusFromReader, StreamCSVRaw, parallel paths)
- Annuities coverage: 78% → 94% (DeferredTerm, ApproxWholeLifeImmediate)
- Server coverage: 68% → 82% (all 9 annuity computations, 4 reserve methods, validation errors)

### Changed
- Updated README code examples to be self-contained and runnable
- Renamed `JSONRecord`/`CSVRecord` → unified `Record` type
- Applied Effective Go patterns across codebase (idiomatic naming, error handling, concurrency)
- All benchmarks migrated to `b.Loop()` (Go 1.24+)
- Cleaned up repo: removed build artifacts, Python `__pycache__`, large test CSVs

### Removed
- Duplicated CSV parsing helpers from `mortality` package (parseLines, splitCSV, detectColumns, parseInt, parseFloat, toLower, maxAge) — now uses `reader` package
- Local `AnnuityResult` struct from `read` command — unified to `writer.Record`

### Performance
- Risk measures (VaR/CTE/ComputeReport): 1.4–1.8× faster. Eliminated 800KB allocation per call by sorting in-place instead of copy-then-sort.

## [0.7.0] - 2026-05-01

### Added
- Parallel Monte Carlo path generation via `GeneratePathsParallel` (2.2× faster at 1M paths)
- Version injected via `ldflags` instead of hardcoded in source
- Merged CI and release into single gated workflow

### Changed
- Updated README and ROADMAP for v0.7.0
- Reformatted all source files with `gofmt`
- Restored README intro and story to original tone

## [0.6.1] - 2026-04-26

### Added
- JSON tags to all serializable models
- Godoc improvements across all packages

### Removed
- 8-core cap on parallel workers (now uses `runtime.NumCPU()`)
- `ProcessBatch` and `ExpectedShortfall` convenience wrappers from concurrency/risk packages
- `StreamCensusWithPV` and `StreamCSVWithPV` from reader API
- `DiscountFactor` type assertions (now accepts any `DiscountFactor`)

## [0.6.0] - 2026-04-24

### Added
- Memory-mapped I/O for CSV parsing (`mmap_unix.go`, `mmap_windows.go`) — 0.80s for 10M rows
- Policy string interning (`whole_life`) for faster streaming
- Dynamic discount table for O(1) discount factor lookups
- Iterative `Pow` replacement for `math.Pow` (micro-optimization)

### Changed
- Flat Monte Carlo buffers with `make([]float64, numPaths*(steps+1))` — fewer allocations
- Ziggurat MC paths with cached drift and diffusion terms
- O(1) mortality `Px` via precomputed `lx` survival table
- O(n) reserve recurrence formula (was O(n²))
- Single file open in `StreamCensus` (was reopening per chunk)

### Performance
- CSV parsing: 28M → 360M rows/sec throughput
- Present value: 22.8 → 2.6 ns/call
- Annuity (whole life): 2μs → 512 ns
- Monte Carlo (100k paths): 40ms → 27ms

## [0.5.2] - 2026-04-20

### Added
- Multiple decrement table (`DecrementTable`) combining death, lapse, disability
- Vasicek mean-reverting interest rate model (`pkg/stochastic/vasicek.go`)
- Confidence intervals for risk report (`StdError`, `Confidence95Lo`, `Confidence95Hi`)
- HTTP API server (`pkg/server`) with CORS, request logging, graceful shutdown
- Generic `WorkerPool[T any]` with context cancellation (`ProcessBatchContext`)
- `StreamCensusChunked` for batch CSV processing
- CSV and text report export endpoints (`/export/csv`, `/export/report`)
- CSV file upload endpoint (`/upload/csv`) with multipart form support
- Deferred annuity types (`DeferredWholeLife`, `DeferredTerm`)
- Whole life NSP, term NSP, endowment NSP calculations
- Comprehensive test coverage for stochastic edge cases, generic reserve paths, writer `GenerateTextReport`, CLI commands, server upload endpoints

### Changed
- Server middleware refactored (logging, recovery, CORS into separate layers)
- Worker pool now generic (`WorkerPool[T any]`) instead of `CensusRecord`-specific
- Reader API consolidated: `StreamCensus`, `StreamCSV`, `StreamCSVRaw` with consistent options
- Graceful shutdown on `SIGINT`/`SIGTERM` in `serve` command
- Single-sort risk report (sorts once, derives all metrics from sorted data)
- Incremental `Px` annuity accumulation (no recomputation per term)
- Buffer pool for CSV line parsing
- Optional MC paths (set `include_paths=false` to skip returning path data)

### Fixed
- Annuity loop bounds (termination at `maxAge`)
- GBM zero-volatility test (continuous compounding formula)
- Concurrency tests (race conditions in `ProcessBatchContext` canceled context test)
- Server fmt import
- Read test `os.Exit` during tests
- Decrement table `QxByCause` edge cases
- Debug print removal, nil guards, double-pass `Px`, dead code

## [0.5.1] - 2026-04-19

### Fixed
- Removed redundant newlines from `fmt.Println` calls in CLI commands

## [0.5.0] - 2026-04-18

### Added
- Detailed CLI help with subcommand-specific flags and examples
- Python bridge now includes HTTP API client (`VStar` class) and CLI wrapper (`VStarCLI`)
- Comprehensive test coverage across all packages:
  - pkg/server: 45%+ (was 0%)
  - pkg/reader: 41%+ (was 20%)
  - pkg/mortality: 63%+ (was 10%)

### Changed
- Updated go.mod to v0.5.0
- Expanded README Quickstart with curl and Python HTTP examples
- Added Monte Carlo benchmark comparison to README

### Fixed
- Mortality table edge cases (empty tables, out-of-range ages)
- CLI help text now shows all subcommands and flags

## [0.4.0] - 2026-04-16

### Added
- HTTP API server (pkg/server) for non-Go users (Python, R, Excel, etc.)
- CLI serve subcommand (`v-star serve`) to start the HTTP API server
- API endpoints:
  - POST /value - Calculate present value for policy records
  - POST /montecarlo - Run Monte Carlo simulation with VaR/CTE
  - POST /convert-rate - Convert between nominal and effective interest rates
  - GET /mortality/{table} - Retrieve mortality table metadata
- Website documentation with API usage examples and curl commands
- Raycast-inspired design system (docs/DESIGN.md) for UI consistency

### Changed
- Updated README.md to include an API section with endpoint examples
- Improved website styling and layout for better readability

## [0.3.1] - 2026-04-16

### Changed
- README.md rewritten to be more approachable for non-Go users (actuaries, Excel/VBA users, Python/R users)
- Added clearer explanations and analogies for each feature

## [0.3.0] - 2026-04-16

### Added
- Full godoc coverage for all public types and functions in pkg/
- Runnable ExampleXXX functions for pkg.go.dev (rates, annuities, mortality, risk, stochastic)
- Library Quickstart section in README.md with 5 copy-paste import examples

### Changed
- Updated README with "Why Go for Actuaries?" section targeting Python/R/Excel/VBA users
- Fixed VaR calculation (now returns percentile directly, not inverse)
- Improved readability of code examples in documentation

### Fixed
- VaR 95% now returns correct 95th percentile instead of 5th

## [0.2.0] - 2026-03-27

### Added
- Risk measures package with VaR and CTE (Expected Shortfall) calculations
- Full risk report generation with percentile analysis
- Examples directory with runnable demonstrations:
  - Quickstart: present value and duration calculations
  - Monte Carlo risk: stochastic simulation with risk metrics
  - CSV valuation: streaming census data processing
- Python bridge with Jupyter notebook for visualization
- GitHub Actions CI workflow for automated testing
- CONTRIBUTING.md with contribution guidelines

### Changed
- Updated README with CI badge, expanded examples section, and risk measures documentation
- Improved project structure and documentation clarity

## [0.1.0] - 2026-01-01

### Added
- Core actuarial engine with rates, annuities, reserves, and mortality tables
- Parallel CSV streaming with CensusRecord parsing (28M rows/sec)
- Monte Carlo interest rate simulation using Geometric Brownian Motion
- Present value and v-star discount factor calculations
- Duration and convexity calculations for bond analysis
- CLI with read, montecarlo, and bench subcommands
- JSON output support for integration with other tools
- Comprehensive test suite with benchmarks

[0.7.0]: https://github.com/lubasinkal/v-star/compare/v0.6.1...v0.7.0
[0.6.1]: https://github.com/lubasinkal/v-star/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/lubasinkal/v-star/compare/v0.5.2...v0.6.0
[0.5.2]: https://github.com/lubasinkal/v-star/compare/v0.5.1...v0.5.2
[0.5.1]: https://github.com/lubasinkal/v-star/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/lubasinkal/v-star/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/lubasinkal/v-star/compare/v0.3.2...v0.4.0
[0.3.2]: https://github.com/lubasinkal/v-star/compare/v0.3.1...v0.3.2
[0.3.1]: https://github.com/lubasinkal/v-star/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/lubasinkal/v-star/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/lubasinkal/v-star/compare/v0.1.0...v0.2.0