# Roadmap

v-star Version-by-Version Roadmap (April 2026 → 2027)

## Current State

As of your latest commits ~April 2026, you're at **v0.2.0** (tagged March 27, 2026) with a rock-solid foundation:

- Zero-deps Go engine
- Parallel CSV streaming (28 M rows/sec)
- Full actuarial core (rates, annuities, reserves, mortality, stochastic GBM, risk/VaR/CTE)
- Clean pkg/ structure, CLI, examples, Python bridge, CI, godoc stubs, and AGENTS.md

Recent unversioned polish (Cloudflare Pages site, CI tweaks, struct fixes) shows you're already iterating fast.

**Goal of this roadmap**: Go from "impressive personal tool" → "the default high-performance actuarial library for Go (and everyone else via API/Python)".

Every version is scoped to **1–3 weekends** of work so you can ship consistently while studying/working. We keep **zero external dependencies** as the north star. Everything builds directly on what you already have.

---

## v0.3.0 – "Polish & Professional"

**Target release: End of April 2026**

Make the project feel production-ready and attractive to first-time users.

### Must-accomplish

- Bump `go.mod` version + tag `v0.3.0`
- Full godoc coverage for every public type/function in `pkg/` (add package-level comments explaining actuarial notation, e.g. prospective vs retrospective reserves)
- Expand every example in `/examples/` into runnable `ExampleXXX` funcs that appear on pkg.go.dev
- Add a single "Library Quickstart" section in README.md with 5 copy-paste import examples (PV, annuity, reserves, Monte Carlo + VaR, streaming CSV)
- Update CHANGELOG.md + add "Unreleased" section at top
- Add GitHub release with auto-generated notes (via CI)
- Benchmark table in README: v-star vs Polars on 2M_test.csv and a new 10 M-row synthetic file (include memory, speed, and "identical results" note)

### Nice-to-have

- One-sentence "Why Go for actuaries?" blurb in README
- Update Cloudflare site to link directly to pkg.go.dev and new examples

### Why this version first?

It costs almost no new code but 3× the perceived quality. Recruiters and other devs will take it seriously immediately.

---

## v0.4.0 – "HTTP API & Accessibility"

**Target release: Mid-May 2026**

Unlock non-Go users (Excel, Python scripts, R, dashboards).

### Must-accomplish

- New `pkg/server` (pure `net/http` + JSON streaming, no frameworks)
  - `POST /value` → stream CSV + return PV/reserves
  - `POST /montecarlo` → return paths + VaR/CTE
  - `GET /mortality/{table}` and `POST /convert-rate`
- Add CLI flag `--serve` (or new `v-star serve` subcommand) that starts the server on :8080
- Add `examples/http-client` folder with curl + Python requests examples
- Zero-config deployment instructions (Fly.io free tier one-click + Railway)
- Update Python bridge to optionally call the HTTP endpoint instead of subprocess
- Add rate-limit + graceful shutdown (still zero deps)

### Nice-to-have

- OpenAPI spec (just a static JSON file in `/docs`)

This is the single highest-leverage addition — suddenly actuaries can call your engine from anywhere without learning Go.

---

## v1.0.0 – "Stable Library"

**Target release: Early June 2026**

Declare "this is ready for real use."

### Must-accomplish

- Lock public API (no breaking changes after this)
- Add semantic versioning policy in CONTRIBUTING.md
- Full test coverage for new server package + existing pkgs (aim 90%+)
- Comprehensive benchmarks for every major function (add to `cmd/v-star bench`)
- Update all godoc examples to use the new server where it makes sense
- Write the big blog post: "v-star v1.0: From VBA rage to 28 M rows/sec actuarial engine in Go" (cross-post to your personal site + Dev.to + LinkedIn)
- Add "Used by" section in README (even if it's just "used in my own reserving work" for now)

### Milestone

This is the version you can confidently show employers or post in actuarial communities.

---

## v1.1.0 – "New Actuarial Models"

**Target release: Late June 2026**

Expand the actual math you use at work.

### Must-accomplish (pick 2–3 based on your day-job pain)

- Multi-state Markov models (`pkg/markov`) — simple 2/3-state disability/termination (common in group life)
- Credibility theory (`pkg/credibility`) — Bühlmann & Bühlmann-Straub
- Percentiles + confidence intervals in risk package (you already flagged this)
- Ornstein-Uhlenbeck or CIR interest-rate models in `pkg/stochastic` (replace or sit alongside GBM)
- Add example datasets in `/testdata` for Botswana/SADC-style mortality tables (if public data available)

Keep everything generic and concurrent.

---

## v1.2.0 – "Variance Reduction & Performance"

**Target release: July 2026**

### Must-accomplish

- Antithetic variates + control variates in Monte Carlo (`pkg/stochastic`)
- Latin Hypercube sampling option
- Update benchmarks to show 2–5× variance reduction
- Optional CGO-free SIMD hints for CSV parser (still zero runtime deps)

---

## v2.0.0 – "Plugin System & Ecosystem"

**Target release: Q3/Q4 2026**

### Big-ticket items

- Simple plugin interface for user-defined cashflow scripts (JSON config or tiny embedded expressions)
- Official `vstar-py` PyPI package (thin wrapper over the binary or HTTP)
- Botswana insurance example dashboard using the HTTP API (tie-in with actuworry)
- Community features: issue templates, Discord/LinkedIn group invite in README

---

## Long-term (2027+)

- v2.1+: Integration with actuarial open-source standards (if any emerge)
- v3.0+: Optional tiny WASM build for browser-based actuarial tools
- Stretch: Official R bridge or Excel add-in (via HTTP)

---

## How to execute this roadmap

1. **Start today**: Create a new branch `roadmap-v0.3` and knock out v0.3.0 this weekend.
2. **Track progress**: Keep the "Unreleased" section in CHANGELOG.md updated as you go.
3. **Visibility rule**: Every minor version gets a short X/LinkedIn post + update to your personal site.
4. **Decision point after v1.0**: Re-evaluate based on stars, issues, or job feedback.

This roadmap is deliberately **your style**: small, focused, high-impact releases that keep the zero-dep speed obsession alive while turning v-star into something you can actually put on a CV or resume as "maintained open-source actuarial library."

Pick **v0.3.0** as your next milestone and tell me when you start — I can give you exact code sketches for the godoc comments, server package structure, or the benchmark table format.

What's your first target date for v0.3.0? Or do you want me to adjust any version's scope based on what you're feeling at work right now? Let's ship this thing.