package cmd

import "testing"

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
