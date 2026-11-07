package cmd

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSortDuplicatesAcrossFiles keeps duplicate validation ahead of output creation.
func TestSortDuplicatesAcrossFiles(t *testing.T) {
	for _, flags := range [][]string{nil, {"-n"}, {"-s"}, {"-l"}, {"-b"}, {"--id-regexp", "^(r[0-9]+)"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			dir := t.TempDir()
			first := auditInput(t, dir, "first.fa", ">r1_a\nACGT\n")
			id := "r1_a"
			if len(flags) == 2 {
				id = "r1_b"
			}
			second := auditInput(t, dir, "second.fa", ">"+id+"\nTGCA\n")
			output := auditInput(t, dir, "out.fa", "keep\n")
			args := append([]string{"sort", first, second, "-o", output}, flags...)
			_, stderr, err := runCLI(t, "", args...)
			if err == nil || !strings.Contains(stderr, "duplicated sequences found") {
				t.Fatalf("duplicate not rejected: %v %s", err, stderr)
			}
			data, err := os.ReadFile(output)
			if err != nil || string(data) != "keep\n" {
				t.Fatalf("output changed before validation: %v", err)
			}
		})
	}
}

// TestSortEmptyOutput keeps empty two-pass results consistent with one-pass output.
func TestSortEmptyOutput(t *testing.T) {
	assertEmptyOrderingOutput(t, "sort")
}

// assertEmptyOrderingOutput checks output creation, replacement and compression
// with newly built/reused empty indexes and empty compressed inputs.
func assertEmptyOrderingOutput(t *testing.T, command string) {
	t.Helper()
	for _, mode := range []string{"one-pass", "new-index", "reused-index", "compressed-input"} {
		for _, suffix := range []string{".fa", ".fa.gz"} {
			for _, existing := range []bool{false, true} {
				dir := t.TempDir()
				input := auditInput(t, dir, "empty.fa", "")
				if mode == "compressed-input" {
					input = auditInput(t, dir, "empty.fa.gz", string(compressedInput(t, ".gz", "")))
				}
				output := filepath.Join(dir, "output"+suffix)
				if existing {
					auditInput(t, dir, filepath.Base(output), "stale output\n")
				}
				args := []string{command, input, "-o", output}
				if command == "shuffle" {
					args = append(args, "--tmp-dir", dir)
				}
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
				leftovers, err := filepath.Glob(filepath.Join(dir, "seqkit-*"))
				if err != nil || len(leftovers) != 0 {
					t.Fatalf("temporary files remain: %v %v", leftovers, err)
				}
			}
		}
	}
}
