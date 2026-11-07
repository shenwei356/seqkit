package cmd

import (
	"os"
	"strings"
	"testing"
)

// TestReplaceMixedFileFormats preserves each input file's format.
func TestReplaceMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "replace", "-p", "^", "-r", "x")
}

// TestReplaceEditFilters shares the sequence-editing filter contract.
func TestReplaceEditFilters(t *testing.T) {
	assertEditFilterContracts(t, []string{"replace", "-s", "-p", ".", "-r", "X"})
}

// TestReplaceInvalidCapturePreservesOutput validates captures before opening output.
func TestReplaceInvalidCapturePreservesOutput(t *testing.T) {
	for _, pattern := range []string{"(a)", "(?:a)", `\(a\)`} {
		t.Run(pattern, func(t *testing.T) {
			dir := t.TempDir()
			kv := auditInput(t, dir, "kv.tsv", "a\tx\n")
			output := auditInput(t, dir, "out.fa", "keep\n")
			_, stderr, err := runCLI(t, ">a\nAC\n", "replace", "-p", pattern, "-r", "{kv}", "-k", kv, "-I", "2", "-o", output)
			if err == nil || !strings.Contains(stderr, "capt") {
				t.Fatalf("err=%v stderr=%s", err, stderr)
			}
			data, err := os.ReadFile(output)
			if err != nil || string(data) != "keep\n" {
				t.Fatalf("output changed: %q, %v", data, err)
			}
		})
	}
}
