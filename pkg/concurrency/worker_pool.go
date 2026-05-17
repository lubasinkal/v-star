package concurrency

import (
	"context"
	"runtime"
	"sync"
)

// WorkerPool processes items of type T in parallel using goroutines.
type WorkerPool[T any] struct {
	processFn func(T) float64
	wg        sync.WaitGroup
	workers   int
}

// NewWorkerPool creates a new worker pool with the specified number of workers.
func NewWorkerPool[T any](workers int, processFn func(T) float64) *WorkerPool[T] {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	return &WorkerPool[T]{
		workers:   workers,
		processFn: processFn,
	}
}

// ProcessBatch processes a slice of items in parallel and returns the total.
func (wp *WorkerPool[T]) ProcessBatch(items []T) float64 {
	if len(items) == 0 {
		return 0
	}
	if wp.workers == 1 || len(items) < 1000 {
		return wp.processSequential(items)
	}
	return wp.processParallel(items)
}

// ProcessBatchContext processes items with context cancellation support.
func (wp *WorkerPool[T]) ProcessBatchContext(ctx context.Context, items []T) (float64, error) {
	if len(items) == 0 {
		return 0, nil
	}
	if wp.workers == 1 || len(items) < 1000 {
		total := 0.0
		for _, item := range items {
			select {
			case <-ctx.Done():
				return total, ctx.Err()
			default:
				total += wp.processFn(item)
			}
		}
		return total, nil
	}
	return wp.processParallelContext(ctx, items)
}

func (wp *WorkerPool[T]) processSequential(items []T) float64 {
	total := 0.0
	for _, item := range items {
		total += wp.processFn(item)
	}
	return total
}

func (wp *WorkerPool[T]) processParallel(items []T) float64 {
	chunkSize := (len(items) + wp.workers - 1) / wp.workers
	results := make(chan float64, wp.workers)

	for w := 0; w < wp.workers; w++ {
		start := w * chunkSize
		end := min(start+chunkSize, len(items))
		if start >= len(items) {
			break
		}

		wp.wg.Add(1)
		go func(chunk []T) {
			defer wp.wg.Done()
			partial := 0.0
			for _, item := range chunk {
				partial += wp.processFn(item)
			}
			results <- partial
		}(items[start:end])
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

func (wp *WorkerPool[T]) processParallelContext(ctx context.Context, items []T) (float64, error) {
	chunkSize := (len(items) + wp.workers - 1) / wp.workers
	type result struct {
		value float64
		err   error
	}
	results := make(chan result, wp.workers)

	for w := 0; w < wp.workers; w++ {
		start := w * chunkSize
		end := min(start+chunkSize, len(items))
		if start >= len(items) {
			break
		}

		wp.wg.Add(1)
		go func(chunk []T) {
			defer wp.wg.Done()
			partial := 0.0
			for _, item := range chunk {
				select {
				case <-ctx.Done():
					results <- result{err: ctx.Err()}
					return
				default:
					partial += wp.processFn(item)
				}
			}
			results <- result{value: partial}
		}(items[start:end])
	}

	go func() {
		wp.wg.Wait()
		close(results)
	}()

	total := 0.0
	for r := range results {
		if r.err != nil {
			return total, r.err
		}
		total += r.value
	}
	return total, nil
}


