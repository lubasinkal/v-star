package main

import (
	"fmt"
	"os"

	"github.com/lubasinkal/v-star/pkg/reader"
)

func main() {
	sampleFile := "sample_census.csv"
	if err := createSampleCSV(sampleFile); err != nil {
		fmt.Printf("Error creating sample: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(sampleFile)

	converter := newPVCalculator(0.05)
	opts := reader.CSVOptions{Header: true}

	fmt.Println("Streaming CSV and computing present values...")
	totalPV, count := reader.StreamCSVWithPV(sampleFile, opts, converter)

	fmt.Printf("Records processed: %d\n", count)
	fmt.Printf("Total present value: $%.2f\n", totalPV)
	fmt.Println()

	fmt.Println("Raw CSV rows:")
	reader.StreamCSV(sampleFile, opts, func(fields []string) {
		fmt.Printf("  %v\n", fields)
	})
}

func newPVCalculator(rate float64) func(float64, int) float64 {
	v := 1.0 / (1.0 + rate)
	return func(sumAssured float64, term int) float64 {
		if term <= 0 {
			return sumAssured
		}
		pv := sumAssured
		for range term {
			pv *= v
		}
		return pv
	}
}

func createSampleCSV(filepath string) error {
	data := `age,sex,policy_type,sum_assured,term
30,M,term,100000,20
45,F,whole_life,200000,25
25,M,endowment,50000,10
55,F,term,150000,5
35,M,whole_life,300000,25
`
	return os.WriteFile(filepath, []byte(data), 0644)
}
