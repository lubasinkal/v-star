# v-star Roadmap

Zero-deps Go actuarial engine. Current: v0.2.0

## v0.3.0 — Polish & Professional (End of April 2026)

Make it production-ready.

- [ ] Bump go.mod + tag v0.3.0
- [ ] Full godoc on all public types/functions in pkg/
- [ ] Runnable ExampleXXX funcs for pkg.go.dev
- [ ] Library Quickstart in README (5 import examples)
- [ ] Benchmark table: v-star vs Polars on 2M rows
- [ ] GitHub release with auto-notes

## v0.4.0 — HTTP API (Mid-May 2026)

Non-Go users access via HTTP.

- [ ] pkg/server (net/http, no frameworks)
  - POST /value — stream CSV, get PV/reserves
  - POST /montecarlo — get paths + VaR/CTE
  - GET /mortality/{table}
  - POST /convert-rate
- [ ] CLI flag --serve or v-star serve subcommand
- [ ] examples/http-client (curl + Python)
- [ ] Fly.io/Railway deployment docs
- [ ] Python bridge -> HTTP option

## v1.0.0 — Stable Library (Early June 2026)

Lock the API. Production-ready.

- [ ] No breaking changes after this
- [ ] Semver policy in CONTRIBUTING.md
- [ ] 90%+ test coverage
- [ ] Blog post: "v-star v1.0: From VBA rage to 28M rows/sec"
- [ ] "Used by" section in README

## v1.1.0 — New Actuarial Models (Late June 2026)

Pick 2-3 based on work needs:

- [ ] pkg/markov — 2/3-state disability/termination
- [ ] pkg/credibility — Bühlmann & Bühlmann-Straub
- [ ] Percentiles + confidence intervals in risk
- [ ] Vasicek/CIR in pkg/stochastic
- [ ] testdata for Botswana/SADC mortality tables

## v1.2.0 — Variance Reduction (July 2026)

- [ ] Antithetic variates + control variates
- [ ] Latin Hypercube sampling
- [ ] 2-5x variance reduction in benchmarks

## v2.0.0 — Plugin System (Q3/Q4 2026)

- [ ] Plugin interface for custom cashflows
- [ ] vstar-py on PyPI
- [ ] Dashboard example (actuworry tie-in)
- [ ] Community: issues, Discord/LinkedIn

## v2.1+ — Long-term

- WASM build for browser
- R bridge or Excel add-in (via HTTP)

---

Target: 1-3 weekends per version. Zero external deps always.