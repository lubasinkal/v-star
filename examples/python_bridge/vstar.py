"""
v-star Python bridge - HTTP API client for v-star server.

Usage:
    from vstar import VStar
    engine = VStar("http://localhost:8080")
    result = engine.present_value([{"sum_assured": 100000, "term": 20}])
    print(result)
"""

import json
import subprocess
from pathlib import Path
from typing import Any


class VStar:
    """HTTP API client for v-star server."""

    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url.rstrip("/")

    def _request(self, endpoint: str, data: dict = None) -> dict:
        import urllib.request
        import urllib.error

        url = f"{self.base_url}{endpoint}"
        headers = {"Content-Type": "application/json"}

        if data is not None:
            body = json.dumps(data).encode("utf-8")
            req = urllib.request.Request(url, data=body, headers=headers, method="POST")
        else:
            req = urllib.request.Request(url, headers=headers)

        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read().decode("utf-8"))
        except urllib.error.HTTPError as e:
            raise RuntimeError(f"HTTP {e.code}: {e.read().decode()}")

    def health(self) -> dict:
        """Check server health."""
        return self._request("/health")

    def present_value(
        self,
        records: list[dict],
        interest_rate: float = 0.05,
        rate_j: float = 0.0,
    ) -> dict:
        """
        Calculate present value for policy records.

        Args:
            records: List of {"sum_assured": float, "term": int, "age": int}
            interest_rate: Effective annual interest rate
            rate_j: Growth rate for v-star calculation

        Returns:
            {"total_pv": float, "record_count": int, "processing_ms": int}
        """
        data = {"interest_rate": interest_rate, "records": records}
        if rate_j > 0:
            data["rate_j"] = rate_j
        return self._request("/value", data)

    def simulate(
        self,
        model: str = "gbm",
        initial_rate: float = 0.05,
        drift: float = 0.02,
        volatility: float = 0.15,
        long_term_mean: float = 0.05,
        mean_reversion: float = 0.2,
        num_paths: int = 10000,
        steps: int = 10,
        dt: float = 1.0,
        seed: int = 0,
        include_paths: bool = False,
        num_workers: int = 0,
    ) -> dict:
        """
        Run stochastic simulation (GBM or Vasicek) with risk metrics.

        Args:
            model: "gbm" (geometric Brownian motion) or "vasicek"
            initial_rate: Starting interest rate
            drift: GBM drift parameter (μ)
            volatility: Volatility (σ)
            long_term_mean: Vasicek long-term mean (b)
            mean_reversion: Vasicek speed of mean reversion (a)
            num_paths: Number of simulation paths
            steps: Number of time steps per path
            dt: Time increment between steps
            seed: Random seed (0 = random)
            include_paths: Include full path data in response
            num_workers: Parallel workers (0 = auto)

        Returns:
            {"paths": [...], "mean": float, "std_dev": float,
             "var_95": float, "cte_95": float, "processing_ms": int}
        """
        data = {
            "model": model,
            "initial_rate": initial_rate,
            "drift": drift,
            "volatility": volatility,
            "num_paths": num_paths,
            "steps": steps,
            "dt": dt,
        }
        if model == "vasicek":
            data["long_term_mean"] = long_term_mean
            data["mean_reversion"] = mean_reversion
        if seed > 0:
            data["seed"] = seed
        if include_paths:
            data["include_paths"] = True
        if num_workers > 0:
            data["num_workers"] = num_workers
        return self._request("/simulate", data)

    def annuity(
        self,
        interest_rate: float,
        qxs: list[float],
        age: int,
        amount: float,
        computation: str,
        term: int = 0,
        deferment: int = 0,
    ) -> dict:
        """
        Compute life-contingent annuity or net single premium.

        Args:
            interest_rate: Effective annual interest rate
            qxs: Mortality rates (qx) indexed by age, starting at 0
            age: Attained age
            amount: Payment amount or sum assured
            computation: One of:
                whole_life_immediate, whole_life_due,
                term_immediate, term_due,
                deferred_whole_life, deferred_term,
                whole_life_nsp, term_nsp, endowment_nsp
            term: Policy term (required for term/deferred/endowment)
            deferment: Deferral period (for deferred computations)

        Returns:
            {"present_value": float, "processing_ms": int}
        """
        data = {
            "interest_rate": interest_rate,
            "qxs": qxs,
            "age": age,
            "amount": amount,
            "computation": computation,
        }
        if term > 0:
            data["term"] = term
        if deferment > 0:
            data["deferment"] = deferment
        return self._request("/annuity", data)

    def reserve(
        self,
        interest_rate: float,
        qxs: list[float],
        age: int,
        term: int,
        sum_assured: float,
        method: str,
        premium: float = 0.0,
        expenses: float = 0.0,
    ) -> dict:
        """
        Calculate policy reserve.

        Args:
            interest_rate: Effective annual interest rate
            qxs: Mortality rates (qx) indexed by age, starting at 0
            age: Attained age at issue
            term: Policy term
            sum_assured: Face amount of insurance
            method: One of: net_premium, gross_premium, prospective, retrospective
            premium: Annual premium (required for prospective/retrospective)
            expenses: Annual expenses (for gross_premium)

        Returns:
            {"reserve": float, "processing_ms": int}
        """
        data = {
            "interest_rate": interest_rate,
            "qxs": qxs,
            "age": age,
            "term": term,
            "sum_assured": sum_assured,
            "method": method,
        }
        if premium > 0:
            data["premium"] = premium
        if expenses > 0:
            data["expenses"] = expenses
        return self._request("/reserve", data)
