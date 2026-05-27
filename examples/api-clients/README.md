# v-star HTTP API — Examples

These are minimal examples showing how to call v-star's HTTP API from common
languages. Each example uses only the language's standard HTTP client — no
v-star-specific libraries needed.

Start the server:

```bash
v-star serve --port=8080
```

---

## Python

```python
import urllib.request, json

BASE = "http://localhost:8080"

def post(path, data):
    body = json.dumps(data).encode()
    req = urllib.request.Request(f"{BASE}{path}", data=body,
        headers={"Content-Type": "application/json"}, method="POST")
    with urllib.request.urlopen(req) as resp:
        return json.loads(resp.read())

# --- Present value ---
result = post("/value", {
    "interest_rate": 0.05,
    "records": [{"sum_assured": 100000, "term": 20}]
})
print(result["total_pv"], result["record_count"])

# --- Monte Carlo + risk metrics ---
result = post("/simulate", {
    "num_paths": 10000,
    "steps": 10,
    "initial_rate": 0.05,
    "drift": 0.02,
    "volatility": 0.15,
    "include_paths": False
})
print(result["var_95"], result["cte_95"])

# --- Annuity / NSP ---
result = post("/annuity", {
    "interest_rate": 0.05,
    "qxs": [0.001] * 111,          # qx from age 0..110
    "age": 30,
    "amount": 1000,
    "computation": "whole_life_immediate"
})
print(result["present_value"])

# --- Reserve ---
result = post("/reserve", {
    "interest_rate": 0.05,
    "qxs": [0.001] * 111,
    "age": 30,
    "term": 20,
    "sum_assured": 100000,
    "method": "net_premium"
})
print(result["reserve"])
```

---

## R

```r
library(httr)

base <- "http://localhost:8080"

# Present value
body <- list(
  interest_rate = 0.05,
  records = list(list(sum_assured = 100000, term = 20))
)
resp <- POST(paste0(base, "/value"), body = body, encode = "json")
print(content(resp))

# Monte Carlo
body <- list(
  num_paths = 10000, steps = 10,
  initial_rate = 0.05, drift = 0.02, volatility = 0.15
)
resp <- POST(paste0(base, "/simulate"), body = body, encode = "json")
print(content(resp))

# Annuity
body <- list(
  interest_rate = 0.05,
  qxs = rep(0.001, 111),
  age = 30,
  amount = 1000,
  computation = "whole_life_immediate"
)
resp <- POST(paste0(base, "/annuity"), body = body, encode = "json")
print(content(resp))

# Reserve
body <- list(
  interest_rate = 0.05,
  qxs = rep(0.001, 111),
  age = 30,
  term = 20,
  sum_assured = 100000,
  method = "net_premium"
)
resp <- POST(paste0(base, "/reserve"), body = body, encode = "json")
print(content(resp))
```

---

## JavaScript (Node.js / fetch)

```javascript
const BASE = "http://localhost:8080";

async function post(path, data) {
  const resp = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data)
  });
  return resp.json();
}

// Present value
post("/value", {
  interest_rate: 0.05,
  records: [{ sum_assured: 100000, term: 20 }]
}).then(r => console.log(r.total_pv, r.record_count));

// Monte Carlo
post("/simulate", {
  num_paths: 10000, steps: 10,
  initial_rate: 0.05, drift: 0.02, volatility: 0.15
}).then(r => console.log(r.var_95, r.cte_95));

// Annuity
post("/annuity", {
  interest_rate: 0.05,
  qxs: Array(111).fill(0.001),
  age: 30, amount: 1000,
  computation: "whole_life_immediate"
}).then(r => console.log(r.present_value));
```

---

## TypeScript

```typescript
const BASE = "http://localhost:8080";

async function post<T>(path: string, data: unknown): Promise<T> {
  const resp = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  return resp.json();
}

interface PVResponse {
  total_pv: number;
  record_count: number;
  processing_ms: number;
}

interface SimulateResponse {
  mean: number;
  std_dev: number;
  var_95: number;
  cte_95: number;
}

// Present value
const pv = await post<PVResponse>("/value", {
  interest_rate: 0.05,
  records: [{ sum_assured: 100000, term: 20 }],
});
console.log(pv.total_pv);

// Monte Carlo
const mc = await post<SimulateResponse>("/simulate", {
  num_paths: 10000, steps: 10,
  initial_rate: 0.05, drift: 0.02, volatility: 0.15,
});
console.log(mc.var_95, mc.cte_95);
```

---

## cURL

```bash
# Present value
curl -s -X POST http://localhost:8080/value \
  -H "Content-Type: application/json" \
  -d '{"interest_rate":0.05,"records":[{"sum_assured":100000,"term":20}]}' \
  | jq .

# Monte Carlo
curl -s -X POST http://localhost:8080/simulate \
  -H "Content-Type: application/json" \
  -d '{"num_paths":10000,"steps":10,"initial_rate":0.05,"drift":0.02,"volatility":0.15}' \
  | jq .

# Annuity
curl -s -X POST http://localhost:8080/annuity \
  -H "Content-Type: application/json" \
  -d '{"interest_rate":0.05,"qxs":[0.001],"age":30,"amount":1000,"computation":"whole_life_immediate"}' \
  | jq .

# Reserve
curl -s -X POST http://localhost:8080/reserve \
  -H "Content-Type: application/json" \
  -d '{"interest_rate":0.05,"qxs":[0.001],"age":30,"term":20,"sum_assured":100000,"method":"net_premium"}' \
  | jq .
```

---

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/value` | POST | Present value for batch records |
| `/simulate` | POST | Stochastic simulation (GBM / Vasicek) + VaR, CTE |
| `/annuity` | POST | Life-contingent annuity / net single premium |
| `/reserve` | POST | Policy reserve (net/gross/prospective/retrospective) |
