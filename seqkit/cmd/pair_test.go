package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestPairCompressedFilenames inserts paired/unpaired markers before FASTQ extensions.
func TestPairCompressedFilenames(t *testing.T) {
	const paired = "@r\nAC\n+\nIJ\n"
	const unpaired = "@s\nGT\n+\nKL\n"
	for _, extension := range []string{".gz", ".xz", ".zst", ".bz2", ".lz4"} {
		t.Run(extension, func(t *testing.T) {
			dir := t.TempDir()
			first := auditInput(t, dir, "r1.fq"+strings.ToUpper(extension), string(compressedInput(t, extension, paired+unpaired)))
			second := auditInput(t, dir, "r2.fq"+extension, string(compressedInput(t, extension, paired)))
			if _, stderr, err := runCLI(t, "", "pair", "-1", first, "-2", second, "-u"); err != nil {
				t.Fatalf("pair: %v %s", err, stderr)
			}
			for name, want := range map[string]string{"r1.paired.fq": paired, "r2.paired.fq": paired, "r1.unpaired.fq": unpaired} {
				out, stderr, err := runCLI(t, "", "seq", filepath.Join(dir, name+extension))
				if err != nil || out != want {
					t.Errorf("incorrect compressed output name or content: %v %s", err, stderr)
				}
			}
		})
	}
}

// TestPairAlignedDuplicatesRespectPending keeps older occurrences ahead of direct pairs.
func TestPairAlignedDuplicatesRespectPending(t *testing.T) {
	for _, swap := range []bool{false, true} {
		dir := t.TempDir()
		a := auditInput(t, dir, "a.fa", ">a\nAA\n>a\nCC\n>a\nGG\n")
		b := auditInput(t, dir, "b.fq", "@b\nTT\n+\nII\n@a\nAC\n+\nJK\n@a\nGT\n+\nLM\n")
		first, second := a, b
		if swap {
			first, second = b, a
		}
		outdir := filepath.Join(dir, "out")
		_, stderr, err := runCLI(t, "", "pair", "-1", first, "-2", second, "-u", "-O", outdir)
		if err != nil {
			t.Fatalf("pairing: %v %s", err, stderr)
		}
		for name, want := range map[string]string{
			"a.fa": ">a\nAA\n>a\nCC\n", "a.unpaired.fa": ">a\nGG\n",
			"b.fq": "@a\nAC\n+\nJK\n@a\nGT\n+\nLM\n", "b.unpaired.fq": "@b\nTT\n+\nII\n",
		} {
			data, err := os.ReadFile(filepath.Join(outdir, name))
			if err != nil || string(data) != want {
				t.Fatalf("swap=%v %s: %q %v", swap, name, data, err)
			}
		}
	}
}

// TestPairDuplicateQueues keeps occurrence order and each mate's input format.
func TestPairDuplicateQueues(t *testing.T) {
	dir := t.TempDir()
	a := auditInput(t, dir, "r1.fa", ">a\nAAAA\n>a\nCCCC\n>z\nGGGG\n")
	b := auditInput(t, dir, "r2.fq", "@a\nACAC\n+\n5678\n@z\nTTTT\n+\n1234\n@a\nCACA\n+\nABCD\n@a\nGAGA\n+\nEFGH\n")
	outdir := filepath.Join(dir, "out")
	keep := auditInput(t, outdir, "keep.txt", "keep")
	_, stderr, err := runCLI(t, "", "pair", "-1", a, "-2", b, "-u", "-f", "-O", outdir)
	if err != nil || !strings.Contains(stderr, "3 paired-end") || !strings.Contains(stderr, "1 unpaired") {
		t.Fatalf("pairing: %v %s", err, stderr)
	}
	for path, want := range map[string]string{"r1.fa": ">a\nAAAA\n>z\nGGGG\n>a\nCCCC\n", "r2.fq": "@a\nACAC\n+\n5678\n@z\nTTTT\n+\n1234\n@a\nCACA\n+\nABCD\n", "r2.unpaired.fq": "@a\nGAGA\n+\nEFGH\n"} {
		data, _ := os.ReadFile(filepath.Join(outdir, path))
		if string(data) != want {
			t.Fatalf("%s: %q", path, data)
		}
	}
	data, _ := os.ReadFile(keep)
	if string(data) != "keep" {
		t.Fatal("unrelated file deleted")
	}
}

// TestPairInitialReadProtection preserves old targets when either mate is empty.
func TestPairInitialReadProtection(t *testing.T) {
	dir := t.TempDir()
	a := auditInput(t, dir, "r1.fa", ">a\nAC\n")
	b := auditInput(t, dir, "r2.fa", "")
	outdir := filepath.Join(dir, "out")
	old := auditInput(t, outdir, "r1.fa", "old")
	_, stderr, err := runCLI(t, "", "pair", "-1", a, "-2", b, "-f", "-O", outdir)
	data, _ := os.ReadFile(old)
	if err == nil || string(data) != "old" {
		t.Fatalf("initial failure changed target: %q %v %s", data, err, stderr)
	}
}

// TestPairUnequalInputLengths keeps pending mates matchable after either input ends.
func TestPairUnequalInputLengths(t *testing.T) {
	for _, swap := range []bool{false, true} {
		for _, save := range []bool{false, true} {
			dir := t.TempDir()
			a := auditInput(t, dir, "a.fa", ">a\nAA\n>a\nCC\n>b\nGG\n")
			b := auditInput(t, dir, "b.fq", "@x\nAA\n+\nII\n@b\nTT\n+\nJK\n@a\nAC\n+\nLM\n@a\nGT\n+\nNO\n@z\nCC\n+\nPQ\n")
			first, second := a, b
			if swap {
				first, second = b, a
			}
			outdir := filepath.Join(dir, "out")
			args := []string{"pair", "-1", first, "-2", second, "-O", outdir}
			if save {
				args = append(args, "-u")
			}
			_, stderr, err := runCLI(t, "", args...)
			if err != nil {
				t.Fatalf("swap=%v save=%v: %v %s", swap, save, err, stderr)
			}
			wantPaired := map[string]string{
				"a.fa": ">b\nGG\n>a\nAA\n>a\nCC\n",
				"b.fq": "@b\nTT\n+\nJK\n@a\nAC\n+\nLM\n@a\nGT\n+\nNO\n",
			}
			if swap {
				wantPaired["a.fa"] = ">a\nAA\n>b\nGG\n>a\nCC\n"
				wantPaired["b.fq"] = "@a\nAC\n+\nLM\n@b\nTT\n+\nJK\n@a\nGT\n+\nNO\n"
			}
			for name, want := range wantPaired {
				data, err := os.ReadFile(filepath.Join(outdir, name))
				if err != nil || string(data) != want {
					t.Fatalf("%s: %q, want %q: %v", name, data, want, err)
				}
			}
			data, err := os.ReadFile(filepath.Join(outdir, "b.unpaired.fq"))
			if save {
				x, z := "@x\nAA\n+\nII\n", "@z\nCC\n+\nPQ\n"
				if err != nil || (string(data) != x+z && string(data) != z+x) {
					t.Fatalf("unpaired reads lost: %q %v", data, err)
				}
			} else if !os.IsNotExist(err) {
				t.Fatalf("unexpected unpaired output: %v", err)
			}
		}
	}
}
