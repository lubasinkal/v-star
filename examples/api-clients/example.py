#!/usr/bin/env python3
"""v-star API example — present value, simulation, annuity, reserve.

Usage:
    pip install requests
    python example.py

Requires v-star server running on http://localhost:8080.
"""

import requests

BASE = "http://localhost:8080"


def post(path, data):
    resp = requests.post(f"{BASE}{path}", json=data)
    resp.raise_for_status()
    return resp.json()


def main():
    # --- Present value ---
    result = post("/value", {
        "interest_rate": 0.05,
        "records": [{"sum_assured": 100000, "term": 20}],
    })
    print(f"PV: {result['total_pv']:.2f} ({result['record_count']} policies)")

    # --- Monte Carlo simulation + risk metrics ---
    result = post("/simulate", {
        "num_paths": 10000,
        "steps": 10,
        "initial_rate": 0.05,
        "drift": 0.02,
        "volatility": 0.15,
    })
    print(f"MC: VaR95={result['var_95']:.4f}, CTE95={result['cte_95']:.4f}")

    # --- Annuity / NSP ---
    qxs = [0.001] * 111  # qx from age 0..110
    result = post("/annuity", {
        "interest_rate": 0.05,
        "qxs": qxs,
        "age": 30,
        "amount": 1000,
        "computation": "whole_life_immediate",
    })
    print(f"Ann: PV={result['present_value']:.2f}")

    # --- Reserve ---
    result = post("/reserve", {
        "interest_rate": 0.05,
        "qxs": qxs,
        "age": 30,
        "term": 20,
        "sum_assured": 100000,
        "method": "net_premium",
    })
    print(f"Res: reserve={result['reserve']:.2f}")


if __name__ == "__main__":
    main()
