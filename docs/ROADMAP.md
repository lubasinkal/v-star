# v-star ($v^*$) Development Roadmap

This document outlines the phased feature plan for the **v-star** engine, focusing on high-frequency actuarial calculations with zero external dependencies.

## Phase 1: The Core Math (Current)
* [x] **Rate Conversion Module:** Standardize $i \to v$ and the $v^*$ growth logic.
* [x] **Basic CLI:** Argument parsing for manual testing.
* [x] **Comprehensive Test Suite:** 100% code coverage on core actuarial formulas using `go test`.

## Phase 2: Mass Concurrency (Performance Benchmark)
*Goal: Prove that v-star can outperform traditional tools (Excel/R) by utilizing all CPU cores.*
* [x] **Stochastic Rate Engine:** A module that generates 100,000+ interest rate paths using `math/rand/v2`.
* [ ] **The Worker Pool:** Implement a goroutine pool to process 1,000,000 valuations in parallel (structure implemented).
* [x] **Internal Benchmarking Tool:** A sub-command `./v-star bench` that reports:
    * Execution time (ms)
    * Allocated memory (MB)
    * Throughput (Policies per second)

## Phase 3: Data Ingestion & I/O
*Goal: Handle real-world actuarial datasets without memory bloat.*
* [x] **Zero-Alloc CSV Parser:** Use `bufio` and `encoding/csv` to stream census data line-by-line.
* [x] **JSON Export:** Standard library implementation to pipe results into web frontends (like your Nuxt projects).

## Phase 4: The Read Tool
*Goal: Beat traditional valuation tools (Excel/Python/R) with extreme speed.*

### 4.1 Core Features
* [x] **High-Performance CSV Reader:** Zero-allocation streaming parser
* [x] **CLI Command:** `./v-star read <filepath> [flags]`
* [x] **Supported Flags:**
    * `--benchmark` (enable performance metrics)
    * `--header=true` (treat first row as headers)
    * `--limit=N` (limit rows processed)
    * [x] `--output=console|json` (output format)

### 4.2 Performance Targets
| Task | Traditional Tool (Est.) | v-star Target | Actual |
| :--- | :--- | :--- | :--- |
| 1M Row CSV Parse | 5 - 30 Seconds | < 1 Second | ~0.7s |
| 1M Row Valuation | 2 - 5 Minutes | < 2 Seconds | ~0.32s (3.1M rows/sec) |
| 100k Row Valuation | 5 - 10 Seconds | < 1 Second | ~0.076s (1.3M rows/sec) |
| Memory Usage | 200MB+ | < 50MB | TBD |
| Throughput | 30k - 200k rows/sec | 1M+ rows/sec | ~3.1M rows/sec |

### 4.3 Data Models
* [x] **CensusRecord Struct:** Age, Sex, PolicyType, SumAssured, Term
* [x] **Validation:** Data type validation and error handling

### 4.4 Output Options
* [x] **Console Output:** Pretty-printed table format
* [x] **JSON Export:** Streaming JSON writer for large files

### 4.5 Valuation Logic
* [x] **Simple PV Calculation:** Present value of sum assured using standard discount factor
* [x] **V-Star PV Calculation:** Present value using the v-star adjusted discount factor
* [x] **CLI Integration:** Automatic valuation during CSV processing

## 5 FrontEnd
* [ ] **Basic Frontend:** Create a frontend to visualise the work.

---

## Performance Targets

| Task | Traditional Tool (Est.) | v-star Target | Actual |
| :--- | :--- | :--- | :--- |
| 100k Monte Carlo Paths | 10 - 30 Seconds | < 500ms | ~1ms (100k paths/sec) |
| 1M Policy NPV Valuations | 2 - 5 Minutes | < 2 Seconds | ~0.32s (3.1M rows/sec) |
| 1M Row CSV Parse | 5 - 30 Seconds | < 1 Second | ~0.7s |
| Memory Footprint | 200MB+ | < 50MB | TBD |


