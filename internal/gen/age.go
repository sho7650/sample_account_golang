package gen

import "sample_account/internal/repo"

// AgeGen wraps the age-bucket table with derived metric helpers.
// nowYear is cached at construction so BirthYear does not call time.Now
// or os.Getenv on the hot path.
type AgeGen struct {
	repo    *repo.AgeRepo
	nowYear int
}

func NewAgeGen(r *repo.AgeRepo) *AgeGen {
	return &AgeGen{repo: r, nowYear: CurrentTime().Year()}
}

// findBucket returns the bucket whose [start, start+population) range
// contains `total`. Linear scan is fine — there are 19 buckets at most.
func (g *AgeGen) findBucket(total int) int {
	buckets := g.repo.Buckets
	for i := 0; i+1 < len(buckets); i++ {
		if buckets[i+1].Start > total {
			return i
		}
	}
	return len(buckets) - 1
}

func (g *AgeGen) totalAgeMod(n int) int {
	if g.repo.TotalAge == 0 {
		return 0
	}
	v := n % g.repo.TotalAge
	if v < 0 {
		v += g.repo.TotalAge
	}
	return v
}

// Age returns generation + (n % 5), giving a per-bucket spread.
func (g *AgeGen) Age(n int) int {
	total := g.totalAgeMod(n)
	i := g.findBucket(total)
	return g.repo.Buckets[i].Generation + ((n%5)+5)%5
}

// AgeGroup floors the bucket generation to its decade.
func (g *AgeGen) AgeGroup(n int) int {
	total := g.totalAgeMod(n)
	i := g.findBucket(total)
	return (g.repo.Buckets[i].Generation / 10) * 10
}

// BirthYear estimates birth year as currentYear - Age(n).
func (g *AgeGen) BirthYear(n int) int {
	return g.nowYear - g.Age(n)
}

// Reward derives a synthetic income figure from the age group.
func (g *AgeGen) Reward(n int, rng *Rng) int {
	mod := func(v int) int {
		if v < 0 {
			return -v
		}
		return v
	}
	base := 50 - abs(g.AgeGroup(n)-50) + (rng.Next() % 5)
	return base * (mod(rng.Next())%3 + 1) * 100000
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
