# v-star HTTP API — Use from Any Language

v-star's HTTP server is the universal interface. Any language that can make HTTP requests
(Python, R, JavaScript, Excel VBA, Julia, Rust, etc.) can use the full actuarial engine.

## Quick Start

```bash
# Start the server
v-star serve --port=8080
```

## Python Example

The `vstar.py` module in this directory is a ready-to-use Python client:

```python
from vstar import VStar

engine = VStar("http://localhost:8080")

# Present value
result = engine.present_value([
    {"sum_assured": 100000, "term": 20}
])
print(f"Total PV: {result['total_pv']}")

# Monte Carlo with risk metrics
result = engine.monte_carlo(
    num_paths=100000, steps=10,
    initial_rate=0.05, drift=0.02, volatility=0.15
)
print(f"VaR 95%: {result['var_95']}")

# Rate conversion
result = engine.convert_rate(0.05, "effective", 12)
print(f"Nominal rate: {result['nominal_rate']}")
```

## R Example

```r
library(httr)

# Present value
body <- list(
  interest_rate = 0.05,
  records = list(list(sum_assured = 100000, term = 20))
)
resp <- POST("http://localhost:8080/value", body = body, encode = "json")
content(resp)

# Monte Carlo
body <- list(
  num_paths = 100000, steps = 10,
  initial_rate = 0.05, drift = 0.02, volatility = 0.15
)
resp <- POST("http://localhost:8080/montecarlo", body = body, encode = "json")
content(resp)
```

## JavaScript Example

```javascript
// Present value
const resp = await fetch("http://localhost:8080/value", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    interest_rate: 0.05,
    records: [{ sum_assured: 100000, term: 20 }]
  })
});
console.log(await resp.json());

// Monte Carlo
const mcResp = await fetch("http://localhost:8080/montecarlo", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    num_paths: 100000, steps: 10,
    initial_rate: 0.05, drift: 0.02, volatility: 0.15
  })
});
console.log(await mcResp.json());
```

## cURL

```bash
# Present value
curl -X POST http://localhost:8080/value \
  -H "Content-Type: application/json" \
  -d '{"interest_rate":0.05,"records":[{"sum_assured":100000,"term":20}]}'

# Monte Carlo
curl -X POST http://localhost:8080/montecarlo \
  -H "Content-Type: application/json" \
  -d '{"num_paths":100000,"steps":10,"initial_rate":0.05}'
```

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/value` | POST | Present value calculation (batch records) |
| `/simulate` | POST | Stochastic simulation — GBM or Vasicek + risk metrics (VaR, CTE) |
| `/annuity` | POST | Life-contingent annuity and net single premium calculations |
| `/reserve` | POST | Policy reserve calculations (net/gross/premium/prospective/retrospective) |

## See Also

- `vstar.py` — Full Python HTTP client with pandas integration
- `demo.ipynb` — Jupyter notebook with usage examples
- [Main README](https://github.com/lubasinkal/v-star#readme) — Full documentation
