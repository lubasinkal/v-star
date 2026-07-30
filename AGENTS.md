# AGENTS.md – v-star

Zero-dependency actuarial engine. Go 1.26.1, stdlib only.

## Build
`go build ./...` | `go test ./...` | `go fmt ./... && go vet ./...` (required before commits)

## Layout
- `cmd/v-star/` — CLI entry | `cmd/generate/` — data generation
- `pkg/` — rates, stochastic, mortality, annuities, reserves, risk, reader, writer, concurrency
- `examples/` — quickstart, monte_carlo_risk, csv_valuation, python_bridge

## Conventions
- Go stdlib only. `go fmt` before commits. Table-driven tests, `1e-9` float tolerance.
- Doc comments on exported symbols. Document math formulas in comments.

## Graphify
- Knowledge graph at `graphify-out/`. Invoke with `skill: "graphify"`.
- After code edits: `graphify update .` (AST-only, no API cost).
