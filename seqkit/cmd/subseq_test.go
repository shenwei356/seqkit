package cmd

import (
	"bytes"
	"fmt"
	"github.com/shenwei356/bio/seq"
	"os"
	"strconv"
	"strings"
	"testing"
)

// TestSubseqQuietIndexUpdate ensures quiet mode does not suppress requested work.
func TestSubseqQuietIndexUpdate(t *testing.T) {
	dir := t.TempDir()
	file := auditInput(t, dir, "input.fa", ">r1\nAAAACCCC\n")
	_, stderr, err := runCLI(t, "", "subseq", "--chr", "r1", "-r", "1:-1", file)
	if err != nil {
		t.Fatalf("create index: %v %s", err, stderr)
	}
	if err := os.WriteFile(file, []byte(">r1\nTTTTGGGGAAAA\n"), 0600); err != nil {
		t.Fatal(err)
	}
	out, stderr, err := runCLI(t, "", "subseq", "--chr", "r1", "-r", "1:-1", "-U", "--quiet", file)
	index, readErr := os.ReadFile(file + ".seqkit.fai")
	if err != nil || stderr != "" || !strings.HasSuffix(out, "\nTTTTGGGGAAAA\n") || readErr != nil || !strings.HasPrefix(string(index), "r1\t12\t") {
		t.Fatalf("index not updated: %v %s read=%v", err, stderr, readErr)
	}
}

// TestSubseqMixedFileFormats preserves each input file's format.
func TestSubseqMixedFileFormats(t *testing.T) {
	assertMixedFileFormats(t, "subseq", "-r", "1:-1")
}

// TestSubseqIndexedMixedFileFormats covers index paths that bypass the FASTX reader.
func TestSubseqIndexedMixedFileFormats(t *testing.T) {
	dir := t.TempDir()
	fasta := auditInput(t, dir, "input.fa", ">f\nAACT\n")
	fastq := auditInput(t, dir, "input.fq", "@q\nAACT\n+\n1234\n")
	bed := auditInput(t, dir, "regions.bed", "f\t0\t4\tf\t0\t-\nq\t0\t4\tq\t0\t-\n")
	gtf := auditInput(t, dir, "regions.gtf", "f\ttest\tCDS\t1\t4\t.\t-\t.\tgene_id \"f\";\nq\ttest\tCDS\t1\t4\t.\t-\t.\tgene_id \"q\";\n")
	for _, mode := range [][]string{{"--bed", bed}, {"--gtf", gtf}} {
		for _, files := range [][]string{{fastq, fasta}, {fasta, fastq}} {
			args := append([]string{"subseq", "--quiet"}, mode...)
			var want strings.Builder
			for _, file := range files {
				out, stderr, err := runCLI(t, "", append(append([]string(nil), args...), file)...)
				if err != nil || out == "" {
					t.Fatalf("single-file extraction failed: %v %s", err, stderr)
				}
				want.WriteString(out)
			}
			out, stderr, err := runCLI(t, "", append(args, files...)...)
			if err != nil || out != want.String() {
				t.Fatalf("indexed mixed-format extraction differs from separate inputs: %v %s", err, stderr)
			}
		}
	}
}

// TestSubseqStreamingIndexLifecycle avoids unused indexes but honors explicit -U.
func TestSubseqStreamingIndexLifecycle(t *testing.T) {
	dir := t.TempDir()
	input := auditInput(t, dir, "input.fa", ">r\nACGT\n")
	out, stderr, err := runCLI(t, "", "subseq", "-r", "1:2", input)
	if err != nil || out != ">r\nAC\n" {
		t.Fatalf("streaming extraction: %v %s", err, stderr)
	}
	if _, err := os.Stat(input + ".seqkit.fai"); !os.IsNotExist(err) {
		t.Fatal("unused index was created")
	}
	out, stderr, err = runCLI(t, "", "subseq", "-r", "1:2", "-U", input)
	if err != nil || out != ">r\nAC\n" {
		t.Fatalf("explicit index update: %v %s", err, stderr)
	}
	if _, err := os.Stat(input + ".seqkit.fai"); err != nil {
		t.Fatal(err)
	}
}

