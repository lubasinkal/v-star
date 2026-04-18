"""vstar-py: Python wrapper for v-star actuarial engine."""

import json
import os
import platform
import subprocess
import sys
import urllib.request
import urllib.error
from pathlib import Path
from typing import Any, Union

__version__ = "0.5.0"
VSTAR_VERSION = "0.5.0"
DEFAULT_BASE_URL = "http://localhost:8080"

try:
    import pandas as pd
    HAS_PANDAS = True
except ImportError:
    HAS_PANDAS = False


def get_platform_binary_name() -> str:
    """Get binary name for current platform."""
    if platform.system() == "Windows":
        return "v-star.exe"
    return "v-star"


def get_binary_path() -> Path:
    """Get path to v-star binary, downloading if necessary."""
    binary_name = get_platform_binary_name()

    cache_dir = Path(os.path.expanduser("~")) / ".cache" / "vstar"
    binary_path = cache_dir / binary_name

    if binary_path.exists():
        return binary_path

    cache_dir.mkdir(parents=True, exist_ok=True)
    url = f"https://github.com/lubasinkal/v-star/releases/download/v{VSTAR_VERSION}/{binary_name}"

    print(f"Downloading v-star {VSTAR_VERSION}...", file=sys.stderr)
    try:
        urllib.request.urlretrieve(url, binary_path)
        if platform.system() != "Windows":
            os.chmod(binary_path, 0o755)
        return binary_path
    except Exception as e:
        raise RuntimeError(
            f"Failed to download v-star: {e}\n"
            "Install manually: https://github.com/lubasinkal/v-star/releases"
        ) from e


def ensure_binary() -> Path:
    """Ensure v-star binary exists."""
    binary_name = get_platform_binary_name()
    cache_dir = Path(os.path.expanduser("~")) / ".cache" / "vstar"
    binary_path = cache_dir / binary_name

    if binary_path.exists():
        return binary_path

    if platform.system() == "Windows":
        exe_path = Path("v-star.exe")
        if exe_path.exists():
            return exe_path
    else:
        exe_path = Path("v-star")
        if exe_path.exists():
            return exe_path

    return get_binary_path()


def _discount_factor(rate: float, periods: int = 1) -> float:
    """Calculate discount factor: v = 1/(1+i)^n"""
    return 1 / ((1 + rate) ** periods)


def _present_value_single(sum_assured: float, term: int, rate: float) -> float:
    """Calculate present value for a single policy."""
    if term <= 0:
        return sum_assured
    v = _discount_factor(rate)
    pv = 0.0
    for t in range(1, term + 1):
        pv += v ** t
    return sum_assured * pv


