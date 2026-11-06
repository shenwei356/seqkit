package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestSplitOutputRegistrySkippedChecks skips input metadata but keeps lexical
// input protection and filesystem identity checks between generated outputs.
func TestSplitOutputRegistrySkippedChecks(t *testing.T) {
	dir := t.TempDir()
	input := auditInput(t, dir, "input.fa", ">r1\nACGT\n")
	loop := filepath.Join(dir, "loop.fa")
	if err := os.Symlink("loop.fa", loop); err != nil {
		t.Skipf("cannot create symlink: %v", err)
	}
	output := filepath.Join(dir, "output.fa")
	registry := splitOutputRegistry{inputs: []string{loop, input}}
	if w, err := registry.open(output); err == nil {
		w.Close()
		t.Fatal("normal checks accepted a symlink loop")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatalf("output opened before validation: %v", err)
	}
	registry.skipInputFileCheck = true
	w, err := registry.open(output)
	if err != nil {
		t.Fatalf("skipped checks probed unused input: %v", err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(dir, "output-alias.fa")
	if err := os.Symlink(output, alias); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{input, alias} {
		if w, err := registry.open(path); err == nil {
			w.Close()
			t.Fatalf("accepted conflicting output: %s", path)
		} else if !strings.Contains(err.Error(), "cannot be the same") {
			t.Fatalf("unexpected collision error: %v", err)
		}
	}
	got, err := os.ReadFile(input)
	if err != nil || string(got) != ">r1\nACGT\n" {
		t.Fatalf("input changed: read=%v", err)
	}
}

// TestSplitRedirectedStdin protects the source file even after a buffered reader
// has consumed and closed stdin while discovering generated output names.
func TestSplitRedirectedStdin(t *testing.T) {
	for _, name := range []string{"split", "split2"} {
		for _, skip := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/skip=%v", name, skip), func(t *testing.T) {
				dir := t.TempDir()
				const data = ">r1\nACGT\n"
				input := auditInput(t, dir, "stdin.part_001.fasta", data)
				args := []string{name, "-s", "1", "-O", dir}
				if name == "split2" {
					mate := auditInput(t, dir, "mate.fa", data)
					args = append(args, "-1", "-", "-2", mate)
				}
				if skip {
					args = append(args, "--skip-file-check")
				}
				fh, err := os.Open(input)
				if err != nil {
					t.Fatal(err)
				}
				defer fh.Close()
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				command := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)...)
				command.Env = append(os.Environ(), "SEQKIT_CLI_TEST_HELPER=1")
				command.Stdin = fh
				message, err := command.CombinedOutput()
				got, readErr := os.ReadFile(input)
				if err == nil || !strings.Contains(string(message), "cannot be the same") || readErr != nil || string(got) != data {
					t.Fatalf("redirected input not protected: err=%v message=%s read=%v", err, message, readErr)
				}
			})
		}
	}
}
