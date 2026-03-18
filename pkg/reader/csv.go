package reader

import (
	"bufio"
	"io"
	"os"
	"strconv"
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
		_, _ = reader.ReadString('\n')
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
				fn(parseLine(line))
				count++
			}
			break
		}
		if err != nil {
			return err
		}

		fn(parseLine(line))
		count++
	}

	return nil
}

func parseLine(line string) CensusRecord {
	// Fast trim without allocation
	n := len(line)
	if n > 0 && line[n-1] == '\n' {
		n--
	}
	if n > 0 && line[n-1] == '\r' {
		n--
	}
	line = line[:n]

	// Find positions of commas (much faster than Split)
	c0 := 0
	c1 := indexByte(line, ',', 0)
	c2 := indexByte(line, ',', c1+1)
	c3 := indexByte(line, ',', c2+1)
	c4 := indexByte(line, ',', c3+1)

	age := atoi(line[c0:c1])
	sex := line[c1+1 : c2]
	policyType := line[c2+1 : c3]
	sumAssured, _ := strconv.ParseFloat(line[c3+1:c4], 64)
	term := atoi(line[c4+1:])

	return CensusRecord{
		Age:        age,
		Sex:        sex,
		PolicyType: policyType,
		SumAssured: sumAssured,
		Term:       term,
	}
}

func indexByte(s string, c byte, start int) int {
	for i := start; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return len(s)
}

// Fast atoi implementation
func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
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
