# AGENTS.md – v-star Dev Guide

## Overview

Zero-dependency actuarial engine for concurrent financial simulations. Go 1.26.1, stdlib only.

## Build

```bash
go build ./...                    # all packages
go build ./cmd/v-star             # CLI binary
go fmt ./... && go vet ./...      # format & lint (required before commits)
go test ./...                     # all tests
go test -v -run <TestName> ./pkg/rates/...  # single test
go test -bench=. ./pkg/rates/...  # benchmarks
go test -cover ./...              # coverage
```

## Layout

```
cmd/v-star/          CLI entry point + commands (read, montecarlo, bench)
cmd/generate/        Data generation
pkg/rates/           Discount factors, rate conversion
pkg/stochastic/      GBM Monte Carlo simulation
pkg/mortality/       Mortality tables, qx/px, CSV loading
pkg/annuities/       Whole life, term, deferred annuities
pkg/reserves/        Net/gross premium, prospective reserves
pkg/risk/            VaR, CTE, expected shortfall
pkg/reader/          CSV streaming (parallel, zero-alloc)
pkg/writer/          JSON output streaming
pkg/concurrency/     Worker pool
examples/            quickstart, monte_carlo_risk, csv_valuation, python_bridge
```

## Style

- Run `go fmt ./...` before commits.
- **Naming**: PascalCase types/funcs, camelCase vars, lowercase packages/files.
- **Imports**: stdlib → third-party → internal, blank-line separated.
- **Errors**: return errors from funcs; `fmt.Printf("Error: %v", err); os.Exit(1)` in CLI.
- **Concurrency**: `sync.WaitGroup` + channels, `defer wg.Done()`.
- **Tests**: table-driven, `t.Run` subtests, float tolerance `1e-9`.
- **CLI**: `flag` package, `flag.Args()` subcommands, `--help`, exit 0/1.
- **Perf**: zero-alloc CSV, `bufio.Scanner`, pre-alloc slices, `bytes.Buffer`/`bytes.IndexByte`.
- See [Effective Go](https://go.dev/doc/effective_go).

## Dependencies

- **Go stdlib only**.
- CSV: custom zero-alloc streaming parser.
- Monte Carlo: GBM interest rate simulation.
- Risk: VaR, CTE, full risk reports.
- Concurrency: configurable worker pool.

## Doc

- Doc comments on exported symbols (`go doc`).
- Document math formulas in comments where relevant.

## Graphify

- Knowledge graph at `graphify-out/`.
- **`/graphify`**: invoke `skill` tool with `skill: "graphify"` first.
- **Queries**: `graphify query "<q>"`, `graphify path "<A>" "<B>"`, `graphify explain "<concept>"`.
- Skip graphify only if task is about stale/bad graph output or user says not to use it.
- Read `graphify-out/GRAPH_REPORT.md` only as fallback when query/path/explain are insufficient.
- After code edits: `graphify update .` (AST-only, no API cost).
