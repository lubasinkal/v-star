# Changelog

All notable changes to v-star are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

For future plans, see [ROADMAP.md](./ROADMAP.md).

## [Unreleased]

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

[0.3.1]: https://github.com/lubasinkal/v-star/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/lubasinkal/v-star/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/lubasinkal/v-star/compare/v0.1.0...v0.2.0