# API Freeze Audit — v0.8.0

Audit of all exported symbols across `pkg/`. Goal: identify anything that would be a
breaking change to fix after the v0.8.0 tag.

## ✅ pkg/rates — Stable
`DiscountFactor`, `RateConverter`, `NewRateConverter`, all conversion functions.
No changes needed.

## ✅ pkg/risk — Stable
`VaR`, `CTE`, `ComputeReport`, `RiskReport`. Clean API, all used. No changes.

## ✅ pkg/concurrency — Stable
`WorkerPool[T any]`, `NewWorkerPool`. Generic, idiomatic. No changes.

## ✅ pkg/stochastic — Stable
`PathGenerator`, `RateGenerator`, `VasicekGenerator`, `RatePath`, constructors.
All clean.

## ✅ pkg/reserves — Stable
`PolicySpec`, `NetPremiumReserve`, `GrossPremiumReserve`, `ProspectiveReserve`,
`RetrospectiveReserve`. Consistent naming. No changes.

## ✅ pkg/annuities — Stable
`AnnuityCalculator`, `NewAnnuityCalculator`, `ContingencyCalculator`,
`ApproxWholeLifeImmediate`. No changes.

## ✅ pkg/profit — New in v0.8.0
`Assumptions`, `Policy`, `Results`, `Run`. No breaking changes yet.

## ✅ pkg/server/middleware — Stable
`Middleware`, `CreateStack`, `CORS`, `Logging`, `Cache`, `ConcurrencyLimiter`.
All clean.

## ⚠️ pkg/server — Request/response types
`PVRequest`, `PVResponse`, `PVRecord`, `SimulateRequest`, `SimulateResponse`,
`AnnuityRequest`, `AnnuityResponse`, `ReserveRequest`, `ReserveResponse`, `Server`, `New`.
All exported but only used internally. Keep as-is — they document the HTTP API contract
and are needed for `go doc` rendering. No changes.

## ⚠️ pkg/writer — Redundant type aliases
`JSONRecord = Record`, `CSVRecord = Record`. Zero-cost type aliases for documentation.
Used internally in csv.go and json.go signatures. **Keep** — they add clarity at no cost.

## ⚠️ pkg/mortality — StreamCSV is exported but unused
`StreamCSV(filepath, func(age int, qx float64) error)` — exported, documented, but never
called outside the package. Same name as `reader.StreamCSV` with a different signature,
which could confuse. **Recommendation: keep** — it serves a distinct use case (streaming
age/qx pairs without building a Table). The package prefix disambiguates.

## 🛠️ pkg/reader — ParseStats should be unexported
`ParseStats` (`RowsRead int`, `RowsSkipped int`) is exported but **never referenced**
anywhere in the codebase except its declaration. It's leftover from a past refactor.
**Action: un-export** before the freeze.

---

## Summary

| Action | Count |
|--------|-------|
| ✅ Stable, no changes needed | 8 packages |
| ⚠️ Inspected, no action | 3 packages (server, writer, mortality) |
| 🛠️ **Un-export `ParseStats`** | 1 change |

**One change needed**: make `ParseStats` lowercase in `pkg/reader/csv.go`.
