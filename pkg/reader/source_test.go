package reader

import (
	"strings"
	"testing"
)

// --- SliceCensusSource ----------------------------------------------------

func TestSliceCensusSource(t *testing.T) {
	records := []CensusRecord{
		{Age: 30, Sex: "M", PolicyType: "term", SumAssured: 100000, Term: 20},
		{Age: 25, Sex: "F", PolicyType: "whole", SumAssured: 50000, Term: 10},
	}

	src := NewSliceCensusSource(records)
	got, err := src.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestSliceCensusSource_Empty(t *testing.T) {
	src := NewSliceCensusSource(nil)
	got, err := src.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestSliceCensusSource_Stream(t *testing.T) {
	records := []CensusRecord{
		{Age: 30, Sex: "M", PolicyType: "term", SumAssured: 100000, Term: 20},
		{Age: 25, Sex: "F", PolicyType: "whole", SumAssured: 50000, Term: 10},
		{Age: 40, Sex: "M", PolicyType: "term", SumAssured: 200000, Term: 15},
	}

	src := NewSliceCensusSource(records)
	var ages []int
	n, err := src.Stream(func(r CensusRecord) error {
		ages = append(ages, r.Age)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if n != 3 {
		t.Errorf("count = %d, want 3", n)
	}
	if len(ages) != 3 || ages[0] != 30 || ages[1] != 25 || ages[2] != 40 {
		t.Errorf("ages = %v, want [30 25 40]", ages)
	}
}

func TestSliceCensusSource_StreamEarlyStop(t *testing.T) {
	records := []CensusRecord{
		{Age: 30, Sex: "M", PolicyType: "term", SumAssured: 100000, Term: 20},
		{Age: 25, Sex: "F", PolicyType: "whole", SumAssured: 50000, Term: 10},
		{Age: 40, Sex: "M", PolicyType: "term", SumAssured: 200000, Term: 15},
	}

	src := NewSliceCensusSource(records)
	var ages []int
	wantErr := errTestStop
	n, err := src.Stream(func(r CensusRecord) error {
		if r.Age == 25 {
			return wantErr
		}
		ages = append(ages, r.Age)
		return nil
	})
	if err != wantErr {
		t.Fatalf("Stream err = %v, want %v", err, wantErr)
	}
	if n != 1 {
		t.Errorf("count = %d, want 1 (stopped after 1st record)", n)
	}
	if len(ages) != 1 || ages[0] != 30 {
		t.Errorf("ages = %v, want [30]", ages)
	}
}

// --- FileCensusSource -----------------------------------------------------

func TestFileCensusSource(t *testing.T) {
	path := writeTestCSV(t, []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,100000,20",
		"25,F,whole,50000,10",
		"40,M,term,200000,15",
	})

	src := NewFileCensusSource(path, CSVOptions{Header: true})
	got, err := src.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
}

func TestFileCensusSource_NoHeader(t *testing.T) {
	path := writeTestCSV(t, []string{
		"30,M,term,100000,20",
		"25,F,whole,50000,10",
	})

	src := NewFileCensusSource(path, CSVOptions{Header: false})
	got, err := src.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestFileCensusSource_NotFound(t *testing.T) {
	src := NewFileCensusSource("nonexistent.csv", CSVOptions{Header: true})
	_, err := src.ReadAll()
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestFileCensusSource_Limit(t *testing.T) {
	path := writeTestCSV(t, []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,100000,20",
		"25,F,whole,50000,10",
		"40,M,term,200000,15",
	})

	src := NewFileCensusSource(path, CSVOptions{Header: true, Limit: 2})
	got, err := src.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestFileCensusSource_Stream(t *testing.T) {
	path := writeTestCSV(t, []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,100000,20",
		"25,F,whole,50000,10",
	})

	src := NewFileCensusSource(path, CSVOptions{Header: true})
	var ages []int
	n, err := src.Stream(func(r CensusRecord) error {
		ages = append(ages, r.Age)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
	if len(ages) != 2 || ages[0] != 30 || ages[1] != 25 {
		t.Errorf("ages = %v, want [30 25]", ages)
	}
}

func TestFileCensusSource_StreamWithLimit(t *testing.T) {
	path := writeTestCSV(t, []string{
		"age,sex,policy_type,sum_assured,term",
		"30,M,term,100000,20",
		"25,F,whole,50000,10",
		"40,M,term,200000,15",
	})

	// Stream with Limit opts — the underlying StreamCensus stops early
	src := NewFileCensusSource(path, CSVOptions{Header: true, Limit: 2})
	var ages []int
	n, err := src.Stream(func(r CensusRecord) error {
		ages = append(ages, r.Age)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
	if len(ages) != 2 {
		t.Errorf("len(ages) = %d, want 2", len(ages))
	}
}

// --- ReaderCensusSource ---------------------------------------------------

func TestReaderCensusSource(t *testing.T) {
	csv := "age,sex,policy_type,sum_assured,term\n30,M,term,100000,20\n25,F,whole,50000,10\n"
	r := strings.NewReader(csv)

	src := NewReaderCensusSource(r, CSVOptions{Header: true})
	got, err := src.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Age != 30 || got[1].Age != 25 {
		t.Errorf("ages = [%d %d], want [30 25]", got[0].Age, got[1].Age)
	}
}

func TestReaderCensusSource_NoHeader(t *testing.T) {
	csv := "30,M,term,100000,20\n25,F,whole,50000,10\n"
	r := strings.NewReader(csv)

	src := NewReaderCensusSource(r, CSVOptions{Header: false})
	got, err := src.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

func TestReaderCensusSource_Stream(t *testing.T) {
	csv := "age,sex,policy_type,sum_assured,term\n30,M,term,100000,20\n25,F,whole,50000,10\n40,M,term,200000,15\n"
	r := strings.NewReader(csv)

	src := NewReaderCensusSource(r, CSVOptions{Header: true})
	var ages []int
	n, err := src.Stream(func(r CensusRecord) error {
		ages = append(ages, r.Age)
		return nil
	})
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if n != 3 {
		t.Errorf("count = %d, want 3", n)
	}
	if len(ages) != 3 {
		t.Errorf("len(ages) = %d, want 3", len(ages))
	}
}

func TestReaderCensusSource_StreamEarlyStop(t *testing.T) {
	csv := "age,sex,policy_type,sum_assured,term\n30,M,term,100000,20\n25,F,whole,50000,10\n40,M,term,200000,15\n"
	r := strings.NewReader(csv)

	src := NewReaderCensusSource(r, CSVOptions{Header: true})
	var ages []int
	n, err := src.Stream(func(r CensusRecord) error {
		if r.Age == 40 {
			return errTestStop
		}
		ages = append(ages, r.Age)
		return nil
	})
	if err != errTestStop {
		t.Fatalf("Stream err = %v, want %v", err, errTestStop)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
	if len(ages) != 2 || ages[0] != 30 || ages[1] != 25 {
		t.Errorf("ages = %v, want [30 25]", ages)
	}
}

func TestReaderCensusSource_Empty(t *testing.T) {
	r := strings.NewReader("")
	src := NewReaderCensusSource(r, CSVOptions{Header: false})
	got, err := src.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestReaderCensusSource_OnlyHeader(t *testing.T) {
	r := strings.NewReader("age,sex,policy_type,sum_assured,term\n")
	src := NewReaderCensusSource(r, CSVOptions{Header: true})
	got, err := src.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

func TestReaderCensusSource_Limit(t *testing.T) {
	csv := "age,sex,policy_type,sum_assured,term\n30,M,term,100000,20\n25,F,whole,50000,10\n40,M,term,200000,15\n"
	r := strings.NewReader(csv)

	src := NewReaderCensusSource(r, CSVOptions{Header: true, Limit: 2})
	got, err := src.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
}

// --- Interface compliance -------------------------------------------------

var (
	_ CensusSource = (*SliceCensusSource)(nil)
	_ CensusSource = (*FileCensusSource)(nil)
	_ CensusSource = (*ReaderCensusSource)(nil)
)

// errTestStop is used by early-stop tests.
var errTestStop = stopError("stop")

type stopError string

func (e stopError) Error() string { return string(e) }
