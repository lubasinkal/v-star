package writer

import "io"

// Record is a valuation record with both json and csv struct tags.
// JSONRecord and CSVRecord are type aliases for backward compatibility.
type Record struct {
	Sex          string  `json:"sex" csv:"sex"`
	PolicyType   string  `json:"policy_type" csv:"policy_type"`
	Age          int     `json:"age" csv:"age"`
	SumAssured   float64 `json:"sum_assured" csv:"sum_assumed"`
	Term         int     `json:"term" csv:"term"`
	PresentValue float64 `json:"present_value" csv:"present_value"`
}

// RecordWriter writes valuation records to an underlying io.Writer.
// Both JSONWriter and CSVWriter implement this interface.
type RecordWriter interface {
	WriteRecord(Record) error
	Close() error
}

// NewRecordWriter creates a RecordWriter for the given format.
// Supported formats: "json", "csv".
func NewRecordWriter(w io.Writer, format string) RecordWriter {
	switch format {
	case "csv":
		return NewCSVWriter(w)
	default:
		return NewJSONWriter(w)
	}
}
