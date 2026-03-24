package reader

import (
	"bufio"
	"errors"
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
// for each chunk. Returns the total record count and any error.
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

// StreamCensusWithPV reads a census CSV, calculates PV for each record using pvFn,
// and returns the total PV and record count. Uses parallel processing for large files.
func StreamCensusWithPV(filepath string, opts StreamOptions, pvFn func(sumAssured float64, term int) float64) (float64, int) {
	f, headerOffset, dataSize, delimiter, err := openCSV(filepath, opts.CSVOptions)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	if dataSize <= 0 {
		return 0, 0
	}

	chunkThreshold := int64(opts.ChunkSize)
	if chunkThreshold <= 0 {
		chunkThreshold = 100000
	}

	if dataSize < chunkThreshold*100 {
		return streamSequentialWithPV(f, opts, headerOffset, delimiter, pvFn)
	}

	numWorkers := max(min(runtime.NumCPU(), 8), 1)
	chunkSizeBytes := dataSize / int64(numWorkers)
	jobs := buildChunks(headerOffset, dataSize, numWorkers)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var totalPV float64
	var totalCount int

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(j csvJob) {
			defer wg.Done()

			localPV := 0.0
			localCount := 0
			limit := opts.Limit

			processChunk(f, j, chunkSizeBytes, headerOffset, func(line []byte) {
				if limit > 0 && localCount >= limit {
					return
				}
				record, err := parseCensusFastBytes(line, delimiter)
				if err == nil {
					localPV += pvFn(record.SumAssured, record.Term)
					localCount++
				} else if opts.OnParseError != nil {
					opts.OnParseError(-1, err)
				}
			})

			mu.Lock()
			totalPV += localPV
			totalCount += localCount
			mu.Unlock()
		}(jobs[w])
	}

	wg.Wait()

	return totalPV, totalCount
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
	chunkSizeBytes := dataSize / numWorkers
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

			err := processChunk(f, j, int64(chunkSizeBytes), headerOffset, func(line []byte) {
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

func streamSequentialWithPV(f *os.File, opts StreamOptions, headerOffset int64, delimiter byte, pvFn func(sumAssured float64, term int) float64) (float64, int) {
	_, err := f.Seek(headerOffset, io.SeekStart)
	if err != nil {
		return 0, 0
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024*1024), 64*1024*1024)

	totalPV := 0.0
	totalCount := 0
	limit := opts.Limit

	for scanner.Scan() {
		if limit > 0 && totalCount >= limit {
			break
		}

		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) == 0 {
			continue
		}

		record, err := parseCensusFastBytes(line, delimiter)
		if err != nil {
			continue
		}

		totalPV += pvFn(record.SumAssured, record.Term)
		totalCount++
	}

	return totalPV, totalCount
}

var _ error = errors.New("")
