package mortality

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTable(t *testing.T) {
	qx := []float64{0.001, 0.002, 0.005, 0.01, 0.02}
	table := NewTable("test", qx)

	if table.Name() != "test" {
		t.Errorf("Name() = %v, want test", table.Name())
	}
	if table.MaxAge() != 4 {
		t.Errorf("MaxAge() = %v, want 4", table.MaxAge())
	}
	if table.Lx(0) != 100000 {
		t.Errorf("Lx(0) = %v, want 100000", table.Lx(0))
	}
}

func TestNewTableNil(t *testing.T) {
	table := NewTable("empty", nil)
	if table == nil {
		t.Error("NewTable with nil should not return nil")
	}
	if table.MaxAge() != -1 {
		t.Errorf("MaxAge() = %v, want -1", table.MaxAge())
	}
}

func TestNewTableEmpty(t *testing.T) {
	table := NewTable("empty", []float64{})
	if table == nil {
		t.Error("NewTable with empty should not return nil")
	}
	if table.MaxAge() != -1 {
		t.Errorf("MaxAge() = %v, want -1", table.MaxAge())
	}
}

func TestTableQx(t *testing.T) {
	qx := []float64{0.001, 0.002, 0.005, 0.01, 0.02}
	table := NewTable("test", qx)

	tests := []struct {
		age  int
		want float64
	}{
		{-1, 0},
		{0, 0.001},
		{1, 0.002},
		{2, 0.005},
		{3, 0.01},
		{4, 0.02},
		{5, 0},
		{100, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := table.Qx(tt.age)
			if got != tt.want {
				t.Errorf("Qx(%d) = %v, want %v", tt.age, got, tt.want)
			}
		})
	}
}

func TestTablePx(t *testing.T) {
	qx := []float64{0.0, 0.0, 0.0, 0.0, 0.0}
	table := NewTable("test", qx)

	got := table.Px(0, 3)
	want := 1.0
	if got != want {
		t.Errorf("Px(0, 3) = %v, want %v", got, want)
	}

	qx2 := []float64{0.1, 0.1, 0.1}
	table2 := NewTable("test2", qx2)

	got2 := table2.Px(0, 2)
	want2 := 0.9 * 0.9
	if math.Abs(got2-want2) > 1e-9 {
		t.Errorf("Px(0, 2) = %v, want %v", got2, want2)
	}

	got3 := table2.Px(0, 0)
	if got3 != 1.0 {
		t.Errorf("Px(0, 0) = %v, want 1.0", got3)
	}

	got4 := table2.Px(0, -1)
	if got4 != 1.0 {
		t.Errorf("Px(0, -1) = %v, want 1.0", got4)
	}
}

func TestTablePxBeyondMax(t *testing.T) {
	qx := []float64{0.1, 0.1, 0.1}
	table := NewTable("test", qx)

	if table.Px(2, 3) != 0 {
		t.Errorf("Px(2, 3) = %v, want 0", table.Px(2, 3))
	}
}

func TestTableLx(t *testing.T) {
	qx := []float64{0.01, 0.01, 0.01}
	table := NewTable("test", qx)

	if table.Lx(0) != 100000 {
		t.Errorf("Lx(0) = %v, want 100000", table.Lx(0))
	}

	got := table.Lx(1)
	want := 100000 * 0.99
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("Lx(1) = %v, want %v", got, want)
	}

	if table.Lx(-1) != 0 {
		t.Errorf("Lx(-1) = %v, want 0", table.Lx(-1))
	}
	if table.Lx(100) != 0 {
		t.Errorf("Lx(100) = %v, want 0", table.Lx(100))
	}
}

func TestTableMaxAge(t *testing.T) {
	qx := []float64{0.001, 0.002, 0.005}
	table := NewTable("test", qx)

	if table.MaxAge() != 2 {
		t.Errorf("MaxAge() = %v, want 2", table.MaxAge())
	}
}

func TestTableName(t *testing.T) {
	qx := []float64{0.001}
	table := NewTable("CD2021", qx)

	if table.Name() != "CD2021" {
		t.Errorf("Name() = %v, want CD2021", table.Name())
	}
}

func TestTableEx(t *testing.T) {
	qx := []float64{0.0, 0.0, 0.0, 0.0, 0.0}
	table := NewTable("test", qx)

	got := table.Ex(0)
	want := 4.0
	if math.Abs(got-want) > 1e-9 {
		t.Errorf("Ex(0) = %v, want %v", got, want)
	}

	got2 := table.Ex(-1)
	if got2 != 0 {
		t.Errorf("Ex(-1) = %v, want 0", got2)
	}

	got3 := table.Ex(4)
	if got3 != 0 {
		t.Errorf("Ex(4) = %v, want 0", got3)
	}
}

func TestParseLines(t *testing.T) {
	data := []byte("age,qx\n0,0.001\n1,0.002\n")
	lines := parseLines(data)
	if len(lines) == 0 {
		t.Error("parseLines() returned empty")
	}
}

func TestSplitCSV(t *testing.T) {
	fields := splitCSV([]byte("a,b,c"))
	if len(fields) != 3 {
		t.Errorf("splitCSV() = %d, want 3", len(fields))
	}
	if fields[0] != "a" || fields[1] != "b" || fields[2] != "c" {
		t.Errorf("splitCSV() = %v", fields)
	}
}

