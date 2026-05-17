package reader

import (
	"bufio"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
)

// ChunkProcessor is a callback that processes a chunk of CensusRecords.
// Return a non-nil error to abort processing.
type ChunkProcessor func(chunk []CensusRecord) error

// StreamOptions configures chunked parallel streaming behavior.
type StreamOptions struct {
	CSVOptions
	ChunkSize int
	Workers   int
}

// StreamCensusChunked reads a census CSV file in parallel chunks, calling processFn
// for each chunk. Returns total record count and any error.
// Use this when you need to process records in batches (e.g., database inserts, batch API calls).
// For simpler row-by-row processing, use StreamCensus.
func StreamCensusChunked(filepath string, opts StreamOptions, processFn ChunkProcessor) (int, error) {
	delimiter := opts.Delimiter
	if delimiter == 0 {
		delimiter = ','
	}

	chunkSize := opts.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 100000
	}

	numWorkers := opts.Workers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	numWorkers = min(numWorkers, 8)

	f, err := os.Open(filepath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	fileSize := info.Size()
	headerOffset := int64(0)

	if opts.Header {
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024), 1024)
		if scanner.Scan() {
			headerOffset = int64(len(scanner.Bytes())) + 1
		}
		if scanner.Err() != nil {
			return 0, scanner.Err()
		}
	}

	dataSize := fileSize - headerOffset
	if dataSize <= 0 {
		return 0, nil
	}

	if dataSize < int64(chunkSize)*1000 || numWorkers == 1 {
		return streamSequentialChunked(f, opts, headerOffset, delimiter, processFn)
	}

	return streamParallelChunked(f, opts, headerOffset, delimiter, processFn, numWorkers, int(dataSize))
}

func streamSequentialChunked(f *os.File, opts StreamOptions, headerOffset int64, delimiter byte, processFn ChunkProcessor) (int, error) {
	_, err := f.Seek(headerOffset, io.SeekStart)
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024*1024), 64*1024*1024)

	chunk := make([]CensusRecord, 0, opts.ChunkSize)
	totalCount := 0
	limit := opts.Limit

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		record, err := parseCensusFastBytes(line, delimiter)
		if err != nil {
			continue
		}

		chunk = append(chunk, record)
		totalCount++

		if len(chunk) >= opts.ChunkSize {
			if err := processFn(chunk); err != nil {
				return totalCount, err
			}
			chunk = chunk[:0]
		}

		if limit > 0 && totalCount >= limit {
			break
		}
	}

	if len(chunk) > 0 {
		if err := processFn(chunk); err != nil {
			return totalCount, err
		}
	}

	return totalCount, scanner.Err()
}

func streamParallelChunked(f *os.File, opts StreamOptions, headerOffset int64, delimiter byte, processFn ChunkProcessor, numWorkers int, dataSize int) (int, error) {
	chunkSize := opts.ChunkSize
	jobs := buildChunks(headerOffset, int64(dataSize), numWorkers)

	results := make([][]CensusRecord, numWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalCount int32
	var firstErr error

	for w := range numWorkers {
		wg.Add(1)
		go func(j csvJob) {
			defer wg.Done()

			records := make([]CensusRecord, 0, chunkSize)

			err := processChunk(f, j, headerOffset, func(line []byte) {
				if r, err := parseCensusFastBytes(line, delimiter); err == nil {
					records = append(records, r)
				} else if opts.OnParseError != nil {
					opts.OnParseError(-1, err)
				}
			})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			results[j.id] = records
			atomic.AddInt32(&totalCount, int32(len(records)))
		}(jobs[w])
	}

	wg.Wait()

	if firstErr != nil {
		return int(totalCount), firstErr
	}

	limit := opts.Limit
	for _, records := range results {
		toProcess := records
		if limit > 0 {
			remaining := limit - int(totalCount)
			if remaining <= 0 {
				break
			}
			if len(toProcess) > remaining {
				toProcess = toProcess[:remaining]
			}
		}

		if err := processFn(toProcess); err != nil {
			return int(totalCount), err
		}

		if limit > 0 && int(totalCount) >= limit {
			break
		}
	}

	return int(totalCount), nil
}
