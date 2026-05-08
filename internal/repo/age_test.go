package repo

import (
	"strings"
	"testing"
)

func TestLoadAges_stripsThousandSeparators(t *testing.T) {
	in := strings.NewReader("0,4,987,706\n5,5,299,787\n10,5,599,317\n")
	repo, err := LoadAges(in)
	if err != nil {
		t.Fatalf("LoadAges: %v", err)
	}
	if len(repo.Buckets) != 3 {
		t.Fatalf("len(Buckets) = %d, want 3", len(repo.Buckets))
	}
	if got, want := repo.Buckets[0].Population, 4_987_706; got != want {
		t.Errorf("Buckets[0].Population = %d, want %d", got, want)
	}
	if got, want := repo.Buckets[1].Population, 5_299_787; got != want {
		t.Errorf("Buckets[1].Population = %d, want %d", got, want)
	}
	if repo.TotalAge != 4_987_706+5_299_787+5_599_317 {
		t.Errorf("TotalAge = %d", repo.TotalAge)
	}
}

func TestLoadAges_computesCumulativeStarts(t *testing.T) {
	in := strings.NewReader("0,1,000\n5,2,000\n10,3,000\n")
	repo, err := LoadAges(in)
	if err != nil {
		t.Fatalf("LoadAges: %v", err)
	}
	wantStarts := []int{0, 1000, 3000}
	for i, b := range repo.Buckets {
		if b.Start != wantStarts[i] {
			t.Errorf("Buckets[%d].Start = %d, want %d", i, b.Start, wantStarts[i])
		}
	}
}

func TestDefaultAges_loadsRealisticPopulation(t *testing.T) {
	repo, err := DefaultAges()
	if err != nil {
		t.Fatalf("DefaultAges: %v", err)
	}
	if len(repo.Buckets) == 0 {
		t.Fatal("no buckets loaded")
	}
	if repo.Buckets[0].Population < 1_000_000 {
		t.Errorf("Buckets[0].Population = %d (thousand-separators not stripped?)", repo.Buckets[0].Population)
	}
	if repo.TotalAge < 100_000_000 {
		t.Errorf("TotalAge = %d, want > 100,000,000", repo.TotalAge)
	}
}
