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

func TestStreamCensus_EmptyDataAfterHeader(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty_data.csv")
	if err := os.WriteFile(tmpFile, []byte("age,sex,policy_type,sum_assured,term\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCensus(tmpFile, CSVOptions{Header: true}, func(r CensusRecord) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCensus: %v", err)
	}
	if count != 0 {
		t.Errorf("got %d records, want 0", count)
	}
}

func TestStreamCensusFromReader_NoHeaderEmpty(t *testing.T) {
	var count int
	err := StreamCensusFromReader(strings.NewReader(""), CSVOptions{}, func(r CensusRecord) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCensusFromReader: %v", err)
	}
	if count != 0 {
		t.Errorf("got %d records, want 0", count)
	}
}

func TestStreamCensusFromReader_WithParseError(t *testing.T) {
	data := "age,sex,policy_type,sum_assured,term\n30,M,term,100000,20\nbad\n31,F,whole,200000,25\n"
	var errCount int
	var count int
	err := StreamCensusFromReader(strings.NewReader(data), CSVOptions{Header: true, OnParseError: func(_ int, _ error) {
		errCount++
	}}, func(r CensusRecord) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCensusFromReader: %v", err)
	}
	if count != 2 {
		t.Errorf("got %d records, want 2", count)
	}
	if errCount != 1 {
		t.Errorf("got %d errors, want 1", errCount)
	}
}

func TestMmapFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "mmap.csv")
	content := []byte("age,sex\n30,M\n31,F\n")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	f, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	mapped, err := mmapFile(f)
	if err != nil {
		t.Fatalf("mmapFile: %v", err)
	}
	defer munmap(mapped)

	if len(mapped) != len(content) {
		t.Errorf("mmap len = %d, want %d", len(mapped), len(content))
	}
	if string(mapped) != string(content) {
		t.Errorf("mmap content = %q, want %q", string(mapped), string(content))
	}
}

func TestMmapEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.csv")
	if err := os.WriteFile(tmpFile, []byte{}, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	f, err := os.Open(tmpFile)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	mapped, err := mmapFile(f)
	if err != nil {
		t.Fatalf("mmapFile: %v", err)
	}
	defer munmap(mapped)

	if mapped != nil {
		t.Errorf("expected nil for empty file, got %v", mapped)
	}
}

func TestParseMappedCensus(t *testing.T) {
	// Build a synthetic CSV buffer with header + data rows
	header := []byte("age,sex,policy_type,sum_assured,term\n")
	var data []byte
	for i := range 1000 {
		row := []byte("30,M,term,100000,20\n")
		data = append(data, row...)
		_ = i
	}
	mapped := append(header, data...)

	var count int
	var seen []CensusRecord
	err := parseMappedCensus(mapped, int64(len(header)), int64(len(data)), 4, ',', 0, nil, func(r CensusRecord) {
		count++
		seen = append(seen, r)
	})
	if err != nil {
		t.Fatalf("parseMappedCensus: %v", err)
	}
	if count != 1000 {
		t.Errorf("got %d records, want 1000", count)
	}
	if seen[0].Age != 30 || seen[0].Sex != "M" || seen[0].PolicyType != "term" || seen[0].SumAssured != 100000 || seen[0].Term != 20 {
		t.Errorf("first record: %+v", seen[0])
	}
}

func TestParseMappedCensus_WithLimit(t *testing.T) {
	header := []byte("age,sex,policy_type,sum_assured,term\n")
	var data []byte
	for i := range 100 {
		data = append(data, []byte("30,M,term,100000,20\n")...)
		_ = i
	}
	mapped := append(header, data...)

	var count int
	err := parseMappedCensus(mapped, int64(len(header)), int64(len(data)), 2, ',', 10, nil, func(r CensusRecord) {
		count++
	})
	if err != nil {
		t.Fatalf("parseMappedCensus: %v", err)
	}
	if count != 10 {
		t.Errorf("got %d records, want 10", count)
	}
}

func TestParseMappedCensus_WithParseError(t *testing.T) {
	header := []byte("age,sex,policy_type,sum_assured,term\n")
	// Include one bad row
	data := []byte("30,M,term,100000,20\nbadrow\n31,F,whole,200000,25\n")
	mapped := append(header, data...)

	var count int
	var errCount int
	err := parseMappedCensus(mapped, int64(len(header)), int64(len(data)), 1, ',', 0, func(_ int, _ error) {
		errCount++
	}, func(r CensusRecord) {
		count++
	})
	if err != nil {
		t.Fatalf("parseMappedCensus: %v", err)
	}
	if count != 2 {
		t.Errorf("got %d records, want 2", count)
	}
	if errCount != 1 {
		t.Errorf("got %d parse errors, want 1", errCount)
	}
}

func TestParseMappedCensus_EmptyData(t *testing.T) {
	header := []byte("age,sex,policy_type,sum_assured,term\n")
	mapped := append(header, []byte{}...)

	var count int
	err := parseMappedCensus(mapped, int64(len(header)), 0, 2, ',', 0, nil, func(r CensusRecord) {
		count++
	})
	if err != nil {
		t.Fatalf("parseMappedCensus: %v", err)
	}
	if count != 0 {
		t.Errorf("got %d records, want 0", count)
	}
}

func TestStreamCensus_LargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large file test in short mode")
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.csv")

	// Build a CSV >10MB to trigger memory-mapped path
	var buf strings.Builder
	buf.WriteString("age,sex,policy_type,sum_assured,term\n")
	row := "30,M,term,100000,20\n"
	// ~11MB: 200000 rows * ~55 bytes = 11MB
	for i := range 200000 {
		buf.WriteString(row)
		_ = i
	}
	if err := os.WriteFile(tmpFile, []byte(buf.String()), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCensus(tmpFile, CSVOptions{Header: true, Delimiter: ','}, func(r CensusRecord) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCensus: %v", err)
	}
	if count != 200000 {
		t.Errorf("got %d records, want 200000", count)
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
