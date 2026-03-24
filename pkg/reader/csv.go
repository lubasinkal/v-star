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
	Header    bool // First row contains column names
	Limit     int  // Max rows to read (0 = unlimited)
	Delimiter byte // Column delimiter (default ',')
}

// StreamCSV reads any CSV file and calls fn for each row with the parsed fields.
// This is a generic reader — it does not know about specific record types.
// Uses parallel processing for large files.
// Note: This allocates strings for each field. For zero-allocation parsing, use StreamCSVRaw.
func StreamCSV(filepath string, opts CSVOptions, fn func(fields []string)) error {
	delimiter := opts.Delimiter
	if delimiter == 0 {
		delimiter = ','
	}

	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	fileSize := info.Size()
	headerOffset := int64(0)

	if opts.Header {
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		if scanner.Scan() {
			headerOffset = int64(len(scanner.Bytes())) + 1
		}
	}

	dataSize := fileSize - headerOffset
	if dataSize <= 0 {
		return nil
	}

	chunkSize := opts.Limit
	if chunkSize <= 0 {
		chunkSize = 100000
	}

	if dataSize < int64(chunkSize)*100 {
		return streamCSVSequentialStr(file, opts, headerOffset, delimiter, fn)
	}

	numWorkers := max(min(runtime.NumCPU(), 8), 1)
	chunkSizeBytes := dataSize / int64(numWorkers)
	overlap := 8192

	type job struct {
		id    int
		start int64
		end   int64
	}

	jobs := make([]job, numWorkers)
	for w := 0; w < numWorkers; w++ {
		start := headerOffset + int64(w)*chunkSizeBytes
		end := start + chunkSizeBytes
		hasOverlap := w < numWorkers-1

		if hasOverlap {
			end = end + int64(overlap)
			if end > fileSize {
				end = fileSize
			}
		}
		jobs[w] = job{id: w, start: start, end: end}
	}

	type batchResult struct {
		rows [][]string
	}
	results := make([]batchResult, numWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()

			bufSize := int(j.end - j.start)
			buf := make([]byte, bufSize)
			n, err := file.ReadAt(buf, j.start)
			if err != nil && err != io.EOF {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			buf = buf[:n]

			originalEnd := int(chunkSizeBytes)

			offset := 0
			if j.id > 0 && len(buf) > 0 && buf[0] != '\n' {
				i := bytes.IndexByte(buf, '\n')
				if i >= 0 {
					offset = i + 1
				}
			}

			estLines := max(originalEnd/50, 1024)
			batch := make([][]string, 0, estLines)

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

				fields := parseFieldsFast(line, delimiter)
				batch = append(batch, fields)
			}
			results[j.id].rows = batch
		}(jobs[w])
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	// Yield results in order
	count := 0
	limit := opts.Limit
	for _, batch := range results {
		for _, fields := range batch.rows {
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
	delimiter := opts.Delimiter
	if delimiter == 0 {
		delimiter = ','
	}

	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	fileSize := info.Size()
	headerOffset := int64(0)

	if opts.Header {
		scanner := bufio.NewScanner(file)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		if scanner.Scan() {
			headerOffset = int64(len(scanner.Bytes())) + 1
		}
	}

	dataSize := fileSize - headerOffset
	if dataSize <= 0 {
		return nil
	}

	chunkSize := opts.Limit
	if chunkSize <= 0 {
		chunkSize = 100000
	}

	if dataSize < int64(chunkSize)*100 {
		return streamCSVSequentialRaw(file, opts, headerOffset, delimiter, fn)
	}

	numWorkers := max(min(runtime.NumCPU(), 8), 1)
	chunkSizeBytes := dataSize / int64(numWorkers)
	overlap := 8192

	type job struct {
		id    int
		start int64
		end   int64
	}

	jobs := make([]job, numWorkers)
	for w := 0; w < numWorkers; w++ {
		start := headerOffset + int64(w)*chunkSizeBytes
		end := start + chunkSizeBytes
		hasOverlap := w < numWorkers-1

		if hasOverlap {
			end = end + int64(overlap)
			if end > fileSize {
				end = fileSize
			}
		}
		jobs[w] = job{id: w, start: start, end: end}
	}

	type batchResult struct {
		rows [][][]byte
	}
	results := make([]batchResult, numWorkers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()

			bufSize := int(j.end - j.start)
			buf := make([]byte, bufSize)
			n, err := file.ReadAt(buf, j.start)
			if err != nil && err != io.EOF {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
				return
			}
			buf = buf[:n]

			originalEnd := int(chunkSizeBytes)

			offset := 0
			if j.id > 0 && len(buf) > 0 && buf[0] != '\n' {
				i := bytes.IndexByte(buf, '\n')
				if i >= 0 {
					offset = i + 1
				}
			}

			estLines := max(originalEnd/50, 1024)
			batch := make([][][]byte, 0, estLines)

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

				fields := parseFieldsRaw(line, delimiter)
				batch = append(batch, fields)
			}
			results[j.id].rows = batch
		}(jobs[w])
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	// Yield results in order
	count := 0
	limit := opts.Limit
	for _, batch := range results {
		for _, fields := range batch.rows {
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
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		if scanner.Scan() {
			headerOffset = int64(len(scanner.Bytes())) + 1
		}
	}

	dataSize := fileSize - headerOffset
	if dataSize <= 0 {
		return 0, 0
	}

	chunkSize := opts.Limit
	if chunkSize <= 0 {
		chunkSize = 100000
	}

	if dataSize < int64(chunkSize)*100 {
		return streamCSVSequentialWithPV(f, opts, headerOffset, delimiter, pvFn)
	}

	numWorkers := max(min(runtime.NumCPU(), 8), 1)

	chunkSizeBytes := dataSize / int64(numWorkers)
	overlap := 8192
	type job struct {
		id    int
		start int64
		end   int64
	}

	jobs := make([]job, numWorkers)
	for w := 0; w < numWorkers; w++ {
		start := headerOffset + int64(w)*chunkSizeBytes
		end := start + chunkSizeBytes
		hasOverlap := w < numWorkers-1

		if hasOverlap {
			end = end + int64(overlap)
			if end > fileSize {
				end = fileSize
			}
		}
		jobs[w] = job{id: w, start: start, end: end}
	}

	var wg sync.WaitGroup
	var totalPV float64
	var totalCount int
	var mu sync.Mutex

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(j job) {
			defer wg.Done()

			bufSize := int(j.end - j.start)
			buf := make([]byte, bufSize)
			n, err := f.ReadAt(buf, j.start)
			if err != nil && err != io.EOF {
				return
			}
			buf = buf[:n]

			originalEnd := int(chunkSizeBytes)

			offset := 0
			if j.id > 0 && len(buf) > 0 && buf[0] != '\n' {
				i := bytes.IndexByte(buf, '\n')
				if i >= 0 {
					offset = i + 1
				}
			}

			localPV := 0.0
			localCount := 0
			limit := opts.Limit

			processedBytes := offset
			for processedBytes < originalEnd && processedBytes < len(buf) {
				if limit > 0 && localCount >= limit {
					break
				}
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

				record, err := parseCensusFastBytes(line, delimiter)
				if err == nil {
					localPV += pvFn(record.SumAssured, record.Term)
					localCount++
				}
			}

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

// streamSequential is the fallback for small files or single-worker mode.
func streamSequential(f *os.File, opts CSVOptions, delimiter byte, fn func([]string)) error {
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
		fields := parseFields(line, delimiter)
		fn(fields)
		count++
	}

	return scanner.Err()
}

func streamSequentialWithHeader(f *os.File, opts CSVOptions, headerOffset int64, delimiter byte, fn func([]string)) error {
	if headerOffset > 0 {
		_, err := f.Seek(headerOffset, io.SeekStart)
		if err != nil {
			return err
		}
	}
	return streamSequential(f, opts, delimiter, fn)
}

// parseFields splits a CSV line into fields in a single pass.
// Handles quoted fields (double-quotes). Returns field slices into the original buffer.
// The caller must copy field values if they need to retain them beyond the callback.
func parseFields(line []byte, delimiter byte) []string {
	// Fast path: check for quotes in a single scan
	hasQuotes := slices.Contains(line, '"')

	if !hasQuotes {
		return parseFieldsFast(line, delimiter)
	}

	// Slow path: handle quoted fields
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

// parseFieldsQuoted handles lines with quoted fields
func parseFieldsQuoted(line []byte, delimiter byte) []string {
	// Count fields (respecting quoted delimiters)
	count := 0
	inQuotes := false
	for i := range line {
		if line[i] == '"' {
			inQuotes = !inQuotes
		} else if line[i] == delimiter && !inQuotes {
			count++
		}
	}

	fields := make([]string, count+1)
	start := 0
	idx := 0
	inQuotes = false

	for i := range line {
		c := line[i]
		if c == '"' {
			inQuotes = !inQuotes
		} else if c == delimiter && !inQuotes {
			fields[idx] = unsafe.String(unsafe.SliceData(line[start:i]), len(line[start:i]))
			idx++
			start = i + 1
		}
	}
	// Last field
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
