// Package reader provides CSV streaming and parsing with parallel processing.
// Supports memory-mapped files for zero-copy parsing and io.Reader
// streams for piped or HTTP-sourced data.
//
// # Which function to use
//
// | You want to...                                | Use this                 |
// |-----------------------------------------------|--------------------------|
// | Process actuarial census CSV row by row       | StreamCensus             |
// | Process census CSV in batches                 | StreamCensusChunked      |
// | Read a generic CSV with strings               | StreamCSV                |
// | Read a generic CSV with raw bytes             | StreamCSVRaw             |
// | Stream from any io.Reader                     | StreamCensusFromReader   |
// | Inspect column headers before processing      | GetHeaders               |
// | Abstract over census source type              | CensusSource interface   |
// |   — in-memory records                         | NewSliceCensusSource     |
// |   — CSV file                                  | NewFileCensusSource      |
//
// # Quick start — process an actuarial census CSV
//
//	reader.StreamCensus("policies.csv", reader.CSVOptions{Header: true},
//	    func(rec reader.CensusRecord) {
//	        fmt.Printf("Age: %d, Sum: %.2f\n", rec.Age, rec.SumAssured)
//	    })
//
// # Total present value (just use StreamCensus + local accumulator)
//
//	totalPV := 0.0
//	reader.StreamCensus("policies.csv", reader.CSVOptions{Header: true},
//	    func(rec reader.CensusRecord) {
//	        totalPV += converter.PresentValue(rec.SumAssured, rec.Term)
//	    })
//
// # Batch processing (database inserts, API batches)
//
//	sopts := reader.StreamOptions{
//	    CSVOptions: reader.CSVOptions{Header: true},
//	    ChunkSize:  10000,
//	    Workers:    8,
//	}
//	reader.StreamCensusChunked("policies.csv", sopts,
//	    func(chunk []reader.CensusRecord) error {
//	        return insertBatch(chunk)
//	    })
//
// # Stream from any io.Reader
//
//	reader.StreamCensusFromReader(os.Stdin, reader.CSVOptions{Header: true},
//	    func(rec reader.CensusRecord) {
//	        fmt.Println(rec.Age, rec.SumAssured)
//	    })
//
// # Generic CSV (not actuarial census format)
//
//	reader.StreamCSV("data.csv", reader.CSVOptions{Header: true},
//	    func(fields []string) {
//	        fmt.Println(fields[0], fields[1])
//	    })
package reader
