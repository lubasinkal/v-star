// Package reader provides CSV streaming and parsing with parallel processing.
//
// # Which function to use
//
// | You want to...                                  | Use this                          |
// |-------------------------------------------------|-----------------------------------|
// | Process actuarial census CSV row by row         | StreamCensus                     |
// | Process census CSV in batches (DB, API calls)   | StreamCensusChunked              |
// | Read a generic CSV (non-census) with strings    | StreamCSV                        |
// | Read a generic CSV with raw bytes (zero-alloc)  | StreamCSVRaw                     |
// | Inspect column headers before processing        | GetHeaders                       |
//
// # Quick start — process an actuarial census CSV
//
//	reader.StreamCensus("policies.csv", reader.CSVOptions{Header: true}, func(rec reader.CensusRecord) {
//	    fmt.Printf("Age: %d, Sum: %.2f\n", rec.Age, rec.SumAssured)
//	})
//
// # Total present value (just use StreamCensus + local accumulator)
//
//	totalPV := 0.0
//	count := 0
//	reader.StreamCensus("policies.csv", reader.CSVOptions{Header: true}, func(rec reader.CensusRecord) {
//	    totalPV += converter.PresentValue(rec.SumAssured, rec.Term)
//	    count++
//	})
//
// # Batch processing (database inserts, API batches)
//
//	sopts := reader.StreamOptions{
//	    CSVOptions: reader.CSVOptions{Header: true},
//	    ChunkSize:  10000,
//	    Workers:    8,
//	}
//	reader.StreamCensusChunked("policies.csv", sopts, func(chunk []reader.CensusRecord) error {
//	    return insertBatch(chunk)
//	})
//
// # Generic CSV (not actuarial census format)
//
//	reader.StreamCSV("data.csv", reader.CSVOptions{Header: true}, func(fields []string) {
//	    fmt.Println(fields[0], fields[1])
//	})
package reader
