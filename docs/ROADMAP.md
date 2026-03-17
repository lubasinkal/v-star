# v-star ($v^*$) Development Roadmap

This document outlines the phased feature plan for the **v-star** engine, focusing on high-frequency actuarial calculations with zero external dependencies.

## Phase 1: The Core Math (Current)
* [x] **Rate Conversion Module:** Standardize $i \to v$ and the $v^*$ growth logic.
* [x] **Basic CLI:** Argument parsing for manual testing.
* [ ] **Comprehensive Test Suite:** 100% code coverage on core actuarial formulas using `go test`.

## Phase 2: Mass Concurrency (Performance Benchmark)
*Goal: Prove that v-star can outperform traditional tools (Excel/R) by utilizing all CPU cores.*
* [ ] **Stochastic Rate Engine:** A module that generates 100,000+ interest rate paths using `math/rand/v2`.
* [ ] **The Worker Pool:** Implement a goroutine pool to process 1,000,000 valuations in parallel.
* [ ] **Internal Benchmarking Tool:** A sub-command `./vstar bench` that reports:
    * Execution time (ms)
    * Allocated memory (MB)
    * Throughput (Policies per second)

## Phase 3: Data Ingestion & I/O
*Goal: Handle real-world actuarial datasets without memory bloat.*
* [ ] **Zero-Alloc CSV Parser:** Use `bufio` and `encoding/csv` to stream census data line-by-line.
* [ ] **JSON Export:** Standard library implementation to pipe results into web frontends (like your Nuxt projects).

## Phase 4: Financial Extensions
* [ ] **Annuity Schedules:** Generate full amortization or payout schedules.
* [ ] **Black-Scholes Module:** Option pricing for equity-linked insurance products.

---

## Performance Targets

| Task | Traditional Tool (Est.) | v-star Target |
| :--- | :--- | :--- |
| 100k Monte Carlo Paths | 10 - 30 Seconds | < 500ms |
| 1M Policy NPV Valuations | 2 - 5 Minutes | < 2 Seconds |
| Memory Footprint | 200MB+ | < 20MB |


