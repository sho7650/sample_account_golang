package runner

import (
	"bytes"
	"strings"
	"testing"

	"sample_account/internal/field"
	"sample_account/internal/gen"
	"sample_account/internal/repo"
)

func newDeps(t *testing.T) Deps {
	t.Helper()
	persons, err := repo.LoadPersons(strings.NewReader(
		"井村,いむら,imura,一輝,かずき,kazuki,男,A\n" +
			"細川,ほそかわ,hosokawa,芽以,めい,mei,女,O\n",
	))
	if err != nil {
		t.Fatalf("LoadPersons: %v", err)
	}
	prefRepo, err := repo.LoadPrefectures(
		strings.NewReader("01,北海道,100\n02,青森県,200\n"),
		strings.NewReader("01,北海道,札幌市,A\n02,青森県,青森市,X\n"),
	)
	if err != nil {
		t.Fatalf("LoadPrefectures: %v", err)
	}
	ageRepo, err := repo.LoadAges(strings.NewReader("0,1,000\n10,2,000\n"))
	if err != nil {
		t.Fatalf("LoadAges: %v", err)
	}
	return Deps{
		Persons:     persons,
		Prefectures: prefRepo,
		Ages:        ageRepo,
		AddressGen:  gen.NewAddressGen(prefRepo),
	}
}

func TestRun_zeroCountProducesNoOutput(t *testing.T) {
	deps := newDeps(t)
	var buf bytes.Buffer
	fields := []field.Field{&field.IDField{}}
	if err := Run(&buf, 0, fields, deps, 42); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestRun_idFieldYieldsOneBasedSequentialIds(t *testing.T) {
	deps := newDeps(t)
	var buf bytes.Buffer
	fields := []field.Field{&field.IDField{}}
	if err := Run(&buf, 5, fields, deps, 42); err != nil {
		t.Fatalf("Run: %v", err)
	}
	want := "1\n2\n3\n4\n5\n"
	if got := buf.String(); got != want {
		t.Errorf("Run output = %q, want %q", got, want)
	}
}

func TestRun_serialAndParallelProduceIdenticalOutput(t *testing.T) {
	deps := newDeps(t)
	fields := []field.Field{&field.IDField{}, &field.LastNameField{}, &field.FirstNameField{}, &field.MailField{}}
	const masterSeed = uint64(0xDEADBEEF)
	const count = 10000

	var serial, parallel bytes.Buffer
	if err := runSerial(&serial, count, fields, deps, masterSeed); err != nil {
		t.Fatalf("runSerial: %v", err)
	}
	if err := Run(&parallel, count, fields, deps, masterSeed); err != nil {
		t.Fatalf("Run (parallel): %v", err)
	}
	if !bytes.Equal(serial.Bytes(), parallel.Bytes()) {
		t.Fatalf("serial and parallel diverged at count=%d (lengths %d vs %d)",
			count, serial.Len(), parallel.Len())
	}
}

func TestRun_differentMasterSeedsChangeOutput(t *testing.T) {
	deps := newDeps(t)
	fields := []field.Field{&field.AgeField{}, &field.RandomIntField{}}
	var a, b bytes.Buffer
	if err := Run(&a, 100, fields, deps, 1); err != nil {
		t.Fatal(err)
	}
	if err := Run(&b, 100, fields, deps, 2); err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Error("different seeds produced identical output")
	}
}

func TestRun_workersOneForcesSerialPath(t *testing.T) {
	deps := newDeps(t)
	fields := []field.Field{&field.IDField{}, &field.LastNameField{}, &field.AgeField{}}
	const masterSeed = uint64(0xCAFE)
	const count = 5000

	var auto, forced bytes.Buffer
	if err := Run(&auto, count, fields, deps, masterSeed); err != nil {
		t.Fatalf("Run (auto): %v", err)
	}
	if err := RunWithJobs(&forced, count, fields, deps, masterSeed, 1); err != nil {
		t.Fatalf("Run (workers=1): %v", err)
	}
	if !bytes.Equal(auto.Bytes(), forced.Bytes()) {
		t.Errorf("auto and workers=1 outputs differ (lengths %d vs %d)",
			auto.Len(), forced.Len())
	}
}