// TestSubseqIndexAuxiliaryAliases protects annotations and later sequence inputs.
func TestSubseqIndexAuxiliaryAliases(t *testing.T) {
	for _, mode := range []string{"--bed", "--gtf", "sequence"} {
		dir := t.TempDir()
		input := auditInput(t, dir, "input.fa", ">r\nACGT\n")
		data := "r\t0\t2\n"
		if mode == "--gtf" {
			data = "r\ttest\tCDS\t1\t2\t.\t+\t.\tgene_id \"r\";\n"
		}
		if mode == "sequence" {
			data = ">s\nTGCA\n"
		}
		index := auditInput(t, dir, "input.fa.seqkit.fai", data)
		output := auditInput(t, dir, "out.fa", "keep\n")
		args := []string{"subseq", "-U", "-o", output, input}
		if mode == "sequence" {
			args = append(args, "-r", "1:2", index)
		} else {
			args = append(args, mode, index)
		}
		_, stderr, err := runCLI(t, "", args...)
		after, _ := os.ReadFile(index)
		out, _ := os.ReadFile(output)
		if err == nil || string(after) != data || string(out) != "keep\n" {
			t.Fatalf("index replaced input: %v %s", err, stderr)
		}
	}
}

// TestSubseqIndexOutputAliases rejects two generated indexes sharing a destination.
func TestSubseqIndexOutputAliases(t *testing.T) {
	dir := t.TempDir()
	first := auditInput(t, dir, "first.fa", ">r1\nACGT\n")
	second := auditInput(t, dir, "second.fa", ">r2\nTGCATGCA\n")
	firstIndex, secondIndex := first+".seqkit.fai", second+".seqkit.fai"
	if err := os.Symlink(secondIndex, firstIndex); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	output := auditInput(t, dir, "output.fa", "keep\n")
	_, stderr, err := runCLI(t, "", "subseq", "-U", "-r", "1:2", first, second, "-o", output)
	after, _ := os.ReadFile(output)
	_, indexErr := os.Stat(secondIndex)
	if err == nil || !strings.Contains(stderr, "cannot be the same") || string(after) != "keep\n" || !os.IsNotExist(indexErr) {
		t.Fatalf("index collision accepted: %v %s index=%v", err, stderr, indexErr)
	}
}

// TestSubseqRepeatedIndexedInput keeps repeated inputs valid when they reuse an index.
func TestSubseqRepeatedIndexedInput(t *testing.T) {
	dir := t.TempDir()
	input := auditInput(t, dir, "input.fa", ">r1\nACGT\n")
	out, stderr, err := runCLI(t, "", "subseq", "-U", "-r", "1:2", input, dir+"/./input.fa")
	if err != nil || strings.Count(out, "AC\n") != 2 {
		t.Fatalf("repeated indexed input rejected: %v %s", err, stderr)
	}
}

// TestSubseqAnnotationFlanks checks clipping, strand symmetry, and quality slices.
func TestSubseqAnnotationFlanks(t *testing.T) {
	maxInt := strconv.Itoa(int(^uint(0) >> 1))
	cases := []struct {
		name       string
		start, end int
		strand     string
		flags      []string
		want, qual string
	}{
		{"left boundary", 1, 4, "+", []string{"-u", "2", "-f"}, "", ""},
		{"reverse left boundary", 1, 4, "-", []string{"-d", "2", "-f"}, "", ""},
		{"right boundary", 5, 8, "+", []string{"-d", "2", "-f"}, "", ""},
		{"reverse right boundary", 5, 8, "-", []string{"-u", "2", "-f"}, "", ""},
		{"left flank", 3, 5, "+", []string{"-u", "8", "-f"}, "AC", "12"},
		{"reverse left flank", 3, 5, "-", []string{"-d", "8", "-f"}, "GT", "21"},
		{"right flank", 3, 5, "+", []string{"-d", "8", "-f"}, "AGT", "678"},
		{"reverse right flank", 3, 5, "-", []string{"-u", "8", "-f"}, "ACT", "876"},
		{"overflow", 3, 4, "+", []string{"-d", maxInt}, "GTCAGT", "345678"},
		{"reverse overflow", 3, 4, "-", []string{"-u", maxInt}, "ACTGAC", "876543"},
		{"far right flank", int(^uint(0) >> 1), int(^uint(0) >> 1), "+", []string{"-d", "2", "-f"}, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			bed := auditInput(t, dir, "regions.bed", fmt.Sprintf("s\t%d\t%d\t.\t0\t%s\n", tc.start-1, tc.end, tc.strand))
			gtf := auditInput(t, dir, "regions.gtf", fmt.Sprintf("s\ttest\tCDS\t%d\t%d\t.\t%s\t.\tgene_id \"s\";\n", tc.start, tc.end, tc.strand))
			for _, ann := range [][]string{{"--bed", bed}, {"--gtf", gtf}} {
				for _, fastq := range []bool{false, true} {
					input := ">s\nACGTCAGT\n"
					if fastq {
						input = "@s\nACGTCAGT\n+\n12345678\n"
					}
					args := append([]string{"subseq", "--quiet"}, ann...)
					args = append(args, tc.flags...)
					out, stderr, err := runCLI(t, input, args...)
					lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
					wantLines := 2
					if fastq {
						wantLines = 4
					}
					if err != nil || len(lines) != wantLines || lines[1] != tc.want || (fastq && lines[3] != tc.qual) {
						t.Fatalf("%s FASTQ=%v: wrong sequence/quality; err=%v %s", ann[0], fastq, err, stderr)
					}
				}
			}
		})
	}
}

