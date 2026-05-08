package gen

import "sample_account/internal/repo"

// PersonGen wraps person records with field-specific lookups.
//
// lastNameJoined / firstNameJoined hold pre-concatenated "<kanji>,<kana>"
// strings so LastName/FirstName are a single slice lookup with no
// allocation on the hot path. The records slice is retained so MailAddress
// can build "first.romaji_last.romaji@example.com" on demand.
type PersonGen struct {
	records          []repo.PersonRecord
	lastNameJoined   []string
	firstNameJoined  []string
	mailLocalCache   []string // first.romaji + "_" + last.romaji material is too large to precompute pairwise; cache "first_" prefix
}

func NewPersonGen(records []repo.PersonRecord) *PersonGen {
	pg := &PersonGen{
		records:         records,
		lastNameJoined:  make([]string, len(records)),
		firstNameJoined: make([]string, len(records)),
		mailLocalCache:  make([]string, len(records)),
	}
	for i, r := range records {
		pg.lastNameJoined[i] = r.LastKanji + "," + r.LastKana
		pg.firstNameJoined[i] = r.FirstKanji + "," + r.FirstKana
		pg.mailLocalCache[i] = r.FirstName + "_"
	}
	return pg
}

func (p *PersonGen) modIndex(n int) int {
	if len(p.records) == 0 {
		return 0
	}
	idx := n % len(p.records)
	if idx < 0 {
		idx += len(p.records)
	}
	return idx
}

// LastName returns "<kanji>,<kana>" — TWO comma-separated CSV fields.
// Returns a pre-joined string from the cache to avoid per-row allocation.
func (p *PersonGen) LastName(n int) string {
	return p.lastNameJoined[p.modIndex(n)]
}

// FirstName returns "<kanji>,<kana>" — TWO comma-separated CSV fields.
// Returns a pre-joined string from the cache to avoid per-row allocation.
func (p *PersonGen) FirstName(n int) string {
	return p.firstNameJoined[p.modIndex(n)]
}

// MailAddress synthesizes "<first.romaji>_<last.romaji>@example.com".
// Still allocates one string per call; field-level Emit can append parts
// directly to the row buffer for a fully alloc-free path.
func (p *PersonGen) MailAddress(first, last int) string {
	return p.mailLocalCache[p.modIndex(first)] +
		p.records[p.modIndex(last)].LastName + "@example.com"
}

// AppendMailAddress writes the email directly into buf without allocating
// an intermediate string.
func (p *PersonGen) AppendMailAddress(buf []byte, first, last int) []byte {
	buf = append(buf, p.mailLocalCache[p.modIndex(first)]...)
	buf = append(buf, p.records[p.modIndex(last)].LastName...)
	return append(buf, "@example.com"...)
}

func (p *PersonGen) Gender(n int) string { return p.records[p.modIndex(n)].Gender }
func (p *PersonGen) Blood(n int) string  { return p.records[p.modIndex(n)].BloodType }
