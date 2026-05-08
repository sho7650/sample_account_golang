package repo

import (
	"fmt"
	"io"
	"sort"
)

// PrefectureRecord describes one of Japan's 47 prefectures.
type PrefectureRecord struct {
	Number        int
	Name          string
	Population    int
	Zips          int // count of address rows belonging to this prefecture
	AddressOffset int // index into PrefectureRepo.Addresses where this prefecture's rows start
}

// AddressRecord is one row from address.csv.
type AddressRecord struct {
	Number     int
	Prefecture string
	Ward       string
	City       string
}

// PrefectureRepo holds prefecture + address tables along with
// pre-computed lookup structures used by the address generator.
type PrefectureRepo struct {
	Prefectures     []PrefectureRecord
	Addresses       []AddressRecord
	TotalPopulation int

	// cumulativePop[i] = sum of populations[0..=i]; used for weighted
	// prefecture selection via binary search.
	cumulativePop []int
}

// LoadPrefectures parses prefecture and address CSVs and pre-computes
// per-prefecture address offsets and cumulative population weights.
//
// Address rows are assumed to be grouped by prefecture number (matches
// the data file). Out-of-order rows would corrupt the offset table.
func LoadPrefectures(prefR, addrR io.Reader) (*PrefectureRepo, error) {
	repo := &PrefectureRepo{}

	prefLines, err := scanLines(prefR)
	if err != nil {
		return nil, fmt.Errorf("prefectures: %w", err)
	}
	repo.Prefectures = make([]PrefectureRecord, 0, len(prefLines))
	for i, line := range prefLines {
		f := splitN(line, 3)
		num, err := parseInt(f[0])
		if err != nil {
			return nil, fmt.Errorf("prefectures: row %d number: %w", i+1, err)
		}
		pop, err := parseInt(f[2])
		if err != nil {
			return nil, fmt.Errorf("prefectures: row %d population: %w", i+1, err)
		}
		repo.Prefectures = append(repo.Prefectures, PrefectureRecord{
			Number: num, Name: f[1], Population: pop,
		})
		repo.TotalPopulation += pop
	}

	addrLines, err := scanLines(addrR)
	if err != nil {
		return nil, fmt.Errorf("addresses: %w", err)
	}
	repo.Addresses = make([]AddressRecord, 0, len(addrLines))

	// Index Prefectures by Number for O(1) lookup while we walk addresses.
	byNumber := make(map[int]int, len(repo.Prefectures))
	for i, p := range repo.Prefectures {
		byNumber[p.Number] = i
	}

	currentPref := 0
	startIdx := 0
	for i, line := range addrLines {
		f := splitN(line, 4)
		num, err := parseInt(f[0])
		if err != nil {
			return nil, fmt.Errorf("addresses: row %d number: %w", i+1, err)
		}
		repo.Addresses = append(repo.Addresses, AddressRecord{
			Number: num, Prefecture: f[1], Ward: f[2], City: f[3],
		})
		if currentPref == 0 {
			currentPref = num
			startIdx = 0
		} else if num != currentPref {
			if idx, ok := byNumber[currentPref]; ok {
				repo.Prefectures[idx].Zips = i - startIdx
				repo.Prefectures[idx].AddressOffset = startIdx
			}
			currentPref = num
			startIdx = i
		}
	}
	if currentPref != 0 {
		if idx, ok := byNumber[currentPref]; ok {
			repo.Prefectures[idx].Zips = len(repo.Addresses) - startIdx
			repo.Prefectures[idx].AddressOffset = startIdx
		}
	}

	repo.cumulativePop = make([]int, len(repo.Prefectures))
	running := 0
	for i, p := range repo.Prefectures {
		running += p.Population
		repo.cumulativePop[i] = running
	}

	return repo, nil
}

// WeightedPrefectureIndex returns a prefecture index in [0, len(Prefectures))
// proportional to population. n is taken modulo TotalPopulation, matching
// the C++ reference behavior.
func (r *PrefectureRepo) WeightedPrefectureIndex(n int) int {
	if r.TotalPopulation == 0 || len(r.cumulativePop) == 0 {
		return 0
	}
	target := n % r.TotalPopulation
	if target < 0 {
		target += r.TotalPopulation
	}
	// First index whose cumulative population is strictly greater than target.
	i := sort.SearchInts(r.cumulativePop, target+1)
	if i >= len(r.cumulativePop) {
		return len(r.cumulativePop) - 1
	}
	return i
}

// DefaultPrefectures loads the embedded prefecture + address dataset.
func DefaultPrefectures() (*PrefectureRepo, error) {
	pf, err := dataFS.Open(prefecturesPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = pf.Close() }()
	af, err := dataFS.Open(addressesPath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = af.Close() }()
	return LoadPrefectures(pf, af)
}
