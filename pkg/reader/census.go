package reader

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// ColumnMap maps CSV column names to their indices.
// ColumnMap maps CSV column names to their positional indices.
type ColumnMap map[string]int

// StreamCensus reads a census CSV file and yields CensusRecords.
// Uses fast byte-level parallel path when columns match default order,
// falls back to generic reader for non-standard layouts.
func StreamCensus(filepath string, opts CSVOptions, fn func(CensusRecord)) error {
	delimiter := opts.Delimiter
	if delimiter == 0 {
		delimiter = ','
	}

	// Detect column order from header
	var colMap ColumnMap
	var useFastPath bool
	headerOffset := int64(0)

	if opts.Header {
		headers, err := GetHeaders(filepath, delimiter)
		if err != nil {
			return err
		}
		colMap = buildColumnMap(headers)
		useFastPath = isDefaultColumnOrder(colMap)

		// Calculate header offset
		f, err := os.Open(filepath)
		if err != nil {
			return err
		}
		scanner := bufio.NewScanner(f)
		if scanner.Scan() {
			headerOffset = int64(len(scanner.Bytes())) + 1
		}
		f.Close()
	} else {
		useFastPath = true
		colMap = defaultColumnMap()
	}

	if useFastPath {
		return streamCensusFastParallel(filepath, opts, headerOffset, delimiter, fn)
	}
	return streamCensusFlex(filepath, opts, delimiter, colMap, fn)
}

// streamCensusFastParallel reads CensusRecords using parallel chunk reading
// with zero-alloc byte-level parsing. Target: 7M+ rows/sec.
func streamCensusFastParallel(filepath string, opts CSVOptions, headerOffset int64, delimiter byte, fn func(CensusRecord)) error {
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
	dataSize := fileSize - headerOffset

	if dataSize <= 0 {
		return nil
	}

	numWorkers := max(min(runtime.NumCPU(), 8), 1)

	// For small files, use sequential scanner
	if dataSize < 10*1024*1024 || numWorkers == 1 {
		f.Seek(headerOffset, io.SeekStart)
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
			record, err := parseCensusFastBytes(line, delimiter)
			if err == nil {
				fn(record)
				count++
			} else if opts.OnParseError != nil {
				opts.OnParseError(count, err)
			}
		}
		return scanner.Err()
	}

	// Parallel: each goroutine reads its chunk via ReadAt (parallel-safe)
	chunkSize := dataSize / int64(numWorkers)
	overlap := int64(8192)
	type batchResult struct {
		records []CensusRecord
	}
	results := make([]batchResult, numWorkers)
	var wg sync.WaitGroup

	for w := range numWorkers {
		start := headerOffset + int64(w)*chunkSize
		end := start + chunkSize
		hasOverlap := end < fileSize

		if hasOverlap {
			end = min(end+overlap, fileSize)
		}

		wg.Add(1)
		go func(idx int, start, end int64) {
			defer wg.Done()

			// Read chunk into pre-allocated buffer
			bufSize := int(end - start)
			buf := make([]byte, bufSize)
			n, err := f.ReadAt(buf, start)
			if err != nil && err != io.EOF {
				return
			}
			buf = buf[:n]

			originalEnd := int(chunkSize)

			// Skip partial line at start (except for first chunk)
			offset := 0
			if start > headerOffset && len(buf) > 0 && buf[0] != '\n' {
				i := bytes.IndexByte(buf, '\n')
				if i >= 0 {
					offset = i + 1
				}
			}

			// Pre-allocate batch based on estimated line count (~45 bytes/line)
			estLines := max(originalEnd/45, 1024)
			batch := make([]CensusRecord, 0, estLines)

			processedBytes := offset
			for processedBytes < originalEnd && processedBytes < len(buf) {
				i := bytes.IndexByte(buf[processedBytes:], '\n')
				var line []byte
				if i < 0 {
					break
				}
				line = buf[processedBytes : processedBytes+i]
				processedBytes += i + 1

				if len(line) == 0 {
					continue
				}
				if line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}

				record, err := parseCensusFastBytes(line, delimiter)
				if err == nil {
					batch = append(batch, record)
				} else if opts.OnParseError != nil {
					opts.OnParseError(-1, err)
				}
			}
			results[idx].records = batch
		}(w, start, end)
	}

	wg.Wait()

	// Yield results in order
	count := 0
	limit := opts.Limit
	for _, batch := range results {
		for _, record := range batch.records {
			if limit > 0 && count >= limit {
				return nil
			}
			fn(record)
			count++
		}
	}

	return nil
}

