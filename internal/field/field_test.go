package field

import (
	"strings"
	"testing"

	"sample_account/internal/gen"
	"sample_account/internal/repo"
)

func newTestDeps(t *testing.T) Deps {
	t.Helper()
	persons, err := repo.LoadPersons(strings.NewReader(
		"井村,いむら,imura,一輝,かずき,kazuki,男,A\n" +
			"細川,ほそかわ,hosokawa,芽以,めい,mei,女,O\n",
	))
	if err != nil {
		t.Fatalf("LoadPersons: %v", err)
	}
	prefR := strings.NewReader("01,北海道,100\n02,青森県,200\n")
	addrR := strings.NewReader("01,北海道,札幌市,A\n02,青森県,青森市,X\n")
	prefRepo, err := repo.LoadPrefectures(prefR, addrR)
	if err != nil {
		t.Fatalf("LoadPrefectures: %v", err)
	}
	ageRepo, err := repo.LoadAges(strings.NewReader("0,1,000\n10,2,000\n"))
	if err != nil {
		t.Fatalf("LoadAges: %v", err)
	}
	return Deps{
		Person:  gen.NewPersonGen(persons),
		Address: gen.NewAddressGen(prefRepo),
		Age:     gen.NewAgeGen(ageRepo),
		Rng:     gen.NewMasterRng(42),
	}
}

func emit(t *testing.T, f Field, ctx RowContext, deps Deps) string {
	t.Helper()
	return string(f.Emit(nil, ctx, deps))
}

func TestRegistry_addThenFindShortAndLong(t *testing.T) {
	reg := NewRegistry()
	reg.Add(&IDField{})
	reg.Add(&LastNameField{})

	if reg.FindShort('i') == nil {
		t.Error("FindShort('i') = nil")
	}
	if reg.FindLong("lastname") == nil {
		t.Error("FindLong(\"lastname\") = nil")
	}
	if reg.FindShort('z') != nil {
		t.Error("FindShort('z') should be nil")
	}
}

func TestRegistry_telephoneAliasResolves(t *testing.T) {
	reg := DefaultRegistry()
	if reg.FindLong("telehpne") == nil {
		t.Error("legacy --telehpne alias must resolve")
	}
	if reg.FindLong("telephone") == nil {
		t.Error("--telephone must resolve")
	}
}

func TestRegistry_shortOptStringHasAllFlags(t *testing.T) {
	reg := DefaultRegistry()
	s := reg.ShortOptString()
	for _, want := range "ilfmtpwcgbaoyrdnq" {
		if !strings.ContainsRune(s, want) {
			t.Errorf("ShortOptString missing %q (got %q)", want, s)
		}
	}
}

func TestIDField_emitsOneBasedRowNumber(t *testing.T) {
	deps := newTestDeps(t)
	got := emit(t, &IDField{}, RowContext{Row: 41}, deps)
	if got != "42" {
		t.Errorf("IDField row=41 = %q, want %q", got, "42")
	}
}

func TestLastNameField_emitsKanjiKana(t *testing.T) {
	deps := newTestDeps(t)
	got := emit(t, &LastNameField{}, RowContext{Last: 0}, deps)
	if got != "井村,いむら" {
		t.Errorf("LastName = %q", got)
	}
}

func TestMailField_combinesIndexes(t *testing.T) {
	deps := newTestDeps(t)
	got := emit(t, &MailField{}, RowContext{First: 1, Last: 0}, deps)
	if got != "mei_imura@example.com" {
		t.Errorf("Mail = %q", got)
	}
}

func TestPrefectureField_emitsNameForIndex(t *testing.T) {
	deps := newTestDeps(t)
	got := emit(t, &PrefectureField{}, RowContext{Pref: 0}, deps)
	if got != "北海道" {
		t.Errorf("Prefecture = %q", got)
	}
}

func TestTelephoneField_emitsValidPattern(t *testing.T) {
	deps := newTestDeps(t)
	got := emit(t, &TelephoneField{}, RowContext{}, deps)
	if !strings.HasPrefix(got, "090-") {
		t.Errorf("Telephone = %q (no 090- prefix)", got)
	}
	if len(got) != len("090-0000-0000") {
		t.Errorf("Telephone length = %d (got %q)", len(got), got)
	}
}

func TestQuotientField_emitsTwoDecimalFraction(t *testing.T) {
	deps := newTestDeps(t)
	got := emit(t, &QuotientField{}, RowContext{}, deps)
	// Format: "0.NN" where NN is two digits.
	if len(got) != 4 || got[0] != '0' || got[1] != '.' {
		t.Errorf("Quotient = %q, want 0.NN", got)
	}
}

func TestDateField_emitsYearMonthDay(t *testing.T) {
	t.Setenv("SAMPLE_ACCOUNT_NOW", "1700000000")
	deps := newTestDeps(t)
	deps.Rng.RollDate()
	got := emit(t, &DateField{}, RowContext{}, deps)
	if strings.Count(got, "/") != 2 {
		t.Errorf("Date = %q, want YYYY/M/D format", got)
	}
}

func TestRandomIntField_inRange(t *testing.T) {
	deps := newTestDeps(t)
	for i := 0; i < 100; i++ {
		got := emit(t, &RandomIntField{}, RowContext{}, deps)
		if got == "" {
			t.Fatal("empty output")
		}
	}
}

func TestDefaultRegistry_has17Fields(t *testing.T) {
	reg := DefaultRegistry()
	if got := len(reg.All()); got != 17 {
		t.Errorf("DefaultRegistry has %d fields, want 17", got)
	}
}
