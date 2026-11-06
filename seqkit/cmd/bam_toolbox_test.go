package cmd

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biogo/hts/sam"
)

// TestBamToolboxPreflight rejects invalid pipelines before opening any output.
func TestBamToolboxPreflight(t *testing.T) {
	dir := t.TempDir()
	in := bamFixture(t, dir, "in.bam", true)
	ref := auditInput(t, dir, "ref.fa", ">chr1\n"+strings.Repeat("ACGT", 25)+"\n")
	for _, config := range []string{
		"Unknown: {}",
		"Sink: true",
		"{}",
		"Dump: {Fields: [Read]}\nDump: {Fields: [Ref]}",
		fmt.Sprintf("AlnContext: {Ref: %q, LeftShift: -1, RightShift: 1, RegexStart: '['}", ref),
		fmt.Sprintf("AlnContext: {Ref: %q, RightShift: 1}", ref),
	} {
		t.Run(config, func(t *testing.T) {
			out := auditInput(t, dir, "out.bam", "keep")
			_, stderr, err := auditCLI(t, dir, "", "bam", in, "-o", out, "-T", config)
			data, readErr := os.ReadFile(out)
			if err == nil || readErr != nil || string(data) != "keep" || strings.Contains(stderr, "panic:") {
				t.Fatalf("invalid pipeline modified output: err=%v data=%q stderr=%s", err, data, stderr)
			}
		})
	}
}

// TestBamToolboxOutputAliases protects every input and separate stage output.
func TestBamToolboxOutputAliases(t *testing.T) {
	for _, target := range []string{"bam", "yaml", "reference", "index", "output", "other stage"} {
		t.Run(target, func(t *testing.T) {
			dir := t.TempDir()
			in := bamFixture(t, dir, "in.bam", true)
			out := auditInput(t, dir, "out.bam", "keep")
			ref := auditInput(t, dir, "ref.fa", ">chr1\n"+strings.Repeat("ACGT", 25)+"\n")
			index := auditInput(t, dir, "ref.fa.seqkit.fai", "chr1\t100\t6\t100\t101\n")
			yaml := filepath.Join(dir, "config.yaml")
			tsv := in
			extra := ""
			switch target {
			case "yaml":
				tsv = yaml
			case "reference":
				tsv = ref
			case "index":
				tsv = index
			case "output":
				tsv = out
			case "other stage":
				tsv = auditInput(t, dir, "stats.tsv", "keep stats")
				extra = fmt.Sprintf("\nAccStats: {Tsv: %q}", tsv)
			}
			config := fmt.Sprintf("Dump: {Fields: [Read], Tsv: %q}%s", tsv, extra)
			if target == "reference" || target == "index" {
				config += fmt.Sprintf("\nAlnContext: {Ref: %q, LeftShift: 0, RightShift: 1, RegexStart: A}", ref)
			}
			auditInput(t, dir, "config.yaml", config)
			before, err := os.ReadFile(tsv)
			if err != nil {
				t.Fatal(err)
			}
			_, stderr, err := runCLI(t, "", "bam", in, "-o", out, "-T", fmt.Sprintf("Yaml: %q", yaml))
			after, readErr := os.ReadFile(tsv)
			if err == nil || readErr != nil || string(before) != string(after) || !strings.Contains(stderr, "cannot be the same") {
				t.Fatalf("toolbox alias not rejected before writing: err=%v preserved=%v stderr=%s", err, string(before) == string(after), stderr)
			}
		})
	}
}

// TestBamToolboxStageOrder makes downstream reports observe only filtered reads.
func TestBamToolboxStageOrder(t *testing.T) {
	dir := t.TempDir()
	in := bamFixture(t, dir, "in.bam", true)
	ref := auditInput(t, dir, "ref.fa", ">chr1\n"+strings.Repeat("ACGT", 25)+"\n")
	for _, filterFirst := range []bool{true, false} {
		tsv := filepath.Join(dir, "dump.tsv")
		filter := fmt.Sprintf("AlnContext: {Ref: %q, LeftShift: 0, RightShift: 1, RegexStart: '^T$'}", ref)
		dump := fmt.Sprintf("Dump: {Fields: [Read], Tsv: %q}", tsv)
		config := filter + "\n" + dump + "\nSink: true"
		want := "Read\n"
		if !filterFirst {
			config = dump + "\n" + filter + "\nSink: true"
			want += "r1\nr2\n"
		}
		_, stderr, err := runCLI(t, "", "bam", in, "-T", config)
		data, readErr := os.ReadFile(tsv)
		if err != nil || readErr != nil || string(data) != want {
			t.Fatalf("stage order filterFirst=%v: got=%q want=%q err=%v stderr=%s", filterFirst, data, want, err, stderr)
		}
	}
}

