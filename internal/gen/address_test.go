package gen

import (
	"strings"
	"testing"

	"sample_account/internal/repo"
)

func loadAddrFixture(t *testing.T) *repo.PrefectureRepo {
	t.Helper()
	prefR := strings.NewReader("01,北海道,100\n02,青森県,200\n03,岩手県,300\n")
	addrR := strings.NewReader(
		"01,北海道,札幌市,旭ケ丘\n" +
			"01,北海道,札幌市,大通\n" +
			"02,青森県,青森市,本町\n" +
			"03,岩手県,盛岡市,中央\n" +
			"03,岩手県,盛岡市,東\n" +
			"03,岩手県,盛岡市,西\n",
	)
	r, err := repo.LoadPrefectures(prefR, addrR)
	if err != nil {
		t.Fatalf("LoadPrefectures: %v", err)
	}
	return r
}

func TestAddressGen_prefectureNameIndex(t *testing.T) {
	r := loadAddrFixture(t)
	ag := NewAddressGen(r)
	if ag.PrefectureName(0) != "北海道" {
		t.Errorf("PrefectureName(0) = %q", ag.PrefectureName(0))
	}
	if ag.PrefectureName(2) != "岩手県" {
		t.Errorf("PrefectureName(2) = %q", ag.PrefectureName(2))
	}
}

func TestAddressGen_outOfRangePrefIndexFallsBackToZero(t *testing.T) {
	r := loadAddrFixture(t)
	ag := NewAddressGen(r)
	if ag.PrefectureName(99) != "北海道" {
		t.Errorf("out-of-range index should fall back to 0")
	}
	if ag.PrefectureName(-1) != "北海道" {
		t.Errorf("negative index should fall back to 0")
	}
}

func TestAddressGen_wardAndCityWithinPrefectureRange(t *testing.T) {
	r := loadAddrFixture(t)
	ag := NewAddressGen(r)
	// Pref 2 has 3 addresses: indices 3,4,5 in global
	got := ag.Ward(2, 0)
	if got != "盛岡市" {
		t.Errorf("Ward(2,0) = %q, want 盛岡市", got)
	}
	cities := map[string]bool{}
	for n := 0; n < 100; n++ {
		c := ag.City(2, n)
		cities[c] = true
	}
	for _, want := range []string{"中央", "東", "西"} {
		if !cities[want] {
			t.Errorf("City(2, *) never produced %q", want)
		}
	}
}

func TestAddressGen_negativeNHandled(t *testing.T) {
	r := loadAddrFixture(t)
	ag := NewAddressGen(r)
	if got := ag.Ward(0, -1); got == "" {
		t.Errorf("Ward with negative n returned empty")
	}
}
