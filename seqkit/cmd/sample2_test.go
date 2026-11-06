package cmd

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestSample2MultipleInputs matches separate runs, including per-file format defaults.
func TestSample2MultipleInputs(t *testing.T) {
	dir := t.TempDir()
	var fasta, fastq strings.Builder
	fastq.WriteString("@empty\n\n+\n\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&fasta, ">f%d\n%s\n", i, strings.Repeat("ACGT", 20))
		fmt.Fprintf(&fastq, "@q%d\n%s\n+\n%s\n", i, strings.Repeat("ACGT", 20), strings.Repeat("I", 80))
	}
	a := auditInput(t, dir, "a.fa", fasta.String())
	b := auditInput(t, dir, "b.fq", fastq.String())
	for _, files := range [][]string{{a, b}, {b, a}} {
		list := auditInput(t, dir, "inputs", files[1]+"\n")
		for _, proportion := range []string{"0.7", "1"} {
			for _, width := range [][]string{nil, {"-w", "10"}} {
				args := append([]string{"sample2", "-p", proportion, "-s", "42"}, width...)
				var want string
				for _, file := range files {
					out, stderr, err := runCLI(t, "", append(append([]string(nil), args...), file)...)
					if err != nil || out == "" {
						t.Fatalf("single input: %v %s", err, stderr)
					}
					want += out
				}
				out, stderr, err := runCLI(t, "", append(args, files[0], "-X", list)...)
				if err != nil || out != want {
					t.Fatalf("multi-file seed/format mismatch: %v %s", err, stderr)
				}
			}
		}
	}
	out, stderr, err := runCLI(t, ">stdin\nAC\n", "sample2", "-p", "1", a, "-")
	if err != nil || !strings.HasSuffix(out, ">stdin\nAC\n") {
		t.Fatalf("stdin with a file: %v %s", err, stderr)
	}
}

// TestSample2SingleInputModes rejects extra inputs before truncating an existing output.
func TestSample2SingleInputModes(t *testing.T) {
	dir := t.TempDir()
	a := auditInput(t, dir, "a.fa", ">a\nAC\n")
	b := auditInput(t, dir, "b.fa", ">b\nGT\n")
	list := auditInput(t, dir, "inputs", b+"\n")
	output := auditInput(t, dir, "keep", "keep\n")
	for _, mode := range [][]string{{"-n", "1"}, {"-n", "1", "-2"}, {"-p", "0.5", "-2"}} {
		for _, inputs := range [][]string{{a, b}, {a, "-X", list}} {
			args := append([]string{"sample2", "-o", output}, mode...)
			_, stderr, err := runCLI(t, "", append(args, inputs...)...)
			got, _ := os.ReadFile(output)
			if err == nil || !strings.Contains(stderr, "only one file") || string(got) != "keep\n" {
				t.Fatalf("mode=%v accepted extra input: %v %s", mode, err, stderr)
			}
		}
	}
}

// TestSample2TimeSeedPerInvocation reuses one time-based seed for paired inputs.
func TestSample2TimeSeedPerInvocation(t *testing.T) {
	var input strings.Builder
	for i := 0; i < 256; i++ {
		fmt.Fprintf(&input, ">r%d\nACGT\n", i)
	}
	file := auditInput(t, t.TempDir(), "input.fa", input.String())
	out, stderr, err := runCLI(t, "", "sample2", "-p", "0.7", "-r", file, file)
	if err != nil || out == "" || out[:len(out)/2] != out[len(out)/2:] {
		t.Fatalf("time seed was not reused: %v %s", err, stderr)
	}
}

// TestSample2LaterInputErrors reports malformed records in any input file.
func TestSample2LaterInputErrors(t *testing.T) {
	dir := t.TempDir()
	good := auditInput(t, dir, "good.fa", ">r1\nAC\n")
	bad := auditInput(t, dir, "bad.fq", "@r2\nACGT\n+\nII\n")
	_, stderr, err := runCLI(t, "", "sample2", "-p", "1", good, bad)
	if err == nil || !strings.Contains(stderr, "[ERRO]") {
		t.Fatalf("later input error lost: %v %s", err, stderr)
	}
}
