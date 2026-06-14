# Roadmap

**Current**: v0.9.0 — Server API & Profit Endpoint
(Exported `Server.Handler()` for embedding, `POST /profit` endpoint wrapping pkg/profit, response caching for idempotent endpoints, concurrency limiter per route)

## v0.8.0 — CLI & API Polish (Released 2026-05-27)
- [DONE] Reserve methods interface (gross/net premium, prospective, retrospective)
- [DONE] In-memory census source (CensusSource interface)
- [DONE] Profit testing / cashflow projection basics (pkg/profit)
- [DONE] Lock public API surface (no breaking changes after this)
- [PARTIAL] Comprehensive error handling + validation
- [MOVED] Deployment examples (Dockerfile for `serve`, Fly.io / Railway one-click) — moved to v0.9.0

## v0.9.0 — Server API & Profit Endpoint (2026-05-29)
- `POST /profit` endpoint for profit testing / cashflow projection
- Exported `Server.Handler()` for embedding and integration testing
- Response caching on all idempotent endpoints (annuity, reserve, profit)
- Per-route concurrency limiting with bounded wait queues
- Deployment examples: Dockerfile for `serve`, Fly.io / Railway one-click config

## v1.0.0 — Stable Core (Mid-June 2026)
The "show to employers / put on CV" version.

- 90%+ test coverage across all packages
- `POST /valuation` endpoint: batch multiple policies → reserves, PV, risk measures
- Flexible assumptions in profit testing (expense inflation, lapses, partial withdrawals)
- "Used by" section starter in README
- Tag v1.0.0 + announce on r/actuary, LinkedIn, Go subreddit

## v1.1.0 — Advanced Life Models (Late June / Early July 2026)
- Markov chain models (disability, multiple decrements, termination)
- Credibility theory (Bühlmann, Bühlmann-Straub)
- CIR interest rate model
- Percentiles / TVaR improvements

## v1.2.0 — Variance Reduction (July 2026)
- Antithetic variates
- Control variates
- Latin Hypercube sampling
- Target: 2–5x variance reduction on typical risk metrics

## v1.3.0 — Nested Stochastic / ALM (August 2026)
- Couple stochastic interest rates (Vasicek/CIR) with cashflow projections
- Reinvestment risk modelling
- Dynamic policyholder behaviour (surrender when rates move)
- Scenario-dependent asset returns driving liability cashflows

## v2.0.0 — IFRS 17 & Solvency II Module (Q3 2026)
- **Killer feature.** IFRS 17: cashflow grouping, risk adjustment (VaR/CTE → RA),
  CSM (Contractual Service Margin) schedules, coverage units
- Solvency II: SCR calculation, standard formula calibrations
- BEL (Best Estimate Liability) + RA + CSM report output
- P&L / OCI statement projection per cohort
- Distinction between onerous and profitable contracts

## v2.1.0 — Zig as Primary Engine (Q4 2026)
- Complete Zig port passes the full Go test suite (already mirrored)
- Zig exposes C ABI: one shared library callable from any language
- Go becomes the CLI/shim; Zig is the computation kernel
- WASM compile target: run policy illustrations in-browser
- Python/R/Excel bindings via the C ABI (no HTTP dependency)

## v3.0.0 — Ecosystem & Ubiquity (2027)
- `vstar-py` on PyPI (thin CFFI wrapper → C ABI, + pandas integration)
- `vstar-wasm` NPM package (browser-based policy illustration widget)
- Plugin system for custom cashflow logic (no-compile for Go, comptime for Zig)
- Public benchmark repo + GitHub Pages "try it live"
- Excel add-in via the C ABI (XLL)

---

**Guiding principles**
- 1–3 weekends per version max
- Zero external dependencies forever (Go std lib only)
- Performance first: keep the 360M rows/sec streaming and sub-second Monte Carlo as selling points
- Focus on what actuaries actually run daily: valuations, reserves, stochastic risk
- The Zig port isn't a rewrite — it's the deployment target. C ABI + WASM = everywhere
