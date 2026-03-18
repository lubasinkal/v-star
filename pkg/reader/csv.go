package reader

import (
	"encoding/csv"
	"io"
	"os"
	"strconv"
)

type StreamOptions struct {
	Header bool
	Limit  int
}

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

func StreamCSV(filepath string, opts StreamOptions, fn func(CensusRecord)) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.TrimLeadingSpace = true

	// Skip header
	if opts.Header {
		if _, err := reader.Read(); err != nil {
			return err
		}
	}

	count := 0

	for {

		if opts.Limit > 0 && count >= opts.Limit {
			break
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		census := ParseRecord(record)
		fn(census)

		count++
	}

	return nil
}
