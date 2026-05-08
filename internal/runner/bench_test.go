package runner

import (
	"io"
	"strings"
	"testing"

	"sample_account/internal/field"
	"sample_account/internal/gen"
	"sample_account/internal/repo"
)

func benchDeps(b *testing.B) Deps {
	b.Helper()
	persons, err := repo.DefaultPersons()
	if err != nil {
		b.Fatal(err)
	}
	prefRepo, err := repo.DefaultPrefectures()
	if err != nil {
		b.Fatal(err)
	}
	ageRepo, err := repo.DefaultAges()
	if err != nil {
		b.Fatal(err)
	}
	return Deps{
		Persons:     persons,
		Prefectures: prefRepo,
		Ages:        ageRepo,
		AddressGen:  gen.NewAddressGen(prefRepo),
	}
}

func benchAllFlags(b *testing.B, count int) {
	deps := benchDeps(b)
	reg := field.DefaultRegistry()
	fields := []field.Field{
		reg.FindShort('i'), reg.FindShort('l'), reg.FindShort('f'),
		reg.FindShort('m'), reg.FindShort('a'), reg.FindShort('t'),
		reg.FindShort('p'), reg.FindShort('w'), reg.FindShort('c'),
		reg.FindShort('g'), reg.FindShort('b'), reg.FindShort('d'),
		reg.FindShort('o'), reg.FindShort('r'), reg.FindShort('y'),
		reg.FindShort('n'), reg.FindShort('q'),
	}
	b.Setenv("SAMPLE_ACCOUNT_NOW", "1700000000")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := Run(io.Discard, count, fields, deps, 42); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRun_AllFlags100(b *testing.B)      { benchAllFlags(b, 100) }
func BenchmarkRun_AllFlags10K(b *testing.B)      { benchAllFlags(b, 10_000) }
func BenchmarkRun_AllFlags1M(b *testing.B)       { benchAllFlags(b, 1_000_000) }

// Sanity: builder is initialized so import isn't unused on go vet runs that
// strip the rest.
var _ = strings.NewReader
