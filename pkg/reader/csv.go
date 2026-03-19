package reader

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"runtime"
	"sync"
)

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
	var headers []string
	if opts.Header {
		if !scanner.Scan() {
			return scanner.Err()
		}
		headers = parseFields(scanner.Bytes(), delimiter)
		_ = headers // caller can use GetHeaders if needed
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
// Each goroutine reads a chunk of the file, finds line boundaries, and parses fields.
// fn is called concurrently — it must be safe for concurrent use or handle its own sync.
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
	results := make(chan []string, 10000)
	var wg sync.WaitGroup
	var parseErr error
	var errMu sync.Mutex

	for w := 0; w < numWorkers; w++ {
		start := headerOffset + int64(w)*chunkSize
		end := start + chunkSize
		if w == numWorkers-1 {
			end = fileSize
		}

		wg.Add(1)
		go func(start, end int64) {
			defer wg.Done()

			bufSize := end - start
			buf := make([]byte, bufSize)
			n, err := f.ReadAt(buf, start)
			if err != nil && err != io.EOF {
				errMu.Lock()
				if parseErr == nil {
					parseErr = err
				}
				errMu.Unlock()
				return
			}
			buf = buf[:n]

			// Adjust start: skip partial line at the beginning
			if start > headerOffset {
				idx := bytes.IndexByte(buf, '\n')
				if idx < 0 {
					return
				}
				buf = buf[idx+1:]
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

			// Parse lines in this chunk
			for len(buf) > 0 {
				idx := bytes.IndexByte(buf, '\n')
				if idx < 0 {
					// Last line without trailing newline
					if len(buf) > 0 {
						fields := parseFields(buf, delimiter)
						results <- fields
					}
					break
				}
				line := buf[:idx]
				buf = buf[idx+1:]

				// Skip empty lines
				if len(line) == 0 {
					continue
				}
				// Trim \r
				if line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}

				fields := parseFields(line, delimiter)
				results <- fields
			}
		}(start, end)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	count := 0
	limit := opts.Limit
	for fields := range results {
		if limit > 0 && count >= limit {
			break
		}
		fn(fields)
		count++
	}

	return parseErr
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
	fields := make([]string, 0, 8)
	start := 0
	inQuotes := false

	for i := 0; i < len(line); i++ {
		c := line[i]
		if c == '"' {
			inQuotes = !inQuotes
		} else if c == delimiter && !inQuotes {
			fields = append(fields, string(line[start:i]))
			start = i + 1
		}
	}
	// Last field
	fields = append(fields, string(line[start:]))

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