// streamCensusFlex reads CensusRecords using the generic CSV reader
// with flexible column mapping. Slower than fast path, but handles any column order.
func streamCensusFlex(filepath string, opts CSVOptions, delimiter byte, colMap ColumnMap, fn func(CensusRecord)) error {
	return StreamCSV(filepath, opts, func(fields []string) {
		record, err := ParseCensusRow(fields, colMap)
		if err == nil {
			fn(record)
		}
	})
}

// parseCensusFastBytes parses a CSV line directly from bytes into a CensusRecord.
// Assumes default column order: age,sex,policy_type,sum_assured,term
// Single-pass: finds all delimiters, then parses numbers from field slices.
func parseCensusFastBytes(line []byte, delimiter byte) (CensusRecord, error) {
	if len(line) == 0 {
		return CensusRecord{}, errors.New("empty line")
	}

	// Find all 4 delimiter positions in one pass
	c1, c2, c3, c4 := -1, -1, -1, -1
	for i := range line {
		if line[i] == delimiter {
			switch {
			case c1 < 0:
				c1 = i
			case c2 < 0:
				c2 = i
			case c3 < 0:
				c3 = i
			case c4 < 0:
				c4 = i
			}
		}
	}

	if c4 < 0 {
		return CensusRecord{}, errors.New("invalid line format: expected 5 fields")
	}

	// Parse age from field 1
	age, ok := parseFastInt(line[:c1])
	if !ok || age < 0 {
		return CensusRecord{}, errors.New("invalid age field")
	}

	// Extract sex (field 2) and policy_type (field 3) as interned strings
	sex := sexString(line[c1+1 : c2])
	policyType := policyString(line[c2+1 : c3])

	// Parse sum_assured from field 4
	sumAssured, ok := parseFastFloat(line[c3+1 : c4])
	if !ok || sumAssured < 0 {
		return CensusRecord{}, errors.New("invalid sum_assured field")
	}

	// Parse term from field 5 (trim trailing \r)
	termField := line[c4+1:]
	if len(termField) > 0 && termField[len(termField)-1] == '\r' {
		termField = termField[:len(termField)-1]
	}
	term, ok := parseFastInt(termField)
	if !ok || term < 0 {
		return CensusRecord{}, errors.New("invalid term field")
	}

	return CensusRecord{
		Age:        age,
		Sex:        sex,
		PolicyType: policyType,
		SumAssured: sumAssured,
		Term:       term,
	}, nil
}

