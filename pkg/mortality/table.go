package mortality

import (
	"errors"
	"io"
	"os"
)

// MortalityTable defines the interface for mortality data access.
// Implementations provide mortality rates (qx), survival probabilities (px),
// and the maximum age defined in the table.
type MortalityTable interface {
	Qx(age int) float64
	Px(age int, term int) float64
	MaxAge() int
}

// Table implements MortalityTable using slices of mortality rates.
// Uses radix 100000 for lx (survival count) calculations.
type Table struct {
	name   string
	qx     []float64
	lx     []float64
	maxAge int
}

// NewTable constructs a Table from a slice of qx (probability of death) values.
// Computes lx internally using radix 100000. Index 0 corresponds to age 0.
func NewTable(name string, qx []float64) *Table {
	maxAge := len(qx) - 1
	lx := make([]float64, len(qx))
	lx[0] = 100000
	for i := 1; i < len(qx); i++ {
		lx[i] = lx[i-1] * (1 - qx[i-1])
	}
	return &Table{
		name:   name,
		qx:     qx,
		lx:     lx,
		maxAge: maxAge,
	}
}

// Qx returns the probability of death between age x and x+1.
// Returns 0 for out-of-range ages.
func (t *Table) Qx(age int) float64 {
	if age < 0 || age > t.maxAge {
		return 0
	}
	return t.qx[age]
}

// Px returns the cumulative survival probability over term years from age.
// Returns 1 for term <= 0, and 0 when age + term exceeds maxAge.
func (t *Table) Px(age int, term int) float64 {
	if age < 0 || term <= 0 {
		return 1
	}
	endAge := age + term
	if endAge > t.maxAge {
		return 0
	}
	product := 1.0
	for a := age; a < endAge; a++ {
		product *= (1 - t.qx[a])
	}
	return product
}

// Ex returns the curtate expectation of life at the given age.
// This is the sum of Px(age, year) for year >= 1 until maxAge is reached.
func (t *Table) Ex(age int) float64 {
	if age < 0 || age > t.maxAge {
		return 0
	}
	ex := 0.0
	for year := 1; age+year <= t.maxAge+1; year++ {
		ex += t.Px(age, year)
	}
	return ex
}

// MaxAge returns the maximum age defined in the table.
func (t *Table) MaxAge() int {
	return t.maxAge
}

// Name returns the table name.
func (t *Table) Name() string {
	return t.name
}

// Lx returns the number of lives surviving to the given age from radix 100000.
func (t *Table) Lx(age int) float64 {
	if age < 0 || age > t.maxAge {
		return 0
	}
	return t.lx[age]
}

// LoadCSV loads a mortality table from a CSV file.
// Supports columns named "qx" or "px" alongside an "age" column.
func LoadCSV(filepath string) (*Table, error) {
	return loadCSVToMemory(filepath)
}

func loadCSVToMemory(filepath string) (*Table, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	lines := parseLines(data)
	if len(lines) < 2 {
		return nil, errors.New("mortality table must have at least header and one data row")
	}
	header := lines[0]
	colMap := detectColumns(header)
	ageIdx := colMap["age"]
	qxIdx := colMap["qx"]
	pxIdx, hasPx := colMap["px"]
	if !hasPx {
		pxIdx = -1
	}
	if ageIdx < 0 {
		return nil, errors.New("age column required")
	}
	if qxIdx < 0 && pxIdx < 0 {
		return nil, errors.New("either qx or px column required")
	}
	name := extractName(filepath)
	qx := make([]float64, len(lines)-1)
	for i := 1; i < len(lines); i++ {
		fields := splitCSV(lines[i])
		age := parseInt(fields[ageIdx])
		if qxIdx >= 0 {
			qx[age] = parseFloat(fields[qxIdx])
		} else if pxIdx >= 0 {
			px := parseFloat(fields[pxIdx])
			if age > 0 {
				prevPx := qx[age-1]
				if prevPx > 0 {
					qx[age-1] = 1 - px/prevPx
				}
			}
		}
	}
	if pxIdx >= 0 {
		for i := 1; i < len(lines); i++ {
			fields := splitCSV(lines[i])
			age := parseInt(fields[ageIdx])
			px := parseFloat(fields[pxIdx])
			if age > 0 && age < len(qx) {
				prevPx := 1.0
				for a := range age {
					prevPx *= (1 - qx[a])
				}
				if prevPx > 0 {
					qx[age-1] = 1 - px/prevPx
				}
			}
		}
	}
	return NewTable(name, qx), nil
}

func StreamCSV(filepath string, fn func(age int, qx float64)) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() < 1024*1024 {
		return streamCSVSmall(file, fn)
	}
	return streamCSVParallel(filepath, fn)
}

