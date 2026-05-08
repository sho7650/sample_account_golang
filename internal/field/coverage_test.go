package field

import (
	"testing"
)

// All zero-sized field structs satisfy Field. This walks DefaultRegistry
// to make sure each implementation responds correctly to its metadata
// methods, which boosts coverage of the trivial accessor methods.
func TestAllFields_metadataIsConsistent(t *testing.T) {
	reg := DefaultRegistry()
	seen := make(map[byte]bool)
	for _, f := range reg.All() {
		s := f.ShortFlag()
		if s == 0 {
			t.Errorf("field %T has zero short flag", f)
		}
		if seen[s] {
			t.Errorf("field %T duplicates short flag %q", f, s)
		}
		seen[s] = true
		if f.LongName() == "" {
			t.Errorf("field %T has empty long name", f)
		}
		if f.Description() == "" {
			t.Errorf("field %T has empty description", f)
		}
	}
}

func TestAllFields_emitProducesNonEmptyOutput(t *testing.T) {
	deps := newTestDeps(t)
	deps.Rng.RollDate()
	reg := DefaultRegistry()
	ctx := RowContext{Row: 0, First: 1, Last: 0, Pref: 0, Ward: 0, City: 0, Age: 0}
	for _, f := range reg.All() {
		got := f.Emit(nil, ctx, deps)
		if len(got) == 0 {
			t.Errorf("field %T emitted empty output", f)
		}
	}
}

func TestAppendPaddedInt_zeroAndNegative(t *testing.T) {
	if got := string(appendPaddedInt(nil, 0, 4)); got != "0000" {
		t.Errorf("appendPaddedInt(0, 4) = %q, want 0000", got)
	}
	if got := string(appendPaddedInt(nil, -5, 4)); got != "0005" {
		t.Errorf("appendPaddedInt(-5, 4) = %q, want 0005", got)
	}
	if got := string(appendPaddedInt(nil, 1234, 4)); got != "1234" {
		t.Errorf("appendPaddedInt(1234, 4) = %q, want 1234", got)
	}
}
