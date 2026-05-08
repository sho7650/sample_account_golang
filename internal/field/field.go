package field

import "sample_account/internal/gen"

// RowContext holds the per-row state shared by every field. The runner
// fills these once per row from the row's RNG so multiple fields stay
// consistent (e.g. First drives both first-name and email's local-part).
type RowContext struct {
	Row   int // 0-based row index
	First int
	Last  int
	Pref  int // population-weighted prefecture index
	Ward  int
	City  int
	Age   int
}

// Deps bundles the generators a field may need.
type Deps struct {
	Person  *gen.PersonGen
	Address *gen.AddressGen
	Age     *gen.AgeGen
	Rng     *gen.Rng
}

// Field is the strategy interface for a single CSV column.
//
// Emit appends the column's value to buf and returns the resulting slice
// (matches the strconv.Append* convention so callers can chain without
// allocating intermediate strings).
type Field interface {
	ShortFlag() byte
	LongName() string
	Description() string
	Emit(buf []byte, ctx RowContext, deps Deps) []byte
}