func TestRun_explicitWorkersAbove1MatchesAuto(t *testing.T) {
	deps := newDeps(t)
	fields := []field.Field{&field.IDField{}, &field.MailField{}}
	const masterSeed = uint64(0xBABE)
	const count = 8000

	var auto, four bytes.Buffer
	if err := Run(&auto, count, fields, deps, masterSeed); err != nil {
		t.Fatal(err)
	}
	if err := RunWithJobs(&four, count, fields, deps, masterSeed, 4); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(auto.Bytes(), four.Bytes()) {
		t.Errorf("auto and workers=4 diverge")
	}
}

func TestRun_largeCountSpansMultipleSubChunks(t *testing.T) {
	// Drives at least ~5 sub-chunks per worker so the inner sub-chunk loop
	// is exercised. Output must still match the serial path byte-for-byte.
	deps := newDeps(t)
	fields := []field.Field{&field.IDField{}, &field.LastNameField{}}
	const masterSeed = uint64(0x5EED)
	const count = 200_000

	var serial, parallel bytes.Buffer
	if err := runSerial(&serial, count, fields, deps, masterSeed); err != nil {
		t.Fatalf("runSerial: %v", err)
	}
	if err := RunWithJobs(&parallel, count, fields, deps, masterSeed, 4); err != nil {
		t.Fatalf("RunWithJobs: %v", err)
	}
	if !bytes.Equal(serial.Bytes(), parallel.Bytes()) {
		t.Fatalf("sub-chunked parallel diverged from serial (len %d vs %d)",
			parallel.Len(), serial.Len())
	}
}

func TestRun_perWorkerMemoryStaysBounded(t *testing.T) {
	// Synthetic memory check: count * 256 would exceed the safety threshold
	// if the runner pre-allocated per-row capacity. With sub-chunking, peak
	// allocations should stay roughly proportional to workers, not count.
	if testing.Short() {
		t.Skip("skipping memory bound test in short mode")
	}
	deps := newDeps(t)
	fields := []field.Field{&field.IDField{}, &field.LastNameField{}}
	const count = 500_000

	// Run, asserting only that it completes without panicking. A real OOM
	// would have caused a SIGKILL; a more nuanced check needs an external
	// process — left to scripts/bench-compare.sh / manual large-N runs.
	var sink bytes.Buffer
	if err := RunWithJobs(&sink, count, fields, deps, 1, 8); err != nil {
		t.Fatalf("RunWithJobs failed at count=%d: %v", count, err)
	}
	if got := bytes.Count(sink.Bytes(), []byte{'\n'}); got != count {
		t.Errorf("rows = %d, want %d", got, count)
	}
}

func TestRun_workersZeroEqualsAuto(t *testing.T) {
	deps := newDeps(t)
	fields := []field.Field{&field.IDField{}}
	var a, b bytes.Buffer
	if err := Run(&a, 200, fields, deps, 7); err != nil {
		t.Fatal(err)
	}
	if err := RunWithJobs(&b, 200, fields, deps, 7, 0); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Bytes(), b.Bytes()) {
		t.Error("workers=0 should equal auto-detection path")
	}
}

func TestRun_outputHasExactlyOneNewlinePerRow(t *testing.T) {
	deps := newDeps(t)
	var buf bytes.Buffer
	fields := []field.Field{&field.IDField{}, &field.LastNameField{}}
	if err := Run(&buf, 50, fields, deps, 99); err != nil {
		t.Fatal(err)
	}
	if got := bytes.Count(buf.Bytes(), []byte{'\n'}); got != 50 {
		t.Errorf("newline count = %d, want 50", got)
	}
}
