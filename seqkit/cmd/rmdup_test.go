package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRmdupMixedAlphabets matches either supported orientation across file types.
func TestRmdupMixedAlphabets(t *testing.T) {
	for _, second := range []struct{ name, prefix, sequence string }{
		{"protein-identical", ">p0\nMELK\n", "AAG"},
		{"protein-complement", ">p0\nMELK\n", "CTT"},
		{"rna-identical", ">p0\nUUUU\n", "AAG"},
		{"rna-complement", ">p0\nUUUU\n", "CUU"},
	} {
		for _, reverseOrder := range []bool{false, true} {
			for _, ignoreCase := range []bool{false, true} {
				for _, positive := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/reverse=%v/ignore=%v/positive=%v", second.name, reverseOrder, ignoreCase, positive), func(t *testing.T) {
						dir := t.TempDir()
						firstText := ">d\nAAG\n"
						text := second.sequence
						if ignoreCase {
							text = strings.ToLower(text)
						}
						secondText := second.prefix + ">p1\n" + text + "\n"
						first := auditInput(t, dir, "first.fa", firstText)
						last := auditInput(t, dir, "last.fa", secondText)
						files := []string{first, last}
						want := firstText + second.prefix
						wantDup, wantIDs := ">p1\n"+text+"\n", "2\td, p1\n"
						if reverseOrder {
							files = []string{last, first}
							want, wantDup, wantIDs = secondText, firstText, "2\tp1, d\n"
						}
						args := []string{"rmdup", "-s"}
						if ignoreCase {
							args = append(args, "-i")
						}
						if positive {
							args = append(args, "-P")
							if second.sequence != "AAG" {
								want += wantDup
								wantDup, wantIDs = "", ""
							}
						}
						duplicates, ids := filepath.Join(dir, "duplicates.fa"), filepath.Join(dir, "ids.tsv")
						args = append(args, "-d", duplicates, "-D", ids)
						out, stderr, err := runCLI(t, "", append(args, files...)...)
						dup, dupErr := os.ReadFile(duplicates)
						idText, idErr := os.ReadFile(ids)
						if err != nil || dupErr != nil || idErr != nil || out != want || string(dup) != wantDup || string(idText) != wantIDs {
							t.Fatalf("output=%q duplicates=%q ids=%q: %v %s", out, dup, idText, err, stderr)
						}
					})
				}
			}
		}
	}
}

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

// TestRmdupLeadingEmptyRecords recognizes reverse-complement duplicates.
func TestRmdupLeadingEmptyRecords(t *testing.T) {
	assertLeadingEmptyRecords(t, ">a\nTTTC\n>b\nGAAA\n", "rmdup", "-s")
	assertLeadingEmptyRecords(t, "@a\nUUUC\n+\nabcd\n@b\nGAAA\n+\ndcba\n", "rmdup", "-s")
}
