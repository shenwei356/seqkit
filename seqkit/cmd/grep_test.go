package cmd

import (
	"fmt"
	"strings"
	"testing"
)

// TestGrepPerFileStrands keeps protein detection local across all search paths.
func TestGrepPerFileStrands(t *testing.T) {
	dir := t.TempDir()
	dna := auditInput(t, dir, "dna.fa", ">d1\nTTTC\n")
	protein := auditInput(t, dir, "protein.fa", ">p1\nEFILPQ\n")
	for _, mode := range [][]string{
		{"-p", "GAAA"}, {"-p", "GAAA", "-r"}, {"-p", "GAAA", "-d"},
		{"-p", "GAAA", "-m", "1"}, {"-p", "GAAA,CCCC", "-m", "1"},
	} {
		for _, files := range [][]string{{protein, dna}, {dna, protein}} {
			for _, positive := range []bool{false, true} {
				args := append([]string{"grep", "-s", "-j", "4"}, mode...)
				want := ">d1\nTTTC\n"
				if positive {
					args = append(args, "-P")
					want = ""
				}
				out, stderr, err := runCLI(t, "", append(args, files...)...)
				if err != nil || out != want {
					t.Fatalf("mode=%v positive=%v: got %q want %q: %v %s", mode, positive, out, want, err, stderr)
				}
			}
		}
	}
}

// TestGrepParallelFileFormats exercises FM-index workers and ordered mixed-format output.
func TestGrepParallelFileFormats(t *testing.T) {
	dir := t.TempDir()
	var fasta, fastq strings.Builder
	sequence := strings.Repeat("ACGT", 25)
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&fasta, ">f%d\n%s\n", i, sequence)
		fmt.Fprintf(&fastq, "@q%d\n%s\n+\n%s\n", i, sequence, strings.Repeat("I", len(sequence)))
	}
	fa := auditInput(t, dir, "input.fa", fasta.String())
	fq := auditInput(t, dir, "input.fq", fastq.String())
	for _, pattern := range []string{"ACGT", strings.Repeat("ACGT", 20)} {
		for _, files := range [][]string{{fa, fq}, {fq, fa}} {
			args := []string{"grep", "-s", "-m", "1", "-p", pattern, "-j", "4"}
			var want string
			for _, file := range files {
				out, stderr, err := runCLI(t, "", append(append([]string(nil), args...), file)...)
				if err != nil {
					t.Fatalf("single file: %v %s", err, stderr)
				}
				want += out
			}
			out, stderr, err := runCLI(t, "", append(args, files...)...)
			if err != nil || out != want {
				t.Fatalf("parallel output differs from separate files: %v %s", err, stderr)
			}
		}
	}
}
