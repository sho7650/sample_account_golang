package gen

import "sample_account/internal/repo"

// AddressGen wraps a PrefectureRepo with index-resolution helpers.
type AddressGen struct {
	repo *repo.PrefectureRepo
}

func NewAddressGen(r *repo.PrefectureRepo) *AddressGen { return &AddressGen{repo: r} }

func (a *AddressGen) clampPref(idx int) int {
	if len(a.repo.Prefectures) == 0 {
		return 0
	}
	if idx < 0 || idx >= len(a.repo.Prefectures) {
		return 0
	}
	return idx
}

// PrefectureName returns the prefecture's display name (defaults to index 0
// for out-of-range inputs, mirroring the C++ behavior).
func (a *AddressGen) PrefectureName(prefIdx int) string {
	return a.repo.Prefectures[a.clampPref(prefIdx)].Name
}

// addressIndex maps (prefIdx, n) to a global Addresses[] index.
// Uses the precomputed AddressOffset, so this is O(1).
func (a *AddressGen) addressIndex(prefIdx, n int) int {
	prefIdx = a.clampPref(prefIdx)
	p := a.repo.Prefectures[prefIdx]
	if p.Zips <= 0 {
		if p.AddressOffset < len(a.repo.Addresses) {
			return p.AddressOffset
		}
		return 0
	}
	if n < 0 {
		n = -n
	}
	return (n % p.Zips) + p.AddressOffset
}

func (a *AddressGen) Ward(prefIdx, n int) string {
	return a.repo.Addresses[a.addressIndex(prefIdx, n)].Ward
}

func (a *AddressGen) City(prefIdx, n int) string {
	return a.repo.Addresses[a.addressIndex(prefIdx, n)].City
}

// WeightedPrefectureIndex re-exports the repo's binary-search lookup so
// callers don't need to know about the repo type.
func (a *AddressGen) WeightedPrefectureIndex(n int) int {
	return a.repo.WeightedPrefectureIndex(n)
}
