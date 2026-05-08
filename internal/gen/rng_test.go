package gen

import (
	"testing"
	"time"
)

func TestNewMasterRng_isDeterministicForFixedSeed(t *testing.T) {
	a := NewMasterRng(42)
	b := NewMasterRng(42)
	for i := 0; i < 1000; i++ {
		x, y := a.Next(), b.Next()
		if x != y {
			t.Fatalf("rng diverged at i=%d: %d vs %d", i, x, y)
		}
	}
}

func TestNewMasterRng_differsAcrossSeeds(t *testing.T) {
	a := NewMasterRng(42)
	b := NewMasterRng(43)
	mismatch := 0
	for i := 0; i < 100; i++ {
		if a.Next() != b.Next() {
			mismatch++
		}
	}
	if mismatch < 95 {
		t.Errorf("seeds 42 vs 43 produced too few differences: %d/100", mismatch)
	}
}

func TestNewRowRng_independentRowsAreDecorrelated(t *testing.T) {
	const master = uint64(0xCAFEBABE)
	rngs := make([]*Rng, 1000)
	for i := range rngs {
		rngs[i] = NewRowRng(master, uint64(i))
	}
	dup := 0
	for i := 1; i < len(rngs); i++ {
		if rngs[i].Next() == rngs[i-1].Next() {
			dup++
		}
	}
	if dup > 5 {
		t.Errorf("too many adjacent rows produced equal first draw: %d/999", dup)
	}
}

func TestRollDate_returnsCalendarComponents(t *testing.T) {
	t.Setenv("SAMPLE_ACCOUNT_NOW", "1700000000") // 2023-11-14 22:13:20 UTC
	rng := NewMasterRng(7)
	for i := 0; i < 100; i++ {
		y, m, d := rng.RollDate()
		if y < 1970 || y > 2100 {
			t.Fatalf("year out of range: %d", y)
		}
		if m < 1 || m > 12 {
			t.Fatalf("month out of range: %d", m)
		}
		if d < 1 || d > 31 {
			t.Fatalf("day out of range: %d", d)
		}
	}
}

func TestCurrentTime_honorsSampleAccountNow(t *testing.T) {
	t.Setenv("SAMPLE_ACCOUNT_NOW", "1234567890")
	got := CurrentTime()
	if got.Unix() != 1234567890 {
		t.Errorf("CurrentTime = %d, want 1234567890", got.Unix())
	}
}

func TestCurrentTime_fallsBackToWallClock(t *testing.T) {
	t.Setenv("SAMPLE_ACCOUNT_NOW", "")
	got := CurrentTime()
	now := time.Now()
	if got.Unix() < now.Add(-2*time.Second).Unix() || got.Unix() > now.Add(2*time.Second).Unix() {
		t.Errorf("CurrentTime() = %v, want near %v", got, now)
	}
}
