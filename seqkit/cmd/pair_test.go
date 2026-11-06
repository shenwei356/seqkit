package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
