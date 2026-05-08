package repo

import (
	"strings"
	"testing"
)

func TestLoadPrefectures_parsesNumericFields(t *testing.T) {
	prefs := strings.NewReader("01,北海道,5352306\n02,青森県,1293619\n")
	addrs := strings.NewReader("01,北海道,札幌市中央区,旭ケ丘\n01,北海道,札幌市中央区,大通東\n02,青森県,青森市,本町\n")

	repo, err := LoadPrefectures(prefs, addrs)
	if err != nil {
		t.Fatalf("LoadPrefectures: %v", err)
	}
	if len(repo.Prefectures) != 2 {
		t.Fatalf("len(Prefectures) = %d, want 2", len(repo.Prefectures))
	}
	if repo.Prefectures[0].Population != 5352306 {
		t.Errorf("Prefectures[0].Population = %d, want 5352306", repo.Prefectures[0].Population)
	}
	if repo.TotalPopulation != 5352306+1293619 {
		t.Errorf("TotalPopulation = %d, want %d", repo.TotalPopulation, 5352306+1293619)
	}
}

func TestLoadPrefectures_assignsZipsAndOffsets(t *testing.T) {
	prefs := strings.NewReader("01,北海道,100\n02,青森県,200\n03,岩手県,300\n")
	addrs := strings.NewReader(
		"01,北海道,札幌市,A\n" +
			"01,北海道,札幌市,B\n" +
			"02,青森県,青森市,X\n" +
			"03,岩手県,盛岡市,P\n" +
			"03,岩手県,盛岡市,Q\n" +
			"03,岩手県,盛岡市,R\n",
	)
	repo, err := LoadPrefectures(prefs, addrs)
	if err != nil {
		t.Fatalf("LoadPrefectures: %v", err)
	}
	wantZips := []int{2, 1, 3}
	wantOffsets := []int{0, 2, 3}
	for i, p := range repo.Prefectures {
		if p.Zips != wantZips[i] {
			t.Errorf("Prefectures[%d].Zips = %d, want %d", i, p.Zips, wantZips[i])
		}
		if p.AddressOffset != wantOffsets[i] {
			t.Errorf("Prefectures[%d].AddressOffset = %d, want %d", i, p.AddressOffset, wantOffsets[i])
		}
	}
	if len(repo.Addresses) != 6 {
		t.Errorf("len(Addresses) = %d, want 6", len(repo.Addresses))
	}
}

func TestWeightedPrefectureIndex_returnsByCumulativeWeight(t *testing.T) {
	prefs := strings.NewReader("01,A,100\n02,B,200\n03,C,300\n")
	addrs := strings.NewReader("01,A,X,Y\n02,B,X,Y\n03,C,X,Y\n")
	repo, err := LoadPrefectures(prefs, addrs)
	if err != nil {
		t.Fatalf("LoadPrefectures: %v", err)
	}

	cases := []struct {
		n    int
		want int
	}{
		{0, 0},   // [0,100) → A
		{99, 0},  // edge
		{100, 1}, // [100,300) → B
		{299, 1}, // edge
		{300, 2}, // [300,600) → C
		{599, 2}, // edge
		{600, 0}, // wraps via modulo → A
	}
	for _, c := range cases {
		got := repo.WeightedPrefectureIndex(c.n)
		if got != c.want {
			t.Errorf("WeightedPrefectureIndex(%d) = %d, want %d", c.n, got, c.want)
		}
	}
}

func TestDefaultPrefectures_loadsAllJapan(t *testing.T) {
	repo, err := DefaultPrefectures()
	if err != nil {
		t.Fatalf("DefaultPrefectures: %v", err)
	}
	if len(repo.Prefectures) != 47 {
		t.Fatalf("len(Prefectures) = %d, want 47", len(repo.Prefectures))
	}
	if repo.TotalPopulation < 100_000_000 {
		t.Errorf("TotalPopulation = %d, want > 100,000,000", repo.TotalPopulation)
	}
	if len(repo.Addresses) == 0 {
		t.Fatal("Addresses is empty")
	}
	zipSum := 0
	for _, p := range repo.Prefectures {
		if p.Zips < 0 {
			t.Errorf("negative Zips for %s", p.Name)
		}
		zipSum += p.Zips
	}
	if zipSum != len(repo.Addresses) {
		t.Errorf("sum(Zips) = %d, len(Addresses) = %d", zipSum, len(repo.Addresses))
	}
}
