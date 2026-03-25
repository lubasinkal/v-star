package reader

// CensusRecord represents a policy record with core actuarial fields.
type CensusRecord struct {
	Sex        string  `csv:"sex"`
	PolicyType string  `csv:"policy_type"`
	Age        int     `csv:"age"`
	SumAssured float64 `csv:"sum_assured"`
	Term       int     `csv:"term"`
}
