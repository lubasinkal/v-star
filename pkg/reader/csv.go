package reader

import (
	"encoding/csv"
	"os"
	"strconv"
)

func ReadCSV(filepath string) ([]CensusRecord, error) {
	// Open file
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	// Create CSV reader
	reader := csv.NewReader(file)

	// Optional: Better performance
	reader.TrimLeadingSpace = true
	// Read header (if exists) - skip it
	_, err = reader.Read()
	if err != nil {
		return nil, err // No data
	}
	// Read all records
	var records []CensusRecord
	for {
		record, err := reader.Read()
		if err != nil {
			break // EOF
		}

		census := ParseRecord(record)
		records = append(records, census)
	}
	return records, nil
}
func ParseRecord(fields []string) CensusRecord {
	age, _ := strconv.Atoi(fields[0])
	sumAssured, _ := strconv.ParseFloat(fields[3], 64)
	term, _ := strconv.Atoi(fields[4])
	return CensusRecord{
		Age:        age,
		Sex:        fields[1],
		PolicyType: fields[2],
		SumAssured: sumAssured,
		Term:       term,
	}
}
