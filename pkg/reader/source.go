package reader

import (
	"io"
)

// CensusSource abstracts the origin of census policy records.
// Implementations can be file-backed (streamed from CSV), in-memory
// slices, generated data, or HTTP request bodies.
//
// For maximum throughput on large CSV files, bypass the interface
// and use StreamCensus / StreamCensusChunked directly.
type CensusSource interface {
	// ReadAll returns all census records from this source.
	// For large sources that don't fit in memory, use Stream instead.
	ReadAll() ([]CensusRecord, error)

	// Stream calls fn for each census record. Returns the number of
	// records processed. If fn returns a non-nil error, iteration
	// stops and Stream returns that error (along with the count so far).
	Stream(fn func(CensusRecord) error) (int, error)
}

// ---------------------------------------------------------------------------
// SliceCensusSource — in-memory slice
// ---------------------------------------------------------------------------

// SliceCensusSource is a CensusSource backed by an in-memory slice.
// Useful for programmatically generated data, test fixtures, or
// small-to-medium datasets already loaded into memory.
type SliceCensusSource struct {
	records []CensusRecord
}

// NewSliceCensusSource creates a CensusSource from an existing slice.
// The caller retains ownership of the slice; the source does not copy it.
func NewSliceCensusSource(records []CensusRecord) *SliceCensusSource {
	return &SliceCensusSource{records: records}
}

// ReadAll returns the underlying slice of records.
func (s *SliceCensusSource) ReadAll() ([]CensusRecord, error) {
	return s.records, nil
}

// Stream iterates over the in-memory slice, calling fn for each record.
func (s *SliceCensusSource) Stream(fn func(CensusRecord) error) (int, error) {
	for i, r := range s.records {
		if err := fn(r); err != nil {
			return i, err
		}
	}
	return len(s.records), nil
}

// ---------------------------------------------------------------------------
// FileCensusSource — CSV file on disk
// ---------------------------------------------------------------------------

// FileCensusSource is a CensusSource that reads census data from a
// CSV file on disk. Uses StreamCensus internally for efficient
// parallel parsing of large files.
type FileCensusSource struct {
	filepath string
	opts     CSVOptions
}

// NewFileCensusSource creates a CensusSource that reads from a CSV file.
// opts controls header detection, delimiter, row limit, and error handling.
func NewFileCensusSource(filepath string, opts CSVOptions) *FileCensusSource {
	return &FileCensusSource{filepath: filepath, opts: opts}
}

// ReadAll reads all census records from the CSV file into memory.
// For very large files, use Stream to process records without buffering
// the entire dataset, or use StreamCensus directly for maximum throughput.
func (s *FileCensusSource) ReadAll() ([]CensusRecord, error) {
	var records []CensusRecord
	_, err := s.Stream(func(r CensusRecord) error {
		records = append(records, r)
		return nil
	})
	return records, err
}

// Stream reads records from the CSV file, calling fn for each.
// The underlying StreamCensus call runs to completion (file I/O);
// use CSVOptions.Limit to bound the number of records read.
func (s *FileCensusSource) Stream(fn func(CensusRecord) error) (int, error) {
	var stopErr error
	count := 0
	err := StreamCensus(s.filepath, s.opts, func(r CensusRecord) {
		if stopErr != nil {
			return
		}
		if err := fn(r); err != nil {
			stopErr = err // capture and suppress further callbacks
			return
		}
		count++
	})
	if stopErr != nil {
		return count, stopErr
	}
	return count, err
}

// ---------------------------------------------------------------------------
// ReaderCensusSource — any io.Reader (HTTP body, stdin, bytes.Buffer, …)
// ---------------------------------------------------------------------------

// ReaderCensusSource is a CensusSource that reads census data from
// any io.Reader (HTTP request body, stdin, bytes.Buffer, etc.).
// Uses StreamCensusFromReader internally (sequential scanner; no mmap).
//
// For maximum throughput on large files, prefer FileCensusSource.
type ReaderCensusSource struct {
	r    io.Reader
	opts CSVOptions
}

// NewReaderCensusSource creates a CensusSource from any io.Reader.
func NewReaderCensusSource(r io.Reader, opts CSVOptions) *ReaderCensusSource {
	return &ReaderCensusSource{r: r, opts: opts}
}

// ReadAll reads all records from the io.Reader into memory.
func (s *ReaderCensusSource) ReadAll() ([]CensusRecord, error) {
	var records []CensusRecord
	_, err := s.Stream(func(r CensusRecord) error {
		records = append(records, r)
		return nil
	})
	return records, err
}

// Stream reads records from the io.Reader, calling fn for each.
// Uses StreamCensusFromReader internally (sequential scanner).
func (s *ReaderCensusSource) Stream(fn func(CensusRecord) error) (int, error) {
	var stopErr error
	count := 0
	err := StreamCensusFromReader(s.r, s.opts, func(r CensusRecord) {
		if stopErr != nil {
			return
		}
		if err := fn(r); err != nil {
			stopErr = err
			return
		}
		count++
	})
	if stopErr != nil {
		return count, stopErr
	}
	return count, err
}
