// Package concurrency provides a worker pool for parallel actuarial computations.
//
// # Process records in parallel
//
//	converter := rates.NewRateConverter(0.05)
//	records := []reader.CensusRecord{
//	    {Age: 30, SumAssured: 100000, Term: 20},
//	    {Age: 45, SumAssured: 200000, Term: 10},
//	}
//	totalPV := concurrency.ProcessBatch(records, converter, 8) // 8 goroutines
//
// # Use the worker pool directly
//
//	wp := concurrency.NewWorkerPool(4, converter)
//	totalPV = wp.ProcessBatch(records)
//
// Falls back to sequential processing for small batches (< 1000 records).
package concurrency
