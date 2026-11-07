package cmd

import (
	"strconv"
	"strings"
	"testing"
)

// TestRangeMixedFileFormats preserves each input file's format.
func TestRangeMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "range", "-r", "1:-1")
}

// TestRangeStopsAtEnd does not parse records beyond the requested positive range.
func TestRangeStopsAtEnd(t *testing.T) {
	out, stderr, err := runCLI(t, "@r1\nAC\n+\n!!\n@r2\nAC\n+\n!\n", "range", "-r", "1:1")
	if err != nil || out != "@r1\nAC\n+\n!!\n" {
		t.Fatalf("read past selected range: %q %v %s", out, err, stderr)
	}
}

// TestRangeMinimumInteger clips negative ranges without overflowing their magnitude.
func TestRangeMinimumInteger(t *testing.T) {
	minimum := strconv.Itoa(-int(^uint(0)>>1) - 1)
	for _, tc := range []struct{ region, want string }{
		{minimum + ":-1", ">r1\nAC\n>r2\nGT\n"},
		{minimum + ":" + minimum, ""},
	} {
		out, stderr, err := runCLI(t, ">r1\nAC\n>r2\nGT\n", "range", "-r", tc.region)
		if err != nil || out != tc.want {
			t.Fatalf("range %s: %q %v %s", tc.region, out, err, stderr)
		}
	}
}

// TestRangeSkippedFastqRestoresWrapping keeps the next FASTA file's default width.
func TestRangeSkippedFastqRestoresWrapping(t *testing.T) {
	dir := t.TempDir()
	fq := auditInput(t, dir, "short.fq", "@q1\nAC\n+\n!!\n")
	fa := auditInput(t, dir, "long.fa", ">r1\n"+strings.Repeat("A", 70)+"\n>r2\nAC\n")
	out, stderr, err := runCLI(t, "", "range", "-r", "-2:-2", fq, fa)
	want := ">r1\n" + strings.Repeat("A", 60) + "\n" + strings.Repeat("A", 10) + "\n"
	if err != nil || out != want {
		t.Fatalf("FASTA wrapping after skipped FASTQ: %q %v %s", out, err, stderr)
	}
}
