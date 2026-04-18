# vstar-py

Python wrapper for [v-star](https://github.com/lubasinkal/v-star) — high-performance zero-dependency actuarial engine.

## Installation

```bash
pip install vstar-py
```

## Quick Start

### HTTP API (requires running server)

```python
from vstar_py import VStar

# Connect to v-star server
engine = VStar("http://localhost:8080")

# Present value
result = engine.present_value([
    {"sum_assured": 100000, "term": 20, "age": 30}
])
print(f"Total PV: {result['total_pv']}")

# Monte Carlo
result = engine.monte_carlo(num_paths=100000, steps=10)
print(f"VaR 95%: {result['var_95']}, CTE 95%: {result['cte_95']}")

# Convert rate
result = engine.convert_rate(0.05, "effective", 12)  # monthly compounding
print(f"Nominal: {result['nominal_rate']}")
```

### CLI Mode (bundled binary)

```python
from vstar_py import VStarCLI, get_binary_path

# Auto-downloads v-star binary on first use
cli = VStarCLI()

# Or specify custom binary
cli = VStarCLI("/path/to/v-star.exe")

# Run Monte Carlo
output = cli.monte_carlo(paths=10000, steps=10, seed=42)
print(output)

# Read CSV
output = cli.read_csv("policies.csv", benchmark=True)
```

## Starting the Server

```bash
# Install v-star first
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
- (Optional) v-star binary for CLI mode

## License

MIT — see [v-star LICENSE](https://github.com/lubasinkal/v-star/blob/main/LICENSE).