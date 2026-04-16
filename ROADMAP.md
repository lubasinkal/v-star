# Roadmap

Current: v0.2.0 (tagged Mar 27, 2026)

## v0.3.0 — Polish (End April 2026)

Production-ready. No new features, just polish.

- Bump go.mod + tag v0.3.0
- Full godoc on all pkg/ exports
- Runnable Example funcs for pkg.go.dev
- Quickstart section in README
- Benchmarks: v-star vs Polars on 2M rows
- GitHub release

## v0.4.0 — HTTP API (Mid-May 2026)

REST API for non-Go users.

- pkg/server (net/http only)
  - POST /value, POST /montecarlo
  - GET /mortality/{table}, POST /convert-rate
- v-star serve subcommand
- examples/http-client (curl, Python requests)
- Deployment docs (Fly.io, Railway)

## v1.0.0 — Stable (Early June 2026)

Lock API. Show to employers.

- No breaking changes after this
- 90%+ test coverage
- "Used by" section in README

## v1.1.0 — New Models (Late June 2026)

Pick 2-3:

- Markov models (disability/termination)
- Credibility (Bühlmann)
- Vasicek/CIR in stochastic
- Percentiles + confidence intervals

## v1.2.0 — Variance Reduction (July 2026)

- Antithetic + control variates
- Latin Hypercube sampling
- 2-5x variance reduction

## v2.0.0 — Ecosystem (Q3/Q4 2026)

- vstar-py on PyPI
- Plugin system for cashflows
- Dashboard example

---

1-3 weekends per version. Zero deps forever.