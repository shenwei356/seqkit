package cmd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/shenwei356/bio/seq"
	"github.com/shenwei356/bio/seqio/fastx"
)

// TestCommonMixedAlphabets compares supported orientations without losing exact matches.
func TestCommonMixedAlphabets(t *testing.T) {
	for _, second := range []struct{ name, prefix, sequence string }{
		{"protein-identical", ">p0\nMELK\n", "AAG"},
		{"protein-complement", ">p0\nMELK\n", "CTT"},
		{"rna-identical", ">p0\nUUUU\n", "AAG"},
		{"rna-complement", ">p0\nUUUU\n", "CUU"},
	} {
		for _, reverseOrder := range []bool{false, true} {
			for _, embedded := range []bool{false, true} {
				for _, positive := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s/reverse=%v/embedded=%v/positive=%v", second.name, reverseOrder, embedded, positive), func(t *testing.T) {
						dir := t.TempDir()
						first := auditInput(t, dir, "first.fa", ">d\nAAG\n")
						last := auditInput(t, dir, "last.fa", second.prefix+">p1\n"+strings.ToLower(second.sequence)+"\n")
						files := []string{first, last}
						want := ">d\nAAG\n"
						if reverseOrder {
							files = []string{last, first}
							want = ">p1\n" + strings.ToLower(second.sequence) + "\n"
						}
						args := []string{"common", "-s", "-i"}
						if embedded {
							args = append(args, "-e")
						}
						if positive {
							args = append(args, "-P")
							if second.sequence != "AAG" {
								want = ""
							}
						}
						out, stderr, err := runCLI(t, "", append(args, files...)...)
						if err != nil || out != want {
							t.Fatalf("output=%q want=%q: %v %s", out, want, err, stderr)
						}
					})
				}
			}
		}
	}
}

// TestCommonMixedAlphabetFileCounts counts each candidate once per file, even
// when a different alphabet can match two distinct first-file candidates.
func TestCommonMixedAlphabetFileCounts(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "first.fa", ">p0\nMELK\n>p1\nAAG\n>p2\nCTT\n")
	second := auditInput(t, dir, "second.fa", ">d\nAAG\n>d2\nAAG\n")
	third := auditInput(t, dir, "third.fa", ">r0\nUUUU\n>r1\nAAG\n")
	for _, embedded := range []bool{false, true} {
		args := []string{"common", "-s"}
		if embedded {
			args = append(args, "-e")
		}
		out, stderr, err := runCLI(t, "", append(args, first, second)...)
		if err != nil || out != ">p1\nAAG\n>p2\nCTT\n" {
			t.Fatalf("two candidates: %q %v %s", out, err, stderr)
		}
		out, stderr, err = runCLI(t, "", append(args, first, second, third)...)
		if err != nil || out != ">p1\nAAG\n" {
			t.Fatalf("three-file intersection: %q %v %s", out, err, stderr)
		}
	}
}

// TestCommonEmbeddedMixedAlphabetRegions maps reverse-strand hits back to the
// first record's coordinates and preserves the matching quality slice.
func TestCommonEmbeddedMixedAlphabetRegions(t *testing.T) {
	for _, tc := range []struct{ name, first, second, want string }{
		{"dna-longer", "@d\nCCAAGG\n+\n123456\n", ">p0\nMELK\n>p1\nCTT\n", "@d:3-5\nAAG\n+\n345\n"},
		{"dna-shorter", "@d\nAAG\n+\n123\n", ">p0\nMELK\n>p1\nMCTTK\n", "@d\nAAG\n+\n123\n"},
		{"rna-longer", "@r0\nUUUU\n+\nIIII\n@r\nCCAAGG\n+\n123456\n", ">p0\nMELK\n>p1\nCUU\n", "@r:3-5\nAAG\n+\n345\n"},
		{"protein-longer", "@p0\nMELK\n+\nIIII\n@p1\nMCTTK\n+\n12345\n", ">d\nAAG\n", "@p1:2-4\nCTT\n+\n234\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			first := auditInput(t, dir, "first.fq", tc.first)
			second := auditInput(t, dir, "second.fa", tc.second)
			out, stderr, err := runCLI(t, "", "common", "-s", "-e", first, second)
			if err != nil || out != tc.want {
				t.Fatalf("output=%q want=%q: %v %s", out, tc.want, err, stderr)
			}
			out, stderr, err = runCLI(t, "", "common", "-s", "-e", "-P", first, second)
			if err != nil || out != "" {
				t.Fatalf("positive-only output=%q: %v %s", out, err, stderr)
			}
		})
	}
}

