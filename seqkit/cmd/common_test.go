package cmd

import (
	"strings"
	"testing"
)

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
