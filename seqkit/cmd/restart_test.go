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
