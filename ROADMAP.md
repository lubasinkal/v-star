# Roadmap

**Current**: v0.7.0 — Core Interfaces & Parallel Monte Carlo
(Core interfaces stabilized, parallel MC path generation, version injection via ldflags, unified Record type)

## v0.8.0 — CLI & API Polish (Early June 2026)
- Reserve methods interface (gross/net premium, prospective, retrospective)
- In-memory census source (CensusSource interface)
- Profit testing / cashflow projection basics (pkg/profit)
- Lock public API surface (no breaking changes after this)
- Comprehensive error handling + validation
- Deployment examples (Dockerfile for `serve`, Fly.io / Railway one-click)

## v1.0.0 — Stable Core (Mid-June 2026)
The "show to employers / put on CV" version.

- 90%+ test coverage
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

## v2.0.0 — Ecosystem & Usability (Q3 2026)
- Official `vstar-py` on PyPI (thin HTTP client + pandas integration)
- Simple plugin system for custom cashflow logic
- Lightweight dashboard example (pure Go templates + HTMX/Streamlit)
- Public benchmark repo or GitHub Pages "try it live" (hosted `serve` instance)

---

**Guiding principles**
- 1–3 weekends per version max
- Zero external dependencies forever (std lib only)
- Performance first: keep the 360M rows/sec streaming and sub-second Monte Carlo as selling points
- Focus on what actuaries actually run daily: valuations, reserves, stochastic risk