// BenchmarkCommonRecordHash includes the reverse-complement allocation lifecycle.
func BenchmarkCommonRecordHash(b *testing.B) {
	for _, size := range []struct {
		name   string
		length int
	}{{"short", 150}, {"long", 10000}} {
		b.Run(size.name, func(b *testing.B) {
			record := &fastx.Record{Seq: &seq.Seq{
				Alphabet: seq.DNAredundant,
				Seq:      []byte(strings.Repeat("ACGT", (size.length+3)/4)[:size.length]),
				Qual:     []byte(strings.Repeat("I", size.length)),
			}}
			b.ReportAllocs()
			b.SetBytes(int64(size.length))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				commonRecordHash(record, true, false, true, false)
			}
		})
	}
}

// TestCommonEmbeddedExactStrands keeps equal-length matches in both orientations.
func TestCommonEmbeddedExactStrands(t *testing.T) {
	for _, test := range []struct{ alphabet, forward, reverse string }{
		{"dna", "TCGT", "ACGA"}, {"rna", "UCGU", "ACGA"},
	} {
		for _, firstSeq := range []string{test.forward, test.reverse} {
			for _, query := range []string{test.forward, test.reverse} {
				for _, positive := range []bool{false, true} {
					for _, ignoreCase := range []bool{false, true} {
						dir := t.TempDir()
						first := auditInput(t, dir, "a.fa", ">a\n"+firstSeq+"\n")
						text := query
						args := []string{"common", "-s", "-e", "-t", test.alphabet}
						if positive {
							args = append(args, "-P")
						}
						if ignoreCase {
							text = strings.ToLower(text)
							args = append(args, "-i")
						}
						second := auditInput(t, dir, "b.fa", ">b\n"+text+"\n")
						want := ">a\n" + firstSeq + "\n"
						if positive && firstSeq != query {
							want = ""
						}
						out, stderr, err := runCLI(t, "", append(args, first, second)...)
						if err != nil || out != want {
							t.Fatalf("%s %s/%s positive=%v ignoreCase=%v: output=%q: %v %s", test.alphabet, firstSeq, query, positive, ignoreCase, out, err, stderr)
						}
					}
				}
			}
		}
	}
}

// TestCommonFirstFileCandidates retains duplicates without counting them as extra files.
func TestCommonFirstFileCandidates(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "first.fa", ">a\nAACT\n>a\nAGTT\n>b\nCCCC\n")
	second := auditInput(t, dir, "second.fa", ">a\nAACT\n>a\nAACT\n>b\nCCCC\n>extra\nGGAA\n")
	third := auditInput(t, dir, "third.fa", ">a\nAGTT\n>a\nAGTT\n>extra\nGGAA\n")
	for _, flags := range [][]string{nil, {"-s"}, {"-s", "-P"}} {
		want := ">a\nAACT\n>a\nAGTT\n"
		if len(flags) == 2 {
			want = ""
		}
		args := append(append([]string{"common"}, flags...), first, second, third)
		out, stderr, err := runCLI(t, "", args...)
		if err != nil || out != want {
			t.Fatalf("flags=%v changed intersection: %v %s", flags, err, stderr)
		}
	}
}

// TestCommonNoCandidatesStillChecksInput preserves later read errors.
func TestCommonNoCandidatesStillChecksInput(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "first.fa", ">a\nACGT\n")
	second := auditInput(t, dir, "second.fa", ">b\nTGCA\n")
	bad := auditInput(t, dir, "bad.fq", "@b\nACGT\n+\nI\n")
	_, stderr, err := runCLI(t, "", "common", first, second, bad)
	if err == nil || !strings.Contains(stderr, "[ERRO]") {
		t.Fatalf("later read error was lost: %v %s", err, stderr)
	}
}

