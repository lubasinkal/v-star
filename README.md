# v-star: The Actuarial Engine That Doesn't Suck

**Your calculations just got millions of times faster.**

Ever tried to run a valuation on a million-policy census? Watched Excel freeze, crash, or take hours? v-star is the answer. Built in Go — an actually fast language compared to R, Python, or VBA — it handles massive datasets and calculations in milliseconds while your coffee is still hot.

![CI](https://github.com/lubasinkal/v-star/actions/workflows/ci.yml/badge.svg)
![License](https://img.shields.io/badge/License-MIT-green)

---

## The Story

Actuarial science grad. Tired of:

- Excel crashing on big files
- VBA scripts that nobody understands
- Python code that felt slow (still better than VBA)
- Proprietary tools where you can't see the math
- Waiting to get accepted for a job

So I built v-star. Zero dependencies. All the actuarial stuff a gradute would think you'd need. Fast enough to make your laptop feel like a supercomputer.

Why the name? Comes from a joke in class: if premiums compound at rate **j** but you're discounting at **i**, the new discount factor is **v\*** = (1+j) × v. The star marks the difference.

---

## What Can It Do?

| Feature | What it means for you |
|---------|----------------------|
| **Present Value** | Standard & v\* discount factors — the core of everything |
| **Annuities** | Whole life, term, deferred — with real mortality tables |
| **Reserves** | Net premium, gross premium, prospective, retrospective |
| **Monte Carlo** | GBM + Vasicek mean-reverting models. 100k paths in ~27ms |
| **Risk Measures** | VaR, CTE, Expected Shortfall, confidence intervals |
| **Multiple Decrements** | Combine death, lapse, disability into a single table |
| **Big CSV Streaming** | Stream millions of rows without blowing up your RAM |
| **HTTP API** | Call from Python, R, Excel via REST endpoints |
| **Zero Dependencies** | Standard library only. No pip, no npm, no version hell |

**New in v0.6.1:** Cleaned public API — removed redundant wrappers (StreamCensusWithPV, StreamCSVWithPV, ProcessBatch, ExpectedShortfall). Removed 8-core worker cap — uses all available cores. Added JSON tags to all model structs. Reserve functions now accept any DiscountFactor (no internal type assertions). Reader entry points documented with decision table.

---

## Speed

Benchmarked on an Intel i5-8250U laptop (1.6-3.4 GHz, 8 cores, NVMe SSD). Plugged in.

| Benchmark | Time | Throughput |
|-----------|------|------------|
| **CSV Parsing** (10M rows, 288 MB) | 0.80s | **12.6M rows/s** |
| **Present Value** (single call, direct) | 2.6 ns | **380M / second** |
| **Present Value** (single call, constructor) | 22.8 ns | **44M / second** |
| **Annuity** (whole life, 90 terms) | 512 ns | **2M / second** |
| **Monte Carlo** (100k paths, 10 steps) | 27 ms | **3.7M paths/sec** |
| **Risk Report** (VaR + CTE, 100k losses) | 0.68 ms | **147M losses/sec** |
| **Valuation** (10M policies, parallel) | 37 ms | **272M policies/sec** |

### CSV Comparison (10M Rows)

| Tool | Time | Memory |
|------|------|--------|
| **v-star** (mmap) | **0.80s** | **1.3 GB (OS page cache)** |
| **v-star** (streaming) | 1.12s | ~0.2 MB |
| Polars | ~535ms | ~500 MB |
| Pandas | ~30s | >2 GB |

*v-star uses memory-mapped I/O for zero-copy parsing. The mmap path uses OS page cache (lazily paged, released to OS under memory pressure). The streaming path keeps memory constant regardless of file size.*

### Monte Carlo (100k Paths)

| Tool | Time | VaR/CTE |
|------|------|---------|
| **v-star** | **27ms** | ✓ (with confidence intervals) |
| Python/Numpy | ~2s | ✓ |
| R | ~5s | ✓ |

---

## Quick Start

```bash
# Install
go get github.com/lubasinkal/v-star

# Run examples
go run ./examples/quickstart              # PV and duration
go run ./examples/monte_carlo_risk       # Monte Carlo + VaR
go run ./examples/csv_valuation           # Big CSV processing

# Build CLI
go build -o v-star ./cmd/v-star

# Process a policy CSV
./v-star read policies.csv --benchmark

# Run Monte Carlo
./v-star montecarlo --paths=100000 --steps=10

# Start HTTP server
./v-star serve
```

---

## Code Examples

### For Actuarial Students & Professionals

You already know the math:

```go
// Present value — like Excel's =PV(0.05, 20, 0, -100000)
converter := rates.NewRateConverter(0.05)
pv := converter.PresentValue(100000, 20)  // → 37,688.95

// Annuity with mortality table
mort, _ := mortality.LoadCSV("mortality.csv")
ann := annuities.NewAnnuityCalculator(converter, mort)
pv = ann.WholeLifeImmediate(65, 1000)    // one line instead of NPV mess

// Reserve calculation (50 lines of VBA → 1 line)
reserves.NetPremiumReserve(policy, converter, mort)

// Multiple decrements (death + lapse combined)
dt := mortality.NewDecrementTable([]*Table{death, lapse}, nil)
totalQx := dt.Qx(age)        // 1 - (1-qx_death)*(1-qx_lapse)
causeQx := dt.QxByCause(age, 0)  // approximate independent qx
```

### For Developers

```go
// Vasicek mean-reverting rates (instead of GBM)
vg := stochastic.NewVasicekGenerator(0.05, 0.04, 0.5, 0.02)
path := vg.GeneratePath(10, 1.0)

// Monte Carlo + VaR with confidence intervals
rg := stochastic.NewRateGeneratorWithSeed(0.05, 0.02, 0.15, 42)
paths := rg.GeneratePaths(100000, 10, 1.0)
report := risk.ComputeReport(losses)
fmt.Println(report.VaR95, report.CTE95)        // risk metrics
fmt.Println(report.Confidence95Lo, report.Confidence95Hi)  // ±1.96σ/√n

// Stream a million-row CSV without loading into memory
totalPV := 0.0
reader.StreamCensus("policies.csv", reader.CSVOptions{Header: true}, func(rec reader.CensusRecord) {
    totalPV += converter.PresentValue(rec.SumAssured, rec.Term)
})

// Generic parallel worker pool with context cancellation
wp := concurrency.NewWorkerPool(8, func(r reader.CensusRecord) float64 {
    return converter.PresentValue(r.SumAssured, r.Term)
})
totalPV := wp.ProcessBatch(records)
result, err := wp.ProcessBatchContext(ctx, records)
```

### CSV Reader — Which one to use

| Function | When to use |
|----------|-------------|
| `StreamCensus` | Process an actuarial census CSV row by row (auto-detects columns); accumulate your own metrics |
| `StreamCensusChunked` | Batch processing (database inserts, API calls) |
| `StreamCSV` | Generic CSV with string fields (non-standard column layout) |
| `StreamCSVRaw` | Generic CSV with zero-allocation byte slices |
| `GetHeaders` | Inspect column headers before deciding parsing strategy |

**All standard library. Zero external dependencies.**

---

## CLI

```bash
# Calculate discount factors
./v-star -i 0.05 -j 0.02

# Process a policy CSV
./v-star read policies.csv --benchmark
./v-star read policies.csv --table=mortality.csv --output=json
./v-star read policies.csv --output=csv --limit=10000
./v-star read policies.csv --output=report

# Run Monte Carlo
./v-star montecarlo --paths=100000 --steps=10 --seed=42

# Run benchmark suite
./v-star bench

# Start HTTP server
./v-star serve --port=8080
```

---

## HTTP API

Start the server:

```bash
./v-star serve --port=8080
```

All endpoints return JSON. The server includes CORS (cross-origin), request logging, and graceful shutdown on Ctrl+C.

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/value` | POST | Calculate present value for a batch of records |
| `/montecarlo` | POST | Run Monte Carlo simulation, get VaR/CTE |
| `/convert-rate` | POST | Convert between nominal and effective rates |
| `/mortality/{table}` | GET | Get mortality table metadata |
| `/export/csv` | POST | Export valuation records as CSV download |
| `/export/report` | POST | Export valuation as a formatted text report |
| `/upload/csv` | POST | Upload a CSV file for valuation (multipart form) |

### Examples

```bash
# Present value
curl -X POST http://localhost:8080/value \
  -H "Content-Type: application/json" \
  -d '{"interest_rate":0.05,"records":[{"sum_assured":100000,"term":20}]}'

# Monte Carlo
curl -X POST http://localhost:8080/montecarlo \
  -H "Content-Type: application/json" \
  -d '{"num_paths":100000,"steps":10,"initial_rate":0.05,"drift":0.02,"volatility":0.15,"include_paths":true}'

# Rate conversion
curl -X POST http://localhost:8080/convert-rate \
  -H "Content-Type: application/json" \
  -d '{"from_rate":0.05,"from_type":"effective","compounding":12}'

# Export CSV
curl -X POST http://localhost:8080/export/csv \
  -H "Content-Type: application/json" \
  -d '{"interest_rate":0.05,"records":[{"sum_assured":100000,"term":20}]}'

# Upload CSV file
curl -X POST http://localhost:8080/upload/csv \
  -F "file=@policies.csv" -F "rate=0.05"
```

### Python

```python
import requests

# Present value
resp = requests.post("http://localhost:8080/value", json={
    "interest_rate": 0.05,
    "records": [{"sum_assured": 100000, "term": 20}]
})
print(resp.json())

# Monte Carlo
resp = requests.post("http://localhost:8080/montecarlo", json={
    "num_paths": 100000, "steps": 10,
    "initial_rate": 0.05, "drift": 0.02, "volatility": 0.15
})
print(resp.json())  # {"var_95": ..., "cte_95": ...}

# Upload CSV
resp = requests.post("http://localhost:8080/upload/csv",
    files={"file": open("policies.csv", "rb")},
    data={"rate": "0.05"})
print(resp.text)
```

Or use `examples/python_bridge/vstar.py`:

```python
from vstar import VStar
engine = VStar("http://localhost:8080")
result = engine.present_value([{"sum_assured": 100000, "term": 20}])
```

---

## Why Go?

- **Speed** — Compiles to native code, no interpreter overhead. 380M PV calculations/sec
- **Zero deps** — Standard library only. No pip, no npm, no version conflicts
- **Readable** — Every formula is right there in the source. Audit-friendly
- **Concurrent** — Goroutines make parallelism trivial
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

- **v0.7.0** — Markov chain models (disability, multiple decrements), credibility theory (Bühlmann)
- **v0.8.0** — Variance reduction (antithetic variates, control variates, Latin Hypercube)
- **v1.0.0** — Stable API, production-ready, deployment docs (Docker, Fly.io)

Full roadmap: [ROADMAP.md](./ROADMAP.md)

---

## Contributing

PRs welcome. See [CONTRIBUTING.md](./CONTRIBUTING.md). Found a bug? [Open an issue](https://github.com/lubasinkal/v-star/issues).

---

## License

MIT — do whatever you want with it. See [LICENSE](./LICENSE).
