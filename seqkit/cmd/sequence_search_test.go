package cmd

import (
	"strings"
	"testing"
)

// TestConcatDescriptionsAndEmptyInputs checks that skipped files add no padding.
func TestConcatDescriptionsAndEmptyInputs(t *testing.T) {
	dir := t.TempDir()
	a := auditInput(t, dir, "a.fa", ">x  even\nAAA\n>y odd\nCCC\n")
	b := auditInput(t, dir, "b.fa", ">x desc\nGG\n")
	empty := auditInput(t, dir, "empty.fa", "")
	for _, inputs := range [][]string{{empty, a, b}, {a, empty, b}, {a, b, empty}} {
		out, stderr, err := runCLI(t, "", append([]string{"concat", "-f", "-F", "N"}, inputs...)...)
		if err != nil || !strings.Contains(out, ">x even|desc\nAAAGG\n") || !strings.Contains(out, ">y odd\nCCCNN\n") {
			t.Fatalf("padding/header mismatch: %q %v %s", out, err, stderr)
		}
	}
	out, stderr, err := runCLI(t, "", "concat", "--id-regexp", `^(\S+)\s`, a, a)
	if err != nil || !strings.Contains(out, ">x even|even\nAAAAAA\n") {
		t.Fatalf("custom concat description: %q %v %s", out, err, stderr)
	}
	out, stderr, err = runCLI(t, "", "concat", empty, a)
	if err != nil || out != "" {
		t.Fatalf("empty inner join changed: %q %v %s", out, err, stderr)
	}
}

// TestFaidxQueryContracts retains valid results while rejecting missing IDs.
func TestFaidxQueryContracts(t *testing.T) {
	dir := t.TempDir()
	file := auditInput(t, dir, "a.fa", ">s\nACGTACGTAC\n>chr1:100-200\nGG\n>chr1\nAA\n")
	for query, want := range map[string]string{"s:-12--1": ">s:-12--1\nACGTACGTAC\n", "s:2--2": ">s:2--2\nCGTACGTA\n", "s:3-": ">s:3-\nGTACGTAC\n"} {
		out, stderr, err := runCLI(t, "", "faidx", file, query)
		if err != nil || out != want {
			t.Fatalf("query %s: %q %v %s", query, out, err, stderr)
		}
	}
	out, stderr, err := runCLI(t, "", "faidx", file, "s:1-2", "missing", "s:3-4")
	if err == nil || out != ">s:1-2\nAC\n>s:3-4\nGT\n" || !strings.Contains(stderr, "sequence not found") {
		t.Fatalf("missing query lost results: %q %v %s", out, err, stderr)
	}
	out, stderr, err = runCLI(t, "", "faidx", file, "chr1:100-200")
	if err != nil || strings.Contains(out, "GG") {
		t.Fatalf("region precedence changed: %q %v %s", out, err, stderr)
	}
}

// TestLocatePatternSources preserves duplicate records and combines both sources.
func TestLocatePatternSources(t *testing.T) {
	dir := t.TempDir()
	patterns := auditInput(t, dir, "patterns.fa", ">dup\nAC\n>dup\nGT\n>dup\nAC\n>empty\n\n")
	for _, mode := range [][]string{nil, {"-F"}, {"-m", "1"}, {"-r"}} {
		args := append([]string{"locate", "-P", "-p", "AA", "-f", patterns}, mode...)
		out, stderr, err := runCLI(t, ">s\nAAACGT\n", args...)
		if err != nil || strings.Count(out, "\tdup\tAC\t") < 2 || !strings.Contains(out, "\tdup\tGT\t") || !strings.Contains(out, "\tAA\tAA\t") || !strings.Contains(stderr, "empty pattern") {
			t.Fatalf("pattern source loss: %q %v %s", out, err, stderr)
		}
	}
	for _, mode := range [][]string{nil, {"-F"}, {"-m", "1"}} {
		out, stderr, err := runCLI(t, ">s\nACGTTGCA\n", append([]string{"locate", "-c", "-p", "TTGCAACG"}, mode...)...)
		if err != nil || !strings.Contains(out, "\t-\t2\t9\t") {
			t.Fatalf("circular coordinate mismatch: %q %v %s", out, err, stderr)
		}
	}
	_, stderr, err := runCLI(t, ">s\nACGT\n", "locate", "-p", "")
	if err == nil || !strings.Contains(stderr, "no valid patterns") {
		t.Fatalf("empty patterns accepted: %v %s", err, stderr)
	}
}

// TestAmpliconBoundaries covers missing primers, N matching and empty regions.
func TestAmpliconBoundaries(t *testing.T) {
	out, stderr, err := runCLI(t, ">s\nCCC\n", "amplicon", "-F", "ccN", "-P")
	if err != nil || out != ">s\nCCC\n" {
		t.Fatalf("N primer mismatch: %q %v %s", out, err, stderr)
	}
	for _, flags := range [][]string{nil, {"-F", "ccN", "-m", "1"}, {"-F", "CCC", "-r", "10:11"}} {
		out, stderr, err = runCLI(t, ">s\nCCC\n", append([]string{"amplicon", "-P"}, flags...)...)
		if len(flags) == 4 && flags[2] == "-r" {
			if err != nil || out != "" {
				t.Fatalf("outside region emitted: %q %v %s", out, err, stderr)
			}
			continue
		}
		if err == nil || strings.Contains(stderr, "panic:") {
			t.Fatalf("invalid primers accepted: %v %s", err, stderr)
		}
	}
	dir := t.TempDir()
	bad := auditInput(t, dir, "primers.tsv", "# comment\n\np F R\n")
	_, stderr, err = runCLI(t, ">s\nCCC\n", "amplicon", "-p", bad)
	if err == nil || !strings.Contains(stderr, bad+":3:") {
		t.Fatalf("primer row not diagnosed: %v %s", err, stderr)
	}
	if _, _, ok := SubLocationInner(10, 2, 7, 20, 30, false); ok {
		t.Fatal("fully outside region accepted")
	}
	if _, _, ok := SubLocationInner(10, 2, 7, -30, -20, false); ok {
		t.Fatal("fully left region accepted")
	}
}

// TestAmpliconReversedMismatchSites agrees with exact-search ordering.
func TestAmpliconReversedMismatchSites(t *testing.T) {
	out, stderr, err := runCLI(t, ">rev\naaagggccc\n", "amplicon", "-F", "ccc", "-R", "ttt", "-m", "1", "-P", "--bed")
	if err != nil || out != "" {
		t.Fatalf("reversed primer sites emitted: %q %v %s", out, err, stderr)
	}
}

// TestLocateParallelFiles captures each file's strand policy before launching work.
func TestLocateParallelFiles(t *testing.T) {
	dir := t.TempDir()
	dna := auditInput(t, dir, "dna.fa", ">dna\nACGTTGCA\n")
	protein := auditInput(t, dir, "protein.fa", ">protein\nZZZZZZZZ\n")
	out, stderr, err := runCLI(t, "", "locate", "-F", "-c", "-p", "TTGCAACG", dna, protein)
	if err != nil || !strings.Contains(out, "dna\tTTGCAACG\tTTGCAACG\t-\t2\t9\t") {
		t.Fatalf("strand policy crossed files: %q %v %s", out, err, stderr)
	}
}