// TestCommonEmbeddedSnapshots retains independent hits in either query order.
func TestCommonEmbeddedSnapshots(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "first.fq", "@q1\nTTGGGGTT\n+\nABCDEFGH\n@q2\nACGT\n+\nIIII\n")
	want := "@q1:1-4\nTTGG\n+\nABCD\n@q1:3-6\nGGGG\n+\nCDEF\n"
	for _, input := range []string{">v1\nCCCC\n>v2\nTTGG\n", ">v2\nTTGG\n>v1\nCCCC\n"} {
		second := auditInput(t, dir, "second.fa", input)
		for attempt := 0; attempt < 5; attempt++ {
			out, stderr, err := runCLI(t, "", "common", "-s", "-e", first, second)
			if err != nil || out != want || !strings.Contains(stderr, "1 original records, 2 distinct common sequences, 2 output fragments") {
				t.Fatalf("unstable intersection: %q %v %s", out, err, stderr)
			}
		}
	}
	empty := auditInput(t, dir, "empty.fa", ">empty\n\n")
	out, stderr, err := runCLI(t, "", "common", "-s", "-e", first, empty)
	if err != nil || out != "" {
		t.Fatalf("empty input did not participate: %q %v %s", out, err, stderr)
	}
}

// TestCommonFirstStdin matches file output including order, names and multiplicity.
func TestCommonFirstStdin(t *testing.T) {
	dir := t.TempDir()
	input := "@a first\nACGT\n+\nABCD\n@b first\nTTGG\n+\nEFGH\n@a first\nACGT\n+\nIJKL\n"
	first := auditInput(t, dir, "first.fq", input)
	second := auditInput(t, dir, "second.fa", ">a first\nACGT\n>b first\nTTGG\n")
	for _, flags := range [][]string{nil, {"-n"}, {"-s"}} {
		args := append([]string{"common"}, flags...)
		want, se, werr := runCLI(t, "", append(append([]string{}, args...), first, second)...)
		out, stderr, err := runCLI(t, input, append(args, "-", second)...)
		if werr != nil || err != nil || out != want || out != input {
			t.Fatalf("stdin differs: %q %q %v %v %s %s", out, want, err, werr, stderr, se)
		}
	}
	input = "@a first\nTTGGGGTT\n+\nABCDEFGH\n"
	first = auditInput(t, dir, "first.fq", input)
	second = auditInput(t, dir, "second.fa", ">a\nCCCC\n")
	want, _, _ := runCLI(t, "", "common", "-s", "-e", first, second)
	out, stderr, err := runCLI(t, input, "common", "-s", "-e", "-", second)
	if err != nil || out != want || out == "" || !strings.Contains(stderr, "first file: -") {
		t.Fatalf("embedded stdin differs: %q %v %s", out, err, stderr)
	}
}

// TestCommonExactAndEmbeddedHits keeps longer targets even when an exact hit exists.
func TestCommonExactAndEmbeddedHits(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "first.fa", ">a\nCC\n>b\nAACCAA\n")
	second := auditInput(t, dir, "second.fa", ">q\nCC\n")
	out, stderr, err := runCLI(t, "", "common", "-s", "-e", "-P", first, second)
	if err != nil || out != ">a\nCC\n>b:3-4\nCC\n" {
		t.Fatalf("exact hit hid embedded hit: %q %v %s", out, err, stderr)
	}
}

// TestCommonProteinStrands rejects reversed proteins, including embedded matches.
func TestCommonProteinStrands(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "a.fa", ">p1\nMELK\n")
	for _, text := range []string{"KLEM", "XXKLEMXX", "MELK"} {
		second := auditInput(t, dir, "b.fa", ">p2\n"+text+"\n")
		for _, extra := range [][]string{nil, {"-e"}, {"-t", "protein"}, {"-e", "-t", "protein"}} {
			args := append([]string{"common", "-s"}, extra...)
			out, stderr, err := runCLI(t, "", append(args, first, second)...)
			want := ""
			if text == "MELK" {
				want = ">p1\nMELK\n"
			}
			if err != nil || out != want {
				t.Fatalf("protein intersection changed: %v %s", err, stderr)
			}
		}
	}
}

// TestCommonLeadingEmptyRecords covers both passes and embedded intersections.
func TestCommonLeadingEmptyRecords(t *testing.T) {
	for _, mode := range [][]string{{"-s"}, {"-s", "-e"}} {
		for _, leadingFirst := range []bool{false, true} {
			dir := t.TempDir()
			first, second := ">r\nTTTC\n", ">s\nGAAA\n"
			if leadingFirst {
				first = ">empty\n" + first
			} else {
				second = ">empty\n" + second
			}
			a := auditInput(t, dir, "a.fa", first)
			b := auditInput(t, dir, "b.fa", second)
			args := append([]string{"common", a, b}, mode...)
			out, stderr, err := runCLI(t, "", args...)
			if err != nil || out != ">r\nTTTC\n" {
				t.Fatalf("leading empty changed intersection: %v %s", err, stderr)
			}
		}
	}
}
