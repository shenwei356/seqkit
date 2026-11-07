package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReplaceDamagedKeyValueFile rejects incomplete mappings before opening output.
func TestReplaceDamagedKeyValueFile(t *testing.T) {
	data := compressedInput(t, ".gz", strings.Repeat("r\treplacement\n", 10000))
	data[len(data)-8] ^= 0xff
	dir := t.TempDir()
	kv := auditInput(t, dir, "bad.tsv.gz", string(data))
	output := auditInput(t, dir, "out.fa", "keep\n")
	_, stderr, err := runCLI(t, ">r\nACGT\n", "replace", "-p", "(.*)", "-r", "{kv}", "-k", kv, "-o", output)
	got, readErr := os.ReadFile(output)
	if err == nil || !strings.Contains(stderr, "read key-value file:") || readErr != nil || string(got) != "keep\n" {
		t.Fatalf("damaged mapping did not fail safely: %v %s", err, stderr)
	}
}

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

// TestReplaceLiteralFileNames preserves dollar signs through both replacement stages.
func TestReplaceLiteralFileNames(t *testing.T) {
	dir := t.TempDir()
	const base = "sample$1${tag}$$.fa"
	input := auditInput(t, dir, base, ">a\nAC\n")
	for _, tt := range []struct{ symbol, want string }{
		{"{fn}", input}, {"{fbn}", filepath.Base(input)}, {"{fbne}", strings.TrimSuffix(base, ".fa")},
	} {
		t.Run(tt.symbol, func(t *testing.T) {
			out, stderr, err := runCLI(t, "", "replace", "-p", "^(a)$", "-r", tt.symbol+"_${1}", input)
			if err != nil || out != ">"+tt.want+"_a\nAC\n" {
				t.Fatalf("out=%q err=%v stderr=%s", out, err, stderr)
			}
		})
	}
}

// TestReplaceDamagedAuxiliaryInput rejects incomplete auxiliary data.
func TestReplaceDamagedAuxiliaryInput(t *testing.T) {
	assertDamagedTextInput(t, "r\n", "replace", "-p", "^", "-r", "x", "--f-pattern-file")
}
