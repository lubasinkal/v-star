import pandas as pd
import time
import sys


def main():
    if len(sys.argv) < 2:
        print("Usage: python benchmark_pandas.py <csv_file>")
        sys.exit(1)

    filepath = sys.argv[1]

    start = time.time()

    df = pd.read_csv(filepath)

    duration = time.time() - start

    total_sum = df["sum_assured"].sum()
    count = len(df)

    print("First 5 rows:")
    print(df.head())

    print(f"\n=== Benchmark Results (pandas) ===")
    print(f"Total rows: {count}")
    print(f"Duration: {duration:.4f} seconds")
    print(f"Throughput: {count / duration:.0f} rows/sec")
    print(f"Total Sum Assured: {total_sum:.2f}")


if __name__ == "__main__":
    main()
