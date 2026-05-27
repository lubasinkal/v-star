package mortality

import (
	"errors"
	"strconv"
	"strings"

	"github.com/lubasinkal/v-star/pkg/reader"
)

// MortalityTable defines the interface for mortality data access.
// Implementations provide mortality rates (qx), survival probabilities (px),
// curtate life expectancy (ex), survivor count (lx), and the maximum age.
type MortalityTable interface {
	Qx(age int) float64
	Px(age int, term int) float64
	MaxAge() int
	Ex(age int) float64
	Lx(age int) float64
}

// Table implements MortalityTable using slices of mortality rates.
// Uses radix 100000 for lx (survival count) calculations.
type Table struct {
	name   string
	qx     []float64
	lx     []float64
	ex     []float64
	maxAge int
}

// NewTable constructs a Table from a slice of qx (probability of death) values.
// Computes lx internally using radix 100000. Index 0 corresponds to age 0.
// Pre-computes curtate expectation of life ex via recurrence.
// Returns a Table with maxAge -1 if qx is nil or empty.
func NewTable(name string, qx []float64) *Table {
	maxAge := -1
	if len(qx) > 0 {
		maxAge = len(qx) - 1
	}
	lx := make([]float64, max(len(qx), 1))
	lx[0] = 100000
	for i := 1; i < len(qx); i++ {
		lx[i] = lx[i-1] * (1 - qx[i-1])
	}
	ex := make([]float64, len(lx))
	for age := maxAge - 1; age >= 0; age-- {
		p := lx[age+1] / lx[age]
		ex[age] = p * (1 + ex[age+1])
	}
	return &Table{
		name:   name,
		qx:     qx,
		lx:     lx,
		ex:     ex,
		maxAge: maxAge,
	}
}

// Qx returns the probability of death between age x and x+1.
// Returns 0 for out-of-range ages or nil table.
func (t *Table) Qx(age int) float64 {
	if t == nil || age < 0 || age > t.maxAge {
		return 0
	}
	if age >= len(t.qx) {
		return 0
	}
	return t.qx[age]
}

// Px returns the cumulative survival probability over term years from age.
// Returns 1 for term <= 0, and 0 when age + term exceeds maxAge.
// Uses pre-computed lx table for O(1) lookup instead of O(term) iteration.
func (t *Table) Px(age int, term int) float64 {
	if t == nil || age < 0 || term <= 0 {
		return 1
	}
	endAge := age + term
	if endAge > t.maxAge || t.lx[age] == 0 {
		return 0
	}
	return t.lx[endAge] / t.lx[age]
}

// Ex returns the curtate expectation of life at the given age.
// Uses a pre-computed table computed via recurrence at construction time.
func (t *Table) Ex(age int) float64 {
	if t == nil || age < 0 || age > t.maxAge {
		return 0
	}
	return t.ex[age]
}

// MaxAge returns the maximum age defined in the table.
func (t *Table) MaxAge() int {
	if t == nil {
		return -1
	}
	return t.maxAge
}

// Name returns the table name.
func (t *Table) Name() string {
	if t == nil {
		return ""
	}
	return t.name
}

// Lx returns the number of lives surviving to the given age from radix 100000.
func (t *Table) Lx(age int) float64 {
	if t == nil || age < 0 || age > t.maxAge {
		return 0
	}
	return t.lx[age]
}

// LoadCSV loads a mortality table from a CSV file.
// Supports columns named "qx" or "px" alongside an "age" column.
func LoadCSV(filepath string) (*Table, error) {
	headers, err := reader.GetHeaders(filepath, ',')
	if err != nil {
		return nil, err
	}

	colMap := buildColMap(headers)
	ageIdx, ageOk := colMap["age"]
	qxIdx, qxOk := colMap["qx"]
	pxIdx, pxOk := colMap["px"]

	if !ageOk {
		return nil, errors.New("mortality: age column required")
	}
	if !qxOk && !pxOk {
		return nil, errors.New("mortality: either qx or px column required")
	}

	if qxOk {
		return loadQx(filepath, ageIdx, qxIdx)
	}
	return loadPx(filepath, ageIdx, pxIdx)
}

