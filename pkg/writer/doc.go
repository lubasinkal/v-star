// Package writer provides streaming output for valuation results.
// Supports JSON, CSV, and text report formats.
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
// # Write records as CSV
//
//	f, _ := os.Create("output.csv")
//	cw := writer.NewCSVWriter(f)
//	cw.WriteHeader()
//	cw.WriteRecord(writer.CSVRecord{
//	    Age: 30, Sex: "M", PolicyType: "term",
//	    SumAssured: 100000, Term: 20, PresentValue: 37688.95,
//	})
//	cw.Close()
//
// # Stream CSV to stdout
//
//	records := []writer.CSVRecord{
//	    {Age: 30, SumAssured: 100000, PresentValue: 37688.95},
//	    {Age: 45, SumAssured: 200000, PresentValue: 78000.00},
//	}
//	writer.StreamCSV(records, os.Stdout)
//
// # Generate text report
//
//	data := writer.ReportData{
//	    Title: "Valuation Report",
//	    InterestRate: 0.05,
//	    RecordCount: 100,
//	    TotalPresentValue: 5000000,
//	}
//	writer.StreamTextReport(data, os.Stdout)
//
// # Combine with CSV streaming for full pipeline
//
//	converter := rates.NewRateConverter(0.05)
//	cw := writer.NewCSVWriter(os.Stdout)
//	cw.WriteHeader()
//	reader.StreamCensus("policies.csv", reader.CSVOptions{Header: true}, func(rec reader.CensusRecord) {
//	    pv := converter.PresentValue(rec.SumAssured, rec.Term)
//	    cw.WriteRecord(writer.CSVRecord{
//	        Age: rec.Age, SumAssured: rec.SumAssured, PresentValue: pv,
//	    })
//	})
//	cw.Close()
package writer
