package cmd

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestSortEmptyOutput keeps empty two-pass results consistent with one-pass output.
func TestSortEmptyOutput(t *testing.T) {
	assertEmptyOrderingOutput(t, "sort")
}

// assertEmptyOrderingOutput checks output creation, replacement and compression
// with both newly built and reused empty indexes.
func assertEmptyOrderingOutput(t *testing.T, command string) {
	t.Helper()
	for _, mode := range []string{"one-pass", "new-index", "reused-index"} {
		for _, suffix := range []string{".fa", ".fa.gz"} {
			for _, existing := range []bool{false, true} {
				dir := t.TempDir()
				input := auditInput(t, dir, "empty.fa", "")
				output := filepath.Join(dir, "output"+suffix)
				if existing {
					auditInput(t, dir, filepath.Base(output), "stale output\n")
				}
				args := []string{command, input, "-o", output}
				if mode != "one-pass" {
					args = append(args, "-2")
				}
				if mode == "reused-index" {
					auditInput(t, dir, "empty.fa.seqkit.fai", "")
				}
				_, stderr, err := runCLI(t, "", args...)
				if err != nil {
					t.Fatalf("%s %s: %v %s", command, mode, err, stderr)
				}
				data, err := os.ReadFile(output)
				if err != nil {
					t.Errorf("%s %s existing=%v: output not created: %v", command, mode, existing, err)
					continue
				}
				if suffix == ".fa.gz" {
					gz, err := gzip.NewReader(bytes.NewReader(data))
					if err != nil {
						t.Errorf("%s %s: invalid gzip output: %v", command, mode, err)
						continue
					}
					data, err = io.ReadAll(gz)
					gz.Close()
					if err != nil {
						t.Fatal(err)
					}
				}
				if len(data) != 0 {
					t.Errorf("%s %s: stale output retained", command, mode)
				}
			}
		}
	}
}
