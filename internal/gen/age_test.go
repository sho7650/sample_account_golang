package gen

import (
	"strings"
	"testing"

	"sample_account/internal/repo"
)

func loadAgeFixture(t *testing.T) *repo.AgeRepo {
	t.Helper()
	in := strings.NewReader("0,1,000\n5,2,000\n10,3,000\n15,4,000\n")
	r, err := repo.LoadAges(in)
	if err != nil {
		t.Fatalf("LoadAges: %v", err)
	}
	return r
}

func TestAgeGen_ageRespectsBucketBoundaries(t *testing.T) {
	r := loadAgeFixture(t)
	g := NewAgeGen(r)
	// Bucket [0,1000) → generation 0 → ages 0..4
	for n := 0; n < 1000; n++ {
		a := g.Age(n)
		if a < 0 || a >= 5 {
			t.Errorf("Age(%d) = %d, want 0..4", n, a)
		}
	}
}

func TestAgeGen_ageGroupRoundsDownToDecade(t *testing.T) {
	r := loadAgeFixture(t)
	g := NewAgeGen(r)
	if got := g.AgeGroup(500); got != 0 {
		t.Errorf("AgeGroup(500) = %d, want 0", got)
	}
	if got := g.AgeGroup(2500); got != 0 {
		t.Errorf("AgeGroup(2500) = %d, want 0 (gen=5 floored)", got)
	}
	if got := g.AgeGroup(5500); got != 10 {
		t.Errorf("AgeGroup(5500) = %d, want 10", got)
	}
}

func TestAgeGen_birthYearDerivesFromCurrentYear(t *testing.T) {
	t.Setenv("SAMPLE_ACCOUNT_NOW", "1700000000") // 2023 in UTC, 2023 in JST as well
	r := loadAgeFixture(t)
	g := NewAgeGen(r)
	a := g.Age(500)
	by := g.BirthYear(500)
	if by != 2023-a {
		t.Errorf("BirthYear(500) = %d, want %d", by, 2023-a)
	}
}

func TestAgeGen_rewardIsPositive(t *testing.T) {
	r := loadAgeFixture(t)
	g := NewAgeGen(r)
	rng := NewMasterRng(42)
	for n := 0; n < 100; n++ {
		got := g.Reward(n, rng)
		if got < 0 {
			t.Errorf("Reward(%d) = %d, want >= 0", n, got)
		}
	}
}
