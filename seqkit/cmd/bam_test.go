package cmd

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
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

// bamIndexFixture writes index metadata for the records in a small test BAM.
func bamIndexFixture(t *testing.T, file string) {
	t.Helper()
	r, f, err := openBamReader(file, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	defer r.Close()
	index := new(bam.Index)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := index.Add(record, r.LastChunk()); err != nil {
			t.Fatal(err)
		}
	}
	out, err := os.Create(file + ".bai")
	if err != nil {
		t.Fatal(err)
	}
	if err := bam.WriteIndex(out, index); err != nil {
		t.Fatal(err)
	}
	if _, ok := index.Unmapped(); !ok {
		if err := binary.Write(out, binary.LittleEndian, uint64(0)); err != nil {
			t.Fatal(err)
		}
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
}

// TestBamEmptyIndex handles a valid BAI with no reference sequences.
func TestBamEmptyIndex(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "empty.bam")
	f, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	header, err := sam.NewHeader(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	w, err := bam.NewWriter(f, header, 1)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	bamIndexFixture(t, file)
	for _, mode := range []string{"--idx-stat", "--idx-count"} {
		_, stderr, err := runCLI(t, "", "bam", mode, file)
		if err != nil || strings.Contains(stderr, "panic:") {
			t.Fatalf("empty BAI %s: %v %s", mode, err, stderr)
		}
	}
}

// TestBamIndexReferenceMismatch rejects a BAI belonging to another BAM.
func TestBamIndexReferenceMismatch(t *testing.T) {
	dir := t.TempDir()
	file := bamFixture(t, dir, "in.bam", true)
	ref1, err := sam.NewReference("chr1", "", "", 100, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	ref2, err := sam.NewReference("chr2", "", "", 100, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sam.NewHeader(nil, []*sam.Reference{ref1, ref2}); err != nil {
		t.Fatal(err)
	}
	index := new(bam.Index)
	record := &sam.Record{Ref: ref2, Cigar: sam.Cigar{sam.NewCigarOp(sam.CigarMatch, 4)}}
	if err := index.Add(record, bgzf.Chunk{}); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(file + ".bai")
	if err != nil {
		t.Fatal(err)
	}
	if err := bam.WriteIndex(f, index); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	_, stderr, err := runCLI(t, "", "bam", "--idx-count", file)
	if err == nil || !strings.Contains(stderr, "reference") || strings.Contains(stderr, "panic:") {
		t.Fatalf("mismatched index: %v %s", err, stderr)
	}
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

// TestBamAuxiliaryOutputAliases rejects aliases before overwriting BAM input.
func TestBamAuxiliaryOutputAliases(t *testing.T) {
	for _, flag := range []string{"--count", "--top-bam", "--img"} {
		for _, alias := range []string{"same", "symlink", "hardlink"} {
			t.Run(flag+"/"+alias, func(t *testing.T) {
				dir := t.TempDir()
				in := bamFixture(t, dir, "in.png", true)
				before, err := os.ReadFile(in)
				if err != nil {
					t.Fatal(err)
				}
				out := in
				if alias != "same" {
					out = filepath.Join(dir, "alias.png")
					if alias == "symlink" {
						err = os.Symlink(in, out)
					} else {
						err = os.Link(in, out)
					}
					if err != nil {
						t.Fatal(err)
					}
				}
				args := []string{"bam", in, "-f", "MapQual", "-Q", "--delay", "0", flag, out}
				if flag == "--top-bam" {
					args = append(args, "--top-size", "1")
				}
				_, stderr, err := runCLI(t, "", args...)
				after, readErr := os.ReadFile(in)
				if err == nil || readErr != nil || string(before) != string(after) || !strings.Contains(stderr, "cannot be the same") {
					t.Fatalf("auxiliary output alias accepted: err=%v preserved=%v stderr=%s", err, string(before) == string(after), stderr)
				}
			})
		}
	}
}

// TestBamTopDashAlias treats the top-BAM destination as a literal filename.
func TestBamTopDashAlias(t *testing.T) {
	dir := t.TempDir()
	in := bamFixture(t, dir, "-", true)
	before, err := os.ReadFile(in)
	if err != nil {
		t.Fatal(err)
	}
	_, stderr, err := auditCLI(t, dir, "", "bam", in, "-f", "MapQual", "-Q", "--top-bam", "-", "--top-size", "1")
	after, readErr := os.ReadFile(in)
	if err == nil || readErr != nil || string(before) != string(after) || !strings.Contains(stderr, "cannot be the same") {
		t.Fatalf("literal dash output alias: err=%v preserved=%v stderr=%s", err, string(before) == string(after), stderr)
	}
}

// TestBamOutputFailures checks errors deferred until buffered output finalization.
func TestBamOutputFailures(t *testing.T) {
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("/dev/full is unavailable")
	}
	dir := t.TempDir()
	in := bamFixture(t, dir, "in.bam", true)
	ref := auditInput(t, dir, "ref.fa", ">chr1\n"+strings.Repeat("ACGT", 25)+"\n")
	for _, args := range [][]string{
		{"--count", "/dev/full", "-Q", "--delay", "0"},
		{"-T", "Dump: {Fields: [Read]}", "-o", "/dev/full"},
		{"-T", "Dump: {Fields: [Read], Tsv: /dev/full}\nSink: true"},
		{"-T", "AccStats: {Tsv: /dev/full}\nSink: true"},
		{"-T", fmt.Sprintf("AlnContext: {Ref: %q, LeftShift: 0, RightShift: 1, RegexStart: A, Tsv: /dev/full}\nSink: true", ref)},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, stderr, err := runCLI(t, "", append([]string{"bam", in}, args...)...)
			if err == nil || !strings.Contains(stderr, "no space left") || strings.Contains(stderr, "panic:") {
				t.Fatalf("write failure was not reported: %v %s", err, stderr)
			}
		})
	}
}

// TestBamTopOrder preserves stable ordering and evicts the oldest tied record.
func TestBamTopOrder(t *testing.T) {
	for _, size := range []int{0, 1, 5, 100} {
		got := make(TopBuffer, size)
		want := make(TopBuffer, size)
		for i := range got {
			got[i].Value, want[i].Value = math.Inf(-1), math.Inf(-1)
		}
		for i := 0; i < 500; i++ {
			r := &sam.Record{Name: fmt.Sprint(i)}
			value := float64((i*19)%61 - 30)
			got = updateTop(r, value, got, "top.bam")
			want = append(want, topEntry{Record: r, Value: value})
			sort.SliceStable(want, func(i, j int) bool { return want[i].Value < want[j].Value })
			want = want[1:]
			if len(got) != len(want) {
				t.Fatalf("top size %d: got %d entries", size, len(got))
			}
			for j := range want {
				if got[j] != want[j] {
					t.Fatalf("top size %d, input %d, entry %d: got %+v, want %+v", size, i, j, got[j], want[j])
				}
			}
		}
	}
}

// BenchmarkBamTop includes initialization and updates over tied mapping qualities.
func BenchmarkBamTop(b *testing.B) {
	for _, size := range []int{100, 1000} {
		b.Run(fmt.Sprint(size), func(b *testing.B) {
			r := &sam.Record{Name: "r"}
			b.ReportAllocs()
			for n := 0; n < b.N; n++ {
				top := make(TopBuffer, size)
				for i := range top {
					top[i].Value = math.Inf(-1)
				}
				for i := 0; i < 10000; i++ {
					top = updateTop(r, float64((i*19)%61), top, "top.bam")
				}
			}
		})
	}
}
