package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLocateDuplicatePatternIDs checks warnings without losing repeated matches.
func TestLocateDuplicatePatternIDs(t *testing.T) {
	for _, mode := range []struct {
		name string
		args []string
	}{
		{"exact", nil},
		{"fmi", []string{"-F"}},
		{"mismatch", []string{"-m", "1"}},
		{"regexp", []string{"-r"}},
		{"degenerate", []string{"-d"}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "patterns.fa")
			if err := os.WriteFile(file, []byte(">dup first\nAC\n>dup second\nGT\n>dup third\nAC\n"), 0600); err != nil {
				t.Fatal(err)
			}
			for _, quiet := range []bool{false, true} {
				args := append([]string{"locate", "-P", "-f", file}, mode.args...)
				if quiet {
					args = append(args, "--quiet")
				}
				out, stderr, err := runCLI(t, ">s\nACGT\n", args...)
				if err != nil || strings.Count(out, "\tdup\tAC\t") != 2 || strings.Count(out, "\tdup\tGT\t") != 1 {
					t.Fatalf("quiet=%v: patterns lost: out=%q, err=%v, stderr=%s", quiet, out, err, stderr)
				}
				warnings := strings.Count(stderr, "duplicate pattern ID")
				if quiet {
					if stderr != "" {
						t.Fatalf("--quiet emitted diagnostics: %s", stderr)
					}
				} else if warnings != 2 || !strings.Contains(stderr, `duplicate pattern ID "dup" in `+file) || !strings.Contains(stderr, "all patterns with this ID will be searched") {
					t.Fatalf("missing duplicate-ID diagnostics: %s", stderr)
				}
			}
		})
	}
}

// TestLocateUniquePatternIDs distinguishes repeated sequences from repeated IDs.
func TestLocateUniquePatternIDs(t *testing.T) {
	file := filepath.Join(t.TempDir(), "patterns.fa")
	if err := os.WriteFile(file, []byte(">one\nAC\n>two\nAC\n"), 0600); err != nil {
		t.Fatal(err)
	}
	out, stderr, err := runCLI(t, ">s\nAC\n", "locate", "-P", "-f", file)
	if err != nil || stderr != "" || strings.Count(out, "\tAC\t+\t1\t2\tAC\n") != 2 {
		t.Fatalf("unique IDs: out=%q, err=%v, stderr=%s", out, err, stderr)
	}
}
