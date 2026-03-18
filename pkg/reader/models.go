package reader

// CensusRecord represents a policy record with core actuarial fields
type CensusRecord struct {
	Age        int     `csv:"age"`
	Sex        string  `csv:"sex"`
	PolicyType string  `csv:"policy_type"`
	SumAssured float64 `csv:"sum_assured"`
	Term       int     `csv:"term"`
}

// ColumnMap maps CSV column names to their indices for efficient lookup
type ColumnMap map[string]int

// ParseResult holds the parsed record and any parsing errors
type ParseResult struct {
	Record CensusRecord
	Errors []error
	Line   int
}

// CSVConfig holds configuration for CSV parsing
type CSVConfig struct {
	Header       bool
	Limit        int
	RequiredCols []string
	AllowUnknown bool
}
