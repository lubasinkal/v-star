# Changelog

Notable changes to v-star. For future plans, see [ROADMAP.md](./ROADMAP.md).

## [0.9.0] - 2026-05-29
- Server API: `POST /profit` endpoint, exported `Server.Handler()`, response caching, concurrency limiter
- Deployment examples: Dockerfile, Fly.io / Railway config

## [0.8.0] - 2026-05-27
- `pkg/profit` — profit testing / cashflow projection (profit signature, PV of profits, IRR, payback year)
- Reserve methods interface (gross/net premium, prospective, retrospective)
- `CensusSource` interface with 3 implementations
- Unified `Record` type (replaces `JSONRecord`/`CSVRecord`)
- `StreamCensusFromReader`, `Ex()`, `Lx()` on `MortalityTable`
- Knowledge graph with `graphify` for codebase navigation
- Multi-language API client examples (Python, R, JS, TS, cURL)
- API freeze audit: 8 packages stable, public API locked
- Performance: risk measures 1.4-1.8x faster, 800KB allocation eliminated

## [0.7.0] - 2026-05-01
- Parallel Monte Carlo path generation (2.2x faster at 1M paths)
- Version injected via `ldflags`

## [0.6.0] - 2026-04-24
- Memory-mapped I/O for CSV parsing — 360M rows/sec throughput
- Dynamic discount table for O(1) discount factor lookups
- O(n) reserve recurrence formula (was O(n²))
- Iterative Pow replacement, flat Monte Carlo buffers, O(1) mortality Px via precomputed lx
- Present value: 22.8 → 2.6 ns/call. Whole life annuity: 2μs → 512 ns. MC 100k: 40ms → 27ms

## [0.5.2] - 2026-04-20
- Multiple decrement tables, Vasicek interest rate model
- Confidence intervals on risk reports
- HTTP API server with CORS, logging, graceful shutdown
- Generic `WorkerPool[T any]` with context cancellation
- Deferred annuity types, whole life/term/endowment NSP
- CSV upload & export endpoints

## [0.5.0] - 2026-04-18
- Detailed CLI help with subcommand flags/examples
- Python HTTP API client

## [0.4.0] - 2026-04-16
- HTTP API server (`v-star serve`): `/value`, `/montecarlo`, `/convert-rate`, `/mortality`
- Website documentation with API examples

## [0.3.0] - 2026-04-16
- Full godoc coverage, `ExampleXXX` functions for pkg.go.dev
- Fixed VaR 95% (now returns correct 95th percentile)

## [0.2.0] - 2026-03-27
- Risk measures (VaR, CTE), full risk reports
- Python bridge with Jupyter notebook
- GitHub Actions CI

## [0.1.0] - 2026-01-01
- Core actuarial engine: rates, annuities, reserves, mortality tables
- Parallel CSV streaming (28M rows/sec), GBM Monte Carlo
- CLI: read, montecarlo, bench
