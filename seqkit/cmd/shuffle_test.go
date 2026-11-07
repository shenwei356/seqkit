package cmd

import (
	"strings"
	"testing"
)

// TestShuffleEmptyOutput creates valid empty output in both processing modes.
func TestShuffleEmptyOutput(t *testing.T) {
	assertEmptyOrderingOutput(t, "shuffle")
}

// TestShuffleMixedFileFormats retains formats after records change order.
func TestShuffleMixedFileFormats(t *testing.T) {
	dir := t.TempDir()
	fasta := ">f\nACGT\n"
	fastq := "@empty\n\n+\n\n@q\nTGCA\n+\nIIII\n"
	fa := auditInput(t, dir, "input.fa", fasta)
	fq := auditInput(t, dir, "input.fq", fastq)
	for _, files := range [][]string{{fa, fq}, {fq, fa}} {
		out, stderr, err := runCLI(t, "", append([]string{"shuffle", "-s", "23"}, files...)...)
		if err != nil {
			t.Fatalf("shuffle: %v %s", err, stderr)
		}
		for _, record := range []string{fasta, "@empty\n\n+\n\n", "@q\nTGCA\n+\nIIII\n"} {
			if strings.Count(out, record) != 1 {
				t.Fatal("shuffled record changed format or multiplicity")
			}
		}
		if len(out) != len(fasta)+len(fastq) {
			t.Fatal("unexpected shuffled output")
		}
	}
}

// TestShuffleTwoPassCRLF matches one-pass output, including a final bare CR.
func TestShuffleTwoPassCRLF(t *testing.T) {
	dir := t.TempDir()
	input := ">a description\nACGT\n>b description\nGGCC\n"
	lf := auditInput(t, dir, "lf.fa", input)
	for _, text := range []string{strings.ReplaceAll(input, "\n", "\r\n"), strings.TrimSuffix(strings.ReplaceAll(input, "\n", "\r\n"), "\n")} {
		crlf := auditInput(t, dir, "crlf.fa", text)
		want, se, werr := runCLI(t, "", "shuffle", lf, "-s", "23")
		out, stderr, err := runCLI(t, "", "shuffle", crlf, "-2", "-U", "-s", "23")
		if werr != nil || err != nil || out != want || strings.Contains(out, "\r") {
			t.Fatalf("CRLF mismatch: %q %q %v %v %s %s", out, want, err, werr, stderr, se)
		}
	}
}
