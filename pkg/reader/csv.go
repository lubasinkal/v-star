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

// StreamCSVParallel reads a CSV file using multiple goroutines for maximum throughput.
// Each goroutine reads its own chunk via ReadAt (concurrent disk I/O), finds line
// boundaries, parses fields, and stores results in a batch. The main goroutine
// yields results in order. fn must be safe for concurrent use or handle its own sync.
func StreamCSVParallel(filepath string, opts CSVOptions, fn func(fields []string)) error {
	delimiter := opts.Delimiter
	if delimiter == 0 {
		delimiter = ','
	}

	f, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}
	fileSize := info.Size()

	// Read and skip header
	headerOffset := int64(0)
	if opts.Header {
		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
		if scanner.Scan() {
			headerOffset = int64(len(scanner.Bytes())) + 1 // +1 for newline
		}
	}

	dataSize := fileSize - headerOffset
	if dataSize <= 0 {
		return nil
	}

	numWorkers := runtime.NumCPU()
	if numWorkers > 8 {
		numWorkers = 8
	}
	if numWorkers < 1 {
		numWorkers = 1
	}

	// For small files or single worker, use sequential path
	if dataSize < 10*1024*1024 || numWorkers == 1 {
		f.Seek(headerOffset, io.SeekStart)
		return streamSequential(f, opts, delimiter, fn)
	}

	chunkSize := dataSize / int64(numWorkers)

	type batchResult struct {
		rows [][]string
	}
	batchResults := make([]batchResult, numWorkers)
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		start := headerOffset + int64(w)*chunkSize
		end := start + chunkSize
		if w == numWorkers-1 {
			end = fileSize
		}

		wg.Add(1)
		go func(idx int, start, end int64) {
			defer wg.Done()

			// Read chunk via ReadAt (concurrent-safe, no mutex needed)
			bufSize := int(end - start)
			buf := make([]byte, bufSize)
			n, err := f.ReadAt(buf, start)
			if err != nil && err != io.EOF {
				return
			}
			buf = buf[:n]

			// Skip partial line at start (except first chunk)
			if start > headerOffset {
				i := bytes.IndexByte(buf, '\n')
				if i < 0 {
					return
				}
				buf = buf[i+1:]
			}

			// Trim trailing partial line
			if end < fileSize {
				lastNL := bytes.LastIndexByte(buf, '\n')
				if lastNL >= 0 {
					buf = buf[:lastNL]
				} else {
					buf = nil
				}
			}

			// Pre-allocate batch based on estimated line count (~50 bytes/line)
			estLines := len(buf) / 50
			if estLines < 1024 {
				estLines = 1024
			}
			rows := make([][]string, 0, estLines)

			// Parse lines in this chunk
			for len(buf) > 0 {
				i := bytes.IndexByte(buf, '\n')
				if i < 0 {
					// Last line without trailing newline
					if len(buf) > 0 {
						line := buf
						if line[len(line)-1] == '\r' {
							line = line[:len(line)-1]
						}
						if len(line) > 0 {
							fields := parseFields(line, delimiter)
							rows = append(rows, fields)
						}
					}
					break
				}

				line := buf[:i]
				buf = buf[i+1:]

				if len(line) == 0 {
					continue
				}
				if line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				if len(line) == 0 {
					continue
				}

				fields := parseFields(line, delimiter)
				rows = append(rows, fields)
			}

			batchResults[idx].rows = rows
		}(w, start, end)
	}

	wg.Wait()

	// Yield results in order
	count := 0
	limit := opts.Limit
	for _, batch := range batchResults {
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
