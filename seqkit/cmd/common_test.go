package cmd

import (
	"strings"
	"testing"
)

// TestCommonFirstFileCandidates retains duplicates without counting them as extra files.
func TestCommonFirstFileCandidates(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "first.fa", ">a\nAACT\n>a\nAGTT\n>b\nCCCC\n")
	second := auditInput(t, dir, "second.fa", ">a\nAACT\n>a\nAACT\n>b\nCCCC\n>extra\nGGAA\n")
	third := auditInput(t, dir, "third.fa", ">a\nAGTT\n>a\nAGTT\n>extra\nGGAA\n")
	for _, flags := range [][]string{nil, {"-s"}, {"-s", "-P"}} {
		want := ">a\nAACT\n>a\nAGTT\n"
		if len(flags) == 2 {
			want = ""
		}
		args := append(append([]string{"common"}, flags...), first, second, third)
		out, stderr, err := runCLI(t, "", args...)
		if err != nil || out != want {
			t.Fatalf("flags=%v changed intersection: %v %s", flags, err, stderr)
		}
	}
}

// TestCommonNoCandidatesStillChecksInput preserves later read errors.
func TestCommonNoCandidatesStillChecksInput(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "first.fa", ">a\nACGT\n")
	second := auditInput(t, dir, "second.fa", ">b\nTGCA\n")
	bad := auditInput(t, dir, "bad.fq", "@b\nACGT\n+\nI\n")
	_, stderr, err := runCLI(t, "", "common", first, second, bad)
	if err == nil || !strings.Contains(stderr, "[ERRO]") {
		t.Fatalf("later read error was lost: %v %s", err, stderr)
	}
}

// TestCommonEmbeddedSnapshots retains independent hits in either query order.
func TestCommonEmbeddedSnapshots(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "first.fq", "@q1\nTTGGGGTT\n+\nABCDEFGH\n@q2\nACGT\n+\nIIII\n")
	want := "@q1:1-4\nTTGG\n+\nABCD\n@q1:3-6\nGGGG\n+\nCDEF\n"
	for _, input := range []string{">v1\nCCCC\n>v2\nTTGG\n", ">v2\nTTGG\n>v1\nCCCC\n"} {
		second := auditInput(t, dir, "second.fa", input)
		for attempt := 0; attempt < 5; attempt++ {
			out, stderr, err := runCLI(t, "", "common", "-s", "-e", first, second)
			if err != nil || out != want || !strings.Contains(stderr, "1 original records, 2 distinct common sequences, 2 output fragments") {
				t.Fatalf("unstable intersection: %q %v %s", out, err, stderr)
			}
		}
	}
	empty := auditInput(t, dir, "empty.fa", ">empty\n\n")
	out, stderr, err := runCLI(t, "", "common", "-s", "-e", first, empty)
	if err != nil || out != "" {
		t.Fatalf("empty input did not participate: %q %v %s", out, err, stderr)
	}
}

// TestCommonFirstStdin matches file output including order, names and multiplicity.
func TestCommonFirstStdin(t *testing.T) {
	dir := t.TempDir()
	input := "@a first\nACGT\n+\nABCD\n@b first\nTTGG\n+\nEFGH\n@a first\nACGT\n+\nIJKL\n"
	first := auditInput(t, dir, "first.fq", input)
	second := auditInput(t, dir, "second.fa", ">a first\nACGT\n>b first\nTTGG\n")
	for _, flags := range [][]string{nil, {"-n"}, {"-s"}} {
		args := append([]string{"common"}, flags...)
		want, se, werr := runCLI(t, "", append(append([]string{}, args...), first, second)...)
		out, stderr, err := runCLI(t, input, append(args, "-", second)...)
		if werr != nil || err != nil || out != want || out != input {
			t.Fatalf("stdin differs: %q %q %v %v %s %s", out, want, err, werr, stderr, se)
		}
	}
	input = "@a first\nTTGGGGTT\n+\nABCDEFGH\n"
	first = auditInput(t, dir, "first.fq", input)
	second = auditInput(t, dir, "second.fa", ">a\nCCCC\n")
	want, _, _ := runCLI(t, "", "common", "-s", "-e", first, second)
	out, stderr, err := runCLI(t, input, "common", "-s", "-e", "-", second)
	if err != nil || out != want || out == "" || !strings.Contains(stderr, "first file: -") {
		t.Fatalf("embedded stdin differs: %q %v %s", out, err, stderr)
	}
}

// TestCommonExactAndEmbeddedHits keeps longer targets even when an exact hit exists.
func TestCommonExactAndEmbeddedHits(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "first.fa", ">a\nCC\n>b\nAACCAA\n")
	second := auditInput(t, dir, "second.fa", ">q\nCC\n")
	out, stderr, err := runCLI(t, "", "common", "-s", "-e", "-P", first, second)
	if err != nil || out != ">a\nCC\n>b:3-4\nCC\n" {
		t.Fatalf("exact hit hid embedded hit: %q %v %s", out, err, stderr)
	}
}

// TestCommonProteinStrands rejects reversed proteins, including embedded matches.
func TestCommonProteinStrands(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "a.fa", ">p1\nMELK\n")
	for _, text := range []string{"KLEM", "XXKLEMXX", "MELK"} {
		second := auditInput(t, dir, "b.fa", ">p2\n"+text+"\n")
		for _, extra := range [][]string{nil, {"-e"}, {"-t", "protein"}, {"-e", "-t", "protein"}} {
			args := append([]string{"common", "-s"}, extra...)
			out, stderr, err := runCLI(t, "", append(args, first, second)...)
			want := ""
			if text == "MELK" {
				want = ">p1\nMELK\n"
			}
			if err != nil || out != want {
				t.Fatalf("protein intersection changed: %v %s", err, stderr)
			}
		}
	}
}
