package repo

import (
	"fmt"
	"io"
	"strings"
)

// PersonRecord is a single row from sample_account.csv.
type PersonRecord struct {
	LastKanji  string
	LastKana   string
	LastName   string
	FirstKanji string
	FirstKana  string
	FirstName  string
	Gender     string
	BloodType  string
}

// LoadPersons reads CSV rows from r into PersonRecords.
// Each line must have exactly eight comma-separated fields.
func LoadPersons(r io.Reader) ([]PersonRecord, error) {
	lines, err := scanLines(r)
	if err != nil {
		return nil, err
	}
	out := make([]PersonRecord, 0, len(lines))
	for i, line := range lines {
		if strings.Count(line, ",") < 7 {
			return nil, fmt.Errorf("persons: row %d malformed (need 8 fields): %q", i+1, line)
		}
		f := splitN(line, 8)
		out = append(out, PersonRecord{
			LastKanji: f[0], LastKana: f[1], LastName: f[2],
			FirstKanji: f[3], FirstKana: f[4], FirstName: f[5],
			Gender: f[6], BloodType: f[7],
		})
	}
	return out, nil
}

// DefaultPersons loads the embedded person dataset.
func DefaultPersons() ([]PersonRecord, error) {
	f, err := dataFS.Open(personsPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return LoadPersons(f)
}
