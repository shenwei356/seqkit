package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestSanaFASTARecovery checks CRLF, empty IDs and complete physical-line logs.
func TestSanaFASTARecovery(t *testing.T) {
	input := "junk\n;comment\n\n>a\nACGT\n>\nGG\n>empty\n>b\nTT\n>bad\nGG\nG1G\nTT"
	want := ">a\nACGT\n>\nGG\n>b\nTT\n"
	for _, text := range []string{input, strings.ReplaceAll(input, "\n", "\r\n")} {
		out, stderr, err := runCLI(t, text, "sana", "-i", "fasta")
		if err != nil || out != want || !strings.Contains(stderr, "Discarded lines: 7") {
			t.Fatalf("FASTA recovery: %q %v %s", out, err, stderr)
		}
		for _, line := range []string{"\t1:\t junk", "\t2:\t ;comment", "\t8:\t >empty", "\t11:\t >bad", "\t12:\t GG", "\t13:\t G1G", "\t14:\t TT"} {
			if !strings.Contains(stderr, line) {
				t.Fatalf("missing physical line %q: %s", line, stderr)
			}
		}
	}
}

// TestSanaFileFailuresAreNotRecoveredRecords protects output during preflight.
func TestSanaFileFailuresAreNotRecoveredRecords(t *testing.T) {
	dir := t.TempDir()
	output := auditInput(t, dir, "out", "keep")
	_, stderr, err := runCLI(t, "", "sana", "-i", "fasta", dir, "-o", output)
	data, _ := os.ReadFile(output)
	if err == nil || string(data) != "keep" || !strings.Contains(stderr, "directory") {
		t.Fatalf("directory accepted: %q %v %s", data, err, stderr)
	}
	for _, flags := range [][]string{{"-i", ""}, {"-i", "", "-I", "fasta"}} {
		_, stderr, err = runCLI(t, "", append([]string{"sana", "-o", output}, flags...)...)
		data, _ = os.ReadFile(output)
		if err == nil || string(data) != "keep" || !strings.Contains(stderr, "Invalid format specified") {
			t.Fatalf("empty format accepted: %v %s", err, stderr)
		}
	}
	good := auditInput(t, dir, "a.fa", ">a\nAC\n")
	bad := auditInput(t, dir, "bad.gz", "\x1f\x8b\x08")
	last := auditInput(t, dir, "z.fa", ">z\nGG\n")
	// A directory is rejected even when it follows a good input.
	_, _, err = runCLI(t, "", "sana", "-i", "fasta", good, dir, last)
	if err == nil {
		t.Fatal("file-level failure accepted")
	}
	out, stderr, err := runCLI(t, "", "sana", "-i", "fasta", good, bad, last)
	if err == nil || out != ">a\nAC\n" || !strings.Contains(stderr, bad) {
		t.Fatalf("I/O error did not stop after valid prefix: %q %v %s", out, err, stderr)
	}
}
