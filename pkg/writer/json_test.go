package writer

import (
	"bytes"
	"strings"
	"testing"
)

func TestStreamJSON_EmptySlice(t *testing.T) {
	var buf bytes.Buffer
	err := StreamJSON(nil, &buf)
	if err != nil {
		t.Fatalf("StreamJSON(nil) error: %v", err)
	}
	if got := buf.String(); got != "[]" {
		t.Errorf("StreamJSON(nil) = %q, want %q", got, "[]")
	}
}

func TestStreamJSON_SingleRecord(t *testing.T) {
	records := []JSONRecord{
		{Age: 30, Sex: "M", PolicyType: "term", SumAssured: 100000, Term: 20, PresentValue: 37688.95},
	}
	var buf bytes.Buffer
	err := StreamJSON(records, &buf)
	if err != nil {
		t.Fatalf("StreamJSON error: %v", err)
	}

	got := buf.String()
	if !strings.HasPrefix(got, "[") || !strings.HasSuffix(got, "]") {
		t.Errorf("StreamJSON output = %q, want JSON array", got)
	}
	if !strings.Contains(got, `"age":30`) {
		t.Errorf("StreamJSON output missing age field: %s", got)
	}
	if !strings.Contains(got, `"present_value":37688.95`) {
		t.Errorf("StreamJSON output missing present_value field: %s", got)
	}
}

func TestJSONWriter_MultipleRecords(t *testing.T) {
	var buf bytes.Buffer
	jw := NewJSONWriter(&buf)

	err := jw.WriteRecord(JSONRecord{Age: 30, SumAssured: 100000, PresentValue: 37688.95})
	if err != nil {
		t.Fatalf("WriteRecord error: %v", err)
	}

	err = jw.WriteRecord(JSONRecord{Age: 45, SumAssured: 200000, PresentValue: 78000.00})
	if err != nil {
		t.Fatalf("WriteRecord error: %v", err)
	}

	err = jw.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}

	got := buf.String()
	if !strings.HasPrefix(got, "[\n") {
		t.Errorf("JSONWriter output should start with '[\\n', got: %q", got[:5])
	}
	if !strings.HasSuffix(got, "\n]") {
		t.Errorf("JSONWriter output should end with '\\n]', got: %q", got[len(got)-5:])
	}
	if !strings.Contains(got, ",\n") {
		t.Errorf("JSONWriter should separate records with ',\\n'")
	}
}

func TestJSONWriter_EmptyClose(t *testing.T) {
	var buf bytes.Buffer
	jw := NewJSONWriter(&buf)

	err := jw.Close()
	if err != nil {
		t.Fatalf("Close error: %v", err)
	}

	if got := buf.String(); got != "[]" {
		t.Errorf("JSONWriter.Close() with no records = %q, want %q", got, "[]")
	}
}