class VStar:
    """HTTP API client for v-star server.
    
    Usage:
        engine = VStar()  # Connect to local server
        # or:
        engine = VStar("http://server:8080")
    """

    def __init__(self, base_url: str = DEFAULT_BASE_URL):
        self.base_url = base_url.rstrip("/")

    def _request(self, endpoint: str, data: dict = None) -> dict:
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
            raise RuntimeError(f"HTTP {e.code}: {e.read().decode()}") from e

    def health(self) -> dict:
        """Check server health."""
        return self._request("/health")

    def present_value(self, records: list[dict], interest_rate: float = 0.05, rate_j: float = 0.0) -> dict:
        """Calculate present value for policy records."""
        data = {"interest_rate": interest_rate, "records": records}
        if rate_j > 0:
            data["rate_j"] = rate_j
        return self._request("/value", data)

    def present_value_df(
        self,
        df: "pd.DataFrame",
        sum_col: str = "sum_assured",
        term_col: str = "term",
        interest_rate: float = 0.05,
    ) -> "pd.DataFrame":
        """Calculate present value for a DataFrame of policies.
        
        Args:
            df: DataFrame with policy data
            sum_col: Column name for sum assured
            term_col: Column name for term
            interest_rate: Discount rate
            
        Returns:
            DataFrame with 'present_value' column added
            
        Example:
            >>> import pandas as pd
            >>> from vstar import VStar
            >>> df = pd.read_csv("policies.csv")
            >>> df = VStar().present_value_df(df)
            >>> print(df.head())
        """
        if not HAS_PANDAS:
            raise ImportError("pandas required: pip install vstar-py[pandas]")

        v = _discount_factor(interest_rate)

        def calc_pv(row):
            sa = row[sum_col]
            term = row[term_col]
            if term <= 0:
                return sa
            pv = 0.0
            for t in range(1, term + 1):
                pv += v ** t
            return sa * pv

        result = df.copy()
        result["present_value"] = result.apply(calc_pv, axis=1)
        return result

    def monte_carlo(
        self,
        num_paths: int = 100000,
        steps: int = 10,
        initial_rate: float = 0.05,
        drift: float = 0.02,
        volatility: float = 0.15,
        seed: int = 0,
    ) -> dict:
        """Run Monte Carlo simulation."""
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

    def monte_carlo_df(
        self,
        num_paths: int = 100000,
        steps: int = 10,
        initial_rate: float = 0.05,
        drift: float = 0.02,
        volatility: float = 0.15,
        seed: int = 0,
    ) -> "pd.DataFrame":
        """Run Monte Carlo and return paths as DataFrame.
        
        Args:
            num_paths: Number of simulation paths
            steps: Time steps per path
            initial_rate: Starting interest rate
            drift: Drift (mean) of the process
            volatility: Volatility (sigma)
            seed: Random seed for reproducibility
            
        Returns:
            DataFrame with columns for each path (or just final values)
            
        Example:
            >>> result = VStar().monte_carlo_df(num_paths=10000)
            >>> print(result.tail())
        """
        if not HAS_PANDAS:
            raise ImportError("pandas required: pip install vstar-py[pandas]")

        data = {
            "initial_rate": initial_rate,
            "drift": drift,
            "volatility": volatility,
            "num_paths": num_paths,
            "steps": steps,
        }
        if seed > 0:
            data["seed"] = seed
            
        response = self._request("/montecarlo", data)
        
        paths = response.get("paths", [])
        if not paths:
            raise ValueError("No paths returned")
            
        return pd.DataFrame(paths, columns=[f"path_{i}" for i in range(len(paths))])

    def risk_report(
        self,
        num_paths: int = 100000,
        steps: int = 10,
        initial_rate: float = 0.05,
        drift: float = 0.02,
        volatility: float = 0.15,
        seed: int = 0,
    ) -> dict:
        """Run Monte Carlo and get risk metrics (VaR, CTE).
        
        Returns:
            dict with var_95, cte_95, mean, std_dev
        """
        return self.monte_carlo(num_paths, steps, initial_rate, drift, volatility, seed)

    def value_batch(
        self,
        df: "pd.DataFrame",
        sum_col: str = "sum_assured",
        term_col: str = "term",
        age_col: str = "age",
        interest_rate: float = 0.05,
        mortality_table: str = None,
    ) -> "pd.DataFrame":
        """Process batch of policies with optional mortality.
        
        For large DataFrames, processes in chunks for efficiency.
        
        Args:
            df: DataFrame with policy data
            sum_col: Column name for sum assured
            term_col: Column name for term  
            age_col: Column name for age (used with mortality)
            interest_rate: Discount rate
            mortality_table: Optional mortality table name
            
        Returns:
            DataFrame with 'present_value' column added
        """
        if not HAS_PANDAS:
            raise ImportError("pandas required: pip install vstar-py[pandas]")

        result = df.copy()
        
        if mortality_table:
            records = df[[age_col, sum_col, term_col]].to_dict("records")
            response = self.present_value(
                records,
                interest_rate=interest_rate,
            )
            result["present_value"] = [r.get("total_pv", 0) for r in response.get("records", [])]
        else:
            result = self.present_value_df(
                result,
                sum_col=sum_col,
                term_col=term_col,
                interest_rate=interest_rate,
            )
            
        return result

    def convert_rate(self, from_rate: float, from_type: str = "effective", compounding: int = 1) -> dict:
        """Convert between nominal and effective rates."""
        data = {"from_rate": from_rate, "from_type": from_type, "compounding": compounding}
        return self._request("/convert-rate", data)

    def mortality(self, table: str) -> dict:
        """Get mortality table info."""
        url = f"{self.base_url}/mortality/{table}"
        req = urllib.request.Request(url)
        try:
            with urllib.request.urlopen(req) as resp:
                return json.loads(resp.read().decode("utf-8"))
        except urllib.error.HTTPError as e:
            raise RuntimeError(f"HTTP {e.code}: {e.read().decode()}") from e


