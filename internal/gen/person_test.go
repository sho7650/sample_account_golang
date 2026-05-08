package gen

import (
	"strings"
	"testing"

	"sample_account/internal/repo"
)

func newPersonGen(t *testing.T) *PersonGen {
	t.Helper()
	recs := []repo.PersonRecord{
		{LastKanji: "井村", LastKana: "いむら", LastName: "imura", FirstKanji: "一輝", FirstKana: "かずき", FirstName: "kazuki", Gender: "男", BloodType: "A"},
		{LastKanji: "細川", LastKana: "ほそかわ", LastName: "hosokawa", FirstKanji: "芽以", FirstKana: "めい", FirstName: "mei", Gender: "女", BloodType: "O"},
	}
	return NewPersonGen(recs)
}

func TestPersonGen_lastNameJoinsKanjiAndKana(t *testing.T) {
	pg := newPersonGen(t)
	got := pg.LastName(0)
	if got != "井村,いむら" {
		t.Errorf("LastName(0) = %q, want %q", got, "井村,いむら")
	}
}

func TestPersonGen_firstNameJoinsKanjiAndKana(t *testing.T) {
	pg := newPersonGen(t)
	got := pg.FirstName(1)
	if got != "芽以,めい" {
		t.Errorf("FirstName(1) = %q, want %q", got, "芽以,めい")
	}
}

func TestPersonGen_mailAddressUsesRomajiAndAtSign(t *testing.T) {
	pg := newPersonGen(t)
	got := pg.MailAddress(1, 0)
	want := "mei_imura@example.com"
	if got != want {
		t.Errorf("MailAddress(1, 0) = %q, want %q", got, want)
	}
	if !strings.Contains(got, "@example.com") {
		t.Errorf("missing @example.com")
	}
}

func TestPersonGen_indexWrapsModulo(t *testing.T) {
	pg := newPersonGen(t)
	if pg.LastName(2) != pg.LastName(0) {
		t.Error("expected modulo wrap on index 2")
	}
}

func TestPersonGen_blood(t *testing.T) {
	pg := newPersonGen(t)
	if pg.Blood(0) != "A" || pg.Blood(1) != "O" {
		t.Errorf("Blood failed: %s %s", pg.Blood(0), pg.Blood(1))
	}
}

func TestPersonGen_gender(t *testing.T) {
	pg := newPersonGen(t)
	if pg.Gender(0) != "男" || pg.Gender(1) != "女" {
		t.Errorf("Gender failed: %s %s", pg.Gender(0), pg.Gender(1))
	}
}
