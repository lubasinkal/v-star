# v-star HTTP API — Examples

These are standalone, runnable scripts showing how to call v-star's HTTP API
from common languages. Each uses only standard HTTP libraries for that language.

Start the server:

```bash
v-star serve --port=8080
```

Then run any example:

```bash
# Python (requires: pip install requests)
python3 examples/api-clients/example.py

# JavaScript (requires: Node.js 18+)
node examples/api-clients/example.js

# cURL (requires: curl + jq)
bash examples/api-clients/curl.sh
```

---

## What each script does

All scripts exercise 4 endpoints and print results:

1. **Present value** — `POST /value`
2. **Monte Carlo + risk metrics** — `POST /simulate`
3. **Life annuity** — `POST /annuity`
4. **Policy reserve** — `POST /reserve`

---

## Scripts

| File | Language | HTTP client |
|------|----------|-------------|
| [example.py](./example.py) | Python | `requests` |
| [example.js](./example.js) | JavaScript | `fetch` (built-in in Node 18+) |
| [curl.sh](./curl.sh) | Bash | `curl` + `jq` |

---

## From other languages

The API is plain JSON over HTTP — any language with an HTTP client works:

**R** (httr):
```r
library(httr)
resp <- POST("http://localhost:8080/value",
  body = '{"interest_rate":0.05,"records":[{"sum_assured":100000,"term":20}]}',
  encode = "raw", content_type_json())
content(resp)
```

**TypeScript** (Deno / Bun):
```typescript
const resp = await fetch("http://localhost:8080/value", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({ interest_rate: 0.05, records: [{ sum_assured: 100000, term: 20 }] }),
});
const data = await resp.json();
console.log(data);
```

---

## API Reference

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/value` | POST | Present value for batch records |
| `/simulate` | POST | Stochastic simulation (GBM / Vasicek) |
| `/annuity` | POST | Life-contingent annuity / net single premium |
| `/reserve` | POST | Policy reserve (net/gross/prospective/retrospective) |
