package cmd

import (
	"strings"
	"testing"
)

// assertMixedFileFormats checks format and wrapping against independent invocations.
func assertMixedFileFormats(t *testing.T, args ...string) {
	t.Helper()
	dir := t.TempDir()
	suffix := ""
	if args[0] == "head-genome" {
		suffix = " " + strings.Repeat("prefix ", 3)
	}
	bases := strings.Repeat("ACGT", 20)
	fasta := ">f1" + suffix + "\n" + bases + "\n>f0" + suffix + "\n\n"
	fastq := "@q0" + suffix + "\n\n+\n\n@q1" + suffix + "\n" + bases + "\n+\n" + strings.Repeat("I", len(bases)) + "\n"
	fa := auditInput(t, dir, "a.fa", fasta)
	fq := auditInput(t, dir, "b.fq", fastq)
	for _, files := range [][]string{{fa, fq}, {fq, fa}} {
		for _, width := range [][]string{nil, {"-w", "7"}} {
			command := append(append([]string(nil), args...), width...)
			var want strings.Builder
			for _, file := range files {
				out, stderr, err := runCLI(t, "", append(append([]string(nil), command...), file)...)
				if err != nil {
					t.Fatalf("single-file invocation: %v %s", err, stderr)
				}
				want.WriteString(out)
			}
			out, stderr, err := runCLI(t, "", append(command, files...)...)
			if err != nil || out != want.String() {
				t.Fatalf("mixed-file format/wrapping mismatch: args=%v width=%v: %v %s", args, width, err, stderr)
			}
		}
	}
}
