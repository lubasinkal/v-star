package reader

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
)

type ChunkProcessor func(chunk []CensusRecord) error

type StreamOptions struct {
	Header    bool
	Limit     int
	Delimiter byte
	ChunkSize int
	Workers   int
}

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
	if numWorkers > 8 {
		numWorkers = 8
	}

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

func StreamCensusWithPV(filepath string, opts StreamOptions, pvFn func(sumAssured float64, term int) float64) (float64, int) {
	delimiter := opts.Delimiter
	if delimiter == 0 {
		delimiter = ','
	}

	f, err := os.Open(filepath)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return 0, 0
	}
	fileSize := info.Size()
	headerOffset := int64(0)

	if opts.Header {
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024), 1024)
		if scanner.Scan() {
			headerOffset = int64(len(scanner.Bytes())) + 1
		}
	}

	dataSize := fileSize - headerOffset
	if dataSize <= 0 {
		return 0, 0
	}

	chunkSize := opts.ChunkSize
	if chunkSize <= 0 {
		chunkSize = 100000
	}

	if dataSize < int64(chunkSize)*100 {
		return streamSequentialWithPV(f, opts, headerOffset, delimiter, pvFn)
	}

	numWorkers := min(runtime.NumCPU(), 8)

	chunkSizeBytes := dataSize / int64(numWorkers)
	type job struct {
		id    int
		start int64
		end   int64
	}

	jobs := make([]job, numWorkers)
	for w := 0; w < numWorkers; w++ {
		start := headerOffset + int64(w)*chunkSizeBytes
		end := start + chunkSizeBytes
		if w == numWorkers-1 {
			end = fileSize
		}
		jobs[w] = job{id: w, start: start, end: end}
	}

	var wg sync.WaitGroup
	var totalPV float64
	var totalCount int
	var countMu sync.Mutex

	processJob := func(j job) {
		overlap := int64(8192)
		readStart := j.start
		readEnd := j.end

		if j.end < fileSize {
			readEnd = j.end + overlap
			if readEnd > fileSize {
				readEnd = fileSize
			}
		}

		buf := make([]byte, readEnd-readStart)
		n, err := f.ReadAt(buf, readStart)
		if err != nil && err != io.EOF {
			return
		}
		buf = buf[:n]

		offset := 0
		if j.id > 0 && len(buf) > 0 && buf[0] != '\n' {
			newlineIdx := bytes.IndexByte(buf, '\n')
			if newlineIdx >= 0 {
				offset = newlineIdx + 1
			}
		}

		originalEnd := int(j.end - j.start)
		localPV := 0.0
		localCount := 0
		limit := opts.Limit

		processedBytes := offset

		for processedBytes < originalEnd && processedBytes < len(buf) {
			if limit > 0 && localCount >= limit {
				break
			}
			newlineIdx := bytes.IndexByte(buf[processedBytes:], '\n')
			var line []byte
			if newlineIdx < 0 {
				break
			} else {
				line = buf[processedBytes : processedBytes+newlineIdx]
				processedBytes += newlineIdx + 1
			}

			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}

			if len(line) == 0 {
				continue
			}

			if r, err := parseCensusFastBytes(line, delimiter); err == nil {
				localPV += pvFn(r.SumAssured, r.Term)
				localCount++
			}
		}

		countMu.Lock()
		totalPV += localPV
		totalCount += localCount
		countMu.Unlock()
	}

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()
			processJob(j)
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
	overlap := 8192

	type job struct {
		id    int
		start int64
		end   int64
	}

	jobs := make([]job, numWorkers)
	for w := range numWorkers {
		start := headerOffset + int64(w)*int64(chunkSizeBytes)
		end := start + int64(chunkSizeBytes)
		hasOverlap := w < numWorkers-1

		if hasOverlap {
			end = end + int64(overlap)
			if end > int64(dataSize)+headerOffset {
				end = int64(dataSize) + headerOffset
			}
		}
		jobs[w] = job{id: w, start: start, end: end}
	}

	results := make([][]CensusRecord, numWorkers)
	var wg sync.WaitGroup
	var totalCount int32
	var firstErr error

	processJob := func(job job) ([]CensusRecord, error) {
		readStart := job.start
		readEnd := job.end

		bufSize := int(readEnd - readStart)
		buf := make([]byte, bufSize)
		n, err := f.ReadAt(buf, readStart)
		if err != nil && err != io.EOF {
			return nil, err
		}
		buf = buf[:n]

		originalEnd := int(chunkSizeBytes)

		offset := 0
		if job.id > 0 && len(buf) > 0 && buf[0] != '\n' {
			i := bytes.IndexByte(buf, '\n')
			if i >= 0 {
				offset = i + 1
			}
		}

		records := make([]CensusRecord, 0, chunkSize)
		processedBytes := offset
		for processedBytes < originalEnd && processedBytes < len(buf) {
			i := bytes.IndexByte(buf[processedBytes:], '\n')
			var line []byte
			if i < 0 {
				break
			}
			line = buf[processedBytes : processedBytes+i]
			processedBytes += i + 1

			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}

			if len(line) == 0 {
				continue
			}

			if r, err := parseCensusFastBytes(line, delimiter); err == nil {
				records = append(records, r)
			}
		}

		return records, nil
	}

	for w := range numWorkers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			records, err := processJob(jobs[idx])
			if err != nil && firstErr == nil {
				firstErr = err
				return
			}
			results[idx] = records
			atomic.AddInt32(&totalCount, int32(len(records)))
		}(w)
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
