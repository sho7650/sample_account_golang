//go:build snapshot

package tests

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// runCase runs the binary with the supplied args under deterministic env
// vars and compares stdout to the checked-in expected file.
func runCase(t *testing.T, name, expected string, args ...string) {
	t.Helper()
	bin := findBinary(t)
	cmd := exec.Command(bin, args...)
	cmd.Env = append(os.Environ(),
		"SAMPLE_ACCOUNT_SEED=42",
		"SAMPLE_ACCOUNT_NOW=1700000000",
		"TZ=Asia/Tokyo",
	)
	got, err := cmd.Output()
	if err != nil {
		t.Fatalf("[%s] run failed: %v", name, err)
	}
	want, err := os.ReadFile(expected)
	if err != nil {
		t.Fatalf("[%s] read expected: %v", name, err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("[%s] mismatch\n--- want ---\n%s\n--- got ---\n%s",
			name, string(want), string(got))
	}
}

// findBinary returns the absolute path to the sample_account binary at
// the repo root, building it if missing.
func findBinary(t *testing.T) string {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Dir(filepath.Dir(thisFile))
	bin := filepath.Join(root, "sample_account")
	if _, err := os.Stat(bin); err == nil {
		return bin
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/sample_account")
	cmd.Dir = root
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build sample_account: %v\n%s", err, out)
	}
	return bin
}

func TestSnapshot_AllFlags(t *testing.T) {
	runCase(t, "all flags", expectedPath("all-flags-seed-42.csv"), "-ilfmatpwcgbdorynq", "5")
}

func TestSnapshot_IdNameMail(t *testing.T) {
	runCase(t, "id+name+mail", expectedPath("ilfm-seed-42.csv"), "-ilfm", "5")
}

func TestSnapshot_DefaultNoFlags(t *testing.T) {
	runCase(t, "default", expectedPath("default-seed-42.csv"), "3")
}

func TestSnapshot_LongAliases(t *testing.T) {
	runCase(t, "long aliases", expectedPath("long-aliases-seed-42.csv"), "--telephone", "--agegroup", "--birthyear", "4")
}

func expectedPath(name string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "expected", name)
}

var _ = fmt.Sprintf // keep fmt referenced for future debug formatting
