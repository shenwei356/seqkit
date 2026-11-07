package cmd

import (
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestSortTwoPassLengthOrder retains empty records and ID tie-breaks in both directions.
func TestSortTwoPassLengthOrder(t *testing.T) {
	file := auditInput(t, t.TempDir(), "input.fa", ">c\nACG\n>a\nT\n>b\nGTA\n>e\n\n")
	for _, reverse := range []bool{false, true} {
		args := []string{"sort", "-2", "-l", file}
		want := ">e\n\n>a\nT\n>b\nGTA\n>c\nACG\n"
		if reverse {
			args = append(args, "-r")
			want = ">b\nGTA\n>c\nACG\n>a\nT\n>e\n\n"
		}
		out, stderr, err := runCLI(t, "", args...)
		if err != nil || out != want {
			t.Fatalf("indexed length order changed: reverse=%v: %v %s", reverse, err, stderr)
		}
	}
}

// TestSortTwoPassHeaderTabs uses the same full-header keys as index creation.
func TestSortTwoPassHeaderTabs(t *testing.T) {
	for _, separator := range []string{"\t", "\t\t"} {
		for _, extension := range []string{"", ".gz"} {
			dir := t.TempDir()
			input := ">a" + separator + "x\nTGCA\n>b" + separator + "x\nACGT\n"
			if extension != "" {
				input = string(compressedInput(t, extension, input))
			}
			file := auditInput(t, dir, "input.fa"+extension, input)
			// Plain input reuses its index on the second invocation.
			for attempt := 0; attempt < 2; attempt++ {
				out, _, err := runCLI(t, "", "sort", "-2", "-s", file)
				if err != nil {
					t.Fatalf("two-pass sequence sort failed: extension=%q attempt=%d: %v", extension, attempt, err)
				}
				table, stderr, err := runCLI(t, out, "fx2tab", "-i", "-Q")
				if err != nil || table != "b\tACGT\na\tTGCA\n" {
					t.Fatalf("sequence order or content changed: %v %s", err, stderr)
				}
			}
		}
	}
}

// TestSortDuplicatesAcrossFiles keeps duplicate validation ahead of output creation.
func TestSortDuplicatesAcrossFiles(t *testing.T) {
	for _, flags := range [][]string{nil, {"-n"}, {"-s"}, {"-l"}, {"-b"}, {"--id-regexp", "^(r[0-9]+)"}} {
		t.Run(strings.Join(flags, " "), func(t *testing.T) {
			dir := t.TempDir()
			first := auditInput(t, dir, "first.fa", ">r1_a\nACGT\n")
			id := "r1_a"
			if len(flags) == 2 {
				id = "r1_b"
			}
			second := auditInput(t, dir, "second.fa", ">"+id+"\nTGCA\n")
			output := auditInput(t, dir, "out.fa", "keep\n")
			args := append([]string{"sort", first, second, "-o", output}, flags...)
			_, stderr, err := runCLI(t, "", args...)
			if err == nil || !strings.Contains(stderr, "duplicated sequences found") {
				t.Fatalf("duplicate not rejected: %v %s", err, stderr)
			}
			data, err := os.ReadFile(output)
			if err != nil || string(data) != "keep\n" {
				t.Fatalf("output changed before validation: %v", err)
			}
		})
	}
}

// TestSortEmptyOutput keeps empty two-pass results consistent with one-pass output.
func TestSortEmptyOutput(t *testing.T) {
	assertEmptyOrderingOutput(t, "sort")
}

// assertEmptyOrderingOutput checks output creation, replacement and compression
// with newly built/reused empty indexes and empty compressed inputs.
func assertEmptyOrderingOutput(t *testing.T, command string) {
	t.Helper()
	for _, mode := range []string{"one-pass", "new-index", "reused-index", "compressed-input"} {
		for _, suffix := range []string{".fa", ".fa.gz"} {
			for _, existing := range []bool{false, true} {
				dir := t.TempDir()
				input := auditInput(t, dir, "empty.fa", "")
				if mode == "compressed-input" {
					input = auditInput(t, dir, "empty.fa.gz", string(compressedInput(t, ".gz", "")))
				}
				output := filepath.Join(dir, "output"+suffix)
				if existing {
					auditInput(t, dir, filepath.Base(output), "stale output\n")
				}
				args := []string{command, input, "-o", output}
				if command == "shuffle" {
					args = append(args, "--tmp-dir", dir)
				}
				if mode != "one-pass" {
					args = append(args, "-2")
				}
				if mode == "reused-index" {
					auditInput(t, dir, "empty.fa.seqkit.fai", "")
				}
				_, stderr, err := runCLI(t, "", args...)
				if err != nil {
					t.Fatalf("%s %s: %v %s", command, mode, err, stderr)
				}
				data, err := os.ReadFile(output)
				if err != nil {
					t.Errorf("%s %s existing=%v: output not created: %v", command, mode, existing, err)
					continue
				}
				if suffix == ".fa.gz" {
					gz, err := gzip.NewReader(bytes.NewReader(data))
					if err != nil {
						t.Errorf("%s %s: invalid gzip output: %v", command, mode, err)
						continue
					}
					data, err = io.ReadAll(gz)
					gz.Close()
					if err != nil {
						t.Fatal(err)
					}
				}
				if len(data) != 0 {
					t.Errorf("%s %s: stale output retained", command, mode)
				}
				leftovers, err := filepath.Glob(filepath.Join(dir, "seqkit-*"))
				if err != nil || len(leftovers) != 0 {
					t.Fatalf("temporary files remain: %v %v", leftovers, err)
				}
			}
		}
	}
}

// TestSortNaturalLargeNumbers compares digit runs without machine-integer limits
// and retains input order for numerically equal spellings in both directions.
func TestSortNaturalLargeNumbers(t *testing.T) {
	ids := []string{"r100000000000000000000", "r2", "r99999999999999999999", "r01", "r1", "r1x10", "r01x2", "r00000000000000000000000000000000000000001"}
	ascending := []int{3, 4, 7, 6, 5, 1, 2, 0}
	descending := []int{0, 2, 1, 5, 6, 3, 4, 7}
	var input strings.Builder
	for _, id := range ids {
		input.WriteString(">" + id + "\nAC\n")
	}
	file := auditInput(t, t.TempDir(), "input.fa", input.String())
	for _, twoPass := range []bool{false, true} {
		for _, byName := range []bool{false, true} {
			for _, reverse := range []bool{false, true} {
				args := []string{"sort", "-N", file}
				if twoPass {
					args = append(args, "-2")
				}
				if byName {
					args = append(args, "-n")
				}
				order := ascending
				if reverse {
					args = append(args, "-r")
					order = descending
				}
				var want strings.Builder
				for _, i := range order {
					want.WriteString(">" + ids[i] + "\nAC\n")
				}
				out, stderr, err := runCLI(t, "", args...)
				if err != nil || out != want.String() {
					t.Fatalf("twoPass=%v byName=%v reverse=%v: incorrect natural order: %q; %v %s", twoPass, byName, reverse, out, err, stderr)
				}
			}
		}
	}
}

// TestNaturalNameLessOrdering checks strict ordering and numeric equivalence,
// including text boundaries, UTF-8 names, leading zeros, and long digit runs.
func TestNaturalNameLessOrdering(t *testing.T) {
	groups := [][]string{{""}, {"-"}, {"0", "00"}, {"1", "0001"}, {"9"}, {"10"},
		{"99999999999999999999"}, {"100000000000000000000"}, {"A"}, {"a"},
		{"a0"}, {"a1", "a01"}, {"a1b2"}, {"a01b10"}, {"a2"}, {"a-"}, {"b"}, {"β2"}, {"β10"}}
	for i, as := range groups {
		for j, bs := range groups {
			for _, a := range as {
				for _, b := range bs {
					if got := naturalNameLess(a, b); got != (i < j) {
						t.Fatalf("%q < %q = %v, want %v", a, b, got, i < j)
					}
				}
			}
		}
	}
}

// TestSortNaturalFoldedNames retains original identities and stable numeric ties
// when name keys are case-folded before comparing them.
func TestSortNaturalFoldedNames(t *testing.T) {
	keys := []sequenceSortKey{{Key: "R2", Name: "r2"}, {Key: "r01", Name: "r01"}, {Key: "R1", Name: "r1"}, {Key: "r10", Name: "r10"}}
	for _, reverse := range []bool{false, true} {
		records := append([]sequenceSortKey(nil), keys...)
		sortBySequenceName(records, true, reverse)
		got := make([]string, len(records))
		for i, record := range records {
			got[i] = record.Key
		}
		want := []string{"r01", "R1", "R2", "r10"}
		if reverse {
			want = []string{"r10", "R2", "r01", "R1"}
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("reverse=%v: %v, want %v", reverse, got, want)
		}
	}
}
