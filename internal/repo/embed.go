package repo

import "embed"

//go:embed data/*.csv
var dataFS embed.FS

const (
	personsPath     = "data/sample_account.csv"
	prefecturesPath = "data/prefectures.csv"
	addressesPath   = "data/address.csv"
	agesPath        = "data/ages.csv"
)
