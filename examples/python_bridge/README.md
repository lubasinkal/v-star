# vstar-py

Python wrapper for [v-star](https://github.com/lubasinkal/v-star) — high-performance zero-dependency actuarial engine.

## Installation

```bash
# Basic (no pandas)
pip install vstar-py

# With pandas support
pip install vstar-py[pandas]

# Full (pandas + requests for HTTP)
pip install vstar-py[all]
```

## Quick Start

### Local Mode (no server needed)

```python
import pandas as pd
from vstar_py import VStarLocal

# Create engine
engine = VStarLocal()

# Single calculation
pv = engine.present_value(100000, 20)
print(f"PV: {pv:,.2f}")  # PV: 37,688.95

# DataFrame with pandas
df = pd.read_csv("policies.csv")
df = engine.present_value_df(df)
print(df.head())
```

### HTTP Mode (requires server)

```python
from vstar_py import VStar

# Connect to server (start with: v-star serve --port=8080)
engine = VStar("http://localhost:8080")

# Present value
result = engine.present_value([
    {"sum_assured": 100000, "term": 20, "age": 30}
])
print(f"Total PV: {result['total_pv']}")

# Monte Carlo with risk metrics
result = engine.risk_report(num_paths=100000)
print(f"VaR 95%: {result['var_95']}")
print(f"CTE 95%: {result['cte_95']}")

# Convert rate
result = engine.convert_rate(0.05, "effective", 12)
print(f"Nominal: {result['nominal_rate']}")
```

### Using Data Files

```python
import pandas as pd
from vstar_py import VStar, VStarLocal

# CSV file with policies
df = pd.read_csv("policies.csv")
print(df.head())
#     age  sex  policy_type  sum_assured  term
# 0    30   M          term     100000     20
# 1    35   F        whole     250000     10

# Local mode (pure Python, no server)
engine = VStarLocal()
df = engine.present_value_df(df, sum_col="sum_assured", term_col="term")
print(df)
#     age  sex  policy_type  sum_assured  term  present_value
# 0    30   M          term     100000     20       37689.50
# 1    35   F        whole     250000     10       95146.88

# Or use server for larger datasets
engine = VStar()
df = engine.value_batch(df, sum_col="sum_assured", term_col="term", age_col="age")
```

## API Reference

### `VStarLocal()` - Local mode (no server)

Pure Python calculations, no v-star server needed.

```python
engine = VStarLocal(rate=0.05)  # Default 5% interest

# Single PV
engine.present_value(100000, 20)

# DataFrame
engine.present_value_df(df, sum_col="sum_assured", term_col="term")

# Annuity
engine.annuity_immediate(20)  # term annuity immediate
engine.annuity_due(20)        # term annuity due

# Discount factor
engine.discount_factor(10)  # v^10
```

### `VStar(url)` - HTTP mode

Requires v-star server running.

```python
engine = VStar("http://localhost:8080")

# Present value
engine.present_value(records, interest_rate=0.05)

# DataFrame with server
engine.present_value_df(df, sum_col="sum_col", term_col="term_col")

# Monte Carlo
engine.monte_carlo_df(num_paths=100000)
engine.risk_report(num_paths=100000)

# Batch processing
engine.value_batch(df, sum_col="sum", term_col="term")
```

### `VStarCLI()` - CLI mode

Uses v-star CLI binary directly.

```python
cli = VStarCLI()
cli.monte_carlo(paths=100000, steps=10)
cli.read_csv("policies.csv", benchmark=True)
```

## Starting the Server

```bash
# Install v-star
go install github.com/lubasinkal/v-star/cmd/v-star

# Start server
v-star serve --port=8080
```

Or with Python:

```python
from vstar_py import VStarCLI
proc = cli.serve(port=8080)
# ...
proc.terminate()
```

## Requirements

- Python 3.9+
- (Optional) pandas for DataFrame support

## Examples

See `demo.ipynb` for Jupyter notebook examples.

## License

MIT — see [v-star LICENSE](https://github.com/lubasinkal/v-star/blob/main/LICENSE).