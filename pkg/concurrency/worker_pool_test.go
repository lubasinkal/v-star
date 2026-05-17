package concurrency

import (
	"context"
	"testing"

	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
)

func pvFn(converter *rates.RateConverter) func(reader.CensusRecord) float64 {
	return func(r reader.CensusRecord) float64 {
		return converter.PresentValue(r.SumAssured, r.Term)
	}
}

func TestNewWorkerPool_EmptyRecords(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	wp := NewWorkerPool(4, pvFn(converter))
	got := wp.ProcessBatch(nil)
	if got != 0 {
		t.Errorf("ProcessBatch(nil) = %v, want 0", got)
	}
}

func TestNewWorkerPool_SingleRecord(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	wp := NewWorkerPool(1, pvFn(converter))
	records := []reader.CensusRecord{
		{Age: 30, SumAssured: 100000, Term: 20},
	}

	got := wp.ProcessBatch(records)
	expected := converter.PresentValue(100000, 20)
	if got != expected {
		t.Errorf("ProcessBatch single record = %v, want %v", got, expected)
	}
}

func TestNewWorkerPool_MultipleRecords(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	wp := NewWorkerPool(4, pvFn(converter))
	records := []reader.CensusRecord{
		{Age: 30, SumAssured: 100000, Term: 20},
		{Age: 45, SumAssured: 200000, Term: 10},
		{Age: 50, SumAssured: 150000, Term: 15},
	}

	got := wp.ProcessBatch(records)
	expected := 0.0
	for _, r := range records {
		expected += converter.PresentValue(r.SumAssured, r.Term)
	}

	if got != expected {
		t.Errorf("ProcessBatch = %v, want %v", got, expected)
	}
}

func TestNewWorkerPool_ParallelMatchesSequential(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	records := make([]reader.CensusRecord, 2000)
	for i := range records {
		records[i] = reader.CensusRecord{
			Age:        30 + i%50,
			SumAssured: 100000.0 + float64(i)*1000.0,
			Term:       10 + i%20,
		}
	}

	seqWP := NewWorkerPool(1, pvFn(converter))
	parWP := NewWorkerPool(8, pvFn(converter))
	sequential := seqWP.ProcessBatch(records)
	parallel := parWP.ProcessBatch(records)

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
	wp := NewWorkerPool(0, func(r reader.CensusRecord) float64 { return 0 })
	if wp == nil {
		t.Error("NewWorkerPool returned nil")
	}
}

func TestWorkerPool_ProcessBatch(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	wp := NewWorkerPool(4, pvFn(converter))
	records := []reader.CensusRecord{
		{Age: 30, SumAssured: 100000, Term: 20},
	}

	got := wp.ProcessBatch(records)
	expected := converter.PresentValue(100000, 20)
	if got != expected {
		t.Errorf("WorkerPool.ProcessBatch = %v, want %v", got, expected)
	}
}

func TestWorkerPool_ProcessBatchContext(t *testing.T) {
	converter := rates.NewRateConverter(0.05)
	wp := NewWorkerPool(4, pvFn(converter))
	records := []reader.CensusRecord{
		{Age: 30, SumAssured: 100000, Term: 20},
	}

	got, err := wp.ProcessBatchContext(context.Background(), records)
	if err != nil {
		t.Fatalf("ProcessBatchContext: %v", err)
	}
	expected := converter.PresentValue(100000, 20)
	if got != expected {
		t.Errorf("ProcessBatchContext = %v, want %v", got, expected)
	}
}

func TestWorkerPool_ProcessBatchContext_Cancelled(t *testing.T) {
	wp := NewWorkerPool(4, func(r reader.CensusRecord) float64 { return 1 })
	records := make([]reader.CensusRecord, 10000)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := wp.ProcessBatchContext(ctx, records)
	if err == nil {
		t.Error("expected error from cancelled context")
	}
}
