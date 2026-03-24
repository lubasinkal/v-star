package concurrency

import (
	"runtime"
	"sync"

	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
)

// WorkerPool processes CensusRecords in parallel using goroutines.
type WorkerPool struct {
	workers   int
	converter rates.RateConverter
	wg        sync.WaitGroup
}

// NewWorkerPool creates a new worker pool with the specified number of workers.
// If workers <= 0, defaults to runtime.NumCPU().
func NewWorkerPool(workers int, converter rates.RateConverter) *WorkerPool {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &WorkerPool{
		workers:   workers,
		converter: converter,
	}
}

// ProcessBatch processes a slice of records in parallel using the worker pool.
// It returns the total present value across all records.
func (wp *WorkerPool) ProcessBatch(records []reader.CensusRecord) float64 {
	if len(records) == 0 {
		return 0
	}

	if wp.workers == 1 || len(records) < 1000 {
		return wp.processSequential(records)
	}

	return wp.processParallel(records)
}

// processSequential calculates PV without goroutine overhead for small batches.
func (wp *WorkerPool) processSequential(records []reader.CensusRecord) float64 {
	total := 0.0
	for _, record := range records {
		total += wp.converter.PresentValue(record.SumAssured, record.Term)
	}
	return total
}

// processParallel distributes work across goroutines and collects partial sums.
func (wp *WorkerPool) processParallel(records []reader.CensusRecord) float64 {
	chunkSize := (len(records) + wp.workers - 1) / wp.workers
	results := make(chan float64, wp.workers)

	for w := 0; w < wp.workers; w++ {
		start := w * chunkSize
		end := min(start+chunkSize, len(records))
		if start >= len(records) {
			break
		}

		wp.wg.Add(1)
		go func(chunk []reader.CensusRecord) {
			defer wp.wg.Done()
			partial := 0.0
			for _, record := range chunk {
				partial += wp.converter.PresentValue(record.SumAssured, record.Term)
			}
			results <- partial
		}(records[start:end])
	}

	go func() {
		wp.wg.Wait()
		close(results)
	}()

	total := 0.0
	for partial := range results {
		total += partial
	}
	return total
}

// ProcessBatch is a convenience function that creates a WorkerPool and processes records.
// Workers <= 0 defaults to runtime.NumCPU().
func ProcessBatch(records []reader.CensusRecord, converter rates.RateConverter, workers int) float64 {
	wp := NewWorkerPool(workers, converter)
	return wp.ProcessBatch(records)
}
