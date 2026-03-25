package reader

import (
	"os"
	"testing"
)

// BenchmarkStreamCensus measures pure CSV parsing throughput.
// Run with: go test ./pkg/reader -bench=BenchmarkStreamCensus -benchmem
func BenchmarkStreamCensus(b *testing.B) {
	filepath := "../../10M.csv"
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		b.Skip("10M.csv not found")
	}

	b.SetBytes(288 * 1024 * 1024) // file size for throughput calc
	for b.Loop() {
		count := 0
		StreamCensus(filepath, CSVOptions{Header: true, Limit: 5000000}, func(r CensusRecord) {
			count++
		})
		_ = count
	}
}

// BenchmarkStreamCensusNoStore measures parsing speed without storing records.
func BenchmarkStreamCensusNoStore(b *testing.B) {
	filepath := "../../10M.csv"
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		b.Skip("10M.csv not found")
	}

	b.SetBytes(288 * 1024 * 1024)
	for b.Loop() {
		StreamCensus(filepath, CSVOptions{Header: true, Limit: 5000000}, func(r CensusRecord) {
			_ = r.Age // minimal callback
		})
	}
}

// BenchmarkParseCensusFastBytes measures raw byte-level parsing speed.
func BenchmarkParseCensusFastBytes(b *testing.B) {
	line := []byte("30,male,term,100000.50,20")

	for b.Loop() {
		_, _ = parseCensusFastBytes(line, ',')
	}
}

// BenchmarkStreamCSV measures sequential CSV reading throughput.
// Run with: go test ./pkg/reader -bench=BenchmarkStreamCSV$ -benchmem
func BenchmarkStreamCSV(b *testing.B) {
	filepath := "../../10M.csv"
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		b.Skip("10M.csv not found")
	}

	b.SetBytes(288 * 1024 * 1024)
	for b.Loop() {
		count := 0
		StreamCSV(filepath, CSVOptions{Header: true, Limit: 5000000}, func(fields []string) {
			count++
		})
		_ = count
	}
}

// BenchmarkStreamCSVParallel measures parallel CSV reading throughput.
// Run with: go test ./pkg/reader -bench=BenchmarkStreamCSVParallel -benchmem
func BenchmarkStreamCSVParallel(b *testing.B) {
	filepath := "../../10M.csv"
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		b.Skip("10M.csv not found")
	}

	b.SetBytes(288 * 1024 * 1024)
	for b.Loop() {
		StreamCSVWithPV(filepath, CSVOptions{Header: true, Limit: 5000000}, func(sumAssured float64, term int) float64 {
			return sumAssured
		})
	}
}

// BenchmarkStreamCSVRaw measures raw byte slice CSV reading throughput (zero-allocation).
// Run with: go test ./pkg/reader -bench=BenchmarkStreamCSVRaw -benchmem
func BenchmarkStreamCSVRaw(b *testing.B) {
	filepath := "../../10M.csv"
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		b.Skip("10M.csv not found")
	}

	b.SetBytes(288 * 1024 * 1024)
	for b.Loop() {
		count := 0
		StreamCSVRaw(filepath, CSVOptions{Header: true, Limit: 5000000}, func(fields [][]byte) {
			count++
		})
		_ = count
	}
}

// BenchmarkParseFields measures generic field splitting speed.
func BenchmarkParseFields(b *testing.B) {
	line := []byte("30,male,term,100000.50,20")

	for b.Loop() {
		_ = parseFields(line, ',')
	}
}

