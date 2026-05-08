package repo

import (
	"strings"
	"testing"
)

func TestLoadPersons_parsesAllEightColumns(t *testing.T) {
	in := strings.NewReader("井村,いむら,imura,一輝,かずき,kazuki,男,A\n" +
		"細川,ほそかわ,hosokawa,芽以,めい,mei,女,O\n")

	got, err := LoadPersons(in)
	if err != nil {
		t.Fatalf("LoadPersons: unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(records) = %d, want 2", len(got))
	}
	want0 := PersonRecord{
		LastKanji: "井村", LastKana: "いむら", LastName: "imura",
		FirstKanji: "一輝", FirstKana: "かずき", FirstName: "kazuki",
		Gender: "男", BloodType: "A",
	}
	if got[0] != want0 {
		t.Errorf("records[0] = %+v, want %+v", got[0], want0)
	}
	if got[1].FirstName != "mei" || got[1].Gender != "女" {
		t.Errorf("records[1] = %+v", got[1])
	}
}

func TestLoadPersons_skipsBlankLines(t *testing.T) {
	in := strings.NewReader("井村,いむら,imura,一輝,かずき,kazuki,男,A\n\n\n")
	got, err := LoadPersons(in)
	if err != nil {
		t.Fatalf("LoadPersons: unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1", len(got))
	}
}

func TestLoadPersons_rejectsTooFewColumns(t *testing.T) {
	in := strings.NewReader("井村,いむら,imura\n")
	if _, err := LoadPersons(in); err == nil {
		t.Fatal("expected error for malformed row, got nil")
	}
}

func TestDefaultPersons_loadsEmbeddedDataset(t *testing.T) {
	got, err := DefaultPersons()
	if err != nil {
		t.Fatalf("DefaultPersons: %v", err)
	}
	if len(got) < 1000 {
		t.Fatalf("expected at least 1000 records, got %d", len(got))
	}
	for i, r := range got {
		if r.LastKanji == "" {
			t.Fatalf("records[%d].LastKanji is empty", i)
		}
	}
}
