package writer

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

// CSVRecord represents a record to be written as CSV
type CSVRecord struct {
	Sex          string  `json:"sex"`
	PolicyType   string  `json:"policy_type"`
	Age          int     `json:"age"`
	SumAssured   float64 `json:"sum_assured"`
	Term         int     `json:"term"`
	PresentValue float64 `json:"present_value"`
}

// CSVWriter streams CSV records to an io.Writer
type CSVWriter struct {
	writer      *csv.Writer
	wroteHeader bool
}

// NewCSVWriter creates a new CSV writer
func NewCSVWriter(w io.Writer) *CSVWriter {
	return &CSVWriter{
		writer:      csv.NewWriter(w),
		wroteHeader: false,
	}
}

// WriteHeader writes the CSV header row
func (cw *CSVWriter) WriteHeader() error {
	if cw.wroteHeader {
		return nil
	}
	err := cw.writer.Write([]string{"sex", "policy_type", "age", "sum_assured", "term", "present_value"})
	if err != nil {
		return err
	}
	cw.wroteHeader = true
	return nil
}

// WriteRecord writes a single CSV record
func (cw *CSVWriter) WriteRecord(record CSVRecord) error {
	if !cw.wroteHeader {
		if err := cw.WriteHeader(); err != nil {
			return err
		}
	}
	return cw.writer.Write([]string{
		record.Sex,
		record.PolicyType,
		strconv.Itoa(record.Age),
		fmt.Sprintf("%.2f", record.SumAssured),
		strconv.Itoa(record.Term),
		fmt.Sprintf("%.2f", record.PresentValue),
	})
}

// Close flushes the CSV writer
func (cw *CSVWriter) Close() error {
	cw.writer.Flush()
	return cw.writer.Error()
}

// StreamCSV writes records as a CSV
func StreamCSV(records []CSVRecord, w io.Writer) error {
	cw := NewCSVWriter(w)
	if err := cw.WriteHeader(); err != nil {
		return err
	}
	for _, record := range records {
		if err := cw.WriteRecord(record); err != nil {
			return err
		}
	}
	return cw.Close()
}
