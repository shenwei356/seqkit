package cmd

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"os"
	"strings"
	"testing"
)

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
