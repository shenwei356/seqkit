package cmd

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestGrepImmediateOutputBeforeEOF leaves input open after one slow and many fast reads.
func TestGrepImmediateOutputBeforeEOF(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	args := []string{"-test.run=^TestCLIProcess$", "--", "grep", "-s", "-P", "-m", "1", "-p", "TTTTTTTT", "-I", "-w", "0", "-j", "4"}
	command := exec.CommandContext(ctx, os.Args[0], args...)
	command.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
	stdin, err := command.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		stdin.Close()
		if command.ProcessState == nil {
			command.Process.Kill()
			command.Wait()
		}
	}()
	const shortReads = 200
	headers := make(chan struct{}, shortReads+1)
	scanned := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		scanner.Buffer(make([]byte, 64<<10), 6<<20)
		for scanner.Scan() {
			if line := scanner.Bytes(); len(line) > 0 && line[0] == '>' {
				headers <- struct{}{}
			}
		}
		scanned <- scanner.Err()
	}()
	written := make(chan error, 1)
	go func() {
		var input strings.Builder
		input.WriteString(">slow\n" + strings.Repeat("A", 5_000_000) + "TTTTTTTT\n")
		for i := 0; i < shortReads; i++ {
			fmt.Fprintf(&input, ">r%d\nTTTTTTTT\n", i)
		}
		// This header terminates the last matching record without ending stdin.
		input.WriteString(">pending\nACGT\n")
		_, err := io.WriteString(stdin, input.String())
		written <- err
	}()
	for count := 0; count <= shortReads; count++ {
		select {
		case <-headers:
		case <-ctx.Done():
			t.Fatalf("only %d records were emitted before EOF", count)
		}
	}
	if err := <-written; err != nil {
		t.Fatal(err)
	}
	if err := stdin.Close(); err != nil {
		t.Fatal(err)
	}
	if err := <-scanned; err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err != nil {
		t.Fatalf("grep failed: %v %s", err, stderr.String())
	}
}

// TestGrepPerFileStrands keeps protein detection local across all search paths.
func TestGrepPerFileStrands(t *testing.T) {
	dir := t.TempDir()
	dna := auditInput(t, dir, "dna.fa", ">d1\nTTTC\n")
	protein := auditInput(t, dir, "protein.fa", ">p1\nEFILPQ\n")
	for _, mode := range [][]string{
		{"-p", "GAAA"}, {"-p", "GAAA", "-r"}, {"-p", "GAAA", "-d"},
		{"-p", "GAAA", "-m", "1"}, {"-p", "GAAA,CCCC", "-m", "1"},
	} {
		for _, files := range [][]string{{protein, dna}, {dna, protein}} {
			for _, positive := range []bool{false, true} {
				args := append([]string{"grep", "-s", "-j", "4"}, mode...)
				want := ">d1\nTTTC\n"
				if positive {
					args = append(args, "-P")
					want = ""
				}
				out, stderr, err := runCLI(t, "", append(args, files...)...)
				if err != nil || out != want {
					t.Fatalf("mode=%v positive=%v: got %q want %q: %v %s", mode, positive, out, want, err, stderr)
				}
			}
		}
	}
}

// TestGrepParallelFileFormats exercises FM-index workers and ordered mixed-format output.
func TestGrepParallelFileFormats(t *testing.T) {
	dir := t.TempDir()
	var fasta, fastq strings.Builder
	sequence := strings.Repeat("ACGT", 25)
	for i := 0; i < 100; i++ {
		fmt.Fprintf(&fasta, ">f%d\n%s\n", i, sequence)
		fmt.Fprintf(&fastq, "@q%d\n%s\n+\n%s\n", i, sequence, strings.Repeat("I", len(sequence)))
	}
	fa := auditInput(t, dir, "input.fa", fasta.String())
	fq := auditInput(t, dir, "input.fq", fastq.String())
	for _, pattern := range []string{"ACGT", strings.Repeat("ACGT", 20)} {
		for _, files := range [][]string{{fa, fq}, {fq, fa}} {
			args := []string{"grep", "-s", "-m", "1", "-p", pattern, "-j", "4"}
			var want string
			for _, file := range files {
				out, stderr, err := runCLI(t, "", append(append([]string(nil), args...), file)...)
				if err != nil {
					t.Fatalf("single file: %v %s", err, stderr)
				}
				want += out
			}
			out, stderr, err := runCLI(t, "", append(args, files...)...)
			if err != nil || out != want {
				t.Fatalf("parallel output differs from separate files: %v %s", err, stderr)
			}
		}
	}
}
