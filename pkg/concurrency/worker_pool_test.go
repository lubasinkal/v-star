package concurrency

import (
	"testing"

	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
)

func TestProcessBatch_EmptyRecords(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	got := ProcessBatch(nil, converter, 4)
	if got != 0 {
		t.Errorf("ProcessBatch(nil) = %v, want 0", got)
	}
}

func TestProcessBatch_SingleRecord(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	records := []reader.CensusRecord{
		{Age: 30, SumAssured: 100000, Term: 20},
	}

	got := ProcessBatch(records, converter, 1)
	expected := converter.PresentValue(100000, 20)
	if got != expected {
		t.Errorf("ProcessBatch single record = %v, want %v", got, expected)
	}
}

func TestProcessBatch_MultipleRecords(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	records := []reader.CensusRecord{
		{Age: 30, SumAssured: 100000, Term: 20},
		{Age: 45, SumAssured: 200000, Term: 10},
		{Age: 50, SumAssured: 150000, Term: 15},
	}

	got := ProcessBatch(records, converter, 4)
	expected := 0.0
	for _, r := range records {
		expected += converter.PresentValue(r.SumAssured, r.Term)
	}

	if got != expected {
		t.Errorf("ProcessBatch = %v, want %v", got, expected)
	}
}

func TestProcessBatch_ParallelMatchesSequential(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	records := make([]reader.CensusRecord, 2000)
	for i := range records {
		records[i] = reader.CensusRecord{
			Age:        30 + i%50,
			SumAssured: 100000.0 + float64(i)*1000.0,
			Term:       10 + i%20,
		}
	}

	sequential := ProcessBatch(records, converter, 1)
	parallel := ProcessBatch(records, converter, 8)

	// Floating point accumulation order differs, use relative tolerance
	diff := sequential - parallel
	if diff < 0 {
		diff = -diff
	}
	relative := diff / sequential
	if relative > 1e-10 {
		t.Errorf("sequential=%v != parallel=%v (relative diff=%v)", sequential, parallel, relative)
	}
}

func TestNewWorkerPool_DefaultWorkers(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	wp := NewWorkerPool(0, converter)
	if wp == nil {
		t.Error("NewWorkerPool returned nil")
	}
}

func TestWorkerPool_ProcessBatch(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	wp := NewWorkerPool(4, converter)
	records := []reader.CensusRecord{
		{Age: 30, SumAssured: 100000, Term: 20},
	}

	got := wp.ProcessBatch(records)
	expected := converter.PresentValue(100000, 20)
	if got != expected {
		t.Errorf("WorkerPool.ProcessBatch = %v, want %v", got, expected)
	}
}