func streamCSVSmall(file *os.File, fn func(age int, qx float64)) error {
	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	lines := parseLines(data)
	if len(lines) < 2 {
		return nil
	}
	header := lines[0]
	colMap := detectColumns(header)
	ageIdx := colMap["age"]
	qxIdx := colMap["qx"]
	pxIdx, hasPx := colMap["px"]
	if !hasPx {
		pxIdx = -1
	}
	if qxIdx < 0 && pxIdx < 0 {
		return errors.New("either qx or px column required")
	}
	var qx []float64
	if qxIdx >= 0 {
		for i := 1; i < len(lines); i++ {
			fields := splitCSV(lines[i])
			age := parseInt(fields[ageIdx])
			q := parseFloat(fields[qxIdx])
			fn(age, q)
			if age >= len(qx) {
				qx = append(qx, q)
			} else {
				qx[age] = q
			}
		}
	} else {
		pxVals := make(map[int]float64)
		for i := 1; i < len(lines); i++ {
			fields := splitCSV(lines[i])
			age := parseInt(fields[ageIdx])
			px := parseFloat(fields[pxIdx])
			pxVals[age] = px
		}
		for age, px := range pxVals {
			var q float64
			if age == 0 {
				q = 1 - px
			} else {
				if prevPx, ok := pxVals[age-1]; ok && prevPx > 0 {
					q = 1 - px/prevPx
				}
			}
			fn(age, q)
		}
	}
	return nil
}

func streamCSVParallel(filepath string, fn func(age int, qx float64)) error {
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
	data := make([]byte, fileSize)
	_, err = f.ReadAt(data, 0)
	if err != nil && err != io.EOF {
		return err
	}
	lines := parseLines(data)
	if len(lines) < 2 {
		return nil
	}
	header := lines[0]
	colMap := detectColumns(header)
	ageIdx := colMap["age"]
	qxIdx := colMap["qx"]
	pxIdx, hasPx := colMap["px"]
	if !hasPx {
		pxIdx = -1
	}
	if ageIdx < 0 || (qxIdx < 0 && pxIdx < 0) {
		return errors.New("invalid column structure")
	}
	pxVals := make(map[int]float64)
	if pxIdx >= 0 {
		for i := 1; i < len(lines); i++ {
			fields := splitCSV(lines[i])
			age := parseInt(fields[ageIdx])
			pxVals[age] = parseFloat(fields[pxIdx])
		}
		for age := 0; age <= maxAge(pxVals); age++ {
			var q float64
			if age == 0 {
				if px, ok := pxVals[0]; ok {
					q = 1 - px
				}
			} else {
				if px, ok := pxVals[age]; ok {
					if prevPx, ok := pxVals[age-1]; ok && prevPx > 0 {
						q = 1 - px/prevPx
					}
				}
			}
			fn(age, q)
		}
	} else {
		for i := 1; i < len(lines); i++ {
			fields := splitCSV(lines[i])
			age := parseInt(fields[ageIdx])
			q := parseFloat(fields[qxIdx])
			fn(age, q)
		}
	}
	return nil
}

func parseLines(data []byte) [][]byte {
	// Skip UTF-8 BOM if present
	if len(data) >= 3 && data[0] == 0xef && data[1] == 0xbb && data[2] == 0xbf {
		data = data[3:]
	}

	var lines [][]byte
	start := 0
	for i := 0; i < len(data); i++ {
		if data[i] == '\n' {
			if i > start {
				line := data[start:i]
				// Trim trailing \r (Windows line endings)
				if len(line) > 0 && line[len(line)-1] == '\r' {
					line = line[:len(line)-1]
				}
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(data) {
		line := data[start:]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}
		if len(line) > 0 {
			lines = append(lines, line)
		}
	}
	return lines
}

func splitCSV(line []byte) []string {
	var fields []string
	start := 0
	inQuotes := false
	for i := range line {
		c := line[i]
		if c == '"' {
			inQuotes = !inQuotes
		} else if c == ',' && !inQuotes {
			fields = append(fields, string(line[start:i]))
			start = i + 1
		}
	}
	fields = append(fields, string(line[start:]))
	return fields
}

func detectColumns(header []byte) map[string]int {
	fields := splitCSV(header)
	colMap := make(map[string]int)
	for i, f := range fields {
		lower := toLower(f)
		colMap[lower] = i
	}
	return colMap
}

func toLower(s string) string {
	result := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		result[i] = c
	}
	return string(result)
}

func parseInt(s string) int {
	if len(s) == 0 {
		return 0
	}
	n := 0
	negative := false
	start := 0
	if s[0] == '-' {
		negative = true
		start = 1
	}
	for i := start; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0
		}
		n = n*10 + int(c-'0')
	}
	if negative {
		return -n
	}
	return n
}

func parseFloat(s string) float64 {
	if len(s) == 0 {
		return 0
	}
	val := 0.0
	divisor := 1
	inDecimal := false
	hasDigit := false
	negative := false
	start := 0

	if s[0] == '-' {
		negative = true
		start = 1
	}

	for i := start; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			if inDecimal {
				return 0
			}
			inDecimal = true
			continue
		}
		if c < '0' || c > '9' {
			return 0
		}
		hasDigit = true
		val = val*10 + float64(c-'0')
		if inDecimal {
			divisor *= 10
		}
	}
	if !hasDigit {
		return 0
	}
	if divisor > 1 {
		val /= float64(divisor)
	}
	if negative {
		val = -val
	}
	return val
}

func extractName(filepath string) string {
	for i := len(filepath) - 1; i >= 0; i-- {
		if filepath[i] == '/' || filepath[i] == '\\' {
			return filepath[i+1 : len(filepath)-4]
		}
	}
	return filepath[:len(filepath)-4]
}

func maxAge(m map[int]float64) int {
	max := 0
	for k := range m {
		if k > max {
			max = k
		}
	}
	return max
}
