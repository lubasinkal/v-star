package reader

type CensusRecord struct {
	Age        int     `csv:"age"`
	Sex        string  `csv:"sex"`
	PolicyType string  `csv:"policy_type"`
	SumAssured float64 `csv:"sum_assured"`
	Term       int     `csv:"term"`
}
