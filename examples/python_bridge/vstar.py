"""
v-star Python bridge - calls the v-star Go CLI from Python.

Usage:
    from vstar import VStar
    engine = VStar("./v-star.exe")
    result = engine.monte_carlo(paths=100000, steps=10, drift=0.02, volatility=0.15)
    print(result)
"""

import subprocess
import json
import os
from pathlib import Path


class VStar:
    """Python wrapper for the v-star Go CLI."""

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
        return {"output": result.stdout.strip()}

    def monte_carlo(
        self,
        paths: int = 100000,
        steps: int = 10,
        drift: float = 0.02,
        volatility: float = 0.15,
        seed: int = -1,
        rate: float = 0.05,
    ) -> str:
        """Run Monte Carlo simulation."""
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
        """Read a CSV file and calculate valuations."""
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


def generate_monte_carlo_paths(
    binary_path: str = "./v-star.exe",
    num_paths: int = 10000,
    steps: int = 10,
    drift: float = 0.02,
    volatility: float = 0.15,
    seed: int = 42,
) -> str:
    """Standalone function to generate Monte Carlo paths."""
    engine = VStar(binary_path)
    return engine.monte_carlo(
        paths=num_paths, steps=steps, drift=drift, volatility=volatility, seed=seed
    )
