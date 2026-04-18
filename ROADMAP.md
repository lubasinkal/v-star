# Roadmap

**Current**: v0.4.0 (tagged April 16, 2026) — HTTP API & Accessibility  
(CLI + `v-star serve`, REST endpoints for value/montecarlo, mortality tables, rate conversion, Python/R/Excel friendly)

## v0.5.0 — Polish & Documentation (End of April / Early May 2026)
Make it feel production-ready and easy to recommend.

- Full godoc comments on **all** exported types/functions in `pkg/`
- Runnable `Example_` funcs for pkg.go.dev
- Expanded Quickstart in README (include `v-star serve` + curl + Python requests examples)
- Better CLI help (`v-star --help`, subcommand flags, config file support)
- Update benchmarks section with clearer table + Monte Carlo timing
- Polish CHANGELOG.md + GitHub release notes
- Fix any small issues (Python bridge example if still rough)

## v1.0.0 — Stable Core (Mid-May 2026)
The “show to employers / put on CV” version. No breaking changes after this.

- Lock public API (pkg/vstar)
- 90%+ test coverage (add table-driven tests for annuities, reserves, stochastic)
- Comprehensive error handling + validation
- Deployment examples (Dockerfile for `serve`, Fly.io / Railway one-click)
- “Used by” section starter in README (even if just “personal projects” for now)
- Tag v1.0.0 + announce on r/actuary, LinkedIn, Go subreddit

## v1.1.0 — Advanced Life Models (Late May / Early June 2026)
Add the models actuaries actually ask for next.

Pick any 3 (or all if momentum is good):
- Markov chain models (disability, multiple decrements, termination)
- Credibility theory (Bühlmann, Bühlmann-Straub)
- Enhanced stochastic: Vasicek / CIR interest rate models
- Percentiles, confidence intervals, and TVaR improvements

## v1.2.0 — Variance Reduction & Speed (June 2026)
Make Monte Carlo even more impressive.

- Antithetic variates
- Control variates
- Latin Hypercube sampling
- Target: 2–5x variance reduction on typical risk metrics
- Optional: parallel workers (runtime.GOMAXPROCS) for multi-core speed boost

## v2.0.0 — Ecosystem & Usability (Q3 2026)
Wider adoption push.

- Official `vstar-py` wrapper on PyPI (thin HTTP client + pandas integration)
- Simple plugin system for custom cashflow logic
- Lightweight dashboard example (pure Go templates + HTMX or separate Streamlit/Gradio)
- Public benchmark repo or GitHub Pages “try it live” (hosted `serve` instance)

---

**Guiding principles**  
- 1–3 weekends per version max  
- Zero external dependencies forever (std lib only)  
- Performance first: keep the 28M rows/sec streaming and sub-second Monte Carlo as selling points  
- Focus on what actuaries actually run daily: valuations, reserves, stochastic risk

