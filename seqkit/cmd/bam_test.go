package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/biogo/hts/bam"
	"github.com/biogo/hts/bgzf"
	"github.com/biogo/hts/sam"
)

// bamFixture creates a small mapped BAM, optionally omitting the last NM tag.
func bamFixture(t *testing.T, dir, name string, allNM bool) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	ref, err := sam.NewReference("chr1", "", "", 100, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	header, err := sam.NewHeader(nil, []*sam.Reference{ref})
	if err != nil {
		t.Fatal(err)
	}
	w, err := bam.NewWriter(f, header, 1)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		var aux []sam.Aux
		if i == 0 || allNM {
			nm, err := sam.NewAux(sam.NewTag("NM"), 0)
			if err != nil {
				t.Fatal(err)
			}
			aux = []sam.Aux{nm}
		}
		r, err := sam.NewRecord(fmt.Sprintf("r%d", i+1), ref, nil, i*4, -1, 0, 60, []sam.CigarOp{sam.NewCigarOp(sam.CigarMatch, 4)}, []byte("ACGT"), []byte{30, 30, 30, 30}, aux)
		if err != nil {
			t.Fatal(err)
		}
		if err := w.Write(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

// readBAMFixture verifies both record contents and the compressed stream terminator.
func readBAMFixture(t *testing.T, path string) (*sam.Header, []*sam.Record) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if ok, err := bgzf.HasEOF(f); err != nil || !ok {
		t.Fatalf("missing BAM EOF marker: %v", err)
	}
	r, err := bam.NewReader(f, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var records []*sam.Record
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
	return r.Header(), records
}

// TestBamFieldsAndPassthrough checks lazy NM access and complete BAM output.
func TestBamFieldsAndPassthrough(t *testing.T) {
	dir := t.TempDir()
	file := bamFixture(t, dir, "missing-nm.bam", false)
	output := filepath.Join(dir, "out.bam")
	_, stderr, err := runCLI(t, "", "bam", "-f", "Read,Ref,MapQual", "-x", file, "-o", output)
	if err != nil || !strings.Contains(stderr, "r2\tchr1\t60") {
		t.Fatalf("unrelated field required NM: %v %s", err, stderr)
	}
	_, records := readBAMFixture(t, output)
	if len(records) != 2 {
		t.Fatalf("truncated pass-through: %d", len(records))
	}
	for _, flags := range [][]string{{"-f", "Acc", "-Q"}, {"-T", "Dump: {Fields: [Read, Acc]}"}} {
		_, stderr, err = runCLI(t, "", append([]string{"bam", file}, flags...)...)
		if err == nil || !strings.Contains(stderr, "r2") || !strings.Contains(stderr, "NM tag") || strings.Contains(stderr, "panic:") {
			t.Fatalf("missing NM not diagnosed: %v %s", err, stderr)
		}
	}
	_, stderr, err = runCLI(t, "", "bam", file, "-T", "Dump: {Fields: [Read, MapQual]}")
	if err != nil || !strings.Contains(stderr, "r2\t60") {
		t.Fatalf("Dump eagerly required NM: %v %s", err, stderr)
	}
}

// TestBamPreflight rejects unsupported modes without truncating output targets.
func TestBamPreflight(t *testing.T) {
	dir := t.TempDir()
	file := bamFixture(t, dir, "good.bam", true)
	for _, flags := range [][]string{{"-f", "Read,Foo"}, {"-x", "-s"}, {"--top-bam", filepath.Join(dir, "top"), "--top-size", "-1", "-f", "MapQual"}, {"--top-bam", filepath.Join(dir, "top")}, {"--top-bam", filepath.Join(dir, "top"), "-f", "Read"}, {"--top-bam", filepath.Join(dir, "top"), "-f", "MapQual", "-s"}, {"-T", "Dump: {Fields: [Read, Foo]}"}} {
		output := auditInput(t, dir, "keep", "keep")
		out, stderr, err := runCLI(t, "", append([]string{"bam", file, "-o", output}, flags...)...)
		data, _ := os.ReadFile(output)
		if err == nil || out != "" || string(data) != "keep" || !strings.Contains(stderr, "[ERRO]") || strings.Contains(stderr, "panic:") {
			t.Fatalf("late mode validation: %q %v %s", data, err, stderr)
		}
	}
	top := filepath.Join(dir, "zero.bam")
	_, stderr, err := runCLI(t, "", "bam", file, "-f", "MapQual", "--top-bam", top, "--top-size", "0", "-Q")
	if err != nil {
		t.Fatalf("zero top size: %v %s", err, stderr)
	}
	_, records := readBAMFixture(t, top)
	if len(records) != 0 {
		t.Fatal("zero top size wrote records")
	}
	counts := filepath.Join(dir, "counts.tsv")
	_, stderr, err = runCLI(t, "", "bam", file, "-c", counts, "-f", "Count", "-o", filepath.Join(dir, "unused.out"), "-Q")
	data, readErr := os.ReadFile(counts)
	if err != nil || readErr != nil || !strings.Contains(string(data), "chr1\t2\t") {
		t.Fatalf("count mode close/field error: %q %v %s", data, err, stderr)
	}

}

// TestBamInputErrors retains completed statistics and stops at the failed file.
func TestBamInputErrors(t *testing.T) {
	dir := t.TempDir()
	first := bamFixture(t, dir, "first.bam", true)
	last := bamFixture(t, dir, "last.bam", true)
	bad := auditInput(t, dir, "bad.bam", ">not-bam\nACGT\n")
	for _, pretty := range [][]string{nil, {"-k"}} {
		out, stderr, err := runCLI(t, "", append([]string{"bam", "-s", first, bad, last}, pretty...)...)
		if err == nil || out != "" || !strings.Contains(stderr, "first.bam") || !strings.Contains(stderr, "bad.bam") || strings.Contains(stderr, "last.bam") || !strings.Contains(stderr, "TotalRec") {
			t.Fatalf("statistics lost/continued: %q %v %s", out, err, stderr)
		}
	}
	_, stderr, err := runCLI(t, "", "bam", bad, "-T", "Dump: {Fields: [Read]}")
	if err == nil || !strings.Contains(stderr, "bad.bam") || strings.Contains(stderr, "panic:") {
		t.Fatalf("toolbox input error: %v %s", err, stderr)
	}
}
