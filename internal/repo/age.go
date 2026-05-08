package repo

import (
	"fmt"
	"io"
)

// AgeBucket represents one row from ages.csv.
// Start is the running sum of prior populations and is used to map
// a uniform random integer to a bucket via WeightedAgeIndex-style logic.
type AgeBucket struct {
	Generation int
	Population int
	Start      int
}

// AgeRepo holds the loaded age buckets and total population.
type AgeRepo struct {
	Buckets  []AgeBucket
	TotalAge int
}

// LoadAges parses ages.csv content. The population column may contain
// thousand-separators (e.g. "4,987,706"); the parser strips all
// non-digit bytes before parsing, mirroring the C++ behavior.
func LoadAges(r io.Reader) (*AgeRepo, error) {
	lines, err := scanLines(r)
	if err != nil {
		return nil, err
	}
	repo := &AgeRepo{Buckets: make([]AgeBucket, 0, len(lines))}
	for i, line := range lines {
		// First comma separates generation; everything after is population
		// (which itself contains commas as thousand-separators).
		gen := 0
		idx := 0
		for j := 0; j < len(line); j++ {
			if line[j] == ',' {
				g, err := parseInt(line[:j])
				if err != nil {
					return nil, fmt.Errorf("ages: row %d generation: %w", i+1, err)
				}
				gen = g
				idx = j + 1
				break
			}
		}
		pop := parseDigits(line[idx:])
		b := AgeBucket{
			Generation: gen,
			Population: pop,
			Start:      repo.TotalAge,
		}
		repo.TotalAge += pop
		repo.Buckets = append(repo.Buckets, b)
	}
	return repo, nil
}

// parseDigits drops every non-digit byte and parses the remainder.
// Empty input yields 0.
func parseDigits(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

// DefaultAges loads the embedded ages dataset.
func DefaultAges() (*AgeRepo, error) {
	f, err := dataFS.Open(agesPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return LoadAges(f)
}
