package reader

import (
	"bufio"
	"errors"
	"io"
	"os"
	"strconv"
	"strings"
)

type StreamOptions struct {
	Header bool
	Limit  int
}

// StreamCSV reads a CSV file and processes each record
func StreamCSV(filepath string, opts StreamOptions, fn func(CensusRecord)) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 64*1024*1024)
	var colMap ColumnMap
	lineNum := 0
	var errors []error
	useFastPath := false

	// Read header if needed
	if opts.Header {
		headerLine, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}
		colMap = parseHeader(headerLine)
		lineNum++

		// Check if we can use fast path (default column order)
		if isDefaultColumnOrder(colMap) {
			useFastPath = true
		}
	} else {
		// No header - use fast path with default column order
		useFastPath = true
		colMap = ColumnMap{
			"age":         0,
			"sex":         1,
			"policy_type": 2,
			"sum_assured": 3,
			"term":        4,
		}
	}

	count := 0
	limit := opts.Limit

	for {
		if limit > 0 && count >= limit {
			break
		}

		line, err := reader.ReadString('\n')
		if err == io.EOF {
			if len(strings.TrimSpace(line)) > 0 {
				var record CensusRecord
				var parseErr error
				if useFastPath {
					record, parseErr = parseLineFast(line)
				} else {
					record, parseErr = parseLineFlex(line, colMap, lineNum+1)
				}
				if parseErr == nil {
					fn(record)
					count++
				} else {
					errors = append(errors, parseErr)
				}
			}
			break
		}
		if err != nil {
			return err
		}

		lineNum++
		var record CensusRecord
		var parseErr error
		if useFastPath {
			record, parseErr = parseLineFast(line)
		} else {
			record, parseErr = parseLineFlex(line, colMap, lineNum)
		}
		if parseErr == nil {
			fn(record)
			count++
		} else {
			errors = append(errors, parseErr)
		}
	}

	// Report errors if any
	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}

// isDefaultColumnOrder checks if the column mapping matches the default order
func isDefaultColumnOrder(colMap ColumnMap) bool {
	return len(colMap) == 5 &&
		colMap["age"] == 0 &&
		colMap["sex"] == 1 &&
		colMap["policy_type"] == 2 &&
		colMap["sum_assured"] == 3 &&
		colMap["term"] == 4
}

// parseHeader extracts column names from the header line
func parseHeader(headerLine string) ColumnMap {
	colMap := make(ColumnMap)
	cols := splitCSVLine(headerLine)
	for i, col := range cols {
		normalized := normalizeColumnName(strings.TrimSpace(col))
		colMap[normalized] = i
	}
	return colMap
}

// normalizeColumnName converts various column name formats to a standard format
func normalizeColumnName(name string) string {
	// Convert to lowercase and replace common separators with underscores
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "-", "_")

	// Handle common variations
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

// parseLine parses a single CSV line into a CensusRecord
func parseLine(line string, colMap ColumnMap, lineNum int) (CensusRecord, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return CensusRecord{}, errors.New("empty line")
	}

	fields := splitCSVLine(line)
	record := CensusRecord{}

	// Parse each field based on column mapping
	for colName, colIdx := range colMap {
		if colIdx >= len(fields) {
			continue // Missing column, skip
		}

		value := strings.TrimSpace(fields[colIdx])
		if value == "" {
			continue // Empty value, skip
		}

		var err error
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
			return record, errors.New("line " + strconv.Itoa(lineNum) + ": invalid " + colName + " value '" + value + "'")
		}
	}

	return record, nil
}

// splitCSVLine splits a CSV line into fields, handling basic quoted values
func splitCSVLine(line string) []string {
	var fields []string
	var currentField strings.Builder
	inQuotes := false
	escaped := false

	for i := 0; i < len(line); i++ {
		c := line[i]

		if escaped {
			currentField.WriteByte(c)
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			continue
		}

		if c == '"' {
			inQuotes = !inQuotes
			continue
		}

		if c == ',' && !inQuotes {
			fields = append(fields, currentField.String())
			currentField.Reset()
			continue
		}

		currentField.WriteByte(c)
	}

	// Add the last field
	fields = append(fields, currentField.String())

	return fields
}

