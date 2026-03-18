import csv
import time
import sys


def main():
    if len(sys.argv) < 2:
        print("Usage: python benchmark.py <csv_file>")
        sys.exit(1)

    filepath = sys.argv[1]

    total_sum = 0
    count = 0

    start = time.time()

    with open(filepath, "r") as f:
        reader = csv.reader(f)
        next(reader)  # Skip header

        for row in reader:
            count += 1
            sum_assured = float(row[3])
            total_sum += sum_assured

            if count <= 5:
                print(
                    f"{count}: Age={row[0]}, Sex={row[1]}, Type={row[2]}, Sum={row[3]}, Term={row[4]}"
                )

    duration = time.time() - start

    print(f"\n=== Benchmark Results ===")
    print(f"Total rows: {count}")
    print(f"Duration: {duration:.4f} seconds")
    print(f"Throughput: {count / duration:.0f} rows/sec")
    print(f"Total Sum Assured: {total_sum:.2f}")


if __name__ == "__main__":
    main()
