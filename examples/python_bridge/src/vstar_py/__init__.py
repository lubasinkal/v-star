"""vstar-py: Python wrapper for v-star actuarial engine."""

import json
import os
import platform
import subprocess
import sys
import urllib.request
import urllib.error
from pathlib import Path
from typing import Any

__version__ = "0.5.0"
VSTAR_VERSION = "0.5.0"
DEFAULT_BASE_URL = "http://localhost:8080"


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
    """Ensure v-star binary exists, prompt user if not."""
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


class VStar:
    """HTTP API client for v-star server."""

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

    def discount_factors(self, rate: float = 0.05, growth: float = 0.02) -> dict:
        """Calculate discount factors."""
        result = subprocess.run(
            [str(self.binary), "-i", str(rate), "-j", str(growth)],
            capture_output=True, text=True, check=True,
        )
        return {"output": result.stdout}

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
        """Read CSV file and calculate valuations."""
        cmd = [str(self.binary), "read", filepath, f"--interest={interest}", f"--output={output}"]
        if benchmark:
            cmd.append("--benchmark")
        if table:
            cmd.append(f"--table={table}")

        result = subprocess.run(cmd, capture_output=True, text=True, check=True)
        return result.stdout

    def serve(self, port: int = 8080) -> subprocess.Popen:
        """Start v-star server (returns process handle)."""
        return subprocess.Popen([str(self.binary), "serve", "--port", str(port)])


def generate_monte_carlo_paths(
    num_paths: int = 10000,
    steps: int = 10,
    drift: float = 0.02,
    volatility: float = 0.15,
    seed: int = 42,
) -> str:
    """Standalone function for CLI Monte Carlo."""
    cli = VStarCLI()
    return cli.monte_carlo(paths=num_paths, steps=steps, drift=drift, volatility=volatility, seed=seed)


__all__ = ["VStar", "VStarCLI", "get_binary_path", "generate_monte_carlo_paths"]