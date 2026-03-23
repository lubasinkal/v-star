# v-star: Roadmap to a High-Performance Open-Source Actuarial Engine

**v-star** is a high-performance, zero-dependency Go library + CLI for fast actuarial calculations: present value of policies (standard & v* discount factors), streaming CSV processing of large policy datasets, and Monte Carlo interest rate simulations.

**Current Status:** v0.2.0 - Production ready for basic actuarial valuations

Goal: evolve it into a **reusable, performant actuarial calculation engine** that actuaries, researchers, and small/medium insurers can embed in their pipelines — fast, transparent, auditable, and free of vendor lock-in.

## Performance Targets Achieved ✅

| Metric | Target | Achieved |
|--------|--------|----------|
| CSV Parsing | 1M+ rows/sec | 11M rows/sec |
| PV Calculation | 1M calcs/sec | 10M calcs/sec |
| Monte Carlo | 100k paths/sec | 100k paths/sec |

## Core Philosophy & Differentiation

- **Zero external dependencies** — Pure stdlib Go, maximum portability
- **Blazing speed** — 11M+ rows/sec via goroutines + streaming
- **Auditability first** — Deterministic, clear math, reproducible with seeds
- **Focus** — Life/annuity valuation + stochastic interest modeling
- **Not trying to be Prophet/MG-ALFA** — Fast prototyping, research, education, big CSV processing

## Current Features

### ✅ CSV Streaming
- [x] **Generic CSV Parser** (`StreamCSV`) — Parse any CSV format
- [x] **Parallel CSV Parser** (`StreamCSVParallel`) — Multi-core CSV processing  
- [x] **Actuarial CSV Parser** (`StreamCensus`) — Direct CensusRecord parsing
- [x] **PV Streaming** (`StreamCSVWithPV`) — Read + Calculate in one pass
- [x] **Header Detection** — Automatic column order detection
- [x] **Custom Delimiters** — Support for tab, pipe, etc.

### ✅ Actuarial Calculations
- [x] **Present Value** — Standard v^term discount factor
- [x] **V-Star Discount** — Custom (1+j)*v factor
- [x] **Rate Conversion** — Nominal to effective rates
- [x] **Annuity Calculations** — Term life, whole life, endowment
- [x] **Policy Reserves** — Net premium, gross premium reserves
- [x] **Mortality Tables** — CSV-based table loading

### ✅ Monte Carlo
- [x] **GBM Interest Paths** — Geometric Brownian Motion simulation
- [x] **Configurable Drift/Volatility** — Flexible parameterization
- [x] **Parallel Generation** — Multi-core path generation

## Roadmap

### Phase 1: Polish & Documentation (Next 2-4 weeks)

- [ ] Add **MIT License** — Essential for adoption
- [ ] Create **pkg/reader/doc.go** — Package documentation with examples
- [ ] Add **Example Tests** — Go doc examples for all exported functions
- [ ] Improve **Error Messages** — Include line numbers, sample data
- [ ] Add **CSV Validation** — `ValidateCSV()` function to check structure
- [ ] **Performance Documentation** — Benchmark results, tuning tips

### Phase 2: API Improvements (1-2 months)

- [ ] Add **Result Structs** — Clean return types for all functions
  ```go
  type PVResult struct {
      TotalPV     float64
      RecordCount int
      Duration    time.Duration
  }
  ```
- [ ] Add **Progress Callback** — Optional progress reporting
  ```go
  type CSVOptions struct {
      Progress func(processed, total int)
  }
  ```
- [ ] Add **Context Support** — Graceful cancellation
  ```go
  StreamCSV(ctx context.Context, path string, opts CSVOptions, fn func(...))
  ```

### Phase 3: Modeling Extensions (3-6 months)

- [ ] **Interface-based Interest Models**
  ```go
  type InterestModel interface {
      GeneratePath(steps int, seed int64) []float64
      Drift() float64
      Volatility() float64
  }
  ```
- [ ] **Additional Stochastic Models** — Vasicek, CIR, Hull-White
- [ ] **Risk Measures** — VaR, CTE/TVaR, Expected Shortfall on MC output
- [ ] **Cashflow Projection** — Beyond single PV, yearly projections

### Phase 4: Ecosystem (6-12 months)

- [ ] **CLI Config Files** — TOML/JSON for repeated runs
- [ ] **Output Formats** — Arrow/Parquet for big data pipelines
- [ ] **Python Bindings** — cgo wrapper for lifelib compatibility
- [ ] **R Bindings** — For actuarial research community

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- How to add a new model
- Coding style (effective Go + small functions)
- Benchmark requirements
- Pull request process

## Inspiration

| Project | Language | Lessons |
|---------|----------|---------|
| [lifelib](https://github.com/f2py/lifelib) | Python | Modular models, Jupyter examples |
| [chainladder-python](https://github.com/casact/chainladder-python) | Python | Community, tests, docs |
| [pyliferisk](https://github.com/pyliferisk/pyliferisk) | Python | Clean life contingencies math |
| [actuary](https://github.com/vondoy/epolymorphic-actuary) | Elixir | Functional approach |

## Success Metrics

- [ ] 50+ GitHub stars
- [ ] 5+ forks
- [ ] 2-3 external contributors
- [ ] Used in a real project or research paper
- [ ] Someone reports: "10× faster than my Python script"

## Getting Started

```bash
# Install
go get github.com/lubasinkal/v-star

# Build CLI
go build -o v-star ./cmd/v-star

# Run benchmark
./v-star bench

# Or use as library
go run examples/quickstart.go
```

## Next Immediate Actions

1. Add MIT License
2. Create pkg/reader/doc.go
3. Add CSV validation function
4. Tag v0.2.0 release
