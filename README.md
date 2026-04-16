# v-star

**Fast actuarial calculations in your pocket.**

v-star is a lightning-fast actuarial engine that processes millions of policy records in seconds. Whether you're an actuary, analyst, or developer — if you need present values, reserves, or risk metrics, v-star delivers.

![CI](https://github.com/lubasinkal/v-star/actions/workflows/ci.yml/badge.svg)
![License](https://img.shields.io/badge/License-MIT-green)

## Why v-star?

| Problem | v-star Solution |
|---------|-----------------|
| Excel crashes on 1M+ rows | Handles 10M+ rows without blinking |
| Python/Pandas slow for big CSVs | 28M rows/sec — ~1.5x faster than Polars |
| VBA scripts are a mess | Clean, auditable Go code you can actually read |
| Black-box libraries | Every formula visible in the source |

**Zero dependencies.** No pip installs, no version conflicts. Just Go.

## What Can It Do?

- **Present Value** — Standard and v* discount factors (the v-star stuff from actuarial exams)
- **Annuities** — Whole life, term, deferred — with real mortality tables
- **Reserves** — Net premium, gross premium, prospective, retrospective
- **Monte Carlo** — 100k+ interest rate paths in under a second
- **Risk Measures** — VaR, CTE (Expected Shortfall) — the stuff risk managers actually need
- **Big CSV Processing** — Stream millions of rows without loading everything into RAM

## Quick Comparison

| Tool | 10M Rows (PV calc) | Memory |
|------|-------------------|--------|
| **v-star** | 347ms | 349 MB → 0.2 MB |
| Polars | 535ms | ~500 MB |
| Pandas | ~30s | >2 GB |

v-star is **~1.5x faster** than Polars with **30% less memory**. And it's zero-dependency Go.

## For Actuarial Folks

You already know the math. Here's how v-star maps to what you do every day:

```
Excel: =PV(0.05, 20, 0, -100000)
v-star: converter.PresentValue(100000, 20)  // → 37688.95

Excel: NPV + mortality tables (messy)
v-star: ann.WholeLifeImmediate(65, 1000)   // clean, one line

Excel: 50-line VBA for reserves
v-star: reserves.NetPremiumReserve(policy, converter, mort)  // 1 line
```

## For Developers

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

## For Everyone Else

No code? No problem. Use the CLI:

```bash
# Build
go build -o v-star ./cmd/v-star

# Calculate discount factors
./v-star -i 0.05 -j 0.02

# Process a policy CSV
./v-star read policies.csv --benchmark

# Run Monte Carlo simulation
./v-star montecarlo --paths=100000 --steps=10
```

## Installation

```bash
go get github.com/lubasinkal/v-star
```

Or grab the latest release from GitHub — single binary, runs anywhere.

## Run Examples

```bash
go run ./examples/quickstart              # PV and duration
go run ./examples/monte_carlo_risk       # Monte Carlo + VaR
go run ./examples/csv_valuation           # Big CSV processing

# Python users: Jupyter notebook demo
cd examples/python_bridge && jupyter notebook demo.ipynb
```

## Who's It For?

- **Actuaries** tired of slow Excel/VBA
- **Analysts** who need to process big census files fast
- **Developers** building insurance/risk systems
- **Students** learning actuarial science (the code is readable!)

## Roadmap

- v0.4.0 — HTTP API (call v-star from Python, R, Excel, anywhere)
- v1.0.0 — Locked API, production-ready
- v1.1.0 — Markov models, credibility theory

See [ROADMAP.md](./ROADMAP.md) for the full plan.

## License

MIT — see [LICENSE](LICENSE). Use it however you want.