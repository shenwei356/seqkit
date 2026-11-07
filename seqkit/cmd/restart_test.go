package cmd

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

// TestRestartMultipleRecords covers repeated, shorter, and reverse-complement records.
func TestRestartMultipleRecords(t *testing.T) {
	const first = ">r1\nAAAGGCTT\n>r2\nAAAGGCTT\n"
	const second = ">r3\nGGCA\n>r4\nAAGCCTTT\n"
	const want = ">r1\nGGCTTAAA\n>r2\nGGCTTAAA\n>r3\nGGCA\n>r4\nGGCTTAAA\n"
	dir := t.TempDir()
	a := auditInput(t, dir, "a.fa", first)
	b := auditInput(t, dir, "b.fa", second)
	for _, files := range [][]string{nil, {a, b}} {
		out, stderr, err := runCLI(t, first+second, append([]string{"restart", "-s", "GGC"}, files...)...)
		if err != nil || stderr != "" || out != want {
			t.Fatalf("records differ: %q %v %s", out, err, stderr)
		}
	}
	fq := auditInput(t, dir, "input.fq", "@q1\nAAAGGCTT\n+\nABCDEFGH\n")
	out, stderr, err := runCLI(t, "", "restart", "-s", "GGC", fq, b)
	if err != nil || stderr != "" || out != "@q1\nGGCTTAAA\n+\nDEFGHABC\n>r3\nGGCA\n>r4\nGGCTTAAA\n" {
		t.Fatalf("mixed formats/qualities changed: %q %v %s", out, err, stderr)
	}
}

// TestRestartOverlappingStarts rejects ambiguous starts without affecting later records.
func TestRestartOverlappingStarts(t *testing.T) {
	out, stderr, err := runCLI(t, ">r1\nAAAAC\n>r2\nAAACG\n", "restart", "-s", "AAA")
	if err != nil || out != ">r2\nAAACG\n" || !strings.Contains(stderr, "found at 2 positions") {
		t.Fatalf("overlapping starts: %q %v %s", out, err, stderr)
	}
}

// TestRestartThreads checks the effective runtime limit in an isolated CLI process.
func TestRestartThreads(t *testing.T) {
	if os.Getenv("SEQKIT_RESTART_THREADS_TEST") == "1" {
		runtime.GOMAXPROCS(4)
		RootCmd.SetArgs([]string{"restart", "-i", "1", "-j", "2"})
		if err := RootCmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if got := runtime.GOMAXPROCS(0); got != 2 {
			t.Fatalf("thread limit ignored: got %d, want 2", got)
		}
		return
	}
	command := exec.Command(os.Args[0], "-test.run=^TestRestartThreads$")
	command.Env = append(os.Environ(), "SEQKIT_RESTART_THREADS_TEST=1")
	command.Stdin = strings.NewReader(">s\nACGT\n")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("thread limit: %v %s", err, output)
	}
}

// TestRestartEmptySearchRecords skips empty records in both exact and mismatch searches.
func TestRestartEmptySearchRecords(t *testing.T) {
	for _, mismatch := range []string{"0", "1"} {
		out, stderr, err := runCLI(t, ">empty\n\n>r1\nACGTT\n", "restart", "-s", "ACG", "-m", mismatch)
		if err != nil || out != ">r1\nACGTT\n" || !strings.Contains(stderr, "not found") {
			t.Fatalf("empty record with -m %s: %q %v %s", mismatch, out, err, stderr)
		}
	}
}

// TestRestartLeadingEmptyRecords keeps reverse circular searches after empty reads.
func TestRestartLeadingEmptyRecords(t *testing.T) {
	for _, mismatch := range []string{"0", "1"} {
		args := []string{"restart", "-s", "GAAA", "-m", mismatch}
		assertLeadingEmptyRecords(t, ">r\nTTTC\n", args...)
		assertLeadingEmptyRecords(t, "@r\nUUUC\n+\nabcd\n", args...)
	}
}

// TestRestartProteinStrands never reverses a protein to invent a matching start.
func TestRestartProteinStrands(t *testing.T) {
	for _, mismatch := range []string{"0", "1"} {
		out, stderr, err := runCLI(t, ">p1\nMELK\n>p2\nKLEM\n", "restart", "-t", "protein", "-s", "KLEM", "-m", mismatch)
		if err != nil || out != ">p2\nKLEM\n" || !strings.Contains(stderr, "not found") {
			t.Fatalf("protein with -m %s: %q %v %s", mismatch, out, err, stderr)
		}
	}
}

// TestRestartCircularStarts finds boundary-spanning starts and rotates qualities with bases.
func TestRestartCircularStarts(t *testing.T) {
	for _, tt := range []struct {
		name, input, pattern, mismatch, want string
	}{
		{"forward", "@r\nACGATGCA\n+\nABCDEFGH\n", "CAAC", "0", "@r\nCAACGATG\n+\nGHABCDEF\n"},
		{"reverse", "@r\nTGCATCGT\n+\nHGFEDCBA\n", "CAAC", "0", "@r\nCAACGATG\n+\nGHABCDEF\n"},
		{"mismatch", ">r\nACGATGCA\n", "GAAC", "1", ">r\nCAACGATG\n"},
		{"reverse mismatch", "@r\nTGCATCGT\n+\nHGFEDCBA\n", "GAAC", "1", "@r\nCAACGATG\n+\nGHABCDEF\n"},
		{"full rotation", ">r\nACGA\n", "GAAC", "0", ">r\nGAAC\n"},
		{"single base", ">r\nACGA\n", "G", "0", ">r\nGAAC\n"},
		{"ignore case", ">r\nacgatgca\n", "CAAC", "0", ">r\ncaacgatg\n"},
		{"ambiguous across boundary", ">r\nAACA\n", "AA", "0", ""},
		{"too long", ">r\nAC\n", "ACA", "0", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out, stderr, err := runCLI(t, tt.input, "restart", "-I", "-s", tt.pattern, "-m", tt.mismatch)
			if err != nil || out != tt.want {
				t.Fatalf("got %q, want %q: %v %s", out, tt.want, err, stderr)
			}
		})
	}
}

// TestRestartExactMatchCounts preserves overlapping counts on both strands and
// still processes the next record after skipping a highly repetitive record.
func TestRestartExactMatchCounts(t *testing.T) {
	for _, base := range []string{"A", "T"} {
		input := ">repeat\n" + strings.Repeat(base, 10000) + "\n>unique\nCCAAAG\n"
		out, stderr, err := runCLI(t, input, "restart", "-s", "AAA")
		if err != nil || out != ">unique\nAAAGCC\n" || !strings.Contains(stderr, "found at 10000 positions") {
			t.Fatalf("repeated %s: %q %v %s", base, out, err, stderr)
		}
	}
}

// TestRestartExactMatches covers the first position and overlapping multiplicity.
func TestRestartExactMatches(t *testing.T) {
	for _, tt := range []struct {
		sequence, pattern string
		first, count      int
	}{
		{"", "A", -1, 0},
		{"AC", "AAA", -1, 0},
		{"AC", "C", 1, 1},
		{"AAAAA", "AAA", 0, 3},
		{"CACA", "CA", 0, 2},
	} {
		first, count := restartExactMatches([]byte(tt.sequence), []byte(tt.pattern))
		if first != tt.first || count != tt.count {
			t.Errorf("%q / %q = (%d, %d), want (%d, %d)", tt.sequence, tt.pattern, first, count, tt.first, tt.count)
		}
	}
}
