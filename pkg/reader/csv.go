package reader

import (
	"bufio"
	"io"
	"os"
	"strconv"
	"strings"
)

type StreamOptions struct {
	Header bool
	Limit  int
}

func StreamCSV(filepath string, opts StreamOptions, fn func(CensusRecord)) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, 64*1024*1024)

	if opts.Header {
		if _, err := reader.ReadString('\n'); err != nil {
			return err
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
			if len(line) > 0 {
				parseAndCall(line, fn, &count)
			}
			break
		}
		if err != nil {
			return err
		}

		parseAndCall(line, fn, &count)
	}

	return nil
}

func parseAndCall(line string, fn func(CensusRecord), count *int) {
	// Fast trim - avoid allocations
	if len(line) == 0 {
		return
	}

	// Remove trailing newline
	if line[len(line)-1] == '\n' {
		line = line[:len(line)-1]
	}
	if len(line) > 0 && line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	if len(line) == 0 {
		return
	}

	fields := strings.SplitN(line, ",", 5)
	if len(fields) < 5 {
		return
	}

	age, _ := strconv.Atoi(fields[0])
	sumAssured, _ := strconv.ParseFloat(fields[3], 64)
	term, _ := strconv.Atoi(fields[4])

	fn(CensusRecord{
		Age:        age,
		Sex:        fields[1],
		PolicyType: fields[2],
		SumAssured: sumAssured,
		Term:       term,
	})

	*count++
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
