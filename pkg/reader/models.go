package reader

// CensusRecord represents a policy record with core actuarial fields.
type CensusRecord struct {
	Sex        string  `csv:"sex" json:"sex"`
	PolicyType string  `csv:"policy_type" json:"policy_type"`
	Age        int     `csv:"age" json:"age"`
	SumAssured float64 `csv:"sum_assured" json:"sum_assured"`
	Term       int     `csv:"term" json:"term"`
}