// TestSubseqRegionIndexedParity preserves the -R contract in both input paths.
func TestSubseqRegionIndexedParity(t *testing.T) {
	dir := t.TempDir()
	input := ">s\nACGTCAGT\n"
	file := auditInput(t, dir, "input.fa", input)
	for _, region := range []string{"2:4", "-3:-1", "20:30"} {
		for _, annotate := range []bool{false, true} {
			args := []string{"subseq", "--quiet", "--chr", "s", "-r", region}
			if annotate {
				args = append(args, "-R")
			}
			want, stderr, err := runCLI(t, input, args...)
			if err != nil {
				t.Fatalf("stream: %v %s", err, stderr)
			}
			got, stderr, err := runCLI(t, "", append(args, file)...)
			if err != nil || got != want {
				t.Fatalf("indexed region %s annotate=%v differs from stream: %v %s", region, annotate, err, stderr)
			}
		}
	}
}

// TestSubseqEmptyInputs keeps later files and honors explicit empty-index updates.
func TestSubseqEmptyInputs(t *testing.T) {
	dir := t.TempDir()
	empty := auditInput(t, dir, "empty.fa", "")
	input := auditInput(t, dir, "input.fa", ">s\nACGT\n")
	for _, extra := range [][]string{nil, {"--chr", "s"}, {"-U"}} {
		args := append([]string{"subseq", "--quiet", "-r", "1:2"}, extra...)
		out, stderr, err := runCLI(t, "", append(args, empty, input)...)
		if err != nil || out != ">s\nAC\n" {
			t.Fatalf("empty input prevented later extraction: %v %s", err, stderr)
		}
	}
	if _, err := os.Stat(empty + ".seqkit.fai"); err != nil {
		t.Fatalf("explicit empty index missing: %v", err)
	}
}

// TestSubseqInvalidGTFCoordinates rejects invalid absolute positions before output.
func TestSubseqInvalidGTFCoordinates(t *testing.T) {
	for _, positions := range [][2]int{{0, 3}, {-2, 3}, {-9, -2}} {
		dir := t.TempDir()
		gtf := auditInput(t, dir, "regions.gtf", fmt.Sprintf("s\ttest\tCDS\t%d\t%d\t.\t+\t.\tgene_id \"s\";\n", positions[0], positions[1]))
		output := auditInput(t, dir, "output.fa", "keep\n")
		_, stderr, err := runCLI(t, ">s\nACGT\n", "subseq", "--gtf", gtf, "-o", output)
		after, _ := os.ReadFile(output)
		if err == nil || !strings.Contains(stderr, "coordinates must be positive") || string(after) != "keep\n" {
			t.Fatalf("invalid GTF accepted or output changed: %v %s", err, stderr)
		}
	}
}

// TestSubseqAnnotationFASTQWidth applies explicit wrapping to sequence and quality.
func TestSubseqAnnotationFASTQWidth(t *testing.T) {
	dir := t.TempDir()
	bed := auditInput(t, dir, "regions.bed", "s\t0\t4\n")
	out, stderr, err := runCLI(t, "@s\nACGT\n+\n1234\n", "subseq", "--bed", bed, "-w", "2")
	if err != nil || !strings.HasSuffix(out, "\nAC\nGT\n+\n12\n34\n") {
		t.Fatalf("annotation FASTQ width ignored: %v %s", err, stderr)
	}
}

// TestSubseqMinimumCoordinate clips the native minimum integer without overflow.
func TestSubseqMinimumCoordinate(t *testing.T) {
	dir := t.TempDir()
	input := ">s\nACGT\n"
	file := auditInput(t, dir, "input.fa", input)
	region := strconv.Itoa(-int(^uint(0)>>1)-1) + ":-1"
	for _, extra := range [][]string{nil, {"--chr", "s", file}} {
		out, stderr, err := runCLI(t, input, append([]string{"subseq", "-r", region}, extra...)...)
		if err != nil || out != input {
			t.Fatalf("minimum start not clipped: %v %s", err, stderr)
		}
	}
}