// parseLineFast uses the original hardcoded inline parsing for maximum speed
func parseLineFast(line string) (CensusRecord, error) {
	// Find commas and parse in a single pass
	// Assumes format: age,sex,policy_type,sum_assured,term\n

	// Find first comma (end of age)
	c1 := 0
	for c1 < len(line) && line[c1] != ',' {
		c1++
	}
	if c1 >= len(line) {
		return CensusRecord{}, errors.New("invalid line format")
	}

	// Find second comma (end of sex)
	c2 := c1 + 1
	for c2 < len(line) && line[c2] != ',' {
		c2++
	}
	if c2 >= len(line) {
		return CensusRecord{}, errors.New("invalid line format")
	}

	// Find third comma (end of policy_type)
	c3 := c2 + 1
	for c3 < len(line) && line[c3] != ',' {
		c3++
	}
	if c3 >= len(line) {
		return CensusRecord{}, errors.New("invalid line format")
	}

	// Find fourth comma (end of sum_assured)
	c4 := c3 + 1
	for c4 < len(line) && line[c4] != ',' {
		c4++
	}
	if c4 >= len(line) {
		return CensusRecord{}, errors.New("invalid line format")
	}

	// Parse age (first field)
	age := 0
	for i := 0; i < c1; i++ {
		c := line[i]
		if c >= '0' && c <= '9' {
			age = age*10 + int(c-'0')
		}
	}

	// Parse sum_assured (fourth field)
	sumVal := 0.0
	decimal := false
	divisor := 1
	for i := c3 + 1; i < c4; i++ {
		c := line[i]
		if c == '.' {
			decimal = true
			continue
		}
		if c >= '0' && c <= '9' {
			sumVal = sumVal*10 + float64(c-'0')
			if decimal {
				divisor *= 10
			}
		}
	}
	if divisor > 1 {
		sumVal = sumVal / float64(divisor)
	}

	// Parse term (fifth field, until end of line)
	term := 0
	for i := c4 + 1; i < len(line); i++ {
		c := line[i]
		if c >= '0' && c <= '9' {
			term = term*10 + int(c-'0')
		} else if c == '\n' || c == '\r' {
			break
		}
	}

	// Extract string fields using substring (minimal allocation)
	// Note: These substrings reference the original line's memory
	sexStart := c1 + 1
	policyTypeStart := c2 + 1

	// Trim trailing newline/carriage return from policyType field if present
	policyTypeEnd := c3
	if policyTypeEnd > policyTypeStart && line[policyTypeEnd-1] == '\r' {
		policyTypeEnd--
	}

	sexEnd := c2
	if sexEnd > sexStart && line[sexEnd-1] == '\r' {
		sexEnd--
	}

	return CensusRecord{
		Age:        age,
		Sex:        line[sexStart:sexEnd],
		PolicyType: line[policyTypeStart:policyTypeEnd],
		SumAssured: sumVal,
		Term:       term,
	}, nil
}

// parseLineFlex parses a single CSV line with flexible column mapping
func parseLineFlex(line string, colMap ColumnMap, lineNum int) (CensusRecord, error) {
	// Trim whitespace
	if len(line) == 0 {
		return CensusRecord{}, errors.New("empty line")
	}
	last := len(line) - 1
	if line[last] == '\n' {
		line = line[:last]
		last--
	}
	if last >= 0 && line[last] == '\r' {
		line = line[:last]
	}

	// Find all comma positions
	commaPositions := make([]int, 0, 10)
	for i := 0; i < len(line); i++ {
		if line[i] == ',' {
			commaPositions = append(commaPositions, i)
		}
	}

	// Helper to extract field value
	getField := func(colIdx int) string {
		if colIdx == 0 {
			if len(commaPositions) > 0 {
				return line[0:commaPositions[0]]
			}
			return line
		}
		if colIdx > len(commaPositions) {
			return ""
		}
		start := commaPositions[colIdx-1] + 1
		end := len(line)
		if colIdx < len(commaPositions) {
			end = commaPositions[colIdx]
		}
		return line[start:end]
	}

	record := CensusRecord{}
	var err error

	// Parse each field based on column mapping
	for colName, colIdx := range colMap {
		value := getField(colIdx)
		if value == "" {
			continue
		}

		switch colName {
		case "age":
			record.Age, err = parseInlineInt(value)
		case "sex":
			record.Sex = value
		case "policy_type":
			record.PolicyType = value
		case "sum_assured":
			record.SumAssured, err = parseInlineFloat(value)
		case "term":
			record.Term, err = parseInlineInt(value)
		}

		if err != nil {
			return record, errors.New("line " + strconv.Itoa(lineNum) + ": invalid " + colName + " value '" + value + "'")
		}
	}

	return record, nil
}

// parseInlineInt parses an integer inline (no allocation)
func parseInlineInt(s string) (int, error) {
	val := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			val = val*10 + int(c-'0')
		} else if c != ' ' && c != '\t' {
			return 0, errors.New("invalid character")
		}
	}
	return val, nil
}

// parseInlineFloat parses a float inline (no allocation)
func parseInlineFloat(s string) (float64, error) {
	val := 0.0
	decimal := false
	divisor := 1
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			decimal = true
			continue
		}
		if c >= '0' && c <= '9' {
			val = val*10 + float64(c-'0')
			if decimal {
				divisor *= 10
			}
		} else if c != ' ' && c != '\t' {
			return 0, errors.New("invalid character")
		}
	}
	if divisor > 1 {
		val = val / float64(divisor)
	}
	return val, nil
}
