package repo

import (
	"bufio"
	"io"
	"strconv"
	"strings"
)

// scanLines yields trimmed, non-empty lines from r.
// Returns the slice in source order so callers can iterate without
// retaining an open reader.
func scanLines(r io.Reader) ([]string, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	var lines []string
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			continue
		}
		lines = append(lines, line)
	}
	return lines, scanner.Err()
}

// splitN slices line on commas into at most n fields.
// If the line has fewer than n commas, the trailing fields are empty.
// If it has more, the final field absorbs the remaining commas verbatim
// (matches std::getline ',' semantics for the last column).
func splitN(line string, n int) []string {
	out := make([]string, n)
	idx := 0
	for i := 0; i < n-1; i++ {
		comma := strings.IndexByte(line[idx:], ',')
		if comma < 0 {
			out[i] = line[idx:]
			idx = len(line)
			continue
		}
		out[i] = line[idx : idx+comma]
		idx += comma + 1
	}
	out[n-1] = line[idx:]
	return out
}

// parseInt parses a base-10 signed int without strconv overhead.
// Treats empty input as 0 to mirror C `atoi` permissiveness.
func parseInt(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	neg := false
	i := 0
	if s[0] == '-' {
		neg = true
		i = 1
	}
	n := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, &parseIntError{Input: s}
		}
		n = n*10 + int(c-'0')
	}
	if neg {
		n = -n
	}
	return n, nil
}

type parseIntError struct{ Input string }

func (e *parseIntError) Error() string { return "parseInt: invalid input " + strconv.Quote(e.Input) }
