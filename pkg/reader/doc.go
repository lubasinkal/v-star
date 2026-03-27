// Package reader provides CSV streaming and parsing with parallel processing.
//
// # Quick Start
//
// Stream a CSV file and process each row:
//
//	reader.StreamCSV("data.csv", reader.CSVOptions{Header: true}, func(fields []string) {
//	    fmt.Println(fields[0], fields[1]) // first two columns
//	})
//
// # Calculate the average of a column
//
//	sum := 0.0
//	count := 0
//	reader.StreamCSV("policies.csv", reader.CSVOptions{Header: true}, func(fields []string) {
//	    val, err := strconv.ParseFloat(fields[3], 64) // 4th column
//	    if err == nil {
//	        sum += val
//	        count++
//	    }
//	})
//	fmt.Printf("Average: %.2f\n", sum/float64(count))
//
// # Stream with zero allocations (faster for large files)
//
//	reader.StreamCSVRaw("big.csv", reader.CSVOptions{Header: true}, func(fields [][]byte) {
//	    // fields are raw byte slices — no string allocation
//	    // convert with string(fields[0]) if needed
//	})
//
// # Parse actuarial census records directly
//
//	reader.StreamCensus("policies.csv", reader.CSVOptions{Header: true}, func(rec reader.CensusRecord) {
//	    fmt.Printf("Age: %d, Sum: %.2f\n", rec.Age, rec.SumAssured)
//	})
//
// # Calculate total present value from a CSV
//
//	converter := rates.NewRateConverter(0.05)
//	opts := reader.CSVOptions{Header: true}
//	totalPV, count := reader.StreamCSVWithPV("policies.csv", opts, converter.PresentValue)
//	fmt.Printf("Total PV: %.2f from %d records\n", totalPV, count)
//
// # Parallel chunked processing
//
//	sopts := reader.StreamOptions{
//	    CSVOptions: reader.CSVOptions{Header: true},
//	    ChunkSize:  10000,
//	    Workers:    8,
//	}
//	reader.StreamCensusChunked("policies.csv", sopts, func(chunk []reader.CensusRecord) error {
//	    // process 10,000 records per chunk, in parallel across 8 goroutines
//	    return nil
//	})
//
// # Output as JSON
//
//	jw := writer.NewJSONWriter(os.Stdout)
//	reader.StreamCensus("policies.csv", reader.CSVOptions{Header: true}, func(rec reader.CensusRecord) {
//	    pv := converter.PresentValue(rec.SumAssured, rec.Term)
//	    jw.WriteRecord(writer.JSONRecord{
//	        Age: rec.Age, SumAssured: rec.SumAssured, PresentValue: pv,
//	    })
//	})
//	jw.Close()
package reader