// TestBamToolboxOutputDirectories validates all destinations before truncation.
func TestBamToolboxOutputDirectories(t *testing.T) {
	dir := t.TempDir()
	in := bamFixture(t, dir, "in.bam", true)
	for _, tsv := range []string{dir, filepath.Join(dir, "missing", "stats.tsv")} {
		out := auditInput(t, dir, "out.bam", "keep")
		_, stderr, err := runCLI(t, "", "bam", in, "-o", out, "-T", fmt.Sprintf("Dump: {Fields: [Read], Tsv: %q}", tsv))
		data, readErr := os.ReadFile(out)
		if err == nil || readErr != nil || string(data) != "keep" || strings.Contains(stderr, "panic:") {
			t.Fatalf("late output validation: err=%v data=%q stderr=%s", err, data, stderr)
		}
	}
}

// TestBamDumpSoftClipFields accepts both lengths and sequences for both ends.
func TestBamDumpSoftClipFields(t *testing.T) {
	dir := t.TempDir()
	in := bamFixture(t, dir, "in.bam", true)
	_, stderr, err := runCLI(t, "", "bam", in, "-T", "Dump: {Fields: [LeftSoftClip, RightSoftClip, LeftSoftClipSeq, RightSoftClipSeq]}\nSink: true")
	if err != nil || !strings.Contains(stderr, "0\t0\t\t\n") {
		t.Fatalf("supported clipping fields rejected: %v %s", err, stderr)
	}
}

// TestSamAlnDetailsIndels separates substitutions from inserted/deleted bases.
func TestSamAlnDetailsIndels(t *testing.T) {
	nm, err := sam.NewAux(sam.NewTag("NM"), 4)
	if err != nil {
		t.Fatal(err)
	}
	for _, reverse := range []bool{false, true} {
		r := &sam.Record{Name: "r", Cigar: sam.Cigar{
			sam.NewCigarOp(sam.CigarMatch, 4),
			sam.NewCigarOp(sam.CigarInsertion, 2),
			sam.NewCigarOp(sam.CigarDeletion, 1),
			sam.NewCigarOp(sam.CigarMatch, 4),
		}, AuxFields: []sam.Aux{nm}}
		if reverse {
			r.Flags |= sam.Reverse
			for i, j := 0, len(r.Cigar)-1; i < j; i, j = i+1, j-1 {
				r.Cigar[i], r.Cigar[j] = r.Cigar[j], r.Cigar[i]
			}
		}
		got := GetSamAlnDetails(r)
		wantAcc := 100.0 * 7 / 11
		if got.Match != 7 || got.Mismatch != 1 || got.Insertion != 2 || got.Deletion != 1 || got.Len != 11 || math.Abs(got.Acc-wantAcc) > 1e-10 || math.Abs(GetSamAcc(r)-wantAcc) > 1e-10 {
			t.Fatalf("wrong indel statistics: %+v", got)
		}
	}
}

// TestSamMissingSequence retains hard-clipped bases when length comes from CIGAR.
func TestSamMissingSequence(t *testing.T) {
	r := &sam.Record{Name: "r", Cigar: sam.Cigar{
		sam.NewCigarOp(sam.CigarHardClipped, 3),
		sam.NewCigarOp(sam.CigarSoftClipped, 2),
		sam.NewCigarOp(sam.CigarMatch, 4),
		sam.NewCigarOp(sam.CigarInsertion, 1),
		sam.NewCigarOp(sam.CigarMatch, 2),
		sam.NewCigarOp(sam.CigarSoftClipped, 1),
		sam.NewCigarOp(sam.CigarHardClipped, 5),
	}}
	if got := GetSamReadLen(r); got != 18 {
		t.Errorf("read length = %d, want 18", got)
	}
	if got := GetSamReadAln(r); got != 7 {
		t.Errorf("aligned length = %d, want 7", got)
	}
	if got := GetSamReadCov(r); math.Abs(got-100.0*7/18) > 1e-10 {
		t.Errorf("coverage = %g, want %g", got, 100.0*7/18)
	}
}

// TestBamContextBounds uses absolute half-open coordinates, never FAI negatives.
func TestBamContextBounds(t *testing.T) {
	dir := t.TempDir()
	ref := auditInput(t, dir, "ref.fa", ">chr1\nACGTTGCA\n")
	idx := NewRefWitdFaidx(ref, false, true)
	defer idx.faidx.Close()
	for _, tt := range []struct {
		start, end int
		want       string
	}{
		{-2, 2, "AC"}, {-4, -1, ""}, {7, 12, "A"}, {9, 12, ""}, {2, 5, "GTT"},
	} {
		got, err := idx.IdxSubSeq("chr1", tt.start, tt.end)
		if err != nil || got != tt.want {
			t.Errorf("[%d,%d): got %q, want %q, err=%v", tt.start, tt.end, got, tt.want, err)
		}
	}
}
