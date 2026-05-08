package cli

import (
	"strings"
	"testing"

	"sample_account/internal/field"
)

func newReg() *field.Registry { return field.DefaultRegistry() }

func selectedShortFlags(args ParseResult) string {
	var b strings.Builder
	for _, f := range args.Selected {
		b.WriteByte(f.ShortFlag())
	}
	return b.String()
}

func TestParse_clusterFlagsPreserveOrder(t *testing.T) {
	r := newReg()
	args, err := Parse([]string{"prog", "-ilfm", "5"}, r)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := selectedShortFlags(args); got != "ilfm" {
		t.Errorf("selected = %q, want %q", got, "ilfm")
	}
	if args.Count != 5 {
		t.Errorf("Count = %d, want 5", args.Count)
	}
}

func TestParse_individualShortFlagsPreserveOrder(t *testing.T) {
	r := newReg()
	args, err := Parse([]string{"prog", "-i", "-l", "-f", "-m"}, r)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := selectedShortFlags(args); got != "ilfm" {
		t.Errorf("selected = %q, want %q", got, "ilfm")
	}
	if args.Count != DefaultRowCount {
		t.Errorf("Count = %d, want %d", args.Count, DefaultRowCount)
	}
}

func TestParse_clusterArbitraryOrder(t *testing.T) {
	r := newReg()
	args, _ := Parse([]string{"prog", "-imlf", "5"}, r)
	if got := selectedShortFlags(args); got != "imlf" {
		t.Errorf("selected = %q, want %q (order preservation)", got, "imlf")
	}
}

func TestParse_longFlagsPreserveOrder(t *testing.T) {
	r := newReg()
	args, err := Parse([]string{"prog", "--id", "--lastname", "--firstname", "--mail"}, r)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := selectedShortFlags(args); got != "ilfm" {
		t.Errorf("selected = %q, want %q", got, "ilfm")
	}
}

func TestParse_telehpneAliasResolves(t *testing.T) {
	r := newReg()
	args, err := Parse([]string{"prog", "--telehpne", "--agegroup", "--birthyear", "4"}, r)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got := selectedShortFlags(args); got != "toy" {
		t.Errorf("selected = %q, want %q", got, "toy")
	}
	if args.Count != 4 {
		t.Errorf("Count = %d, want 4", args.Count)
	}
}

func TestParse_unknownShortFlagErrors(t *testing.T) {
	r := newReg()
	if _, err := Parse([]string{"prog", "-Z"}, r); err == nil {
		t.Error("expected error for unknown short flag")
	}
}

func TestParse_unknownLongFlagErrors(t *testing.T) {
	r := newReg()
	if _, err := Parse([]string{"prog", "--unknown"}, r); err == nil {
		t.Error("expected error for unknown long flag")
	}
}

func TestParse_helpShortCircuits(t *testing.T) {
	r := newReg()
	args, err := Parse([]string{"prog", "--help"}, r)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if !args.Help {
		t.Error("Help flag should be true")
	}
}

func TestParse_noFlagsDefaultsToIdField(t *testing.T) {
	r := newReg()
	args, _ := Parse([]string{"prog", "3"}, r)
	if got := selectedShortFlags(args); got != "i" {
		t.Errorf("default selected = %q, want %q", got, "i")
	}
	if args.Count != 3 {
		t.Errorf("Count = %d, want 3", args.Count)
	}
}

func TestParse_invalidCountIgnored(t *testing.T) {
	r := newReg()
	args, _ := Parse([]string{"prog", "-i", "abc"}, r)
	if args.Count != DefaultRowCount {
		t.Errorf("Count = %d (non-numeric trailing arg should be ignored)", args.Count)
	}
}

func TestParse_jobsLongFormSpaceSeparated(t *testing.T) {
	r := newReg()
	args, err := Parse([]string{"prog", "--jobs", "4", "-i", "10"}, r)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if args.Jobs != 4 {
		t.Errorf("Jobs = %d, want 4", args.Jobs)
	}
	if args.Count != 10 {
		t.Errorf("Count = %d, want 10", args.Count)
	}
}

func TestParse_jobsLongFormEquals(t *testing.T) {
	r := newReg()
	args, err := Parse([]string{"prog", "--jobs=8", "-i", "5"}, r)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if args.Jobs != 8 {
		t.Errorf("Jobs = %d, want 8", args.Jobs)
	}
}

func TestParse_jobsShortFormSpaceSeparated(t *testing.T) {
	r := newReg()
	args, err := Parse([]string{"prog", "-j", "1", "-i"}, r)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if args.Jobs != 1 {
		t.Errorf("Jobs = %d, want 1", args.Jobs)
	}
}

func TestParse_jobsShortFormAttached(t *testing.T) {
	r := newReg()
	args, err := Parse([]string{"prog", "-j2", "-i"}, r)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if args.Jobs != 2 {
		t.Errorf("Jobs = %d, want 2", args.Jobs)
	}
}

func TestParse_jobsDefaultsToZero(t *testing.T) {
	r := newReg()
	args, _ := Parse([]string{"prog", "-i"}, r)
	if args.Jobs != 0 {
		t.Errorf("Jobs = %d, want 0 (auto)", args.Jobs)
	}
}

func TestParse_jobsNegativeRejected(t *testing.T) {
	r := newReg()
	if _, err := Parse([]string{"prog", "--jobs", "-1"}, r); err == nil {
		t.Error("expected error for negative jobs value")
	}
}

func TestParse_jobsNonNumericRejected(t *testing.T) {
	r := newReg()
	if _, err := Parse([]string{"prog", "--jobs", "abc"}, r); err == nil {
		t.Error("expected error for non-numeric jobs value")
	}
}

func TestParse_jobsMissingArgRejected(t *testing.T) {
	r := newReg()
	if _, err := Parse([]string{"prog", "--jobs"}, r); err == nil {
		t.Error("expected error when --jobs has no argument")
	}
}

func TestPrintHelp_listsAllFlags(t *testing.T) {
	r := newReg()
	var b strings.Builder
	PrintHelp(&b, "prog", r)
	out := b.String()
	for _, want := range []string{"--id", "--lastname", "--telephone", "--telehpne", "SAMPLE_ACCOUNT_SEED", "Usage:"} {
		if !strings.Contains(out, want) {
			t.Errorf("help missing %q", want)
		}
	}
}
