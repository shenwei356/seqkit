package cmd

import (
	"os"
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
