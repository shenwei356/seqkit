package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRmdupMixedFileFormats preserves each input file's format.
func TestRmdupMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "rmdup")
}

// TestRmdupProteinStrands keeps distinct reversed proteins and nucleotide complements.
func TestRmdupProteinStrands(t *testing.T) {
	dir := t.TempDir()
	p := auditInput(t, dir, "p.fa", ">p1\nMELK\n>p2\nKLEM\n>p3\nMELK\n")
	d := auditInput(t, dir, "d.fa", ">d1\nAACT\n>d2\nAGTT\n")
	for _, files := range [][]string{{p, d}, {d, p}} {
		out, stderr, err := runCLI(t, "", append([]string{"rmdup", "-s"}, files...)...)
		want := ">p1\nMELK\n>p2\nKLEM\n>d1\nAACT\n"
		if files[0] == d {
			want = ">d1\nAACT\n>p1\nMELK\n>p2\nKLEM\n"
		}
		if err != nil || out != want {
			t.Fatalf("per-file strand policy changed: %v %s", err, stderr)
		}
	}
	out, stderr, err := runCLI(t, "", "rmdup", "-s", "-t", "protein", p)
	if err != nil || out != ">p1\nMELK\n>p2\nKLEM\n" {
		t.Fatalf("explicit protein type failed: %v %s", err, stderr)
	}
}

// TestRmdupAuxiliaryOutputsProtectInputList checks both auxiliary writers before truncation.
func TestRmdupAuxiliaryOutputsProtectInputList(t *testing.T) {
	for _, flag := range []string{"-d", "-D"} {
		for _, alias := range []string{"same", "symlink", "hardlink"} {
			t.Run(flag+"/"+alias, func(t *testing.T) {
				dir := t.TempDir()
				input := auditInput(t, dir, "input.fa", ">r1\nAC\n>r1\nAC\n")
				listData := input + "\n"
				list := auditInput(t, dir, "inputs.txt", listData)
				output := list
				if alias != "same" {
					output = filepath.Join(dir, "alias")
					link := os.Link
					if alias == "symlink" {
						link = os.Symlink
					}
					if err := link(list, output); err != nil {
						t.Skipf("links unavailable: %v", err)
					}
				}
				main := auditInput(t, dir, "main.fa", "keep\n")
				_, stderr, err := runCLI(t, "", "rmdup", "-X", list, flag, output, "-o", main, "--skip-file-check")
				got, _ := os.ReadFile(list)
				kept, _ := os.ReadFile(main)
				if err == nil || !strings.Contains(stderr, "cannot be the same") || string(got) != listData || string(kept) != "keep\n" {
					t.Fatalf("input list or main output modified: %v %s", err, stderr)
				}
			})
		}
	}
}
