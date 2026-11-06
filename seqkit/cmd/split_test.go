package cmd

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSplitTwoPassNamesAndCompression compares every grouping mode with single-pass output.
func TestSplitTwoPassNamesAndCompression(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "a.fa.gz")
	fh, err := os.Create(file)
	if err != nil {
		t.Fatal(err)
	}
	zw := gzip.NewWriter(fh)
	_, err = io.WriteString(zw, ">a first\nACGT\n>a second\nACCC\n>b third\nGGTT\n")
	if err != nil {
		t.Fatal(err)
	}
	if err = zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err = fh.Close(); err != nil {
		t.Fatal(err)
	}
	for i, flags := range [][]string{{"-s", "2"}, {"-p", "2"}, {"-i"}, {"-r", "1:2"}} {
		for _, extension := range []string{"", ".xz"} {
			var want map[string]string
			for _, two := range []bool{false, true} {
				outdir := filepath.Join(dir, fmt.Sprintf("%d-%s-%t", i, extension, two))
				args := append([]string{"split", file, "-O", outdir}, flags...)
				if two {
					args = append(args, "-2")
				}
				if extension != "" {
					args = append(args, "-e", extension)
				}
				_, stderr, err := runCLI(t, "", args...)
				if err != nil {
					t.Fatalf("split: %v %s", err, stderr)
				}
				entries, err := os.ReadDir(outdir)
				if err != nil {
					t.Fatal(err)
				}
				got := make(map[string]string)
				for _, entry := range entries {
					out, stderr, err := runCLI(t, "", "seq", "-w", "0", filepath.Join(outdir, entry.Name()))
					if err != nil {
						t.Fatalf("read compressed part: %v %s", err, stderr)
					}
					got[entry.Name()] = out
				}
				if !two {
					want = got
					continue
				}
				if len(got) != len(want) {
					t.Fatalf("different part counts: %v != %v", got, want)
				}
				for name, content := range want {
					if got[name] != content {
						t.Fatalf("different names/content in mode %v: %s: %q != %q; outputs: %v", flags, name, got[name], content, got)
					}
				}
			}
		}
	}
}

// TestSplitBalancedParts compares ordered, nonempty parts in both reading modes.
func TestSplitBalancedParts(t *testing.T) {
	dir := t.TempDir()
	var input strings.Builder
	for i := 0; i < 10; i++ {
		fmt.Fprintf(&input, ">s%d\nACGT\n", i)
	}
	file := auditInput(t, dir, "a.fa", input.String())
	for _, mode := range []string{"one", "two"} {
		for _, parts := range []int{1, 6, 15} {
			outdir := filepath.Join(dir, fmt.Sprintf("%s-%d", mode, parts))
			args := []string{"split", file, "-p", fmt.Sprint(parts), "-O", outdir}
			if mode == "two" {
				args = append(args, "-2")
			}
			_, stderr, err := runCLI(t, "", args...)
			if err != nil {
				t.Fatalf("split: %v %s", err, stderr)
			}
			paths, _ := filepath.Glob(filepath.Join(outdir, "*.fa"))
			if len(paths) != min(10, parts) {
				t.Fatalf("parts = %v", paths)
			}
			var joined strings.Builder
			minimum, maximum := 10, 0
			for _, path := range paths {
				data, _ := os.ReadFile(path)
				joined.Write(data)
				count := strings.Count(string(data), ">")
				minimum, maximum = min(minimum, count), max(maximum, count)
			}
			if joined.String() != input.String() || minimum < 1 || maximum-minimum > 1 {
				t.Fatalf("unbalanced or reordered: %d..%d %q", minimum, maximum, joined.String())
			}
		}
	}
}

// TestSplitPreservesUnrelatedFiles covers force, initial errors and lazy part creation.
func TestSplitPreservesUnrelatedFiles(t *testing.T) {
	for _, command := range []string{"split", "split2"} {
		dir := t.TempDir()
		file := auditInput(t, dir, "a.fa", ">a\nAC\n>b\nGG\n")
		outdir := filepath.Join(dir, "out")
		keep := auditInput(t, outdir, "keep.txt", "keep")
		extra := auditInput(t, outdir, "a.part_003.fa", "old extra")
		for _, mode := range [][]string{nil, {"-f"}} {
			_, stderr, err := runCLI(t, "", append([]string{command, file, "-s", "1", "-O", outdir}, mode...)...)
			if err != nil {
				t.Fatalf("split: %v %s", err, stderr)
			}
			for path, want := range map[string]string{keep: "keep", extra: "old extra"} {
				data, _ := os.ReadFile(path)
				if string(data) != want {
					t.Fatalf("unrelated output changed: %s %q", path, data)
				}
			}
		}
		empty := auditInput(t, dir, "empty.fa", "")
		_, stderr, err := runCLI(t, "", command, empty, "-s", "1", "-O", outdir, "-f")
		data, _ := os.ReadFile(keep)
		if err != nil || string(data) != "keep" {
			t.Fatalf("empty input cleared output: %v %s %q", err, stderr, data)
		}
	}
}

// TestSplitInvalidRegions rejects empty intersections but retains clipping.
func TestSplitInvalidRegions(t *testing.T) {
	dir := t.TempDir()
	file := auditInput(t, dir, "a.fa", ">s\nACGTA\n")
	for _, two := range []bool{false, true} {
		for _, region := range []string{"-7:-6", "1:-6", "2:-5", "4:2"} {
			args := []string{"split", file, "-r", region, "-O", filepath.Join(dir, fmt.Sprintf("%t-%s", two, region))}
			if two {
				args = append(args, "-2")
			}
			_, stderr, err := runCLI(t, "", args...)
			if err == nil || !strings.Contains(stderr, "not match sequence") {
				t.Fatalf("bad region accepted: %v %s", err, stderr)
			}
		}
	}
}

