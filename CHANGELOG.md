# Changelog

All notable changes to v-star are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

For future plans, see [ROADMAP.md](./ROADMAP.md).

## [Unreleased]

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

[Unreleased]: https://github.com/lubasinkal/v-star/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/lubasinkal/v-star/compare/v0.1.0...v0.2.0