class VStarLocal:
    """Local mode - no server needed, calculates directly in Python.
    
    Usage:
        engine = VStarLocal()  # Uses pure Python calculations
    """

    def __init__(self):
        self.rate = 0.05

    def present_value(self, sum_assured: float, term: int, rate: float = None) -> float:
        """Calculate present value for a single policy."""
        r = rate or self.rate
        return _present_value_single(sum_assured, term, r)

    def present_value_df(
        self,
        df: "pd.DataFrame",
        sum_col: str = "sum_assured",
        term_col: str = "term",
        rate: float = None,
    ) -> "pd.DataFrame":
        """Calculate present value for DataFrame.
        
        Example:
            >>> import pandas as pd
            >>> from vstar import VStarLocal
            >>> df = pd.read_csv("policies.csv")
            >>> df = VStarLocal().present_value_df(df)
        """
        if not HAS_PANDAS:
            raise ImportError("pandas required: pip install vstar-py[pandas]")

        r = rate or self.rate
        v = _discount_factor(r)

        def calc_pv(row):
            sa = row[sum_col]
            term = row[term_col]
            if term <= 0:
                return sa
            pv = 0.0
            for t in range(1, term + 1):
                pv += v ** t
            return sa * pv

        result = df.copy()
        result["present_value"] = result.apply(calc_pv, axis=1)
        return result

    def discount_factor(self, periods: int = 1) -> float:
        """Calculate v^n discount factor."""
        return _discount_factor(self.rate, periods)

    def annuity_immediate(self, term: int, rate: float = None) -> float:
        """Calculate term annuity immediate (ax_."""
        r = rate or self.rate
        if r == 0:
            return term
        v = _discount_factor(r)
        ax = (1 - v ** term) / r
        return ax

    def annuity_due(self, term: int, rate: float = None) -> float:
        """Calculate term annuity due (a¨x."""
        r = rate or self.rate
        if r == 0:
            return term
        v = _discount_factor(r)
        ax_dot = (1 - v ** term) / (1 - v)
        return ax_dot


class VStarCLI:
    """CLI wrapper calling v-star binary directly."""

    def __init__(self, binary_path: str = None):
        if binary_path:
            self.binary = Path(binary_path)
        else:
            self.binary = get_binary_path()

        if not self.binary.exists():
            raise FileNotFoundError(f"v-star binary not found. Install: pip install vstar-py --no-binary")

    @property
    def version(self) -> str:
        """Get v-star version."""
        result = subprocess.run([str(self.binary), "--version"], capture_output=True, text=True, check=True)
        return result.stdout.strip()

    def monte_carlo(self, paths: int = 100000, steps: int = 10, drift: float = 0.02, volatility: float = 0.15, seed: int = -1, rate: float = 0.05) -> str:
        """Run Monte Carlo via CLI."""
        cmd = [
            str(self.binary), "montecarlo",
            f"--paths={paths}", f"--steps={steps}",
            f"--drift={drift}", f"--volatility={volatility}",
            "-i", str(rate),
        ]
        if seed >= 0:
            cmd.append(f"--seed={seed}")
        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        return result.stdout

    def read_csv(self, filepath: str, interest: float = 0.05, output: str = "console", benchmark: bool = False, table: str = "") -> str:
        """Read CSV and calculate valuations."""
        cmd = [str(self.binary), "read", filepath, f"--interest={interest}", f"--output={output}"]
        if benchmark:
            cmd.append("--benchmark")
        if table:
            cmd.append(f"--table={table}")
        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        return result.stdout

    def serve(self, port: int = 8080) -> subprocess.Popen:
        """Start v-star server."""
        return subprocess.Popen([str(self.binary), "serve", "--port", str(port)])


def generate_monte_carlo_paths(num_paths: int = 10000, steps: int = 10, drift: float = 0.02, volatility: float = 0.15, seed: int = 42) -> str:
    """Standalone Monte Carlo function."""
    cli = VStarCLI()
    return cli.monte_carlo(paths=num_paths, steps=steps, drift=drift, volatility=volatility, seed=seed)


__all__ = [
    "VStar",
    "VStarLocal", 
    "VStarCLI", 
    "get_binary_path", 
    "generate_monte_carlo_paths",
    "HAS_PANDAS",
]