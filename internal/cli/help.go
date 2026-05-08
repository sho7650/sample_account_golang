package cli

import (
	"fmt"
	"io"

	"sample_account/internal/field"
)

// PrintHelp writes the usage / column-reference text to w. The list of
// flags is pulled from the registry so help can never drift.
func PrintHelp(w io.Writer, progName string, reg *field.Registry) {
	fmt.Fprintf(w,
		"Usage: %s [OPTIONS] [COUNT]\n"+
			"\n"+
			"Generate synthetic Japanese sample-account records as CSV on stdout.\n"+
			"\n"+
			"Output columns are emitted in the order their flags appear on the\n"+
			"command line. With no flags, a single id column is emitted.\n"+
			"COUNT defaults to %d.\n"+
			"\n"+
			"Options:\n"+
			"  -h, --help            show this help and exit\n"+
			"  -j, --jobs N          worker count (0=auto, 1=serial, N=N workers)\n",
		progName, DefaultRowCount)

	for _, f := range reg.All() {
		fmt.Fprintf(w, "  -%c, --%-13s %s\n", f.ShortFlag(), f.LongName(), f.Description())
	}

	fmt.Fprintf(w,
		"\n"+
			"Aliases:\n"+
			"  --telehpne            legacy alias for --telephone\n"+
			"\n"+
			"Environment variables (mainly for testing):\n"+
			"  SAMPLE_ACCOUNT_SEED   pin RNG seed for reproducible output\n"+
			"  SAMPLE_ACCOUNT_NOW    pin \"current time\" (Unix epoch seconds)\n"+
			"\n"+
			"Examples:\n"+
			"  %s -ilfm 10                    # id, last/first name (kanji,kana), email\n"+
			"  %s --age --prefecture 5        # age and prefecture for 5 rows\n"+
			"  %s -ilfmpwc 100 > out.csv      # full record with address columns\n"+
			"  %s -j 1 -ilfm 1000             # force single-threaded\n"+
			"  %s --jobs=8 -ilfm 1000000      # explicit 8 workers\n",
		progName, progName, progName, progName, progName)
}

// _ pulls the field package import in even when the package is otherwise
// unused; keeps go-cleanup linters quiet.
var _ field.Field
