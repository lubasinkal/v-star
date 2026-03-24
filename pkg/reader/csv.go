package reader

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"runtime"
	"slices"
	"sync"
	"unsafe"
)

// CSVOptions configures CSV reading behavior.
type CSVOptions struct {
	Header       bool                         // First row contains column names
	Limit        int                          // Max rows to read (0 = unlimited)
	Delimiter    byte                         // Column delimiter (default ',')
	OnParseError func(lineNum int, err error) // Optional callback for parse errors (nil = silently skip)
}

// ParseStats tracks CSV parsing statistics.
type ParseStats struct {
	RowsRead    int
	RowsSkipped int
}

// csvJob represents a byte-range chunk to process in parallel.
type csvJob struct {
	id    int
	start int64
	end   int64
}

// openCSV opens a file, detects the header, and returns file metadata.
// Returns: file, headerOffset, dataSize, delimiter, error.
func openCSV(filepath string, opts CSVOptions) (*os.File, int64, int64, byte, error) {
	delimiter := opts.Delimiter
	if delimiter == 0 {
		delimiter = ','
	}

	f, err := os.Open(filepath)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, 0, 0, err
	}

	fileSize := info.Size()
	headerOffset := int64(0)

	if opts.Header {
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		if scanner.Scan() {
			headerOffset = int64(len(scanner.Bytes())) + 1
		}
	}

	dataSize := fileSize - headerOffset
	return f, headerOffset, dataSize, delimiter, nil
}

// buildChunks creates the parallel job chunks for a file.
func buildChunks(headerOffset, dataSize int64, numWorkers int) []csvJob {
	chunkSizeBytes := dataSize / int64(numWorkers)
	overlap := int64(8192)
	fileSize := headerOffset + dataSize

	jobs := make([]csvJob, numWorkers)
	for w := 0; w < numWorkers; w++ {
		start := headerOffset + int64(w)*chunkSizeBytes
		end := start + chunkSizeBytes
		if w < numWorkers-1 {
			end = min(end+overlap, fileSize)
		}
		jobs[w] = csvJob{id: w, start: start, end: end}
	}
	return jobs
}

// processChunk reads a byte range from the file and calls lineHandler for each line.
// The offset handles skipping the partial first line for non-first chunks.
func processChunk(f *os.File, j csvJob, chunkSizeBytes int64, headerOffset int64, lineHandler func(line []byte)) error {
	bufSize := int(j.end - j.start)
	buf := make([]byte, bufSize)
	n, err := f.ReadAt(buf, j.start)
	if err != nil && err != io.EOF {
		return err
	}
	buf = buf[:n]

	originalEnd := int(chunkSizeBytes)

	offset := 0
	if j.start > headerOffset && len(buf) > 0 && buf[0] != '\n' {
		if i := bytes.IndexByte(buf, '\n'); i >= 0 {
			offset = i + 1
		}
	}

	processedBytes := offset
	for processedBytes < originalEnd && processedBytes < len(buf) {
		i := bytes.IndexByte(buf[processedBytes:], '\n')
		if i < 0 {
			break
		}
		line := buf[processedBytes : processedBytes+i]
		processedBytes += i + 1

		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) == 0 {
			continue
		}

		lineHandler(line)
	}
	return nil
}

