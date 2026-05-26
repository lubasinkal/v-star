package writer

import "io"

// Record is a valuation record with both json and csv struct tags.
type Record struct {
	Sex          string  `json:"sex" csv:"sex"`
	PolicyType   string  `json:"policy_type" csv:"policy_type"`
	Age          int     `json:"age" csv:"age"`
	SumAssured   float64 `json:"sum_assured" csv:"sum_assured"`
	Term         int     `json:"term" csv:"term"`
	PresentValue float64 `json:"present_value" csv:"present_value"`
}

// JSONRecord and CSVRecord are type aliases for backward compatibility.
type JSONRecord = Record
type CSVRecord = Record

// RecordWriter defines an interface for writing valuation records.
// Implementations include JSON and CSV formats.
type RecordWriter interface {
	WriteRecord(Record) error
	Close() error
}

// recordWriter wraps a format-specific writer implementation.
type recordWriter struct {
	writeRecordFn func(Record) error
	closeFn       func() error
}

func (w *recordWriter) WriteRecord(r Record) error { return w.writeRecordFn(r) }
func (w *recordWriter) Close() error               { return w.closeFn() }

// NewRecordWriter creates a RecordWriter for the given format.
// Supported formats: "json", "csv".
func NewRecordWriter(w io.Writer, format string) *recordWriter {
	switch format {
	case "csv":
		cw := NewCSVWriter(w)
		return &recordWriter{
			writeRecordFn: cw.WriteRecord,
			closeFn:       cw.Close,
		}
	default:
		jw := NewJSONWriter(w)
		return &recordWriter{
			writeRecordFn: jw.WriteRecord,
			closeFn:       jw.Close,
		}
	}
}

// compile-time check: *recordWriter satisfies RecordWriter
var _ RecordWriter = (*recordWriter)(nil)
