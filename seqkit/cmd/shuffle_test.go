package cmd

import (
	"strings"
	"testing"
)

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
