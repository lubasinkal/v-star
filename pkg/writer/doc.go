// Package writer provides streaming JSON output for valuation results.
//
// # Write records as JSON
//
//	f, _ := os.Create("output.json")
//	jw := writer.NewJSONWriter(f)
//	defer jw.Close()
//
//	jw.WriteRecord(writer.JSONRecord{
//	    Age: 30, Sex: "M", PolicyType: "term",
//	    SumAssured: 100000, Term: 20, PresentValue: 37688.95,
//	})
//	jw.Close()
//
// # Stream JSON to stdout
//
//	records := []writer.JSONRecord{
//	    {Age: 30, SumAssured: 100000, PresentValue: 37688.95},
//	    {Age: 45, SumAssured: 200000, PresentValue: 78000.00},
//	}
//	writer.StreamJSON(records, os.Stdout)
//
// # Combine with CSV streaming for full pipeline
//
//	converter := rates.NewRateConverter(0.05)
//	jw := writer.NewJSONWriter(os.Stdout)
//	reader.StreamCensus("policies.csv", reader.CSVOptions{Header: true}, func(rec reader.CensusRecord) {
//	    pv := converter.PresentValue(rec.SumAssured, rec.Term)
//	    jw.WriteRecord(writer.JSONRecord{
//	        Age: rec.Age, SumAssured: rec.SumAssured, PresentValue: pv,
//	    })
//	})
//	jw.Close()
package writer
