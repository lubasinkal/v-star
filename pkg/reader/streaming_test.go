package reader

import (
	"fmt"
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

func TestStreamCensusChunked_Parallel(t *testing.T) {
	// Trigger parallel path by setting small chunkSize so dataSize >= chunkSize*1000
	// and Workers > 1.
	var lines []string
	lines = append(lines, "age,sex,policy_type,sum_assured,term")
	for range 500 {
		lines = append(lines, "30,M,term,100000,20")
	}
	path := writeTestCSV(t, lines)

	opts := StreamOptions{
		CSVOptions: CSVOptions{Header: true},
		ChunkSize:  2,
		Workers:    4,
	}
	count, err := StreamCensusChunked(path, opts, func(chunk []CensusRecord) error {
		return nil
	})
	if err != nil {
		t.Fatalf("StreamCensusChunked: %v", err)
	}
	if count != 500 {
		t.Errorf("count = %d, want 500", count)
	}
}

func TestStreamCensusChunked_ParallelLimit(t *testing.T) {
	// Parallel path — limit applies to processFn calls.
	// Note: the returned count is the total parsed, not the limited count.
	var lines []string
	lines = append(lines, "age,sex,policy_type,sum_assured,term")
	for i := range 50 {
		lines = append(lines, fmt.Sprintf("%d,%s,%s,%d,%d", i%50+20, "M", "term", 100000+i, i%20+1))
	}
	path := writeTestCSV(t, lines)

	var processed int
	opts := StreamOptions{
		CSVOptions: CSVOptions{Header: true, Limit: 3},
		ChunkSize:  2,
		Workers:    4,
	}
	count, err := StreamCensusChunked(path, opts, func(chunk []CensusRecord) error {
		processed += len(chunk)
		return nil
	})
	if err != nil {
		t.Fatalf("StreamCensusChunked: %v", err)
	}
	if processed != 3 {
		t.Errorf("processed = %d, want 3", processed)
	}
	_ = count // total parsed, may differ from limited count
}

func TestStreamCensusChunked_NoDataAfterHeader(t *testing.T) {
	path := writeTestCSV(t, []string{"age,sex,policy_type,sum_assured,term"})
	opts := StreamOptions{
		CSVOptions: CSVOptions{Header: true},
		Workers:    2,
	}
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
