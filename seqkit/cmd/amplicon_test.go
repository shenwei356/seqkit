package cmd

import (
	"reflect"
	"strings"
	"sync"
	"testing"
)

// TestAmpliconMixedFileFormats preserves each input file's format.
func TestAmpliconMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "amplicon", "-P", "-F", "ACGT", "-u")
}

// TestAmpliconParallelMixedFileFormats preserves each input file's format.
func TestAmpliconParallelMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "amplicon", "-P", "-F", "ACGT", "-u", "-m", "1", "-j", "4")
}

// TestAmpliconProteinStrands preserves unmatched input without reversing proteins.
func TestAmpliconProteinStrands(t *testing.T) {
	const input = "@s\nEACG\n+\n1234\n"
	for _, alphabet := range []string{"protein", "unlimit"} {
		for _, mismatch := range []string{"0", "1"} {
			for _, mode := range []string{"", "--bed", "-u"} {
				args := []string{"amplicon", "-t", alphabet, "-F", "GCA", "-m", mismatch, "-j", "2"}
				want := ""
				if mode != "" {
					args = append(args, mode)
					if mode == "-u" {
						want = input
					}
				}
				out, stderr, err := runCLI(t, input, args...)
				if err != nil || out != want {
					t.Fatalf("%v: got %q, want %q: %v %s", args, out, want, err, stderr)
				}
			}
		}
	}
	// Alphabet restrictions from one file must not suppress later DNA reverse hits.
	dir := t.TempDir()
	protein := auditInput(t, dir, "protein.fq", input)
	dna := auditInput(t, dir, "dna.fq", "@dna\nTTGC\n+\n1234\n")
	for _, mismatch := range []string{"0", "1"} {
		out, stderr, err := runCLI(t, "", "amplicon", "-F", "GCA", "-m", mismatch, "-j", "2", protein, dna)
		if err != nil || out != "@dna\nGCA\n+\n432\n" {
			t.Fatalf("mixed alphabets with -m %s: %q %v %s", mismatch, out, err, stderr)
		}
	}
}

// TestAmpliconEmptyRecords never searches empty input and preserves -u formatting.
func TestAmpliconEmptyRecords(t *testing.T) {
	for _, mismatch := range []string{"0", "1"} {
		for _, input := range []string{">empty\n\n", "@empty\n\n+\n\n"} {
			for _, bed := range []bool{false, true} {
				args := []string{"amplicon", "-F", "AC", "-m", mismatch, "-j", "2"}
				if bed {
					args = append(args, "--bed")
				}
				out, stderr, err := runCLI(t, input, args...)
				if err != nil || out != "" {
					t.Fatalf("empty record matched: %q %v %s", out, err, stderr)
				}
				if !bed {
					out, stderr, err = runCLI(t, input, append(args, "-u")...)
					if err != nil || out != input {
						t.Fatalf("unmatched format changed: %v %s", err, stderr)
					}
				}
			}
		}
		out, stderr, err := runCLI(t, ">hit\nACGT\n>empty\n\n>miss\nTTTT\n", "amplicon", "-P", "-F", "AC", "-m", mismatch, "-j", "1")
		if err != nil || strings.Contains(out, ">empty") {
			t.Fatalf("previous hit leaked: %v %s", err, stderr)
		}
	}
}

// TestAmpliconOverlappingReversePrimer keeps the longest hit and its qualities on both strands.
func TestAmpliconOverlappingReversePrimer(t *testing.T) {
	for _, input := range []string{">r\nAAAAA\n", ">r\nTTTTT\n", "@r\nAAAAA\n+\nABCDE\n", "@r\nTTTTT\n+\nEDCBA\n"} {
		want, stderr, err := runCLI(t, input, "amplicon", "-F", "A", "-R", "TTT")
		if err != nil {
			t.Fatalf("exact primer: %v %s", err, stderr)
		}
		got, stderr, err := runCLI(t, input, "amplicon", "-F", "A", "-R", "TTN")
		if err != nil || got != want || !strings.Contains(got, "AAAAA\n") {
			t.Fatalf("overlapping primer differs: %v %s", err, stderr)
		}
	}
}

// TestAmpliconFinderLifecycle clears old results and returns consistent cached coordinates.
func TestAmpliconFinderLifecycle(t *testing.T) {
	finder, err := NewAmpliconFinder([]byte("A"), []byte("AC"), nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	pool := &sync.Pool{New: func() any {
		f, e := NewAmpliconFinder([]byte("A"), []byte("AC"), nil, 0)
		if e != nil {
			panic(e)
		}
		// sync.Pool may discard entries, so fresh idle finders retain no input either.
		f.Seq = nil
		return f
	}}
	for _, mismatch := range []int{1, 0} {
		if err := finder.Reset([]byte("TACGT"), mismatch); err != nil {
			t.Fatal(err)
		}
		first, mis, err := finder.Locate()
		again, againMis, againErr := finder.Locate()
		if err != nil || againErr != nil || !reflect.DeepEqual(first, again) || !reflect.DeepEqual(mis, againMis) {
			t.Fatal("cached coordinates changed")
		}
		if err := finder.Reset(nil, mismatch); err == nil {
			t.Fatal("empty reset accepted")
		}
		if loc, _, err := finder.Locate(); err != nil || loc != nil {
			t.Fatal("failed reset retained a hit")
		}
		if err := finder.Reset([]byte("TTTT"), mismatch); err != nil {
			t.Fatal(err)
		}
		if loc, _, err := finder.Locate(); err != nil || loc != nil {
			t.Fatal("miss retained a hit")
		}
		recycleAmpliconFinder(pool, finder)
		finder = pool.Get().(*AmpliconFinder)
		if finder.Seq != nil || finder.FMindex != nil {
			t.Fatal("idle finder retains input or index")
		}
	}
}