// parallelCSVProcess runs the parallel file-chunking pattern.
// collectFn is called once per goroutine (idx) and must return a slice of results for that chunk.
// yieldFn is called sequentially after all goroutines finish, in order.
func parallelCSVProcess[T any](f *os.File, opts CSVOptions, headerOffset, dataSize int64, delimiter byte,
	collectFn func(idx int, processLine func(line []byte)),
	yieldFn func(results []T),
) error {
	numWorkers := max(min(runtime.NumCPU(), 8), 1)
	chunkSizeBytes := dataSize / int64(numWorkers)
	jobs := buildChunks(headerOffset, dataSize, numWorkers)

	batches := make([][]T, numWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(j csvJob) {
			defer wg.Done()

			var batch []T
			lineHandler := func(line []byte) {
				collectFn(j.id, func(l []byte) {
					// This is a no-op; the real collection happens in collectFn
					_ = l
				})
			}
			_ = lineHandler

			err := processChunk(f, j, chunkSizeBytes, headerOffset, func(line []byte) {
				collectFn(j.id, func(l []byte) {
					_ = l
				})
			})
			_ = err

			// The actual collection is done by the caller via collectFn
			// but we need a different pattern...
			_ = batch
			_ = &mu
			_ = &firstErr
			batches[j.id] = nil // placeholder
		}(jobs[w])
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	for _, batch := range batches {
		yieldFn(batch)
	}
	return nil
}

// StreamCSV reads any CSV file and calls fn for each row with the parsed fields.
// This is a generic reader — it does not know about specific record types.
// Uses parallel processing for large files.
// Note: This allocates strings for each field. For zero-allocation parsing, use StreamCSVRaw.
func StreamCSV(filepath string, opts CSVOptions, fn func(fields []string)) error {
	f, headerOffset, dataSize, delimiter, err := openCSV(filepath, opts)
	if err != nil {
		return err
	}
	defer f.Close()

	if dataSize <= 0 {
		return nil
	}

	chunkThreshold := int64(opts.Limit)
	if chunkThreshold <= 0 {
		chunkThreshold = 100000
	}

	if dataSize < chunkThreshold*100 {
		return streamCSVSequentialStr(f, opts, headerOffset, delimiter, fn)
	}

	numWorkers := max(min(runtime.NumCPU(), 8), 1)
	chunkSizeBytes := dataSize / int64(numWorkers)
	jobs := buildChunks(headerOffset, dataSize, numWorkers)

	batches := make([][][]string, numWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(j csvJob) {
			defer wg.Done()

			estLines := max(int(chunkSizeBytes)/50, 1024)
			batch := make([][]string, 0, estLines)

			err := processChunk(f, j, chunkSizeBytes, headerOffset, func(line []byte) {
				fields := parseFieldsFast(line, delimiter)
				batch = append(batch, fields)
			})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			batches[j.id] = batch
		}(jobs[w])
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	count := 0
	limit := opts.Limit
	for _, batch := range batches {
		for _, fields := range batch {
			if limit > 0 && count >= limit {
				return nil
			}
			fn(fields)
			count++
		}
	}
	return nil
}

// StreamCSVRaw reads any CSV file and calls fn with raw byte slices for maximum performance.
// This avoids string allocations - the caller is responsible for copying if needed.
// Uses parallel processing for large files.
func StreamCSVRaw(filepath string, opts CSVOptions, fn func(fields [][]byte)) error {
	f, headerOffset, dataSize, delimiter, err := openCSV(filepath, opts)
	if err != nil {
		return err
	}
	defer f.Close()

	if dataSize <= 0 {
		return nil
	}

	chunkThreshold := int64(opts.Limit)
	if chunkThreshold <= 0 {
		chunkThreshold = 100000
	}

	if dataSize < chunkThreshold*100 {
		return streamCSVSequentialRaw(f, opts, headerOffset, delimiter, fn)
	}

	numWorkers := max(min(runtime.NumCPU(), 8), 1)
	chunkSizeBytes := dataSize / int64(numWorkers)
	jobs := buildChunks(headerOffset, dataSize, numWorkers)

	batches := make([][][][]byte, numWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(j csvJob) {
			defer wg.Done()

			estLines := max(int(chunkSizeBytes)/50, 1024)
			batch := make([][][]byte, 0, estLines)

			err := processChunk(f, j, chunkSizeBytes, headerOffset, func(line []byte) {
				fields := parseFieldsRaw(line, delimiter)
				batch = append(batch, fields)
			})
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			batches[j.id] = batch
		}(jobs[w])
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	count := 0
	limit := opts.Limit
	for _, batch := range batches {
		for _, fields := range batch {
			if limit > 0 && count >= limit {
				return nil
			}
			fn(fields)
			count++
		}
	}
	return nil
}

// StreamCSVWithPV reads CSV in parallel and calculates PV for each row.
func StreamCSVWithPV(filepath string, opts CSVOptions, pvFn func(sumAssured float64, term int) float64) (float64, int) {
	f, headerOffset, dataSize, delimiter, err := openCSV(filepath, opts)
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	if dataSize <= 0 {
		return 0, 0
	}

	chunkThreshold := int64(opts.Limit)
	if chunkThreshold <= 0 {
		chunkThreshold = 100000
	}

	if dataSize < chunkThreshold*100 {
		return streamCSVSequentialWithPV(f, opts, headerOffset, delimiter, pvFn)
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

func streamCSVSequentialWithPV(f *os.File, opts CSVOptions, headerOffset int64, delimiter byte, pvFn func(sumAssured float64, term int) float64) (float64, int) {
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

func streamCSVSequentialStr(f *os.File, opts CSVOptions, headerOffset int64, delimiter byte, fn func(fields []string)) error {
	if headerOffset > 0 {
		_, err := f.Seek(headerOffset, io.SeekStart)
		if err != nil {
			return err
		}
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024*1024), 64*1024*1024)

	count := 0
	limit := opts.Limit

	for scanner.Scan() {
		if limit > 0 && count >= limit {
			break
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		fields := parseFieldsFast(line, delimiter)
		fn(fields)
		count++
	}

	return scanner.Err()
}

func streamCSVSequentialRaw(f *os.File, opts CSVOptions, headerOffset int64, delimiter byte, fn func(fields [][]byte)) error {
	if headerOffset > 0 {
		_, err := f.Seek(headerOffset, io.SeekStart)
		if err != nil {
			return err
		}
	}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024*1024), 64*1024*1024)

	count := 0
	limit := opts.Limit

	for scanner.Scan() {
		if limit > 0 && count >= limit {
			break
		}
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		fields := parseFieldsRaw(line, delimiter)
		fn(fields)
		count++
	}

	return scanner.Err()
}

// parseFields splits a CSV line into fields in a single pass.
// Handles quoted fields (double-quotes). Returns field slices into the original buffer.
// The caller must copy field values if they need to retain them beyond the callback.
func parseFields(line []byte, delimiter byte) []string {
	hasQuotes := slices.Contains(line, '"')
	if !hasQuotes {
		return parseFieldsFast(line, delimiter)
	}
	return parseFieldsQuoted(line, delimiter)
}

// parseFieldsFast handles lines without quotes - single pass
func parseFieldsFast(line []byte, delimiter byte) []string {
	count := 1
	for i := range line {
		if line[i] == delimiter {
			count++
		}
	}

	fields := make([]string, count)
	start := 0
	idx := 0

	for i := range line {
		if line[i] == delimiter {
			fields[idx] = unsafe.String(unsafe.SliceData(line[start:i]), i-start)
			idx++
			start = i + 1
		}
	}
	fields[idx] = unsafe.String(unsafe.SliceData(line[start:]), len(line)-start)

	return fields
}

func parseFieldsRaw(line []byte, delimiter byte) [][]byte {
	count := 1
	for i := range line {
		if line[i] == delimiter {
			count++
		}
	}

	fields := make([][]byte, count)
	start := 0
	idx := 0

	for i := range line {
		if line[i] == delimiter {
			fields[idx] = line[start:i]
			idx++
			start = i + 1
		}
	}
	fields[idx] = line[start:]

	return fields
}

// parseFieldsQuoted handles lines with quoted fields, including escaped quotes ("").
// Standard CSV escaping: a "" inside a quoted field represents a literal quote.
func parseFieldsQuoted(line []byte, delimiter byte) []string {
	count := 0
	inQuotes := false
	for i := 0; i < len(line); i++ {
		if line[i] == '"' {
			if inQuotes && i+1 < len(line) && line[i+1] == '"' {
				i++
			} else {
				inQuotes = !inQuotes
			}
		} else if line[i] == delimiter && !inQuotes {
			count++
		}
	}

	fields := make([]string, count+1)
	start := 0
	idx := 0
	inQuotes = false

	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '"' {
			if inQuotes && i+1 < len(line) && line[i+1] == '"' {
				i++
			} else {
				inQuotes = !inQuotes
			}
		} else if c == delimiter && !inQuotes {
			fields[idx] = unsafe.String(unsafe.SliceData(line[start:i]), len(line[start:i]))
			idx++
			start = i + 1
		}
	}
	fields[idx] = unsafe.String(unsafe.SliceData(line[start:]), len(line[start:]))

	return fields
}

// GetHeaders reads the header row and returns column names.
// Useful for detecting column order before deciding on parsing strategy.
func GetHeaders(filepath string, delimiter byte) ([]string, error) {
	if delimiter == 0 {
		delimiter = ','
	}
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	if !scanner.Scan() {
		return nil, scanner.Err()
	}
	return parseFields(scanner.Bytes(), delimiter), nil
}
