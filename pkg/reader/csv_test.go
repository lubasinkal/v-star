package reader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenCSVNotFound(t *testing.T) {
	_, _, _, _, err := openCSV("/nonexistent/file.csv", CSVOptions{})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestStreamCSV(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "age,sex,policy_type,sum_assured,term\n30,M,term,100000,20\n25,F,whole,50000,10\n"

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCSV(tmpFile, CSVOptions{Header: true}, func(fields []string) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestStreamCSVNoHeader(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "30,M,term,100000,20\n25,F,whole,50000,10\n"

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var fields []string
	err := StreamCSV(tmpFile, CSVOptions{Header: false}, func(flds []string) {
		fields = append(fields, flds...)
	})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if len(fields) < 4 {
		t.Errorf("fields = %v", fields)
	}
}

func TestStreamCSVLimit(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "age,term,sum\n" + makeRows(100)

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCSV(tmpFile, CSVOptions{Header: true, Limit: 10}, func(fields []string) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if count != 10 {
		t.Errorf("count = %d, want 10", count)
	}
}

func TestStreamCSVRaw(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "age,term,sum\n30,20,100000\n"

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCSVRaw(tmpFile, CSVOptions{Header: true}, func(fields [][]byte) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCSVRaw: %v", err)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
}

func TestStreamCSVRawFields(t *testing.T) {
	fields := parseFieldsRaw([]byte("30,20,100000"), ',')
	if len(fields) != 3 {
		t.Errorf("len(fields) = %d, want 3", len(fields))
	}
}

func TestGetHeaders(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "age,term,sum_assured\n30,20,100000\n"

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	headers, err := GetHeaders(tmpFile, ',')
	if err != nil {
		t.Fatalf("GetHeaders: %v", err)
	}
	if len(headers) != 3 {
		t.Errorf("len(headers) = %d, want 3", len(headers))
	}
}

func TestStreamCSVParallel(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.csv")
	var content strings.Builder
	content.WriteString("age,term,sum\n")
	for i := range 5000 {
		content.WriteString(fmt.Sprintf("%d,%d,%d\n", i%50+20, i%20+1, 100000+i))
	}

	if err := os.WriteFile(tmpFile, []byte(content.String()), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCSV(tmpFile, CSVOptions{Header: true}, func(fields []string) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if count != 5000 {
		t.Errorf("count = %d, want 5000", count)
	}
}

func TestParseFieldsFast(t *testing.T) {
	fields := parseFieldsFast([]byte("a,b,c,d"), ',')
	if len(fields) != 4 {
		t.Errorf("len(fields) = %d, want 4", len(fields))
	}
}

func TestParseFieldsRaw(t *testing.T) {
	fields := parseFieldsRaw([]byte("a,b,c"), ',')
	if len(fields) != 3 {
		t.Errorf("len(fields) = %d, want 3", len(fields))
	}
}

func TestStreamCSV_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "empty.csv")
	if err := os.WriteFile(tmpFile, []byte("age,term,sum\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCSV(tmpFile, CSVOptions{Header: true}, func(fields []string) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestStreamCSV_Parallel(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.csv")
	var content strings.Builder
	content.WriteString("age,term,sum\n")
	for i := range 5000 {
		content.WriteString(fmt.Sprintf("%d,%d,%d\n", i%50+20, i%20+1, 100000+i))
	}

	if err := os.WriteFile(tmpFile, []byte(content.String()), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCSV(tmpFile, CSVOptions{Header: true, Limit: 50}, func(fields []string) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if count != 50 {
		t.Errorf("count = %d, want 50", count)
	}
}

func TestStreamCSV_CustomDelimiter(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "age|term|sum_assured\n30|20|100000\n25|10|50000\n"
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCSV(tmpFile, CSVOptions{Header: true, Delimiter: '|'}, func(fields []string) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestStreamCSVRaw_Parallel(t *testing.T) {
	// Trigger parallel path in StreamCSVRaw with Limit-based threshold.
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.csv")
	var content strings.Builder
	content.WriteString("age,term,sum\n")
	for i := range 5000 {
		content.WriteString(fmt.Sprintf("%d,%d,%d\n", i%50+20, i%20+1, 100000+i))
	}

	if err := os.WriteFile(tmpFile, []byte(content.String()), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCSVRaw(tmpFile, CSVOptions{Header: true, Limit: 100}, func(fields [][]byte) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCSVRaw: %v", err)
	}
	if count != 100 {
		t.Errorf("count = %d, want 100", count)
	}
}

func TestParseFields(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"a,b,c", 3},
		{"a,b,c,d", 4},
		{"a", 1},
		{"", 1},
	}
	for _, tt := range tests {
		got := parseFields([]byte(tt.input), ',')
		if len(got) != tt.want {
			t.Errorf("parseFields(%q) = %d fields, want %d", tt.input, len(got), tt.want)
		}
	}
}

func TestParseFields_Quoted(t *testing.T) {
	fields := parseFields([]byte(`a,"b,c",d`), ',')
	if len(fields) != 3 {
		t.Fatalf("got %d fields, want 3", len(fields))
	}
}

func TestCSVOptions(t *testing.T) {
	opts := CSVOptions{
		Limit:     100,
		Header:    true,
		Delimiter: '|',
	}
	if opts.Limit != 100 {
		t.Errorf("Limit = %d, want 100", opts.Limit)
	}
	if opts.Header != true {
		t.Errorf("Header = %v, want true", opts.Header)
	}
	if opts.Delimiter != '|' {
		t.Errorf("Delimiter = %q, want '|'", opts.Delimiter)
	}
}

func TestCSVOptions_Defaults(t *testing.T) {
	opts := CSVOptions{}
	if opts.Limit != 0 {
		t.Errorf("Limit = %d, want 0", opts.Limit)
	}
}

func makeRows(n int) string {
	var rows strings.Builder
	for i := range n {
		rows.WriteString(fmt.Sprintf("%d,%d,%d\n", i%50+20, i%20+1, 100000+i))
	}
	return rows.String()
}