// parseFastInt parses an integer from a byte slice without allocation.
// Returns the parsed value and true on success, or 0 and false if the input
// contains non-digit characters (other than a leading '-').
func parseFastInt(b []byte) (int, bool) {
	if len(b) == 0 {
		return 0, false
	}
	n := 0
	negative := false
	start := 0
	if b[0] == '-' {
		negative = true
		start = 1
		if len(b) == 1 {
			return 0, false
		}
	}
	for i := start; i < len(b); i++ {
		c := b[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		n = n*10 + int(c-'0')
	}
	if negative {
		return -n, true
	}
	return n, true
}

// parseFastFloat parses a float from a byte slice without allocation.
// Returns the parsed value and true on success, or 0 and false if the input
// is invalid (non-numeric characters, multiple decimal points, etc.).
func parseFastFloat(b []byte) (float64, bool) {
	if len(b) == 0 {
		return 0, false
	}
	val := 0.0
	divisor := 1
	inDecimal := false
	negative := false
	start := 0
	hasDigit := false

	if b[0] == '-' {
		negative = true
		start = 1
		if len(b) == 1 {
			return 0, false
		}
	}

	for i := start; i < len(b); i++ {
		c := b[i]
		if c == '.' {
			if inDecimal {
				return 0, false
			}
			inDecimal = true
			continue
		}
		if c < '0' || c > '9' {
			return 0, false
		}
		hasDigit = true
		val = val*10 + float64(c-'0')
		if inDecimal {
			divisor *= 10
		}
	}
	if !hasDigit {
		return 0, false
	}
	if divisor > 1 {
		val /= float64(divisor)
	}
	if negative {
		val = -val
	}
	return val, true
}

// ParseCensusRow converts generic string fields to a CensusRecord using column mapping.
// ParseCensusRow converts a slice of string fields to a CensusRecord
// using the provided column mapping.
func ParseCensusRow(fields []string, colMap ColumnMap) (CensusRecord, error) {
	if len(fields) == 0 {
		return CensusRecord{}, errors.New("empty row")
	}

	record := CensusRecord{}
	var err error

	for colName, colIdx := range colMap {
		if colIdx >= len(fields) {
			continue
		}
		value := strings.TrimSpace(fields[colIdx])
		if value == "" {
			continue
		}

		switch colName {
		case "age":
			record.Age, err = strconv.Atoi(value)
		case "sex":
			record.Sex = value
		case "policy_type":
			record.PolicyType = value
		case "sum_assured":
			record.SumAssured, err = strconv.ParseFloat(value, 64)
		case "term":
			record.Term, err = strconv.Atoi(value)
		}

		if err != nil {
			return record, err
		}
	}

	return record, nil
}

// sexString returns the interned string for common sex values without allocation.
func sexString(b []byte) string {
	if len(b) == 4 && b[0] == 'm' && b[1] == 'a' && b[2] == 'l' && b[3] == 'e' {
		return "male"
	}
	if len(b) == 6 && b[0] == 'f' && b[1] == 'e' && b[2] == 'm' && b[3] == 'a' && b[4] == 'l' && b[5] == 'e' {
		return "female"
	}
	return string(b)
}

// policyString returns the interned string for common policy types without allocation.
func policyString(b []byte) string {
	switch len(b) {
	case 4:
		if b[0] == 't' && b[1] == 'e' && b[2] == 'r' && b[3] == 'm' {
			return "term"
		}
	case 5:
		if b[0] == 'w' && b[1] == 'h' && b[2] == 'o' && b[3] == 'l' && b[4] == 'e' {
			return "whole"
		}
	case 9:
		if b[0] == 'e' && b[1] == 'n' && b[2] == 'd' && b[3] == 'o' &&
			b[4] == 'w' && b[5] == 'm' && b[6] == 'e' && b[7] == 'n' && b[8] == 't' {
			return "endowment"
		}
	}
	return string(b)
}

// buildColumnMap creates a ColumnMap from header names.
func buildColumnMap(headers []string) ColumnMap {
	colMap := make(ColumnMap, len(headers))
	for i, h := range headers {
		colMap[normalizeColumnName(strings.TrimSpace(h))] = i
	}
	return colMap
}

// isDefaultColumnOrder checks if columns match: age,sex,policy_type,sum_assured,term
func isDefaultColumnOrder(colMap ColumnMap) bool {
	return len(colMap) == 5 &&
		colMap["age"] == 0 &&
		colMap["sex"] == 1 &&
		colMap["policy_type"] == 2 &&
		colMap["sum_assured"] == 3 &&
		colMap["term"] == 4
}

// defaultColumnMap returns the assumed column order when no header is present.
func defaultColumnMap() ColumnMap {
	return ColumnMap{
		"age":         0,
		"sex":         1,
		"policy_type": 2,
		"sum_assured": 3,
		"term":        4,
	}
}

// normalizeColumnName standardizes column names for flexible matching.
func normalizeColumnName(name string) string {
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")

	variations := map[string]string{
		"sumassured":      "sum_assured",
		"policytype":      "policy_type",
		"sumassuredvalue": "sum_assured",
		"policytypecode":  "policy_type",
	}

	if normalized, exists := variations[name]; exists {
		return normalized
	}
	return name
}
