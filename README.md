# v-star: The Actuarial Engine That Doesn't Suck

**Your calculations just got millions of times faster.**

Ever tried to run a valuation on a million policy census data in Excel even running reserving formulas? Watched it freeze, crash, or take hours? That's the problem v-star solves. Built in Go (yes, an actually fast language compared to R, Python or VBA), v-star handles massive datasets and calculations in milliseconds while your coffee is still hot.

![CI](https://github.com/lubasinkal/v-star/actions/workflows/ci.yml/badge.svg)
![License](https://img.shields.io/badge/License-MIT-green)

---

## The Story

I'm an actuarial science graduate who got tired of:

- Excel crashing on big files
- VBA scripts that nobody understands
- Python code that felt slow (still better than VBA)
- Propietary tools and libraries where you can't see the math
- Waiting to get accepted for a job

So I built v-star. Zero dependencies. All the actuarial stuff a gradute would think you'd need. Fast enough to make your laptop feel like a supercomputer.

Why call it **v-star ?** Comes from a joke we had in class: if premiums compound at rate j but you're discounting at i, the new discount factor is **v*** = (1+j) × v. The star marks the difference. 🎯

---

## What Can It Do?

| Feature | What it means for you |
|---------|----------------------|
| **Present Value** | Standard & v* discount factors — the core of everything |
| **Annuities** | Whole life, term, deferred — with real mortality tables |
| **Reserves** | Net premium, gross premium, prospective, retrospective |
| **Monte Carlo** | 100k interest rate paths in under a second |
| **Risk Measures** | VaR, CTE — what risk managers actually care about |
| **Big CSV Processing** | Stream millions of rows without blowing up your RAM |

---

## Speed Comparison (10M Rows)

| Tool | Time | Memory |
|------|------|--------|
| **v-star** | 347ms | 349 MB → 0.2 MB |
| Polars | 535ms | ~500 MB |
| Pandas | ~30s | >2 GB |

v-star is **~1.5x faster** than Polars and uses **30% less memory**. And it's zero-dependency Go — no pip install, no version hell.

---

## For Actuarial Students & Professionals

You already know the math. Here's how it maps to your Excel:

```excel
=PV(0.05, 20, 0, -100000)
```
```go
converter.PresentValue(100000, 20)  // → 37,688.95
```

```excel
NPV mess with mortality tables
```
```go
ann.WholeLifeImmediate(65, 1000)   // clean, one line
```

```excel
50 lines of VBA for reserves
```
```go
reserves.NetPremiumReserve(policy, converter, mort)  // 1 line
```

The formulas are exactly the same. Just faster, auditable, and you can actually read the code.

---

## For Developers, Programmers, Hobbyists

```go
// Present value — 28M calculations per second
converter := rates.NewRateConverter(0.05)
pv := converter.PresentValue(100000, 20)

// Annuity with mortality
mort, _ := mortality.LoadCSV("mortality.csv")
ann := annuities.NewAnnuityCalculator(converter, mort)
pv = ann.WholeLifeImmediate(65, 1000)

// Monte Carlo + VaR — 100k paths in <1 second
rg := stochastic.NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42)
paths := rg.GeneratePaths(100000, 10, 1.0)
report := risk.ComputeReport(losses)

// Stream a million-row CSV without loading into memory
opts := reader.CSVOptions{Header: true}
totalPV, count := reader.StreamCSVWithPV("policies.csv", opts, converter.PresentValue)
```

All standard library. No external dependencies. Just Go.

---

## For Everyone Else (CLI)

Don't want to write code? No problem:

```bash
# Build
go build -o v-star ./cmd/v-star

# Calculate discount factors
./v-star -i 0.05 -j 0.02

# Process a policy CSV (with benchmark)
./v-star read policies.csv --benchmark

# Run Monte Carlo
./v-star montecarlo --paths=100000 --steps=10
```

---

## Quick Start

```bash
# Install
go get github.com/lubasinkal/v-star

# Run examples
go run ./examples/quickstart              # PV and duration
go run ./examples/monte_carlo_risk       # Monte Carlo + VaR
go run ./examples/csv_valuation           # Big CSV processing
```

Python users: check out `examples/python_bridge/` for the Jupyter notebook demo. If you don see it then sorry I'll work on it.

---

## Why Go?

- **Speed** — Compiles to native code, no interpreter overhead
- **Zero deps** — Standard library only. No pip, no npm, no version conflicts
- **Readable** — Every formula is right there in the source. Audit-friendly
- **Concurrent** — Goroutines make parallelism easy
- **Portable** — One binary, runs anywhere

---

## Who's It For?

| Person | Why v-star |
|--------|------------|
| **Actuarial student** | Learn by reading the code. Fast calculations for assignments. |
| **Actuary** | Replace slow Excel/VBA. Process big censuses in seconds. |
| **Analyst** | Stream big CSVs without crashing. Get results, not errors. |
| **Developer** | Build insurance/risk tools without bloated dependencies. |
| **Risk manager** | Run Monte Carlo + VaR in production. Fast. |

---

## What's Coming Next?

- **v0.4.0** — HTTP API (call v-star from Python, R, Excel via HTTP)
- **v1.0.0** — Stable API, production-ready
- **v1.1.0** — Markov models, credibility theory

Full roadmap: [ROADMAP.md](./ROADMAP.md)

---

## Contribute

Found a bug? Want a feature? PRs welcome. See [CONTRIBUTING.md](./CONTRIBUTING.md).

---

## License

MIT — do whatever you want with it. See [LICENSE](./LICENSE).
