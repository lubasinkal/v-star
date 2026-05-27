package reader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStreamCensus(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "age,sex,policy_type,sum_assured,term\n30,M,term,100000,20\n25,F,whole,50000,10\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var records []CensusRecord
	err := StreamCensus(tmpFile, CSVOptions{Header: true}, func(r CensusRecord) {
		records = append(records, r)
	})
	if err != nil {
		t.Fatalf("StreamCensus: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0].Age != 30 || records[0].SumAssured != 100000 || records[0].Sex != "M" {
		t.Errorf("first record: %+v", records[0])
	}
	if records[1].Age != 25 || records[1].SumAssured != 50000 || records[1].PolicyType != "whole" {
		t.Errorf("second record: %+v", records[1])
	}
}

func TestStreamCensus_NoHeader(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "30,M,term,100000,20\n25,F,whole,50000,10\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var records []CensusRecord
	err := StreamCensus(tmpFile, CSVOptions{Header: false}, func(r CensusRecord) {
		records = append(records, r)
	})
	if err != nil {
		t.Fatalf("StreamCensus: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
}

func TestStreamCensus_Limit(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "age,sex,policy_type,sum_assured,term\n30,M,term,100000,20\n25,F,whole,50000,10\n40,M,term,200000,15\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCensus(tmpFile, CSVOptions{Header: true, Limit: 2}, func(r CensusRecord) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCensus: %v", err)
	}
	if count != 2 {
		t.Errorf("got %d records, want 2", count)
	}
}

func TestStreamCensus_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.csv")
	if err := os.WriteFile(tmpFile, []byte("age,sex,policy_type,sum_assured,term\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var records []CensusRecord
	err := StreamCensus(tmpFile, CSVOptions{Header: true}, func(r CensusRecord) {
		records = append(records, r)
	})
	if err != nil {
		t.Fatalf("StreamCensus: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestStreamCensus_NotFound(t *testing.T) {
	err := StreamCensus("/nonexistent/file.csv", CSVOptions{Header: true}, func(r CensusRecord) {})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestStreamCensus_NonDefaultOrder(t *testing.T) {
	// Columns in different order: sum_assured,term,age,sex,policy_type
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "sum_assured,term,age,sex,policy_type\n100000,20,30,M,term\n50000,10,25,F,whole\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var records []CensusRecord
	err := StreamCensus(tmpFile, CSVOptions{Header: true}, func(r CensusRecord) {
		records = append(records, r)
	})
	if err != nil {
		t.Fatalf("StreamCensus: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0].Age != 30 || records[0].SumAssured != 100000 || records[0].Term != 20 {
		t.Errorf("first record: %+v", records[0])
	}
}

func TestStreamCensus_NonDefaultOrderNoSexType(t *testing.T) {
	// Minimal columns: age,term,sum_assured (no sex/policy_type)
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "age,term,sum_assured\n30,20,100000\n25,10,50000\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var records []CensusRecord
	err := StreamCensus(tmpFile, CSVOptions{Header: true}, func(r CensusRecord) {
		records = append(records, r)
	})
	if err != nil {
		t.Fatalf("StreamCensus: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
}

func TestStreamCensus_NonDefaultOrderWithSpaces(t *testing.T) {
	// Column names with spaces
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "age, term, sum_assured, sex, policy_type\n30,20,100000,M,term\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var records []CensusRecord
	err := StreamCensus(tmpFile, CSVOptions{Header: true}, func(r CensusRecord) {
		records = append(records, r)
	})
	if err != nil {
		t.Fatalf("StreamCensus: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Age != 30 || records[0].SumAssured != 100000 {
		t.Errorf("record: %+v", records[0])
	}
}

func TestStreamCensusFromReader(t *testing.T) {
	content := "age,sex,policy_type,sum_assured,term\n30,M,term,100000,20\n25,F,whole,50000,10\n"
	r := strings.NewReader(content)

	var records []CensusRecord
	err := StreamCensusFromReader(r, CSVOptions{Header: true}, func(rec CensusRecord) {
		records = append(records, rec)
	})
	if err != nil {
		t.Fatalf("StreamCensusFromReader: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	if records[0].Age != 30 || records[0].SumAssured != 100000 {
		t.Errorf("first record: %+v", records[0])
	}
}

func TestStreamCensusFromReader_NoHeader(t *testing.T) {
	content := "30,M,term,100000,20\n25,F,whole,50000,10\n"
	r := strings.NewReader(content)

	var records []CensusRecord
	err := StreamCensusFromReader(r, CSVOptions{Header: false}, func(rec CensusRecord) {
		records = append(records, rec)
	})
	if err != nil {
		t.Fatalf("StreamCensusFromReader: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
}

func TestStreamCensusFromReader_Limit(t *testing.T) {
	content := "age,sex,policy_type,sum_assured,term\n30,M,term,100000,20\n25,F,whole,50000,10\n40,M,term,200000,15\n"
	r := strings.NewReader(content)

	var count int
	err := StreamCensusFromReader(r, CSVOptions{Header: true, Limit: 2}, func(rec CensusRecord) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCensusFromReader: %v", err)
	}
	if count != 2 {
		t.Errorf("got %d records, want 2", count)
	}
}

func TestStreamCensusFromReader_Empty(t *testing.T) {
	content := "age,sex,policy_type,sum_assured,term\n"
	r := strings.NewReader(content)

	var records []CensusRecord
	err := StreamCensusFromReader(r, CSVOptions{Header: true}, func(rec CensusRecord) {
		records = append(records, rec)
	})
	if err != nil {
		t.Fatalf("StreamCensusFromReader: %v", err)
	}
	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestBuildColumnMap(t *testing.T) {
	m := buildColumnMap([]string{"Age", "Sex", "Policy Type", "SumAssured", "Term"})
	if m["age"] != 0 {
		t.Errorf("age index = %d, want 0", m["age"])
	}
	if m["sex"] != 1 {
		t.Errorf("sex index = %d, want 1", m["sex"])
	}
	if m["policy_type"] != 2 {
		t.Errorf("policy_type index = %d, want 2", m["policy_type"])
	}
	if m["sum_assured"] != 3 {
		t.Errorf("sum_assured index = %d, want 3", m["sum_assured"])
	}
	if m["term"] != 4 {
		t.Errorf("term index = %d, want 4", m["term"])
	}
}

func TestBuildColumnMap_MixedCase(t *testing.T) {
	m := buildColumnMap([]string{"AGE", "Sex", "sum_assured", "term"})
	if m["age"] != 0 {
		t.Errorf("age index = %d", m["age"])
	}
	if m["sex"] != 1 {
		t.Errorf("sex index = %d", m["sex"])
	}
	if m["sum_assured"] != 2 {
		t.Errorf("sum_assured index = %d", m["sum_assured"])
	}
	if m["term"] != 3 {
		t.Errorf("term index = %d", m["term"])
	}
}

func TestIsDefaultColumnOrder(t *testing.T) {
	m := ColumnMap{"age": 0, "sex": 1, "policy_type": 2, "sum_assured": 3, "term": 4}
	if !isDefaultColumnOrder(m) {
		t.Error("expected default column order to be recognized")
	}
}

func TestIsDefaultColumnOrder_WrongOrder(t *testing.T) {
	m := ColumnMap{"age": 1, "sex": 0, "policy_type": 2, "sum_assured": 3, "term": 4}
	if isDefaultColumnOrder(m) {
		t.Error("wrong order should not be recognized as default")
	}
}

func TestIsDefaultColumnOrder_WrongSize(t *testing.T) {
	m := ColumnMap{"age": 0, "sex": 1}
	if isDefaultColumnOrder(m) {
		t.Error("small map should not be recognized as default")
	}
}

func TestDefaultColumnMap(t *testing.T) {
	m := defaultColumnMap()
	if m["age"] != 0 || m["sex"] != 1 || m["policy_type"] != 2 || m["sum_assured"] != 3 || m["term"] != 4 {
		t.Errorf("unexpected default map: %v", m)
	}
}

func TestReadHeadersAndOffset(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "age,sex,policy_type,sum_assured,term\n30,M,term,100000,20\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	headers, offset, err := readHeadersAndOffset(tmpFile, ',')
	if err != nil {
		t.Fatalf("readHeadersAndOffset: %v", err)
	}
	if len(headers) != 5 {
		t.Errorf("got %d headers, want 5", len(headers))
	}
	if offset <= 0 {
		t.Errorf("offset = %d, want > 0", offset)
	}
}

func TestReadHeadersAndOffset_NotFound(t *testing.T) {
	_, _, err := readHeadersAndOffset("/nonexistent/file.csv", ',')
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseCensusRow_AllFields(t *testing.T) {
	fields := []string{"30", "M", "term", "100000", "20"}
	colMap := ColumnMap{"age": 0, "sex": 1, "policy_type": 2, "sum_assured": 3, "term": 4}

	rec, err := ParseCensusRow(fields, colMap)
	if err != nil {
		t.Fatalf("ParseCensusRow: %v", err)
	}
	if rec.Age != 30 || rec.Sex != "M" || rec.PolicyType != "term" || rec.SumAssured != 100000 || rec.Term != 20 {
		t.Errorf("record: %+v", rec)
	}
}

func TestParseCensusRow_DifferentOrder(t *testing.T) {
	fields := []string{"100000", "20", "30", "M", "term"}
	colMap := ColumnMap{"sum_assured": 0, "term": 1, "age": 2, "sex": 3, "policy_type": 4}

	rec, err := ParseCensusRow(fields, colMap)
	if err != nil {
		t.Fatalf("ParseCensusRow: %v", err)
	}
	if rec.Age != 30 || rec.SumAssured != 100000 || rec.Term != 20 {
		t.Errorf("record: %+v", rec)
	}
}

func TestParseCensusRow_EmptyRow(t *testing.T) {
	_, err := ParseCensusRow([]string{}, ColumnMap{})
	if err == nil {
		t.Error("expected error for empty row")
	}
}

func TestParseCensusRow_MissingField(t *testing.T) {
	fields := []string{"30", "M", "term", "100000", "20"}
	colMap := ColumnMap{"age": 0, "sum_assured": 3}

	rec, err := ParseCensusRow(fields, colMap)
	if err != nil {
		t.Fatalf("ParseCensusRow: %v", err)
	}
	if rec.Age != 30 || rec.SumAssured != 100000 {
		t.Errorf("record: %+v", rec)
	}
	if rec.Sex != "" || rec.PolicyType != "" || rec.Term != 0 {
		t.Error("unexpected fields should be zero-valued")
	}
}

func TestPolicyString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"term", "term"},
		{"whole", "whole"},
		{"endowment", "endowment"},
		{"whole_life", "whole_life"},
		{"other", "other"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := policyString([]byte(tt.input))
		if got != tt.want {
			t.Errorf("policyString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestSexString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"male", "male"},
		{"female", "female"},
		{"M", "M"},
		{"F", "F"},
		{"", ""},
	}
	for _, tt := range tests {
		got := sexString([]byte(tt.input))
		if got != tt.want {
			t.Errorf("sexString(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
