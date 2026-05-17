// Package concurrency provides a generic worker pool for parallel processing.
//
// # Process records in parallel
//
//	wp := concurrency.NewWorkerPool(8, func(r reader.CensusRecord) float64 {
//	    return converter.PresentValue(r.SumAssured, r.Term)
//	})
//	totalPV := wp.ProcessBatch(records)
//
// Falls back to sequential processing for small batches (< 1000 records).
// For context cancellation, use ProcessBatchContext.
package concurrency