// TestSplit2PairedBoundaries ensures read2 follows read1 and unmatched tails fail.
func TestSplit2PairedBoundaries(t *testing.T) {
	dir := t.TempDir()
	a := auditInput(t, dir, "r1.fa", ">a\nAAAAAAAAAAAA\n>b\nCC\n>c\nGG\n")
	b := auditInput(t, dir, "r2.fq", "@a\nTT\n+\n12\n@b\nAA\n+\n34\n@c\nCCCCCCCCCCCC\n+\nABCDEFGHIJKL\n")
	outdir := filepath.Join(dir, "out")
	_, stderr, err := runCLI(t, "", "split2", "-1", a, "-2", b, "-l", "12", "-O", outdir)
	if err != nil {
		t.Fatalf("paired length: %v %s", err, stderr)
	}
	for path, want := range map[string]string{"r1.part_001.fa": ">a\nAAAAAAAAAAAA\n", "r2.part_001.fq": "@a\nTT\n+\n12\n", "r1.part_002.fa": ">b\nCC\n>c\nGG\n", "r2.part_002.fq": "@b\nAA\n+\n34\n@c\nCCCCCCCCCCCC\n+\nABCDEFGHIJKL\n"} {
		data, _ := os.ReadFile(filepath.Join(outdir, path))
		if string(data) != want {
			t.Fatalf("mate boundary %s: %q", path, data)
		}
	}
	b = auditInput(t, dir, "short.fa", ">different-ID\nTT\n")
	_, stderr, err = runCLI(t, "", "split2", "-1", a, "-2", b, "-s", "2", "-O", filepath.Join(dir, "short"))
	if err == nil || !strings.Contains(stderr, "record counts differ") {
		t.Fatalf("unpaired tail accepted: %v %s", err, stderr)
	}
	for _, path := range []string{"r1.part_001.fa", "short.part_001.fa"} {
		data, _ := os.ReadFile(filepath.Join(dir, "short", path))
		if strings.Count(string(data), ">") != 1 {
			t.Fatalf("unpaired tail written: %s %q", path, data)
		}
	}
}

// TestSplit2PreflightAndCollisions protects existing outputs from invalid options.
func TestSplit2PreflightAndCollisions(t *testing.T) {
	dir := t.TempDir()
	file := auditInput(t, dir, "ids.fa", ">dup AAAA\nAAAA\n>other\nCCCC\n>dup BBBB\nGGGG\n")
	for i, flags := range [][]string{{"-s", "1", "-p", "2"}, {"-p", "2", "-l", "10"}, {"-l", "NaN"}, {"-l", "Inf"}, {"-l", "0.5"}, {"-l", "99999999999999999999"}, {"-s", "1", "-P", "../x_"}, {"-s", "1", "--by-size-prefix", `C:\x`}} {
		outdir := filepath.Join(dir, fmt.Sprint(i))
		_, stderr, err := runCLI(t, "", append([]string{"split2", file, "-O", outdir}, flags...)...)
		if err == nil || strings.Contains(stderr, "panic:") {
			t.Fatalf("invalid split2 options accepted: %v %s", err, stderr)
		}
		if _, err := os.Stat(outdir); !os.IsNotExist(err) {
			t.Fatalf("output created before validation: %s", outdir)
		}
	}
	outdir := filepath.Join(dir, "duplicate")
	_, stderr, err := runCLI(t, "", "split2", file, "-s", "1", "-N", "-f", "-O", outdir)
	data, _ := os.ReadFile(filepath.Join(outdir, "dup.fa"))
	if err == nil || !strings.Contains(stderr, "cannot be the same") || string(data) != ">dup AAAA\nAAAA\n" {
		t.Fatalf("duplicate output overwritten: %q %v %s", data, err, stderr)
	}
}

// TestSequenceLengthUnits distinguishes biological lengths from byte buffers.
func TestSequenceLengthUnits(t *testing.T) {
	for value, want := range map[string]int64{"1K": 1000, "0.5K": 500, "1.5": 1, "1M": 1000000, "1G": 1000000000, "1T": 1000000000000, "9223372036854775807": 9223372036854775807} {
		got, err := parseSequenceLength(value)
		if err != nil || got != want {
			t.Fatalf("length %s = %d, %v", value, got, err)
		}
	}

	input := ">a\n" + strings.Repeat("A", 1000) + "\n>b\nC\n"
	out, stderr, err := runCLI(t, input, "head", "-l", "1K")
	if err != nil || strings.Count(out, ">") != 1 {
		t.Fatalf("head uses binary lengths: %q %v %s", out, err, stderr)
	}
	dir := t.TempDir()
	file := auditInput(t, dir, "a.fa", input)
	_, stderr, err = runCLI(t, "", "split2", file, "-l", "1K", "-O", filepath.Join(dir, "out"))
	paths, _ := filepath.Glob(filepath.Join(dir, "out", "*.fa"))
	if err != nil || len(paths) != 2 {
		t.Fatalf("split2 uses binary lengths: %v %v %s", paths, err, stderr)
	}
	if got, err := ParseByteSize("1K"); err != nil || got != 1024 {
		t.Fatalf("byte unit changed: %d %v", got, err)
	}
	for _, command := range []string{"head", "split2", "tab2fx", "merge-slides", "scat"} {
		out, stderr, err := runCLI(t, "", command, "-h")
		base := "base 1024"
		if command == "head" || command == "split2" {
			base = "base 1000"
		}
		if err != nil || !strings.Contains(out, base) {
			t.Fatalf("missing units for %s: %v %s", command, err, stderr)
		}
	}
}
