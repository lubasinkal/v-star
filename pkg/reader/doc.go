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
// # CensusSource — abstract over where census data comes from
//
// The same function can accept a CensusSource parameter and work with
// any source — CSV file, in-memory slice, HTTP body, or stdin:
//
//	func totalPV(src reader.CensusSource, rate float64) float64 {
//	    conv := rates.NewRateConverter(rate)
//	    var total float64
//	    src.Stream(func(r reader.CensusRecord) error {
//	        total += conv.PresentValue(r.SumAssured, r.Term)
//	        return nil
//	    })
//	    return total
//	}
//
//	// From a CSV file:
//	src := reader.NewFileCensusSource("policies.csv",
//	    reader.CSVOptions{Header: true})
//	fmt.Println(totalPV(src, 0.05))
//
//	// From an in-memory slice:
//	src = reader.NewSliceCensusSource([]reader.CensusRecord{
//	    {Age: 30, SumAssured: 100000, Term: 20},
//	})
//	fmt.Println(totalPV(src, 0.05))
//
//	// From an HTTP request body:
//	src = reader.NewReaderCensusSource(r.Body,
//	    reader.CSVOptions{Header: true})
//	fmt.Println(totalPV(src, 0.05))
//
// # ReaderCensusSource — from any io.Reader
//
// Useful for HTTP request bodies, stdin, bytes.Buffer, etc.
// Uses sequential scanning (no mmap), so prefer FileCensusSource
// for large files on disk.
//
//	reader.NewReaderCensusSource(os.Stdin, reader.CSVOptions{Header: true})
//	reader.NewReaderCensusSource(r.Body, reader.CSVOptions{Header: true})
//
// # Generic CSV (not actuarial census format)
//
//	reader.StreamCSV("data.csv", reader.CSVOptions{Header: true},
//	    func(fields []string) {
//	        fmt.Println(fields[0], fields[1])
//	    })
package reader
