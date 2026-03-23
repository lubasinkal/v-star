package reader

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"runtime"
	"sync"
	"unsafe"
)

var fieldSlicePool = sync.Pool{
	New: func() interface{} {
		return make([]string, 0, 32)
	},
}

// CSVOptions configures CSV reading behavior.
type CSVOptions struct {
	Header    bool // First row contains column names
	Limit     int  // Max rows to read (0 = unlimited)
	Delimiter byte // Column delimiter (default ',')
}

// StreamCSV reads any CSV file and calls fn for each row with the parsed fields.
// This is a generic reader — it does not know about specific record types.
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

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024*1024), 64*1024*1024)

	// Read header if present
	if opts.Header {
		if !scanner.Scan() {
			return scanner.Err()
		}
	}

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

	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	chunkSize := dataSize / int64(numWorkers)
	type job struct {
		start int64
		end   int64
	}

	jobs := make([]job, numWorkers)
	for w := 0; w < numWorkers; w++ {
		start := headerOffset + int64(w)*chunkSize
		end := start + chunkSize
		if w == numWorkers-1 {
			end = fileSize
		}
		jobs[w] = job{start: start, end: end}
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

			if j.start > headerOffset {
				i := bytes.IndexByte(buf, '\n')
				if i >= 0 {
					buf = buf[i+1:]
				}
			}

			if j.end < fileSize {
				lastNL := bytes.LastIndexByte(buf, '\n')
				if lastNL >= 0 {
					buf = buf[:lastNL]
				}
			}

			localPV := 0.0
			localCount := 0
			limit := opts.Limit

			for len(buf) > 0 {
				if limit > 0 && localCount >= limit {
					break
				}
				i := bytes.IndexByte(buf, '\n')
				var line []byte
				if i < 0 {
					line = buf
					buf = nil
				} else {
					line = buf[:i]
					buf = buf[i+1:]
				}

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

// parseFields splits a CSV line into fields in a single pass.
// Handles quoted fields (double-quotes). Returns field slices into the original buffer.
// The caller must copy field values if they need to retain them beyond the callback.
func parseFields(line []byte, delimiter byte) []string {
	// Fast path: check for quotes in a single scan
	hasQuotes := false
	for i := 0; i < len(line); i++ {
		if line[i] == '"' {
			hasQuotes = true
			break
		}
	}

	if !hasQuotes {
		return parseFieldsFast(line, delimiter)
	}

	// Slow path: handle quoted fields
	return parseFieldsQuoted(line, delimiter)
}

// parseFieldsFast handles lines without quotes - single pass, no allocations for counting
func parseFieldsFast(line []byte, delimiter byte) []string {
	// Count fields in one pass
	count := 0
	for i := 0; i < len(line); i++ {
		if line[i] == delimiter {
			count++
		}
	}

	// Pre-allocate result slice
	fields := make([]string, count+1)
	start := 0
	idx := 0

	for i := 0; i < len(line); i++ {
		if line[i] == delimiter {
			fields[idx] = unsafe.String(unsafe.SliceData(line[start:i]), len(line[start:i]))
			idx++
			start = i + 1
		}
	}
	// Last field
	fields[idx] = unsafe.String(unsafe.SliceData(line[start:]), len(line[start:]))

	return fields
}

// parseFieldsQuoted handles lines with quoted fields
func parseFieldsQuoted(line []byte, delimiter byte) []string {
	// Count fields (respecting quoted delimiters)
	count := 0
	inQuotes := false
	for i := 0; i < len(line); i++ {
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

	for i := 0; i < len(line); i++ {
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
