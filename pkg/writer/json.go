package writer

import (
	"encoding/json"
	"fmt"
	"io"
)

// JSONRecord represents a record to be written as JSON
type JSONRecord struct {
	Age          int     `json:"age"`
	Sex          string  `json:"sex"`
	PolicyType   string  `json:"policy_type"`
	SumAssured   float64 `json:"sum_assured"`
	Term         int     `json:"term"`
	PresentValue float64 `json:"present_value"`
}

// JSONWriter streams JSON records to an io.Writer
type JSONWriter struct {
	writer io.Writer
	first  bool
}

// NewJSONWriter creates a new JSON writer
func NewJSONWriter(w io.Writer) *JSONWriter {
	return &JSONWriter{
		writer: w,
		first:  true,
	}
}

// WriteRecord writes a single JSON record
func (jw *JSONWriter) WriteRecord(record JSONRecord) error {
	if jw.first {
		jw.first = false
		fmt.Fprint(jw.writer, "[\n")
	} else {
		fmt.Fprint(jw.writer, ",\n")
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	_, err = jw.writer.Write(data)
	return err
}

// Close finalizes the JSON array
func (jw *JSONWriter) Close() error {
	if jw.first {
		// No records written
		fmt.Fprint(jw.writer, "[]")
	} else {
		fmt.Fprint(jw.writer, "\n]")
	}
	return nil
}

// StreamJSON writes records as a streaming JSON array
func StreamJSON(records []JSONRecord, w io.Writer) error {
	jw := NewJSONWriter(w)
	for _, record := range records {
		if err := jw.WriteRecord(record); err != nil {
			return err
		}
	}
	return jw.Close()
}
