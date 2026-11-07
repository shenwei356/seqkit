package cmd

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestFa2fqFASTQWidth wraps both strands without changing default output.
func TestFa2fqFASTQWidth(t *testing.T) {
	for _, tc := range []struct{ reference, quality string }{{"ACGT", "1234"}, {"ACT", "876"}} {
		ref := auditInput(t, t.TempDir(), "reference.fa", ">r\n"+tc.reference+"\n")
		for _, width := range []string{"", "0", "2", "20"} {
			args := []string{"fa2fq", "-f", ref}
			if width != "" {
				args = append(args, "-w", width)
			}
			sequence, quality := tc.reference, tc.quality
			if width == "2" {
				sequence = sequence[:2] + "\n" + sequence[2:]
				quality = quality[:2] + "\n" + quality[2:]
			}
			want := "@r\n" + sequence + "\n+\n" + quality + "\n"
			out, stderr, err := runCLI(t, "@r\nACGTCAGT\n+\n12345678\n", args...)
			if err != nil || out != want {
				t.Fatalf("width=%q output=%q want=%q: %v %s", width, out, want, err, stderr)
			}
		}
	}
}

// TestFa2fqDamagedReference rejects a partial map before opening the output.
func TestFa2fqDamagedReference(t *testing.T) {
	var input strings.Builder
	for i := 0; i < 15; i++ {
		fmt.Fprintf(&input, ">s%d\nAC\n", i)
	}
	var buffer bytes.Buffer
	writer := gzip.NewWriter(&buffer)
	if _, err := writer.Write([]byte(input.String())); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	data := buffer.Bytes()
	data[len(data)-8] ^= 1
	dir := t.TempDir()
	reference := auditInput(t, dir, "reference.fa.gz", string(data))
	output := auditInput(t, dir, "output.fq", "keep\n")
	_, stderr, err := runCLI(t, "@s0\nACGT\n+\n1234\n", "fa2fq", "-f", reference, "-o", output)
	after, _ := os.ReadFile(output)
	if err == nil || !strings.Contains(stderr, reference) || string(after) != "keep\n" {
		t.Fatalf("damaged reference accepted or output changed: %v %s", err, stderr)
	}
}

// TestFa2fqStdinConsumers validates auxiliary stdin before truncating output.
func TestFa2fqStdinConsumers(t *testing.T) {
	output := auditInput(t, t.TempDir(), "output.fq", "keep\n")
	_, stderr, err := runCLI(t, ">s\nAC\n", "fa2fq", "-f", "-", "-o", output)
	after, _ := os.ReadFile(output)
	if err == nil || !strings.Contains(stderr, "stdin may only") || string(after) != "keep\n" {
		t.Fatalf("duplicate stdin was not rejected safely: %v %s", err, stderr)
	}
}

// TestFa2fqStrands preserves trimmed sequence and reverse-complement quality order.
func TestFa2fqStrands(t *testing.T) {
	for _, tc := range []struct{ sequence, want, qual string }{{"AC", "AC", "12"}, {"ACT", "ACT", "876"}} {
		reference := auditInput(t, t.TempDir(), "reference.fa", ">s\n"+tc.sequence+"\n")
		out, stderr, err := runCLI(t, "@s\nACGTCAGT\n+\n12345678\n", "fa2fq", "-f", reference)
		if err != nil || out != "@s\n"+tc.want+"\n+\n"+tc.qual+"\n" {
			t.Fatalf("trimmed FASTQ changed: %v %s", err, stderr)
		}
	}
}

// TestFa2fqProteinStrands never invents matches by reversing proteins or unrestricted input.
func TestFa2fqProteinStrands(t *testing.T) {
	for _, alphabet := range []string{"protein", "unlimit"} {
		for _, referenceSequence := range []string{"MELK", "KLEM"} {
			reference := auditInput(t, t.TempDir(), "reference.fa", ">s\n"+referenceSequence+"\n")
			out, stderr, err := runCLI(t, "@s\nMELK\n+\n1234\n", "fa2fq", "-t", alphabet, "-f", reference)
			want := ""
			if referenceSequence == "MELK" {
				want = "@s\nMELK\n+\n1234\n"
			}
			if err != nil || out != want {
				t.Fatalf("%s reference %s: got %q, want %q: %v %s", alphabet, referenceSequence, out, want, err, stderr)
			}
		}
	}
}

// TestFa2fqLeadingEmptyRecords keeps reverse matches and reverses quality scores.
func TestFa2fqLeadingEmptyRecords(t *testing.T) {
	ref := auditInput(t, t.TempDir(), "query.fa", ">r\nGAAA\n")
	assertLeadingEmptyRecords(t, "@r\nTTTC\n+\nabcd\n", "fa2fq", "-f", ref)
}
