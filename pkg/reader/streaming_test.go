package reader

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTestCSV(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.csv")
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	for _, line := range lines {
		f.WriteString(line)
		f.WriteString("\n")
	}
	f.Close()
	return path
}

func TestStreamCensusChunked(t *testing.T) {
	lines := []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,100000,20",
		"45,F,whole,200000,15",
		"25,M,endowment,50000,10",
	}
	path := writeTestCSV(t, lines)

	var records []CensusRecord
	opts := StreamOptions{
		CSVOptions: CSVOptions{Header: true},
		ChunkSize:  2,
		Workers:    1,
	}
	count, err := StreamCensusChunked(path, opts, func(chunk []CensusRecord) error {
		records = append(records, chunk...)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamCensusChunked: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	if len(records) != 3 {
		t.Errorf("records = %d, want 3", len(records))
	}
	if records[0].Age != 30 || records[0].SumAssured != 100000 {
		t.Errorf("first record mismatch: %+v", records[0])
	}
}

func TestStreamCensusChunked_Empty(t *testing.T) {
	path := writeTestCSV(t, []string{"age,sex,policy_type,sum_assured,term"})
	opts := StreamOptions{CSVOptions: CSVOptions{Header: true}}
	count, err := StreamCensusChunked(path, opts, func(chunk []CensusRecord) error {
		return nil
	})
	if err != nil {
		t.Fatalf("StreamCensusChunked: %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
}

func TestStreamCensusChunked_NoHeader(t *testing.T) {
	lines := []string{
		"30,M,term,100000,20",
		"45,F,whole,200000,15",
	}
	path := writeTestCSV(t, lines)

	var records []CensusRecord
	opts := StreamOptions{
		CSVOptions: CSVOptions{Header: false},
		ChunkSize:  1,
	}
	count, err := StreamCensusChunked(path, opts, func(chunk []CensusRecord) error {
		records = append(records, chunk...)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamCensusChunked: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestStreamCensusChunked_ChunkProcessorError(t *testing.T) {
	lines := []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,100000,20",
		"45,F,whole,200000,15",
	}
	path := writeTestCSV(t, lines)
	opts := StreamOptions{CSVOptions: CSVOptions{Header: true}}

	_, err := StreamCensusChunked(path, opts, func(chunk []CensusRecord) error {
		return os.ErrExist
	})
	if err == nil {
		t.Error("expected error from chunk processor")
	}
}

func TestStreamCensusWithPV(t *testing.T) {
	lines := []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,1000,1",
		"45,F,whole,2000,2",
	}
	path := writeTestCSV(t, lines)
	pvFn := func(sumAssured float64, term int) float64 {
		return sumAssured * 0.95
	}
	opts := StreamOptions{CSVOptions: CSVOptions{Header: true}}
	pv, count := StreamCensusWithPV(path, opts, pvFn)
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	expectedPV := 1000*0.95 + 2000*0.95
	if pv != expectedPV {
		t.Errorf("totalPV = %v, want %v", pv, expectedPV)
	}
}

func TestStreamCensusWithPV_EmptyFile(t *testing.T) {
	path := writeTestCSV(t, []string{"age,sex,policy_type,sum_assured,term"})
	opts := StreamOptions{CSVOptions: CSVOptions{Header: true}}
	pv, count := StreamCensusWithPV(path, opts, func(a float64, t int) float64 { return a })
	if count != 0 || pv != 0 {
		t.Errorf("expected (0,0), got (%v,%v)", pv, count)
	}
}

func TestStreamCensusWithPV_Limit(t *testing.T) {
	lines := []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,100,1",
		"45,F,whole,200,2",
		"25,M,whole,300,3",
	}
	path := writeTestCSV(t, lines)
	opts := StreamOptions{
		CSVOptions: CSVOptions{Header: true, Limit: 2},
	}
	pv, count := StreamCensusWithPV(path, opts, func(a float64, t int) float64 { return 1 })
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if pv != 2 {
		t.Errorf("pv = %v, want 2", pv)
	}
}

func TestStreamCensusChunked_Limit(t *testing.T) {
	lines := []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,100,1",
		"45,F,whole,200,2",
		"25,M,whole,300,3",
		"40,F,term,400,4",
	}
	path := writeTestCSV(t, lines)
	opts := StreamOptions{
		CSVOptions: CSVOptions{Header: true, Limit: 2},
	}
	count, err := StreamCensusChunked(path, opts, func(chunk []CensusRecord) error {
		return nil
	})
	if err != nil {
		t.Fatalf("StreamCensusChunked: %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
}

func TestStreamCensusChunked_AllWorkers(t *testing.T) {
	lines := []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,100,1",
		"45,F,whole,200,2",
	}
	path := writeTestCSV(t, lines)
	opts := StreamOptions{
		CSVOptions: CSVOptions{Header: true},
		Workers:    4,
		ChunkSize:  1,
	}
	_, err := StreamCensusChunked(path, opts, func(chunk []CensusRecord) error {
		return nil
	})
	if err != nil {
		t.Fatalf("StreamCensusChunked: %v", err)
	}
}
