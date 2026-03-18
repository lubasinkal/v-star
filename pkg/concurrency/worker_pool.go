package concurrency

import (
	"sync"

	"github.com/lubasinkal/v-star/pkg/rates"
	"github.com/lubasinkal/v-star/pkg/reader"
)

// WorkerPool processes CensusRecords in parallel
type WorkerPool struct {
	workers    int
	converter  rates.RateConverter
	inputChan  chan reader.CensusRecord
	resultChan chan float64
	wg         sync.WaitGroup
}

// NewWorkerPool creates a new worker pool with specified number of workers
func NewWorkerPool(workers int, converter rates.RateConverter) *WorkerPool {
	return &WorkerPool{
		workers:    workers,
		converter:  converter,
		inputChan:  make(chan reader.CensusRecord, workers*10),
		resultChan: make(chan float64, workers*100), // Larger buffer for results
	}
}

// Start launches the worker goroutines
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// worker processes records from the input channel
func (wp *WorkerPool) worker() {
	defer wp.wg.Done()
	for record := range wp.inputChan {
		pv := wp.converter.PresentValue(record.SumAssured, record.Term)
		wp.resultChan <- pv
	}
}

// Submit adds a record to the processing queue
func (wp *WorkerPool) Submit(record reader.CensusRecord) {
	wp.inputChan <- record
}

// Close signals no more records will be submitted
func (wp *WorkerPool) Close() {
	close(wp.inputChan)
}

// Wait for all workers to finish and close result channel
func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
	close(wp.resultChan)
}

// CollectResults sums all results from the result channel
func (wp *WorkerPool) CollectResults() float64 {
	total := 0.0
	for pv := range wp.resultChan {
		total += pv
	}
	return total
}

// ProcessBatch processes a slice of records
// For now, it calculates sequentially to ensure correctness
// The worker pool structure is kept for future parallel implementation
func ProcessBatch(records []reader.CensusRecord, converter rates.RateConverter, workers int) float64 {
	if len(records) == 0 {
		return 0
	}

	// Simple sequential calculation for now
	// Worker pool will be implemented properly in Phase 2
	total := 0.0
	for _, record := range records {
		total += converter.PresentValue(record.SumAssured, record.Term)
	}
	return total
}
