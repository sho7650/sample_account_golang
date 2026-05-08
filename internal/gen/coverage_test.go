package gen

import (
	"strings"
	"testing"

	"sample_account/internal/repo"
)

func TestNewRowRngWithNow_capturesSuppliedNow(t *testing.T) {
	rng := NewRowRngWithNow(42, 1, 1700000000)
	y, m, d := rng.RollDate()
	if y < 1970 || y > 2024 {
		t.Errorf("RollDate year out of expected range with nowUnix=1700000000: %d", y)
	}
	if m < 1 || m > 12 || d < 1 || d > 31 {
		t.Errorf("invalid month/day: m=%d d=%d", m, d)
	}
}

func TestRollDate_zeroNowFallsBackToEpoch(t *testing.T) {
	rng := NewRowRngWithNow(0, 0, 0)
	y, m, d := rng.RollDate()
	if y != 1970 || m != 1 || d != 1 {
		t.Errorf("RollDate with zero nowUnix = %d/%d/%d, want 1970/1/1", y, m, d)
	}
}

func TestMasterSeed_envOverridesWallClock(t *testing.T) {
	t.Setenv("SAMPLE_ACCOUNT_SEED", "12345")
	if got := MasterSeed(); got != 12345 {
		t.Errorf("MasterSeed() = %d, want 12345", got)
	}
}

func TestMasterSeed_invalidEnvFallsBack(t *testing.T) {
	t.Setenv("SAMPLE_ACCOUNT_SEED", "not-a-number")
	if got := MasterSeed(); got == 0 {
		t.Error("MasterSeed should fall back to non-zero wall clock for invalid env")
	}
}

func TestCurrentTime_invalidEnvFallsBackToWallClock(t *testing.T) {
	t.Setenv("SAMPLE_ACCOUNT_NOW", "garbage")
	got := CurrentTime()
	if got.IsZero() {
		t.Error("CurrentTime should not be zero when env is invalid")
	}
}

func TestAddressGen_proxyHelpers(t *testing.T) {
	prefR := strings.NewReader("01,北海道,100\n02,青森県,200\n")
	addrR := strings.NewReader("01,北海道,札幌市,A\n02,青森県,青森市,X\n")
	r, err := repo.LoadPrefectures(prefR, addrR)
	if err != nil {
		t.Fatalf("LoadPrefectures: %v", err)
	}
	ag := NewAddressGen(r)
	idx := ag.WeightedPrefectureIndex(150)
	if idx < 0 || idx > 1 {
		t.Errorf("WeightedPrefectureIndex(150) = %d, out of range", idx)
	}
}

func TestAgeGen_zeroTotalIsHandled(t *testing.T) {
	empty := &repo.AgeRepo{}
	g := NewAgeGen(empty)
	if g.totalAgeMod(42) != 0 {
		t.Error("totalAgeMod with empty repo should be 0")
	}
}