// TestParseCensusFastBytesCorrectness verifies the fast parser produces correct results.
func TestParseCensusFastBytesCorrectness(t *testing.T) {
	tests := []struct {
		line  string
		age   int
		sex   string
		pType string
		sum   float64
		term  int
	}{
		{"30,male,term,100000.50,20", 30, "male", "term", 100000.50, 20},
		{"25,female,endowment,50000.00,10", 25, "female", "endowment", 50000.00, 10},
		{"65,male,whole,75000.75,5", 65, "male", "whole", 75000.75, 5},
		{"40,female,term,1234567.89,30", 40, "female", "term", 1234567.89, 30},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			record, err := parseCensusFastBytes([]byte(tt.line), ',')
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if record.Age != tt.age {
				t.Errorf("age: got %d, want %d", record.Age, tt.age)
			}
			if record.Sex != tt.sex {
				t.Errorf("sex: got %q, want %q", record.Sex, tt.sex)
			}
			if record.PolicyType != tt.pType {
				t.Errorf("policyType: got %q, want %q", record.PolicyType, tt.pType)
			}
			if record.SumAssured != tt.sum {
				t.Errorf("sumAssured: got %f, want %f", record.SumAssured, tt.sum)
			}
			if record.Term != tt.term {
				t.Errorf("term: got %d, want %d", record.Term, tt.term)
			}
		})
	}
}

// TestParseFieldsCorrectness verifies generic field parsing.
func TestParseFieldsCorrectness(t *testing.T) {
	line := []byte("30,male,term,100000.50,20")
	fields := parseFields(line, ',')
	if len(fields) != 5 {
		t.Fatalf("expected 5 fields, got %d", len(fields))
	}
	expected := []string{"30", "male", "term", "100000.50", "20"}
	for i, f := range fields {
		if f != expected[i] {
			t.Errorf("field %d: got %q, want %q", i, f, expected[i])
		}
	}
}

// TestParseFieldsQuoted verifies quoted field handling.
func TestParseFieldsQuoted(t *testing.T) {
	line := []byte(`30,"male,intersex",term,100000.50,20`)
	fields := parseFields(line, ',')
	if len(fields) != 5 {
		t.Fatalf("expected 5 fields, got %d: %v", len(fields), fields)
	}
	if fields[1] != `"male,intersex"` {
		t.Errorf("field 1: got %q, want %q", fields[1], `"male,intersex"`)
	}
}

// TestParseCensusRow verifies the generic row parser.
func TestParseCensusRow(t *testing.T) {
	fields := []string{"30", "male", "term", "100000.50", "20"}
	colMap := ColumnMap{
		"age": 0, "sex": 1, "policy_type": 2, "sum_assured": 3, "term": 4,
	}
	record, err := ParseCensusRow(fields, colMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.Age != 30 {
		t.Errorf("age: got %d, want 30", record.Age)
	}
	if record.SumAssured != 100000.50 {
		t.Errorf("sumAssured: got %f, want 100000.50", record.SumAssured)
	}
}

// TestParseCensusRowReordered verifies parsing with non-standard column order.
func TestParseCensusRowReordered(t *testing.T) {
	fields := []string{"female", "25", "term", "10", "50000.00"}
	colMap := ColumnMap{
		"sex": 0, "age": 1, "policy_type": 2, "term": 3, "sum_assured": 4,
	}
	record, err := ParseCensusRow(fields, colMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.Age != 25 {
		t.Errorf("age: got %d, want 25", record.Age)
	}
	if record.Sex != "female" {
		t.Errorf("sex: got %q, want %q", record.Sex, "female")
	}
	if record.SumAssured != 50000.00 {
		t.Errorf("sumAssured: got %f, want 50000.00", record.SumAssured)
	}
	if record.Term != 10 {
		t.Errorf("term: got %d, want 10", record.Term)
	}
}

// TestNormalizeColumnName verifies column name normalization.
func TestNormalizeColumnName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Sum Assured", "sum_assured"},
		{"Policy Type", "policy_type"},
		{"sumassured", "sum_assured"},
		{"Age", "age"},
		{"policy-type", "policy_type"},
	}
	for _, tt := range tests {
		got := normalizeColumnName(tt.input)
		if got != tt.expected {
			t.Errorf("normalizeColumnName(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