// TestSubseqIndexedHeaderParsing preserves names and parses custom IDs only once.
func TestSubseqIndexedHeaderParsing(t *testing.T) {
	dir := t.TempDir()
	input := ">prefix|s " + strings.Repeat("x", 3) + "\nACGT\n"
	file := auditInput(t, dir, "input.fa", input)
	for _, custom := range []bool{false, true} {
		for _, annotate := range []bool{false, true} {
			args := []string{"subseq", "--quiet", "-r", "1:2", "--chr", "prefix|s"}
			if custom {
				args[len(args)-1] = "s"
				args = append(args, "--id-regexp", `^prefix\|([^ ]+)`)
			}
			if annotate {
				args = append(args, "-R")
			}
			want, stderr, err := runCLI(t, input, args...)
			if err != nil {
				t.Fatalf("stream: %v %s", err, stderr)
			}
			got, stderr, err := runCLI(t, "", append(args, file)...)
			if err != nil || got != want {
				t.Fatalf("indexed header differs: custom=%v annotate=%v err=%v", custom, annotate, err)
			}
		}
	}
}

// TestAnnotationSubseqStrandSymmetry keeps oriented bases and qualities after revcom.
func TestAnnotationSubseqStrandSymmetry(t *testing.T) {
	sequence := &seq.Seq{Alphabet: seq.DNA, Seq: []byte("ACGTCAGT"), Qual: []byte("12345678")}
	reverse := sequence.RevCom()
	for start := 1; start <= 8; start++ {
		for end := start; end <= 8; end++ {
			for _, flanks := range [][2]int{{0, 0}, {2, 0}, {0, 3}, {2, 3}, {20, 0}, {0, 20}} {
				for _, only := range []bool{false, true} {
					if only && ((flanks[0] == 0) == (flanks[1] == 0)) {
						continue
					}
					forward := annotationSubseq(sequence, start, end, false, only, flanks[0], flanks[1])
					reversed := annotationSubseq(reverse, 9-end, 9-start, true, only, flanks[0], flanks[1])
					if !bytes.Equal(forward.Seq, reversed.Seq) || !bytes.Equal(forward.Qual, reversed.Qual) {
						t.Fatalf("strand asymmetry: %d:%d flanks=%v only=%v", start, end, flanks, only)
					}
				}
			}
		}
	}
}

// TestSubseqRepeatedIndexedID retains the first record when full headers differ.
func TestSubseqRepeatedIndexedID(t *testing.T) {
	dir := t.TempDir()
	input := ">s " + strings.Repeat("x", 3) + "\nACGT\n>s " + strings.Repeat("y", 3) + "\nTGCA\n"
	file := auditInput(t, dir, "input.fa", input)
	for i := 0; i < 3; i++ {
		out, stderr, err := runCLI(t, "", "subseq", "--chr", "s", "-r", "1:2", file)
		if err != nil || !strings.HasSuffix(out, "\nAC\n") || strings.Count(out, ">s") != 1 {
			t.Fatalf("repeated ID changed selection: %v %s", err, stderr)
		}
	}
}

// TestSubseqDamagedAuxiliaryInput rejects incomplete auxiliary data.
func TestSubseqDamagedAuxiliaryInput(t *testing.T) {
	assertDamagedTextInput(t, "r\t0\t2\n", "subseq", "--bed")
}

// TestSubseqDamagedGTF rejects incomplete annotations from the GTF dependency.
func TestSubseqDamagedGTF(t *testing.T) {
	assertDamagedTextInput(t, "r\ttest\tCDS\t1\t2\t.\t+\t.\tgene_id \"r\";\n", "subseq", "--gtf")
}

// TestSubseqAnnotationChromosomeCase keeps --chr filtering case-insensitive
// without changing the exact ID match between annotations and sequences.
func TestSubseqAnnotationChromosomeCase(t *testing.T) {
	dir := t.TempDir()
	input := ">ChrA\nACGT\n>other\nTTTT\n"
	fasta := auditInput(t, dir, "input.fa", input)
	bed := auditInput(t, dir, "regions.bed", "ChrA\t0\t2\nother\t0\t2\n")
	gtf := auditInput(t, dir, "regions.gtf", "ChrA\ttest\tCDS\t1\t2\t.\t+\t.\tgene_id \"a\";\nother\ttest\tCDS\t1\t2\t.\t+\t.\tgene_id \"b\";\n")
	for _, annotation := range [][]string{{"--bed", bed}, {"--gtf", gtf}} {
		for _, indexed := range []bool{false, true} {
			for _, chr := range []string{"ChrA", "chra", "CHRA"} {
				args := append([]string{"subseq", "--quiet", "--chr", chr}, annotation...)
				stdin := input
				if indexed {
					args = append(args, fasta)
					stdin = ""
				}
				out, stderr, err := runCLI(t, stdin, args...)
				if err != nil || strings.Count(out, ">") != 1 || !strings.HasPrefix(out, ">ChrA_1-2:") || !strings.HasSuffix(out, "\nAC\n") {
					t.Fatalf("%s indexed=%v chr=%s: incorrect selection: %v %s", annotation[0], indexed, chr, err, stderr)
				}
			}
		}
	}
}
