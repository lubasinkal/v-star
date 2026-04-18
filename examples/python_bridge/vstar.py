"""
v-star Python bridge - HTTP API client for v-star server.

Usage:
    from vstar import VStar
    engine = VStar("http://localhost:8080")
    result = engine.present_value([{"sum_assured": 100000, "term": 20}])
    print(result)

    # Or use the CLI subprocess wrapper:
    from vstar import VStarCLI
    cli = VStarCLI("./v-star.exe")
    result = cli.monte_carlo(paths=100000, steps=10)
"""

import json
import os
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
            {"total_pv": float, "record_count": int}
        """
        data = {
            "interest_rate": interest_rate,
            "records": records,
        }
        if rate_j > 0:
            data["rate_j"] = rate_j

        return self._request("/value", data)

    def monte_carlo(
        self,
        num_paths: int = 100000,
        steps: int = 10,
        initial_rate: float = 0.05,
        drift: float = 0.02,
        volatility: float = 0.15,
        seed: int = 0,
    ) -> dict:
        """
        Run Monte Carlo simulation.

        Args:
            num_paths: Number of paths to generate
            steps: Number of time steps
            initial_rate: Initial interest rate
            drift: Drift parameter
            volatility: Volatility
            seed: Random seed (0 for random)

        Returns:
            {"paths": [...], "mean": float, "var_95": float, "cte_95": float}
        """
        data = {
            "initial_rate": initial_rate,
            "drift": drift,
            "volatility": volatility,
            "num_paths": num_paths,
            "steps": steps,
        }
        if seed > 0:
            data["seed"] = seed

        return self._request("/montecarlo", data)

    def convert_rate(
        self,
        from_rate: float,
        from_type: str = "effective",
        compounding: int = 1,
    ) -> dict:
        """
        Convert between nominal and effective rates.

        Args:
            from_rate: Rate to convert
            from_type: "effective" or "nominal"
            compounding: Compounding periods (1=annual, 2=semi-annual, 4=quarterly, 12=monthly)

        Returns:
            {"effective_rate": float, "nominal_rate": float}
        """
        data = {
            "from_rate": from_rate,
            "from_type": from_type,
            "compounding": compounding,
        }
        return self._request("/convert-rate", data)

    def mortality(self, table: str) -> dict:
        """Get mortality table info."""
        import urllib.request
        import urllib.error

        url = f"{self.base_url}/mortality/{table}"
        req = urllib.request.Request(url)

        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read().decode("utf-8"))
        except urllib.error.HTTPError as e:
            raise RuntimeError(f"HTTP {e.code}: {e.read().decode()}")


class VStarCLI:
    """CLI wrapper calling v-star binary directly."""

    def __init__(self, binary_path: str = "./v-star.exe"):
        self.binary = Path(binary_path)
        if not self.binary.exists():
            raise FileNotFoundError(
                f"v-star binary not found at {binary_path}. "
                "Build it first: go build -o v-star.exe ./cmd/v-star"
            )

    def version(self) -> str:
        """Get v-star version."""
        result = subprocess.run(
            [str(self.binary), "--version"],
            capture_output=True,
            text=True,
            check=True,
        )
        return result.stdout.strip()

    def discount_factors(self, rate: float = 0.05, growth: float = 0.02) -> dict:
        """Calculate discount factors."""
        result = subprocess.run(
            [str(self.binary), "-i", str(rate), "-j", str(growth)],
            capture_output=True,
            text=True,
            check=True,
        )
        output = result.stdout
        return {"output": output}

    def monte_carlo(
        self,
        paths: int = 100000,
        steps: int = 10,
        drift: float = 0.02,
        volatility: float = 0.15,
        seed: int = -1,
        rate: float = 0.05,
    ) -> str:
        """Run Monte Carlo simulation via CLI."""
        cmd = [
            str(self.binary),
            "montecarlo",
            f"--paths={paths}",
            f"--steps={steps}",
            f"--drift={drift}",
            f"--volatility={volatility}",
            "-i",
            str(rate),
        ]
        if seed >= 0:
            cmd.append(f"--seed={seed}")

        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        return result.stdout

    def read_csv(
        self,
        filepath: str,
        interest: float = 0.05,
        output: str = "json",
        benchmark: bool = False,
        table: str = "",
    ) -> str:
        """Read CSV file and calculate valuations."""
        cmd = [
            str(self.binary),
            "read",
            filepath,
            f"--interest={interest}",
            f"--output={output}",
        ]
        if benchmark:
            cmd.append("--benchmark")
        if table:
            cmd.append(f"--table={table}")

        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        return result.stdout

    def serve(self, port: int = 8080) -> subprocess.Popen:
        """Start v-star server (returns process handle)."""
        return subprocess.Popen(
            [str(self.binary), "serve", "--port", str(port)],
        )


def generate_monte_carlo_paths(
    binary_path: str = "./v-star.exe",
    num_paths: int = 10000,
    steps: int = 10,
    drift: float = 0.02,
    volatility: float = 0.15,
    seed: int = 42,
) -> str:
    """Standalone function for CLI Monte Carlo."""
    cli = VStarCLI(binary_path)
    return cli.monte_carlo(
        paths=num_paths, steps=steps, drift=drift, volatility=volatility, seed=seed
    )