func TestDetectColumns(t *testing.T) {
	cols := detectColumns([]byte("age,qx,px"))
	if cols["age"] != 0 || cols["qx"] != 1 || cols["px"] != 2 {
		t.Errorf("detectColumns() = %v", cols)
	}
}

func TestParseInt(t *testing.T) {
	tests := []struct {
		s    string
		want int
	}{
		{"0", 0},
		{"123", 123},
		{"-45", -45},
		{"", 0},
		{"007", 7},
	}

	for _, tt := range tests {
		if got := parseInt(tt.s); got != tt.want {
			t.Errorf("parseInt(%q) = %d, want %d", tt.s, got, tt.want)
		}
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		s    string
		want float64
	}{
		{"0", 0},
		{"123.456", 123.456},
		{"-45.5", -45.5},
		{"", 0},
		{".5", 0.5},
	}

	for _, tt := range tests {
		if got := parseFloat(tt.s); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("parseFloat(%q) = %v, want %v", tt.s, got, tt.want)
		}
	}
}

func TestExtractName(t *testing.T) {
	tests := []struct {
		filepath string
		want     string
	}{
		{"/path/to/table.csv", "table"},
		{"C:\\path\\to\\table.csv", "table"},
		{"table.csv", "table"},
	}

	for _, tt := range tests {
		if got := extractName(tt.filepath); got != tt.want {
			t.Errorf("extractName(%q) = %q, want %q", tt.filepath, got, tt.want)
		}
	}
}

func TestLoadCSV(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.csv")
	content := "age,qx\n0,0.001\n1,0.002\n2,0.005\n"

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	table, err := LoadCSV(tmpFile)
	if err != nil {
		t.Fatalf("LoadCSV: %v", err)
	}

	if table.Name() != "test" {
		t.Errorf("Name() = %v, want test", table.Name())
	}
	if table.MaxAge() != 2 {
		t.Errorf("MaxAge() = %v, want 2", table.MaxAge())
	}
	if table.Qx(0) != 0.001 {
		t.Errorf("Qx(0) = %v, want 0.001", table.Qx(0))
	}
}

func TestLoadCSVNotFound(t *testing.T) {
	_, err := LoadCSV("/nonexistent/path/table.csv")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestStreamCSV(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "stream.csv")
	content := "age,qx\n0,0.001\n1,0.002\n2,0.005\n"

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCSV(tmpFile, func(age int, q float64) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if count != 3 {
		t.Errorf("StreamCSV got %d records, want 3", count)
	}
}

func TestStreamCSVWithPx(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "px.csv")
	content := "age,px\n0,1.0\n1,0.999\n2,0.997\n"

	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var ages []int
	err := StreamCSV(tmpFile, func(age int, q float64) {
		ages = append(ages, age)
	})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if len(ages) != 3 {
		t.Errorf("StreamCSV got %d records", len(ages))
	}
}

func TestStreamCSVLarge(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.csv")
	var lines []string
	lines = append(lines, "age,qx")
	for i := 0; i < 2000; i++ {
		lines = append(lines, fmt.Sprintf("%d,0.00%d", i, i%10))
	}
	content := []byte(strings.Join(lines, "\n"))

	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCSV(tmpFile, func(age int, q float64) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if count != 2000 {
		t.Errorf("StreamCSV got %d records, want 2000", count)
	}
}

func TestStreamCSVLargeWithPx(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large_px.csv")
	var lines []string
	lines = append(lines, "age,px")
	px := 1.0
	for i := 0; i < 2000; i++ {
		lines = append(lines, fmt.Sprintf("%d,%.4f", i, px))
		px *= 0.999
	}
	content := []byte(strings.Join(lines, "\n"))

	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	var count int
	err := StreamCSV(tmpFile, func(age int, q float64) {
		count++
	})
	if err != nil {
		t.Fatalf("StreamCSV: %v", err)
	}
	if count != 2000 {
		t.Errorf("StreamCSV got %d records, want 2000", count)
	}
}

func BenchmarkQx(b *testing.B) {
	qx := make([]float64, 121)
	for i := range qx {
		qx[i] = float64(i) * 0.0001
	}
	table := NewTable("bench", qx)

	for b.Loop() {
		for age := 0; age <= 120; age++ {
			_ = table.Qx(age)
		}
	}
}

func BenchmarkPx(b *testing.B) {
	qx := make([]float64, 121)
	for i := range qx {
		qx[i] = float64(i) * 0.0001
	}
	table := NewTable("bench", qx)

	for b.Loop() {
		for age := 0; age <= 100; age++ {
			_ = table.Px(age, 20)
		}
	}
}

func ExampleTable_Qx() {
	qx := []float64{0.001, 0.002, 0.005, 0.01, 0.02, 0.03}
	table := NewTable("test", qx)

	q := table.Qx(2)
	println(q)
}

func ExampleTable_Ex() {
	qx := make([]float64, 121)
	for i := range qx {
		qx[i] = 0.001 * float64(i+1)
	}
	table := NewTable("test", qx)

	ex := table.Ex(60)
	println(ex)
}
