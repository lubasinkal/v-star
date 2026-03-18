package reader

import (
	"bufio"
	"io"
	"os"
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
				parseAndCall(line, fn)
				count++
			}
			file.Close()
			break
		}
		if err != nil {
			file.Close()
			return err
		}

		parseAndCall(line, fn)
		count++
	}

	file.Close()
	return nil
}

func parseAndCall(line string, fn func(CensusRecord)) {
	// Trim - avoid allocation
	if len(line) == 0 {
		return
	}
	last := len(line) - 1
	if line[last] == '\n' {
		line = line[:last]
		last--
	}
	if last >= 0 && line[last] == '\r' {
		line = line[:last]
	}

	// Find commas - inline
	c1 := 0
	for c1 < len(line) && line[c1] != ',' {
		c1++
	}
	c2 := c1 + 1
	for c2 < len(line) && line[c2] != ',' {
		c2++
	}
	c3 := c2 + 1
	for c3 < len(line) && line[c3] != ',' {
		c3++
	}
	c4 := c3 + 1
	for c4 < len(line) && line[c4] != ',' {
		c4++
	}

	// Parse ints inline
	age := 0
	for i := 0; i < c1; i++ {
		c := line[i]
		if c >= '0' && c <= '9' {
			age = age*10 + int(c-'0')
		}
	}

	sumVal := 0.0
	decimal := false
	divisor := 1
	for i := c3 + 1; i < c4; i++ {
		c := line[i]
		if c == '.' {
			decimal = true
			continue
		}
		if c >= '0' && c <= '9' {
			sumVal = sumVal*10 + float64(c-'0')
			if decimal {
				divisor *= 10
			}
		}
	}
	if divisor > 1 {
		sumVal = sumVal / float64(divisor)
	}

	term := 0
	for i := c4 + 1; i < len(line); i++ {
		c := line[i]
		if c >= '0' && c <= '9' {
			term = term*10 + int(c-'0')
		}
	}

	fn(CensusRecord{
		Age:        age,
		Sex:        line[c1+1 : c2],
		PolicyType: line[c2+1 : c3],
		SumAssured: sumVal,
		Term:       term,
	})
}