func loadQx(filepath string, ageIdx, qxIdx int) (*Table, error) {
	var qx []float64
	err := reader.StreamCSV(filepath, reader.CSVOptions{Header: true, Delimiter: ','}, func(fields []string) {
		if ageIdx >= len(fields) || qxIdx >= len(fields) {
			return
		}
		age, _ := strconv.Atoi(strings.TrimSpace(fields[ageIdx]))
		q, _ := strconv.ParseFloat(strings.TrimSpace(fields[qxIdx]), 64)
		if age >= len(qx) {
			qx = append(qx, make([]float64, age-len(qx)+1)...)
		}
		qx[age] = q
	})
	if err != nil {
		return nil, err
	}
	return NewTable(extractName(filepath), qx), nil
}

func loadPx(filepath string, ageIdx, pxIdx int) (*Table, error) {
	pxVals := make(map[int]float64)
	err := reader.StreamCSV(filepath, reader.CSVOptions{Header: true, Delimiter: ','}, func(fields []string) {
		if ageIdx >= len(fields) || pxIdx >= len(fields) {
			return
		}
		age, _ := strconv.Atoi(strings.TrimSpace(fields[ageIdx]))
		p, _ := strconv.ParseFloat(strings.TrimSpace(fields[pxIdx]), 64)
		pxVals[age] = p
	})
	if err != nil {
		return nil, err
	}
	maxA := maxKey(pxVals)
	qx := make([]float64, maxA+1)
	for age, px := range pxVals {
		if age == 0 {
			qx[age] = 1 - px
		} else {
			prevPx := 1.0
			for a := range age {
				prevPx *= 1 - qx[a]
			}
			if prevPx > 0 {
				qx[age-1] = 1 - px/prevPx
			}
		}
	}
	return NewTable(extractName(filepath), qx), nil
}

func buildColMap(headers []string) map[string]int {
	m := make(map[string]int, len(headers))
	for i, h := range headers {
		m[strings.ToLower(strings.TrimSpace(h))] = i
	}
	return m
}

func StreamCSV(filepath string, fn func(age int, qx float64)) error {
	headers, err := reader.GetHeaders(filepath, ',')
	if err != nil {
		return err
	}
	colMap := buildColMap(headers)
	ageIdx, ageOk := colMap["age"]
	qxIdx, qxOk := colMap["qx"]
	pxIdx, pxOk := colMap["px"]
	if !ageOk || (!qxOk && !pxOk) {
		return errors.New("mortality: invalid column structure")
	}

	if qxOk {
		return reader.StreamCSV(filepath, reader.CSVOptions{Header: true, Delimiter: ','}, func(fields []string) {
			if ageIdx >= len(fields) || qxIdx >= len(fields) {
				return
			}
			age, _ := strconv.Atoi(strings.TrimSpace(fields[ageIdx]))
			q, _ := strconv.ParseFloat(strings.TrimSpace(fields[qxIdx]), 64)
			fn(age, q)
		})
	}

	// px columns: collect all values, derive qx
	pxVals := make(map[int]float64)
	err = reader.StreamCSV(filepath, reader.CSVOptions{Header: true, Delimiter: ','}, func(fields []string) {
		if ageIdx >= len(fields) || pxIdx >= len(fields) {
			return
		}
		age, _ := strconv.Atoi(strings.TrimSpace(fields[ageIdx]))
		p, _ := strconv.ParseFloat(strings.TrimSpace(fields[pxIdx]), 64)
		pxVals[age] = p
	})
	if err != nil {
		return err
	}
	for age := 0; age <= maxKey(pxVals); age++ {
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
	return nil
}

func extractName(filepath string) string {
	for i := len(filepath) - 1; i >= 0; i-- {
		if filepath[i] == '/' || filepath[i] == '\\' {
			return filepath[i+1 : len(filepath)-4]
		}
	}
	return filepath[:len(filepath)-4]
}

func maxKey(m map[int]float64) int {
	max := 0
	for k := range m {
		if k > max {
			max = k
		}
	}
	return max
}

// compile-time checks: *Table satisfies MortalityTable
var _ MortalityTable = (*Table)(nil)
